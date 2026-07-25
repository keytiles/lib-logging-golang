package kt_logging_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
)

// Ensures GetLogger / logging work without prior InitFromConfig and do not panic.
func TestGetLoggerWithoutInitDoesNotPanic(t *testing.T) {
	// GIVEN — no InitFromConfig called in this scenario (lazy default path)

	// WHEN — resolve a hierarchical logger name and emit a log
	logger := kt_logging.GetLogger("runtime.path.without.init")
	logger.Info("phase1 runtime path ok")

	// THEN — logger is usable and keeps the requested name
	if logger == nil {
		t.Fatal("GetLogger returned nil")
	}
	if logger.GetName() != "runtime.path.without.init" {
		t.Fatalf("unexpected logger name: %q", logger.GetName())
	}
}

// Ensures InitFromConfig rejects an invalid handler encoding with an error (no panic).
func TestInitFromConfigInvalidEncodingReturnsError(t *testing.T) {
	// GIVEN — a temp config file with an unsupported encoding value
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad-encoding.yaml")
	content := []byte(`
loggers:
  root:
    level: info
    handlers:
      - bad
handlers:
  bad:
    level: info
    encoding: not-a-real-encoding
    outputPaths:
      - stdout
`)
	if err := os.WriteFile(cfgPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	// WHEN — loading that config
	err := kt_logging.InitFromConfig(cfgPath)

	// THEN — init fails with an error instead of panicking or succeeding
	if err == nil {
		t.Fatal("expected error for invalid encoding, got nil")
	}
}
