// This file defines the Logger struct along with its methods
//
// Loggers are named (created from the config json/yaml and returned by ktlogging.with(loggerName)) objects
// with a specific level assigned to them - writing log events into a set of named and configured outputs

package ktlogging

import (
	"fmt"

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
func (l Logger) clone() *Logger {
	loggers_clone := map[string]*zap.Logger{}
	for key, value := range l.handlers {
		loggers_clone[key] = value
	}
	return newLogger(l.name, l.level, loggers_clone)
}

// returns the name of the Logger - this can not change after instantiation
func (l Logger) GetName() string {
	return l.name
}

// returns the level
func (l Logger) GetLevel() LogLevel {
	return l.level
}

// returns the attached Handlers
func (l Logger) GetHandlers() map[string]*zap.Logger {
	return l.handlers
}

func (l Logger) isFilteredOut(level LogLevel) bool {
	return l.level < level
}

// internally used method to do the log
func (l *Logger) log(level LogLevel, customLabels []Label, message string, messageParams ...any) {

	// filter for level
	if l.isFilteredOut(level) {
		return
	}

	// this event will be logged - so it makes sense to compile and put together everything!

	// lets build the log string
	msg := fmt.Sprintf(message, messageParams...)

	// we add the name of the logger
	var joinedLabels = []zap.Field{zap.String("logger", l.name)}
	// and context variables - if exists
	if len(zapGlobalLabels) > 0 {
		joinedLabels = append(joinedLabels, zapGlobalLabels...)
	}
	joinedLabels = append(joinedLabels, toZapFieldArray(customLabels)...)

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

// Decorates the upcoming LogEvent (when you invoke .info(), .error() etc method the LogEvent is fired) with the given label. If you have multiple labels to add consider using .WithLabels() method instead.
// Please note: the label will be just used in the upcoming LogEvent and after that forgotten!
func (l *Logger) WithLabel(label Label) LogEvent {
	le := newLogEvent(l)
	le = le.WithLabel(label)
	return le
}

// logs the given message resolved with (optional) messageParams (Printf() style) on the given log level
// in case the the message is filtered out due to configured log level then the message string is not built at all
func (l *Logger) Log(level LogLevel, message string, messageParams ...any) {
	l.log(level, []Label{}, message, messageParams...)
}

// wrapper around .Log() method - fired with Debug level
func (l Logger) Debug(message string, messageParams ...any) {
	l.log(DebugLevel, []Label{}, message, messageParams...)
}

// wrapper around .Log() method - fired with Info level
func (l Logger) Info(message string, messageParams ...any) {
	l.log(InfoLevel, []Label{}, message, messageParams...)
}

// wrapper around .Log() method - fired with Warning level
func (l Logger) Warn(message string, messageParams ...any) {
	l.log(WarningLevel, []Label{}, message, messageParams...)
}

// wrapper around .Log() method - fired with Error level
func (l Logger) Error(message string, messageParams ...any) {
	l.log(ErrorLevel, []Label{}, message, messageParams...)
}
