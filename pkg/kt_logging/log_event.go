package kt_logging

/*
LogEvent is used when the caller adds extra labels before firing a log.
An instance is created by Logger.WithLabel(s) and collects labels until
Info/Warn/etc closes the event.
*/

type LogEvent struct {
	// Logger that will emit the event
	logger *Logger
	// Labels attached to this event only (nil until first WithLabel / WithLabels)
	customLabels []Label
}

// Creates a short-lived LogEvent value (stack-friendly; no heap slices until labels are added).
func newLogEvent(withLogger *Logger) LogEvent {
	return LogEvent{logger: withLogger}
}

// Appends multiple labels to this event.
func (le LogEvent) WithLabels(labels []Label) LogEvent {
	le.customLabels = append(le.customLabels, labels...)
	return le
}

// Appends a single label to this event.
func (le LogEvent) WithLabel(label Label) LogEvent {
	le.customLabels = append(le.customLabels, label)
	return le
}

// Emits the collected labels through the underlying Logger.
// A nil underlying logger is a no-op (same contract as nil *Logger methods).
func (le LogEvent) logWithLogger(level LogLevel, message string, messageParams ...any) {
	if le.logger == nil {
		return
	}
	if le.logger.isFilteredOut(level) || len(le.logger.handlers) == 0 {
		// skip — no point assembling further work
		return
	}

	le.logger.log(level, le.customLabels, message, messageParams...)
}

// Fires a log event on Debug level
func (le LogEvent) Debug(message string, messageParams ...any) {
	le.logWithLogger(DebugLevel, message, messageParams...)
}

// Fires a log event on Info level
func (le LogEvent) Info(message string, messageParams ...any) {
	le.logWithLogger(InfoLevel, message, messageParams...)
}

// Fires a log event on Warning level
func (le LogEvent) Warn(message string, messageParams ...any) {
	le.logWithLogger(WarningLevel, message, messageParams...)
}

// Fires a log event on Error level
func (le LogEvent) Error(message string, messageParams ...any) {
	le.logWithLogger(ErrorLevel, message, messageParams...)
}
