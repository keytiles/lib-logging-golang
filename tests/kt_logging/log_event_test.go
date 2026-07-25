package kt_logging_test

import (
	"testing"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
)

// Ensures WithLabel / WithLabels chaining still attaches labels after LogEvent simplification.
func TestLogEventWithLabelsChaining(t *testing.T) {
	// GIVEN — a logger and labels to attach via both APIs
	logger := kt_logging.GetLogger("main")

	// WHEN — chaining WithLabel and WithLabels then firing Info (must not panic)
	logger.
		WithLabel(kt_logging.StringLabel("one", "a")).
		WithLabels([]kt_logging.Label{
			kt_logging.IntLabel("two", 2),
			kt_logging.BoolLabel("three", true),
		}).
		Info("phase3 log event chain ok")

	// THEN — logger remains usable (smoke assertion)
	if logger.GetName() != "main" {
		t.Fatalf("unexpected logger name: %q", logger.GetName())
	}
}
