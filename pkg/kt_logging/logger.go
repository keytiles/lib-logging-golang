// This file defines the Logger struct along with its methods
//
// Loggers are named (created from the config json/yaml and returned by ktlogging.with(loggerName)) objects
// with a specific level assigned to them - writing log events into a set of named and configured outputs.
// All exported *Logger methods are nil-safe: a nil receiver is a silent no-op (IsSilent() == true).

package kt_logging

import (
	"fmt"
	"maps"

	"go.uber.org/zap"
)

type Logger struct {
	name     string                 // package private field
	level    LogLevel               // package private field
	handlers map[string]*zap.Logger // package private field
}

// Constructor of the Logger - package private
func newLogger(name string, level LogLevel, handlers map[string]*zap.Logger) *Logger {
	instance := &Logger{name: name, level: level, handlers: handlers}
	return instance
}

// returns a clone of the logger - after this the 2 instances are not connected anyhow
func (l *Logger) clone() *Logger {
	if l == nil {
		return nil
	}
	loggers_clone := make(map[string]*zap.Logger, len(l.handlers))
	maps.Copy(loggers_clone, l.handlers)
	return newLogger(l.name, l.level, loggers_clone)
}

// returns the name of the Logger - this can not change after instantiation
func (l *Logger) GetName() string {
	if l == nil {
		return ""
	}
	return l.name
}

// returns the level
func (l *Logger) GetLevel() LogLevel {
	if l == nil {
		return NoneLevel
	}
	return l.level
}

// Returns a shallow copy of the attached handlers map so callers cannot mutate the logger's internal map.
func (l *Logger) GetHandlers() map[string]*zap.Logger {
	if l == nil {
		return nil
	}
	handlersCopy := make(map[string]*zap.Logger, len(l.handlers))
	maps.Copy(handlersCopy, l.handlers)
	return handlersCopy
}

func (l *Logger) isFilteredOut(level LogLevel) bool {
	if l == nil {
		return true
	}
	return l.level < level
}

// returns TRUE if Logger would output Error level logs due to its current configuration - FALSE otherwise
func (l *Logger) IsErrorEnabled() bool {
	return l != nil && l.level >= ErrorLevel && len(l.handlers) > 0
}

// returns TRUE if Logger would output Warning level logs due to its current configuration - FALSE otherwise
func (l *Logger) IsWarningEnabled() bool {
	return l != nil && l.level >= WarningLevel && len(l.handlers) > 0
}

// returns TRUE if Logger would output Info level logs due to its current configuration - FALSE otherwise
func (l *Logger) IsInfoEnabled() bool {
	return l != nil && l.level >= InfoLevel && len(l.handlers) > 0
}

// returns TRUE if Logger would output Debug level logs due to its current configuration - FALSE otherwise
func (l *Logger) IsDebugEnabled() bool {
	return l != nil && l.level >= DebugLevel && len(l.handlers) > 0
}

// Returns TRUE if Logger would not output anything due to its current configuration. This is either because  it's log level is None or does not have any
// (output) handlers at the moment
func (l *Logger) IsSilent() bool {
	return l == nil || l.level == NoneLevel || len(l.handlers) == 0
}

// Internally used method to emit a log event after level / handler filtering.
func (l *Logger) log(level LogLevel, customLabels []Label, message string, messageParams ...any) {
	if l == nil {
		return
	}

	// filter for level and not having any handlers (output)
	if l.isFilteredOut(level) || len(l.handlers) == 0 {
		return
	}

	// this event will be logged - so it makes sense to compile and put together everything!

	// build message only when there are format args
	msg := message
	if len(messageParams) > 0 {
		msg = fmt.Sprintf(message, messageParams...)
	}

	// pre-size fields: logger name + globals + custom labels
	globalLen := 0
	snap := loadGlobalLabelsSnapshot()
	if snap != nil {
		globalLen = len(snap.zapFields)
	}
	joinedLabels := make([]zap.Field, 0, 1+globalLen+len(customLabels))
	joinedLabels = append(joinedLabels, zap.String("logger", l.name))
	if globalLen > 0 {
		joinedLabels = append(joinedLabels, snap.zapFields...)
	}
	joinedLabels = appendZapFields(joinedLabels, customLabels)

	// now lets use all underlying Zap loggers and send the log event to each
	for _, zapLogger := range l.handlers {
		switch level {
		case ErrorLevel:
			zapLogger.Error(msg, joinedLabels...)
		case WarningLevel:
			zapLogger.Warn(msg, joinedLabels...)
		case InfoLevel:
			zapLogger.Info(msg, joinedLabels...)
		case DebugLevel:
			zapLogger.Debug(msg, joinedLabels...)
		default:
			// OK someone has sent us unknown log level
			// we dont want to lose this log event but we need to note the problem - so let's log it on Warning level
			l.log(WarningLevel, customLabels, "the following message was logged on unkown log level! Original message: "+message, messageParams...)
		}
	}
}

// Decorates the upcoming LogEvent (when you invoke .info(), .error() etc method the LogEvent is fired) with the given labels.
// Please note: the labels will be just used in the upcoming LogEvent and after that forgotten!
func (l *Logger) WithLabels(labels []Label) LogEvent {
	le := newLogEvent(l)
	le = le.WithLabels(labels)
	return le
}

// Decorates the upcoming LogEvent (when you invoke .info(), .error() etc method the LogEvent is fired) with the given label. If you have multiple labels to add
// consider using .WithLabels() method instead.
// Please note: the label will be just used in the upcoming LogEvent and after that forgotten!
func (l *Logger) WithLabel(label Label) LogEvent {
	le := newLogEvent(l)
	le = le.WithLabel(label)
	return le
}

// logs the given message resolved with (optional) messageParams (Printf() style) on the given log level
// in case the the message is filtered out due to configured log level then the message string is not built at all
func (l *Logger) Log(level LogLevel, message string, messageParams ...any) {
	l.log(level, nil, message, messageParams...)
}

// Wrapper around .Log() function - firing a log event on Debug level
func (l *Logger) Debug(message string, messageParams ...any) {
	l.log(DebugLevel, nil, message, messageParams...)
}

// Wrapper around .Log() function - firing a log event on Info level
func (l *Logger) Info(message string, messageParams ...any) {
	l.log(InfoLevel, nil, message, messageParams...)
}

// Wrapper around .Log() function - firing a log event on Warning level
func (l *Logger) Warn(message string, messageParams ...any) {
	l.log(WarningLevel, nil, message, messageParams...)
}

// Wrapper around .Log() function - firing a log event on Error level
func (l *Logger) Error(message string, messageParams ...any) {
	l.log(ErrorLevel, nil, message, messageParams...)
}
