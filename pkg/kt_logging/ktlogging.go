package kt_logging

import (
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	//"gopkg.in/yaml.v3"
)

type LogLevel uint8

const (
	NoneLevel    LogLevel = 0
	ErrorLevel   LogLevel = 1
	WarningLevel LogLevel = 2
	InfoLevel    LogLevel = 3
	DebugLevel   LogLevel = 4

	_ROOT_NAME string = "root"
)

var loggers map[string]*Logger

// we need locks to avoid concurrent map operations
var loggersLock = new(sync.RWMutex)

// this is a set of key-value pairs which are added to every log events
// You can use the getter/setter to change these values!
var globalLabels []Label
var zapGlobalLabels []zap.Field

// returns the current GlobalLabels - key-value pairs attached to all log events
func GetGlobalLabels() []Label {
	return globalLabels
}

// you can change the GlobalLabels with this - the key-value pairs attached to all log events
func SetGlobalLabels(labels []Label) {
	globalLabels = labels
	// let's convert immediately to Zap fields
	zapGlobalLabels = toZapFieldArray(labels)
}

// Initializing the logging from the .yaml or .json config file available on the given path
func InitFromConfig(cfgPath string) error {
	// read the config file
	configModel, err := parseFromJsonOrYaml(cfgPath)
	if err != nil {
		return err
	}
	// create and initialize loggers based on that
	configuredLoggers, err := initLoggersFromConfig(configModel)
	if err != nil {
		return err
	}

	// all good - lets store this
	loggers = configuredLoggers

	return nil
}

// returns a Logger with the given name - if does not exist then a new instance is created with this name and registered
// note: Loggers are hierarchical
func GetLogger(loggerName string) *Logger {
	loggersLock.Lock()
	ctxLogger := getLogger(loggerName)
	loggersLock.Unlock()
	return ctxLogger
}

// just a shortcut to the GetLogger method - for builder style readability stuff
func With(loggerName string) *Logger {
	return GetLogger(loggerName)
}

// internal method to get a logger - NOT THREAD SAFE! Already assumes Lock is established so no race condition!
func getLogger(loggerName string) *Logger {
	if loggers == nil {
		// this means that loggers were not initialised. Create a root logger with default config.
		var err error
		loggers, err = initLoggersFromConfig(getDefaultLoggerConfig())
		if err != nil {
			panic(fmt.Sprintf("could not create a root logger with default config: %v", err.Error()))
		}
	}
	ctxLogger := loggers[loggerName]
	if ctxLogger == nil {
		// let's plit by '.' characters
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
		// register the clone
		loggers[loggerName] = loggerCopy
		ctxLogger = loggerCopy
	}
	return ctxLogger
}

func getRootLogger() *Logger {
	rootLogger, contains := loggers[_ROOT_NAME]
	if !contains {
		// this should never happen, as a root logger should have been created if loggers were not initialised
		panic("root logger not found! loggers initialisation likely did not happen correctly.")
	}
	return rootLogger
}

func getDefaultLoggerConfig() ConfigModel {
	handler := "stdout_json"
	return ConfigModel{
		Loggers: map[string]LoggerConfigModel{
			"root": {
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

// creates all Loggers and also Handlers (underlying Zap Loggers) - based on the config we have
func initLoggersFromConfig(config ConfigModel) (map[string]*Logger, error) {

	loggers := make(map[string]*Logger)

	// let's start with the handlers - as we will create a Zap logger for each entry there

	zapEncoderConfig := zapcore.EncoderConfig{
		MessageKey:  "message",
		LevelKey:    "level",
		TimeKey:     "time",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
		EncodeTime:  zapcore.RFC3339NanoTimeEncoder,
	}
	zapLoggers := make(map[string]*zap.Logger)
	for key, element := range config.Handlers {
		// let's assemble a Zap config object!
		zapLevel, err := zap.ParseAtomicLevel(element.Level)
		if err != nil {
			return loggers, fmt.Errorf("unkown log level '%v' in config at /handlers/%v", element.Level, key)
		}

		var zapLogger *zap.Logger
		if element.RollingFile == nil {
			zapCfg := zap.Config{
				Level:             zapLevel,
				Encoding:          element.Encoding,
				OutputPaths:       element.OutputPaths,
				EncoderConfig:     zapEncoderConfig,
				DisableCaller:     true,
				DisableStacktrace: true,
			}
			zapLogger = zap.Must(zapCfg.Build())
		} else {
			if len(element.OutputPaths) > 0 {
				// this is not allowed!
				return loggers, fmt.Errorf("if you use 'rollingFile' on a handler then you can not use 'outputPaths' as well in config at /handlers/%v", key)
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
			if element.Encoding == "console" {
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
				return loggers, fmt.Errorf("problem in config /loggers/%v: invalid handler reference, handler '%v' does not exist", key, handlerName)
			}
			handlers[handlerName] = handler
		}
		level, err := parseLogLevelString(element.Level)
		if err != nil {
			return loggers, fmt.Errorf("problem in config /loggers/%v: %v", key, err)
		}
		logger := newLogger(key, level, handlers)
		loggers[key] = logger
	}

	if _, contains := loggers["root"]; !contains {
		// "root" logger definition is mandatory
		return loggers, fmt.Errorf("log config file must define \"root\" logger")
	}

	return loggers, nil
}
