package main

import (
	"fmt"
	"os"
	"time"

	ktlogging "github.com/keytiles/lib-logging-golang"
)

func main() {

	// === default logger without any config. Json encoded, info level, written to std out.
	ktlogging.GetDefaultLogger().Info("logging with a default logger")

	// === init the logging

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
