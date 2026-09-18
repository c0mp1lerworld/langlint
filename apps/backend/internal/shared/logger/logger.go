package logger

import (
	"io"
	"log/slog"
)

// New returns the single structured logger of the backend. Its handler is
// wrapped by PIIHandler, so any record carrying PII is dropped before it
// reaches the sink (A8). Callers must not build a raw slog handler elsewhere.
func New(w io.Writer, level slog.Level) *slog.Logger {
	inner := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(NewPIIHandler(inner))
}
