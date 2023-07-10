package ktlogging

/*
LogEvent structs just used internally - when user is adding extra labels to the log event.
In that case an instance of this struct is created by the Logger and this struct is respponsible to collect up the extra labels
until finally an .Info(), .Warn() etc call is made to close the log
*/

type LogEvent struct {
	// we will do the log itself with this effective Logger
	logger *Logger
	// a list of pointers - pointing to key-value pair arrays we are intending to attach to this log event if fired
	customLabelList [][]Label
	// and we also have a simple list of pointers - ppointing to key-value pair
	customLabels []Label
}

// constructor - package private
// note: as you can see we do not return pointer but allocated object on stack - this is on purpose!
// since these objects are short lived much better allocate them on stack than on heap (which kicks in GC as well -> slower)
func newLogEvent(withLogger *Logger) LogEvent {
	instance := LogEvent{logger: withLogger}
	// lets initialize with emppty arrays
	instance.customLabelList = [][]Label{}
	instance.customLabels = []Label{}

	return instance
}

func (le LogEvent) WithLabels(labels []Label) LogEvent {
	le.customLabelList = append(le.customLabelList, labels)
	return le
}

func (le LogEvent) WithLabel(label Label) LogEvent {
	le.customLabels = append(le.customLabels, label)
	return le
}

// making this event - actually makes the log itself
func (le LogEvent) logWithLogger(level LogLevel, message string, messageParams ...any) {
	if le.logger.isFilteredOut(level) {
		// we skip this - as this log event will not happen for sure no point to make further efforts
		return
	}

	// this event will be logged - so it makes sense to compile and put together everything!

	var joinedLabels = []Label{}
	// add the custom things
	for _, customLabelsArr := range le.customLabelList {
		joinedLabels = append(joinedLabels, customLabelsArr...)
	}
	joinedLabels = append(joinedLabels, le.customLabels...)

	// finally, lets do the log!
	le.logger.log(level, joinedLabels, message, messageParams...)
}

func (le LogEvent) Debug(message string, messageParams ...any) {
	le.logWithLogger(DebugLevel, message, messageParams...)
}

func (le LogEvent) Info(message string, messageParams ...any) {
	le.logWithLogger(InfoLevel, message, messageParams...)
}

func (le LogEvent) Warn(message string, messageParams ...any) {
	le.logWithLogger(WarningLevel, message, messageParams...)
}

func (le LogEvent) Error(message string, messageParams ...any) {
	le.logWithLogger(ErrorLevel, message, messageParams...)
}
