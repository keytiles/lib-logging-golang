package ktlogging

import (
	"testing"
)

func BenchmarkLogging(b *testing.B) {

	InitFromConfig("./example/log-config.yaml")

	labels := []Label{StringLabel("key", "value")}

	// f, err := os.Create("benchmark.out")
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()

	// err = trace.Start(f)
	// if err != nil {
	// 	panic(err)
	// }

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		With("silent").WithLabels(labels).WithLabel(StringLabel("key2", "value2")).WithLabel(StringLabel("key3", "value3")).Info("hello")
	}

	// trace.Stop()

	b.StopTimer()
}
