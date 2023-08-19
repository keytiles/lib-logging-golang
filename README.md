# lib-logging-golang

A wrapper around (currently! can change!) the popular [go.uber.org/zap](https://pkg.go.dev/go.uber.org/zap) logging library and on top of that brings
 * configurability from yaml/json config (Python style)
 * hierarchical logging
 * bringing fmt.Printf() style .Info("log message with %v", value) logging signature - which will be only evaluated into a string if log event is not filtered out
 * concept of "global labels" - set of key-value papirs which are always logged with every log event
 * builder style to add custom labels (zap.Fields) to particular log events

# Get and install

`go get github.com/keytiles/lib-logging-golang`

# Usage

Here is a simple example (you also find this as a running code in the [example](example) folder!)

```go
import (
    ...
	"github.com/keytiles/lib-logging-golang"
	...
)


func main() {

	// === init the logging with a config file

	//     NOTE: this step is optional!
	//     if you do not init then a default config will be applied automatically at the first time when you request a Logger (see below)
	//     default config means:
	//     - "root" logger on INFO level
	//     - writes to STDOUT (console) in JSON format

	cfgErr := ktlogging.InitFromConfig("./log-config.yaml")
	if cfgErr != nil {
		panic(fmt.Sprintf("Oops! it looks configuring logging failed :-( error was: %v", cfgErr))
	}

	// === global labels

	ktlogging.SetGlobalLabels(buildGlobalLabels())

	// manipulating the GlobalLabels later is also possible
	globalLabels := ktlogging.GetGlobalLabels()
	globalLabels = append(globalLabels, ktlogging.FloatLabel("myVersion", 5.2))
	ktlogging.SetGlobalLabels(globalLabels)

	// === and now let's use the initialized logging!

	// most simple usage
	ktlogging.With("root").Info("very simple info message")

	// message constructued with parameters (will be really evaluated into a string if log event is not filtered out)
	ktlogging.With("root").Info("just an info level message - sent at %v", time.Now())

	// message with only one custom label
	ktlogging.With("root").WithLabel(ktlogging.StringLabel("myKey", "myValue")).Info("just an info level message - sent at %v", time.Now())

	// message with multiple labels
	ktlogging.With("root").WithLabels([]ktlogging.Label{ktlogging.IntLabel("myIntKey", 5), ktlogging.BoolLabel("myBoolKey", true)}).Info("just an info level message - sent at %v", time.Now())

	// and combined also works - multiple labels and one custom
	ktlogging.With("root").WithLabel(ktlogging.StringLabel("myKey", "myValue")).WithLabels([]ktlogging.Label{ktlogging.IntLabel("myIntKey", 5), ktlogging.BoolLabel("myBoolKey", true)}).Info("just an info level message - sent at %v", time.Now())

	// hierarchical logging - we only have "controller" configured (log-config.yaml) so this one will fall back in runtime
	ktlogging.With("controller.something").Info("not visible as logger level is 'warn'")
	ktlogging.With("controller.something").Warn("visible controller log")

	// get a Logger once - and then just use it in all subsequent logs
	// this way you can create package-private Logger instances e.g.
	logger := ktlogging.GetLogger("main")
	logger.Info("with logger instance")
	labels := []ktlogging.Label{ktlogging.StringLabel("key", "value")}
	logger.WithLabels(labels).Info("one more message tagged with 'key=value'")
}

// builds and returns labels we want to add to all log events (this is just an example!!)
func buildGlobalLabels() []ktlogging.Label {
	var globalLabels = []ktlogging.Label{}
	appName := os.Getenv("CONTAINER_NAME")
	appVer := os.Getenv("CONTAINER_VERSION")
	host := os.Getenv("HOSTNAME")

	if appName == "" {
		appName = "?"
	}
	globalLabels = append(globalLabels, ktlogging.StringLabel("appName", appName))
	if appVer == "" {
		appVer = "?"
	}
	globalLabels = append(globalLabels, ktlogging.StringLabel("appVer", appVer))
	if host == "" {
		host = "?"
	}
	globalLabels = append(globalLabels, ktlogging.StringLabel("host", host))

	return globalLabels
}
```

# Config file

We are using the **Python style log config** as that is simple yet effective at the same time.

Take a look into `/example/log-config.yaml` file!

This basically consists of two sections:
 * **loggers** - is a map of Logger instances you want to create.  
   So each Logger is named (by the key) and you can assign a specific log `level` (error|warning|info|debug) and list of `handlers` (see below) to where this Logger
   will forward to each log events passed the level filtering
 * **handlers** - is a map of configured outputs.
   Each Handler is a named (by the key) entity and can represent outputting to STDOUT (console), file or other. For Handlers you can control the encoding format can be 'json' or 'console'


