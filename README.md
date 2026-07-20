# lib-logging-golang

A wrapper around (currently! can change!) the popular [go.uber.org/zap](https://pkg.go.dev/go.uber.org/zap) logging library. On top of that it brings:

- configurability from yaml/json config (Python style)
- hierarchical logging
- `fmt.Printf()` style `.Info("log message with %v", value)` signatures — the string is only built if the log event is not filtered out
- **global labels** — key-value pairs attached to every log event
- builder style to add custom **labels** to particular log events

If you never call `InitFromConfig`, the first `GetLogger` / `With` creates a default **root** logger (JSON to stdout at info level).

# Get and install

```bash
go get github.com/keytiles/lib-logging-golang/v2@latest
```

Import path:

```go
import "github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
```

# Usage

Short happy path below. Full walkthrough (hierarchy, silent loggers, `Is*Enabled`, etc.) lives in the [example](example) folder — see `example/usage_example.go` and `example/log-config.yaml`.

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
)

func main() {
	if err := kt_logging.InitFromConfig("./log-config.yaml"); err != nil {
		panic(fmt.Sprintf("configuring logging failed: %v", err))
	}

	kt_logging.SetGlobalLabels([]kt_logging.Label{
		kt_logging.StringLabel("appName", envOr("CONTAINER_NAME", "?")),
		kt_logging.StringLabel("host", envOr("HOSTNAME", "?")),
	})

	// simple + Printf-style (formatted only if not filtered out)
	kt_logging.With("root").Info("very simple info message")
	kt_logging.With("root").Info("sent at %v", time.Now())

	// per-event labels
	kt_logging.With("root").
		WithLabel(kt_logging.StringLabel("myKey", "myValue")).
		Info("info with a label")

	// hierarchy: only "controller" is configured → "controller.something" inherits it
	kt_logging.With("controller.something").Info("not visible (logger level is warn)")
	kt_logging.With("controller.something").Warn("visible controller log")

	logger := kt_logging.GetLogger("main")
	logger.Info("with a cached logger instance")
	if logger.IsDebugEnabled() {
		logger.Debug("expensive debug assembly only when enabled")
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

# Config file

We use a **Python-style** log config (simple and effective). See [`example/log-config.yaml`](example/log-config.yaml).

Two sections:

- **loggers** — map of named Logger instances (map key = name). Each has:
  - `level` — `error` | `warning`/`warn` | `info` | `debug` | `none`/`off` (case-insensitive)
  - `handlers` — list of handler names to forward to after level filtering (empty ⇒ silent)
  - A **`root`** logger is mandatory
- **handlers** — map of named outputs. Each has:
  - `level` — zap level for that output
  - `encoding` — `json` or `console`
  - either `outputPaths` (e.g. `stdout` or a file path), **or** `rollingFile` (size/age rotation via lumberjack) — not both on the same handler

**Note:** rolling files are not reliable on Windows (lumberjack file-lock issue). Prefer non-rolling outputs there; details in [`CHANGELOG.md`](CHANGELOG.md).

# See also

- [`docs/logging-v2.1.md`](docs/logging-v2.1.md) — how the package works (relations, hierarchy, APIs)
- [`CHANGELOG.md`](CHANGELOG.md) — release history
- [`example/`](example) — runnable example + sample config
