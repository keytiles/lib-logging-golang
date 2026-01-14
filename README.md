# lib-logging-golang

A wrapper around (currently! can change!) the popular [go.uber.org/zap](https://pkg.go.dev/go.uber.org/zap) logging library and on top of that brings

- configurability from yaml/json config (Python style)
- hierarchical logging
- bringing fmt.Printf() style .Info("log message with %v", value) logging signature - which will be only evaluated into a string if log event is not filtered out
- concept of "global labels" - set of key-value papirs which are always logged with every log event
- builder style to add custom labels (zap.Fields) to particular log events

# Get and install

`go get github.com/keytiles/lib-logging-golang`

# Usage

Here is a simple example (you also find this as a running code in the [example](example) folder!)

```go
import (
    ...
	"github.com/keytiles/lib-logging-golang/v2/kt_logging"
	...
)


func main() {

	// === init the logging

	cfgErr := kt_logging.InitFromConfig("./log-config.yaml")
	if cfgErr != nil {
		panic(fmt.Sprintf("Oops! it looks configuring logging failed :-( error was: %v", cfgErr))
	}

	// === global labels

	kt_logging.SetGlobalLabels(buildGlobalLabels())

	// manipulating the GlobalLabels later is also possible
	globalLabels := kt_logging.GetGlobalLabels()
	globalLabels = append(globalLabels, kt_logging.FloatLabel("myVersion", 5.2))
	kt_logging.SetGlobalLabels(globalLabels)

	// === and now let's use the initialized logging!

	// most simple usage
	kt_logging.With("root").Info("very simple info message")

	// message constructued with parameters (will be really evaluated into a string if log event is not filtered out)
	kt_logging.With("root").Info("just an info level message - sent at %v", time.Now())

	// message with only one custom label
	kt_logging.With("root").WithLabel(kt_logging.StringLabel("myKey", "myValue")).Info("just an info level message - sent at %v", time.Now())

	// message with multiple labels
	kt_logging.With("root").WithLabels([]kt_logging.Label{kt_logging.IntLabel("myIntKey", 5), kt_logging.BoolLabel("myBoolKey", true)}).Info("just an info level message - sent at %v", time.Now())

	// and combined also works - multiple labels and one custom
	kt_logging.With("root").WithLabel(kt_logging.StringLabel("myKey", "myValue")).WithLabels([]kt_logging.Label{kt_logging.IntLabel("myIntKey", 5), kt_logging.BoolLabel("myBoolKey", true)}).Info("just an info level message - sent at %v", time.Now())

	// hierarchical logging - we only have "controller" configured (log-config.yaml) so this one will fall back in runtime
	kt_logging.With("controller.something").Info("not visible as logger level is 'warn'")
	kt_logging.With("controller.something").Warn("visible controller log")

	// get a Logger once - and then just use it in all subsequent logs
	// this way you can create package-private Logger instances e.g.
	logger := kt_logging.GetLogger("main")
	logger.Info("with logger instance")
	labels := []kt_logging.Label{kt_logging.StringLabel("key", "value")}
	logger.WithLabels(labels).Info("one more message tagged with 'key=value'")

	// check conditionally if a log event we intend to do on a certain level would be fired or not
	// this way we can omit efforts taken into assembling a log event which later would be simply just dropped anyways
	if logger.IsDebugEnabled() {
		myDebugMsg := "for example"
		myDebugMsg = myDebugMsg + " if we do stuff"
		myDebugMsg = myDebugMsg + " to compile a Debug log message"
		myDebugMsg = myDebugMsg + " this way just done if makes sense"
		logger.Debug(myDebugMsg)
	}
	// the above methods also consider if the Logger has any configured output (handler) or not
	// and return false if however the log level is good but currently the Logger does not output anywhere
	// take a look into the 'log-config.yaml'! this Logger is configured on "debug" level but no handlers attached...
	noOutputLogger := kt_logging.GetLogger("no_handler")
	if noOutputLogger.IsInfoEnabled() {
		// you will never get in here...
	} else {
		// but always here!
		fmt.Println("logger 'no_handler' IsInfoEnabled() returned FALSE")
	}
	if noOutputLogger.IsErrorEnabled() {
		// similarly, you will never get in here either
	} else {
		// but always here!
		fmt.Println("logger 'no_handler' IsErrorEnabled() returned FALSE")
	}
	noOutputLogger.Info("this message will NOT appear")

	// you can also check if a specific logger "is silent" currently either because of the log level or not having configured outputs...
	if noOutputLogger.IsSilent() {
		// you would now get in here as this logger does not have any output (handler)
		fmt.Println("logger 'no_handler' IsSilent() returned TRUE")
	}
	silentLevelLogger := kt_logging.GetLogger("silent_level")
	silentLevelLogger.Error("this message will NOT appear")
	if silentLevelLogger.IsSilent() {
		// you would now get in here too as this logger's log level is "none" at the moment
		fmt.Println("logger 'silent_level' IsSilent() returned TRUE as well")
	}
	if !kt_logging.GetLogger("main").IsSilent() {
		// you would now get in here as 'main' logger is obviously not "silent"
		fmt.Println("logger 'main' IsSilent() returned FALSE - obviously...")
	}
}

// builds and returns labels we want to add to all log events (this is just an example!!)
func buildGlobalLabels() []kt_logging.Label {
	var globalLabels = []kt_logging.Label{}
	appName := os.Getenv("CONTAINER_NAME")
	appVer := os.Getenv("CONTAINER_VERSION")
	host := os.Getenv("HOSTNAME")

	if appName == "" {
		appName = "?"
	}
	globalLabels = append(globalLabels, kt_logging.StringLabel("appName", appName))
	if appVer == "" {
		appVer = "?"
	}
	globalLabels = append(globalLabels, kt_logging.StringLabel("appVer", appVer))
	if host == "" {
		host = "?"
	}
	globalLabels = append(globalLabels, kt_logging.StringLabel("host", host))

	return globalLabels
}
```

# Config file

We are using the **Python style log config** as that is simple yet effective at the same time.

Take a look into `/example/log-config.yaml` file!

This basically consists of two sections:

- **loggers** - is a map of Logger instances you want to create.  
  So each Logger is named (by the key) and you can assign a specific log `level` (error|warning|info|debug) and list of `handlers` (see below) to where this Logger
  will forward to each log events passed the level filtering
- **handlers** - is a map of configured outputs.
  Each Handler is a named (by the key) entity and can represent outputting to STDOUT (console), file or other. For Handlers you can control the encoding format can be 'json' or 'console'
