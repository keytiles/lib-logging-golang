package kt_logging_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
)

// Writes a minimal config with optional loggerMaxCacheSize and returns its path.
func writeCacheTestConfig(t *testing.T, loggerMaxCacheSize *int) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "log-config.yaml")

	cacheLine := ""
	if loggerMaxCacheSize != nil {
		cacheLine = fmt.Sprintf("loggerMaxCacheSize: %d\n", *loggerMaxCacheSize)
	}

	content := []byte(cacheLine + `
loggers:
  root:
    level: info
    handlers:
      - stdout_json
  main:
    level: debug
    handlers:
      - stdout_json
handlers:
  stdout_json:
    level: debug
    encoding: json
    outputPaths:
      - stdout
`)
	if err := os.WriteFile(cfgPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

// Ensures on-demand logger cache respects loggerMaxCacheSize and FIFO eviction by creation order.
func TestLoggerOnDemandCacheFIFO(t *testing.T) {
	t.Run("cache size capped and FIFO evicts oldest created", func(t *testing.T) {
		// ---- GIVEN
		max := 2
		if err := kt_logging.InitFromConfig(writeCacheTestConfig(t, &max)); err != nil {
			t.Fatal(err)
		}

		// ---- WHEN — fill cache with two flat on-demand names, then add a third
		first := kt_logging.GetLogger("ondemand_a")
		_ = kt_logging.GetLogger("ondemand_b")
		_ = kt_logging.GetLogger("ondemand_a") // hit does not change creation order
		_ = kt_logging.GetLogger("ondemand_c") // should evict a (oldest)

		// ---- THEN
		if kt_logging.VisibleForTesting_OnDemandCacheSize() != 2 {
			t.Fatalf("on-demand size: got %d, want 2", kt_logging.VisibleForTesting_OnDemandCacheSize())
		}
		if kt_logging.VisibleForTesting_IsRegistered("ondemand_a") {
			t.Fatal("ondemand_a should have been evicted (oldest / FIFO)")
		}
		if !kt_logging.VisibleForTesting_IsRegistered("ondemand_b") {
			t.Fatal("ondemand_b should still be cached")
		}
		if !kt_logging.VisibleForTesting_IsRegistered("ondemand_c") {
			t.Fatal("ondemand_c should be cached")
		}
		// recreating evicted name works
		recreated := kt_logging.GetLogger("ondemand_a")
		if recreated == nil || recreated.GetName() != "ondemand_a" {
			t.Fatal("evicted name should be recreatable")
		}
		if first.GetName() != "ondemand_a" {
			t.Fatal("original logger instance should keep its name")
		}
	})

	t.Run("config loggers stay registered and are never evicted", func(t *testing.T) {
		// ---- GIVEN
		max := 1
		if err := kt_logging.InitFromConfig(writeCacheTestConfig(t, &max)); err != nil {
			t.Fatal(err)
		}

		// ---- WHEN — overflow on-demand cache repeatedly
		for i := 0; i < 5; i++ {
			kt_logging.GetLogger(fmt.Sprintf("flood_%d", i))
		}

		// ---- THEN — config names were never in creation-order cache, so they remain
		if !kt_logging.VisibleForTesting_IsRegistered("root") || !kt_logging.VisibleForTesting_IsRegistered("main") {
			t.Fatal("config loggers must stay registered")
		}
		if kt_logging.VisibleForTesting_OnDemandCacheSize() > 1 {
			t.Fatalf("on-demand size over max: %d", kt_logging.VisibleForTesting_OnDemandCacheSize())
		}
	})

	t.Run("loggerMaxCacheSize override and default", func(t *testing.T) {
		// ---- GIVEN / WHEN — explicit override
		max := 7
		if err := kt_logging.InitFromConfig(writeCacheTestConfig(t, &max)); err != nil {
			t.Fatal(err)
		}
		// ---- THEN
		if kt_logging.VisibleForTesting_LoggerCacheMax() != 7 {
			t.Fatalf("cache max: got %d, want 7", kt_logging.VisibleForTesting_LoggerCacheMax())
		}

		// ---- GIVEN / WHEN — omitted field uses default
		if err := kt_logging.InitFromConfig(writeCacheTestConfig(t, nil)); err != nil {
			t.Fatal(err)
		}
		// ---- THEN
		if kt_logging.VisibleForTesting_LoggerCacheMax() != kt_logging.DefaultLoggerMaxCacheSize {
			t.Fatalf("cache max: got %d, want default %d", kt_logging.VisibleForTesting_LoggerCacheMax(), kt_logging.DefaultLoggerMaxCacheSize)
		}
	})

	t.Run("negative loggerMaxCacheSize returns error", func(t *testing.T) {
		// ---- GIVEN
		neg := -1
		cfgPath := writeCacheTestConfig(t, &neg)

		// ---- WHEN
		err := kt_logging.InitFromConfig(cfgPath)

		// ---- THEN
		if err == nil {
			t.Fatal("expected error for negative loggerMaxCacheSize")
		}
	})
}
