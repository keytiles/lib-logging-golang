package kt_logging_test

import (
	"fmt"
	"testing"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
)

func BenchmarkLogging(b *testing.B) {

	// err := kt_logging.InitFromConfig("tests-log-config-stdoutjson-only.yaml")
	err := kt_logging.InitFromConfig("tests-log-config-with-plainlog-file.yaml")
	if err != nil {
		panic(fmt.Sprintf("failed to load log config file: %s", err))
	}

	labels := []kt_logging.Label{kt_logging.StringLabel("key", "value")}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		kt_logging.With("main").
			WithLabels(labels).
			WithLabel(kt_logging.StringLabel("key2", "value2")).
			WithLabel(kt_logging.StringLabel("key3", "value3")).
			Info("hello")
	}

	b.StopTimer()
}
