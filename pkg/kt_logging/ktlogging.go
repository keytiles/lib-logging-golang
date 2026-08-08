package kt_logging

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogLevel uint8

const (
	NoneLevel    LogLevel = 0
	ErrorLevel   LogLevel = 1
	WarningLevel LogLevel = 2
	InfoLevel    LogLevel = 3
	DebugLevel   LogLevel = 4

	_ROOT_NAME            string = "root"
	_DEFAULT_HANDLER_NAME string = "stdout_json"

	// DefaultLoggerMaxCacheSize is used when loggerMaxCacheSize is omitted or 0 in config.
	// It limits how many on-demand (non-config) logger names stay cached in memory.
	DefaultLoggerMaxCacheSize = 3000
)

type loggerCacheItem struct {
	logger       *Logger
	creationTime time.Time
}

var loggers map[string]*Logger

// Contains the names of the requested loggers in creation order - excluding those ones which were created by the config statically.
// Index 0 = oldest.
// When we reach loggerCacheMax we simply take the item 0 (must be evicted from loggers map) then shift the array and append new one to the end.
// Config / init loggers are never added here, so they are never evicted.
var loggersCreationOrder []string

// Max number of on-demand (non-config) loggers kept in the registry.
var loggerCacheMax = DefaultLoggerMaxCacheSize

// we need locks to avoid concurrent map operations
var loggersLock = new(sync.RWMutex)

// Snapshot of labels attached to every log event. Stored via atomic.Value for lock-free reads on the log path.
type globalLabelsSnapshot struct {
	labels    []Label
	zapFields []zap.Field
}

// Holds *globalLabelsSnapshot (or nil before the first SetGlobalLabels).
var globalLabelsState atomic.Value

// Returns a copy of the current global labels (snapshot). Mutating the result does not affect logging;
// call SetGlobalLabels to publish a new set.
func GetGlobalLabels() []Label {
	snap := loadGlobalLabelsSnapshot()
	if snap == nil || len(snap.labels) == 0 {
		return nil
	}
	return append([]Label(nil), snap.labels...)
}

// Replaces the global labels attached to every log event. The provided slice is copied;
// later mutation of the caller's slice does not affect logging.
func SetGlobalLabels(labels []Label) {
	var copied []Label
	if len(labels) > 0 {
		copied = append([]Label(nil), labels...)
	}
	globalLabelsState.Store(&globalLabelsSnapshot{
		labels:    copied,
		zapFields: toZapFieldArray(copied),
	})
}

// Loads the current global-labels snapshot, or nil if none was set yet.
func loadGlobalLabelsSnapshot() *globalLabelsSnapshot {
	v := globalLabelsState.Load()
	if v == nil {
		return nil
	}
	return v.(*globalLabelsSnapshot)
}

// Resolves loggerMaxCacheSize from config: 0 → default; negative → error.
func resolveLoggerMaxCacheSize(configured int) (int, error) {
	if configured < 0 {
		return 0, fmt.Errorf("loggerMaxCacheSize must be >= 0 (0 means default %d)", DefaultLoggerMaxCacheSize)
	}
	if configured == 0 {
		return DefaultLoggerMaxCacheSize, nil
	}
	return configured, nil
}

// Replaces the in-memory registry. Caller must hold loggersLock.
// Configured names stay in loggers only (not in loggersCreationOrder); on-demand creation-order cache is cleared.
func resetRegistry(configured map[string]*Logger, cacheMax int) {
	loggers = configured

	// allocate enough size for the ordered names - but make it empty
	loggersCreationOrder = make([]string, 0, cacheMax)

	loggerCacheMax = cacheMax
}

// Registers name as a new on-demand cache entry, evicting the oldest (FIFO) if at capacity.
// Caller must hold loggersLock. Name must already be present in loggers.
// Only on-demand names belong here — config loggers are installed via resetRegistry and never registered.
func registerOnDemand(name string) {
	// already tracked — cache hit does not reorder (FIFO by creation)
	for _, existing := range loggersCreationOrder {
		if existing == name {
			return
		}
	}
	if len(loggersCreationOrder) >= loggerCacheMax {
		evictName := loggersCreationOrder[0]
		delete(loggers, evictName)
		loggersCreationOrder = append(loggersCreationOrder[1:], name)
		return
	}
	loggersCreationOrder = append(loggersCreationOrder, name)
}

// Initializing the logging from the .yaml or .json config file available on the given path
func InitFromConfig(cfgPath string) error {
	// read the config file
	configModel, err := parseFromJsonOrYaml(cfgPath)
	if err != nil {
		return err
	}

	cacheMax, err := resolveLoggerMaxCacheSize(configModel.LoggerMaxCacheSize)
	if err != nil {
		return err
	}
	// create and initialize loggers based on that
	configuredLoggers, err := initLoggersFromConfig(configModel)
	if err != nil {
		return err
	}

	// swap registry under lock so concurrent GetLogger cannot race
	loggersLock.Lock()
	resetRegistry(configuredLoggers, cacheMax)
	loggersLock.Unlock()

	return nil
}

