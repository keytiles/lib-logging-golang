package kt_logging_test

import (
	"testing"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"go.uber.org/zap"
)

// Ensures GetGlobalLabels returns a snapshot copy that does not share backing storage with the published set.
func TestGetGlobalLabelsReturnsSnapshotCopy(t *testing.T) {
	// GIVEN — a published global labels set
	original := []kt_logging.Label{kt_logging.StringLabel("appName", "phase2")}
	kt_logging.SetGlobalLabels(original)

	// WHEN — reading the snapshot and mutating the returned slice
	snapshot := kt_logging.GetGlobalLabels()
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 label, got %d", len(snapshot))
	}
	snapshot[0] = kt_logging.StringLabel("appName", "mutated")

	// THEN — a fresh Get still returns the published value (mutation did not leak)
	again := kt_logging.GetGlobalLabels()
	if len(again) != 1 || again[0].GetStringValue() != "phase2" {
		t.Fatalf("expected published snapshot to stay 'phase2', got %+v", again)
	}
}

// Ensures SetGlobalLabels copies the caller slice so later mutation of the input does not affect logging state.
func TestSetGlobalLabelsCopiesInput(t *testing.T) {
	// GIVEN — a caller-owned labels slice
	input := []kt_logging.Label{kt_logging.StringLabel("host", "original-host")}

	// WHEN — publishing then mutating the caller slice
	kt_logging.SetGlobalLabels(input)
	input[0] = kt_logging.StringLabel("host", "mutated-host")

	// THEN — Get still returns the value from Set time
	got := kt_logging.GetGlobalLabels()
	if len(got) != 1 || got[0].GetStringValue() != "original-host" {
		t.Fatalf("expected 'original-host' after input mutation, got %+v", got)
	}
}

// Ensures GetHandlers returns a map copy; mutating the returned map does not change the logger's handlers.
func TestGetHandlersReturnsShallowCopy(t *testing.T) {
	// GIVEN — a logger with at least one handler (lazy default root path)
	logger := kt_logging.GetLogger("root")
	handlers := logger.GetHandlers()
	if len(handlers) == 0 {
		t.Fatal("expected root logger to have handlers")
	}
	originalLen := len(handlers)

	// WHEN — mutating the returned map
	handlers["injected"] = &zap.Logger{}

	// THEN — a fresh GetHandlers still has the original size / no injected key
	again := logger.GetHandlers()
	if len(again) != originalLen {
		t.Fatalf("expected handler count %d after mutating returned map, got %d", originalLen, len(again))
	}
	if _, exists := again["injected"]; exists {
		t.Fatal("injected handler leaked into logger internal map")
	}
}
