package kt_logging_bench_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
)

// Emit benchmarks for kt_logging — kept separate from unit tests under tests/kt_logging/.
// One config YAML per handler style (see testdata/bench-handler-*.yaml).
//
//	go test -run=^$ -bench=BenchmarkEmit -benchmem ./tests/kt_logging_bench/
//
// Or use the summary script:
//
//	./tests/kt_logging_bench/run-benchmarks.sh              # default -benchtime=3s
//	./tests/kt_logging_bench/run-benchmarks.sh -count=3     # optional: mean over 3 runs

// JSON log file only (no console).
func BenchmarkEmit_FileJson(b *testing.B) {
	runEmitBenchmark(b, "bench-handler-file-json.yaml", false)
}

// Stdout JSON handler. Redirects os.Stdout → DevNull before init so the ns/op table stays readable.
func BenchmarkEmit_StdoutJson(b *testing.B) {
	runEmitBenchmark(b, "bench-handler-stdout-json.yaml", true)
}

// Stdout plain (console encoding) handler. Same stdout redirect as StdoutJson.
func BenchmarkEmit_StdoutPlain(b *testing.B) {
	runEmitBenchmark(b, "bench-handler-stdout-plain.yaml", true)
}

// Plain (console encoding) file only.
func BenchmarkEmit_FilePlain(b *testing.B) {
	runEmitBenchmark(b, "bench-handler-file-plain.yaml", false)
}

// Rolling JSON file only. Skips on Windows (lumberjack limitation — see CHANGELOG).
func BenchmarkEmit_FileRollingJson(b *testing.B) {
	if runtime.GOOS == "windows" {
		b.Skip("rolling file handler is not reliable on Windows (see CHANGELOG)")
	}
	runEmitBenchmark(b, "bench-handler-file-rolling-json.yaml", false)
}

// Init from testdata/<cfgFile>, cache logger, emit labeled Info each iteration.
func runEmitBenchmark(b *testing.B, cfgFile string, redirectStdoutToDevNull bool) {
	b.Helper()

	if redirectStdoutToDevNull {
		restore := redirectStdout(b)
		defer restore()
	}

	// Configs use relative output paths — run with cwd = testdata so files land next to the YAMLs.
	testdata := testdataDir(b)
	cfgPath := filepath.Join(testdata, cfgFile)

	prevWd, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}
	if err := os.Chdir(testdata); err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.Chdir(prevWd) }()

	if err := kt_logging.InitFromConfig(cfgPath); err != nil {
		b.Fatalf("InitFromConfig(%s): %v", cfgPath, err)
	}

	logger := kt_logging.GetLogger("main")
	labels := []kt_logging.Label{kt_logging.StringLabel("key", "value")}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.
			WithLabels(labels).
			WithLabel(kt_logging.StringLabel("key2", "value2")).
			WithLabel(kt_logging.StringLabel("key3", "value3")).
			Info("hello")
	}
}

// Points os.Stdout at DevNull so the stdout handler does not flood the terminal.
func redirectStdout(b *testing.B) (restore func()) {
	b.Helper()
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("open DevNull: %v", err)
	}
	old := os.Stdout
	os.Stdout = devNull
	return func() {
		os.Stdout = old
		_ = devNull.Close()
	}
}

func testdataDir(b *testing.B) string {
	b.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		b.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}