// returns a Logger with the given name - if does not exist then a new instance is created with this name and registered
// note: Loggers are hierarchical
func GetLogger(loggerName string) *Logger {
	// fast path: cache hit under read lock
	loggersLock.RLock()
	if loggers != nil {
		if ctxLogger := loggers[loggerName]; ctxLogger != nil {
			loggersLock.RUnlock()
			return ctxLogger
		}
	}
	loggersLock.RUnlock()

	// slow path: create / lazy-init under write lock (double-checked inside getLogger)
	loggersLock.Lock()
	ctxLogger := getLogger(loggerName)
	loggersLock.Unlock()
	return ctxLogger
}

// just a shortcut to the GetLogger method - for builder style readability stuff
func With(loggerName string) *Logger {
	return GetLogger(loggerName)
}

// Resolves or creates a named logger. NOT THREAD SAFE — caller must hold loggersLock.
// Never panics on the lazy-default path.
func getLogger(loggerName string) *Logger {
	if loggers == nil {
		// Prefer normal default config; on failure install a hard-coded stdout JSON root (no panic).
		configured, err := initLoggersFromConfig(getDefaultLoggerConfig())
		if err == nil {
			resetRegistry(configured, DefaultLoggerMaxCacheSize)
		} else {
			resetRegistry(map[string]*Logger{
				_ROOT_NAME: newHardFallbackRootLogger(),
			}, DefaultLoggerMaxCacheSize)
		}
	}
	ctxLogger := loggers[loggerName]
	if ctxLogger == nil {
		// let's split by '.' characters
		dotIdx := strings.LastIndex(loggerName, ".")
		var parentLogger *Logger
		if dotIdx > 0 {
			// ok it seems to be a structured name... let's retry with logger by cutting down the last part
			parentLogger = getLogger(loggerName[0:dotIdx])
		} else {
			// there are no dots - we return the "root" logger
			parentLogger = getRootLogger()
		}

		// let's register a copy of this returned parentLogger so next time we find this faster
		loggerCopy := parentLogger.clone()
		// let's rename the clone
		loggerCopy.name = loggerName
		// register the clone as on-demand (may evict oldest extras)
		loggers[loggerName] = loggerCopy
		registerOnDemand(loggerName)
		ctxLogger = loggerCopy
	}
	return ctxLogger
}

// Returns the root logger. Never panics: if root is missing, installs a safe fallback.
// Caller must hold loggersLock.
func getRootLogger() *Logger {
	if rootLogger, contains := loggers[_ROOT_NAME]; contains {
		return rootLogger
	}
	if loggers == nil {
		loggers = make(map[string]*Logger)
	}
	if loggersCreationOrder == nil {
		loggersCreationOrder = make([]string, 0, loggerCacheMax)
	}
	root := newHardFallbackRootLogger()
	loggers[_ROOT_NAME] = root
	// root from fallback is config-equivalent: not added to loggersCreationOrder
	return root
}

// Builds a minimal root logger that writes JSON to stdout using only zapcore primitives (no zap.Must).
func newHardFallbackRootLogger() *Logger {
	encoderConfig := defaultZapEncoderConfig()
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)
	zapLogger := zap.New(core)
	return newLogger(_ROOT_NAME, InfoLevel, map[string]*zap.Logger{
		_DEFAULT_HANDLER_NAME: zapLogger,
	})
}

// Shared zap encoder settings used by config-built handlers and the hard fallback root.
func defaultZapEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		MessageKey:  "message",
		LevelKey:    "level",
		TimeKey:     "time",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
		EncodeTime:  zapcore.RFC3339NanoTimeEncoder,
	}
}

// Returns the in-memory default config (root → JSON stdout at info) used when InitFromConfig was never called.
func getDefaultLoggerConfig() ConfigModel {
	handler := _DEFAULT_HANDLER_NAME
	return ConfigModel{
		Loggers: map[string]LoggerConfigModel{
			_ROOT_NAME: {
				Level:        "info",
				HandlerNames: []string{handler},
			},
		},
		Handlers: map[string]HandlerConfigModel{
			handler: {
				Level:       "info",
				Encoding:    "json",
				OutputPaths: []string{"stdout"},
			},
		},
	}
}

// Maps a config level string to LogLevel (case-insensitive).
func parseLogLevelString(levelStr string) (LogLevel, error) {
	var level LogLevel
	switch strings.ToLower(levelStr) {
	case "off", "none":
		level = NoneLevel
	case "error":
		level = ErrorLevel
	case "warning", "warn":
		level = WarningLevel
	case "info":
		level = InfoLevel
	case "debug":
		level = DebugLevel
	default:
		return 0, fmt.Errorf("invalid log level '%v'", levelStr)
	}
	return level, nil
}

