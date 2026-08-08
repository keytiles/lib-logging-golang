package kt_logging_test

import (
	"testing"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
)

// Ensures nil *Logger methods are no-ops / safe defaults and do not panic.
func TestNilLoggerDoesNotPanic(t *testing.T) {
	var logger *kt_logging.Logger

	t.Run("inspectors return silent defaults", func(t *testing.T) {
		// ---- GIVEN
		// nil logger

		// ---- WHEN / THEN
		if logger.GetName() != "" {
			t.Fatalf("GetName: got %q, want empty", logger.GetName())
		}
		if logger.GetLevel() != kt_logging.NoneLevel {
			t.Fatalf("GetLevel: got %v, want NoneLevel", logger.GetLevel())
		}
		if logger.GetHandlers() != nil {
			t.Fatalf("GetHandlers: got %#v, want nil", logger.GetHandlers())
		}
		if logger.IsErrorEnabled() || logger.IsWarningEnabled() || logger.IsInfoEnabled() || logger.IsDebugEnabled() {
			t.Fatal("Is*Enabled should be false on nil logger")
		}
		if !logger.IsSilent() {
			t.Fatal("IsSilent should be true on nil logger")
		}
	})

	t.Run("direct emit methods do not panic", func(t *testing.T) {
		// ---- GIVEN
		// nil logger

		// ---- WHEN / THEN — must not panic
		logger.Debug("debug on nil")
		logger.Info("info on nil")
		logger.Warn("warn on nil")
		logger.Error("error on nil")
		logger.Log(kt_logging.InfoLevel, "log on nil")
	})

	t.Run("WithLabel chain emit does not panic", func(t *testing.T) {
		// ---- GIVEN
		// nil logger

		// ---- WHEN / THEN — must not panic
		logger.WithLabel(kt_logging.StringLabel("k", "v")).Info("labeled info on nil")
		logger.WithLabels([]kt_logging.Label{kt_logging.IntLabel("n", 1)}).Warn("labeled warn on nil")
	})
}
