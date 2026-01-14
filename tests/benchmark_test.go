package kt_logging_test

import (
	"testing"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
)

func BenchmarkLogging(b *testing.B) {

	kt_logging.InitFromConfig("../example/log-config.yaml")

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