// Creates all Loggers and Handlers (underlying Zap Loggers) from the config model.
func initLoggersFromConfig(config ConfigModel) (map[string]*Logger, error) {

	result := make(map[string]*Logger)

	// let's start with the handlers - as we will create a Zap logger for each entry there

	zapEncoderConfig := defaultZapEncoderConfig()
	zapLoggers := make(map[string]*zap.Logger)
	for key, element := range config.Handlers {
		// validate encoding (json | console); reject typos instead of silently defaulting
		var encoding string
		switch strings.ToLower(element.Encoding) {
		case "json":
			encoding = "json"
		case "console":
			encoding = "console"
		default:
			return result, fmt.Errorf("invalid encoding '%v' in config at /handlers/%v (supported: json, console)", element.Encoding, key)
		}

		// let's assemble a Zap config object!
		zapLevel, err := zap.ParseAtomicLevel(element.Level)
		if err != nil {
			return result, fmt.Errorf("unkown log level '%v' in config at /handlers/%v", element.Level, key)
		}

		var zapLogger *zap.Logger
		if element.RollingFile == nil {
			zapCfg := zap.Config{
				Level:             zapLevel,
				Encoding:          encoding,
				OutputPaths:       element.OutputPaths,
				EncoderConfig:     zapEncoderConfig,
				DisableCaller:     true,
				DisableStacktrace: true,
			}
			// return build errors instead of panicking (zap.Must)
			built, buildErr := zapCfg.Build()
			if buildErr != nil {
				return result, fmt.Errorf("failed to build handler '%v': %v", key, buildErr)
			}
			zapLogger = built
		} else {
			if len(element.OutputPaths) > 0 {
				// this is not allowed!
				return result, fmt.Errorf("if you use 'rollingFile' on a handler then you can not use 'outputPaths' as well in config at /handlers/%v", key)
			}
			log := &lumberjack.Logger{
				Filename:   element.RollingFile.File,       // Location of the log file
				MaxSize:    element.RollingFile.MaxSizeMb,  // Maximum file size (in MB)
				MaxBackups: element.RollingFile.MaxBackups, // Maximum number of old files to retain
				MaxAge:     element.RollingFile.MaxAgeDays, // Maximum number of days to retain old files
				Compress:   element.RollingFile.Compress,   // Whether to compress/archive old files
				LocalTime:  true,                           // Use local time for timestamps
			}
			writer := zapcore.AddSync(log)
			var encoder zapcore.Encoder
			if encoding == "console" {
				encoder = zapcore.NewConsoleEncoder(zapEncoderConfig)
			} else {
				encoder = zapcore.NewJSONEncoder(zapEncoderConfig)
			}
			core := zapcore.NewCore(encoder, writer, zapLevel)
			zapLogger = zap.New(core)
		}
		zapLoggers[key] = zapLogger
	}

	// cool! now let's deal with the /loggers part!
	for key, element := range config.Loggers {
		var handlers = map[string]*zap.Logger{}
		for _, handlerName := range element.HandlerNames {
			handler, contains := zapLoggers[handlerName]
			if !contains {
				return result, fmt.Errorf("problem in config /loggers/%v: invalid handler reference, handler '%v' does not exist", key, handlerName)
			}
			handlers[handlerName] = handler
		}
		level, err := parseLogLevelString(element.Level)
		if err != nil {
			return result, fmt.Errorf("problem in config /loggers/%v: %v", key, err)
		}
		logger := newLogger(key, level, handlers)
		result[key] = logger
	}

	if _, contains := result[_ROOT_NAME]; !contains {
		// "root" logger definition is mandatory
		return result, fmt.Errorf("log config file must define \"root\" logger")
	}

	return result, nil
}

// VisibleForTesting_OnDemandCacheSize returns how many on-demand logger names are currently cached.
func VisibleForTesting_OnDemandCacheSize() int {
	loggersLock.RLock()
	defer loggersLock.RUnlock()
	return len(loggersCreationOrder)
}

// VisibleForTesting_IsRegistered reports whether name is currently in the logger registry.
func VisibleForTesting_IsRegistered(name string) bool {
	loggersLock.RLock()
	defer loggersLock.RUnlock()
	if loggers == nil {
		return false
	}
	_, ok := loggers[name]
	return ok
}

// VisibleForTesting_LoggerCacheMax returns the active on-demand cache max.
func VisibleForTesting_LoggerCacheMax() int {
	loggersLock.RLock()
	defer loggersLock.RUnlock()
	return loggerCacheMax
}
