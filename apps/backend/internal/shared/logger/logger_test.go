package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNew_CleanRecord_IsLogged(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, slog.LevelInfo).Info("practice created", "practice_id", "abc")

	if !strings.Contains(buf.String(), "practice created") {
		t.Fatalf("clean record was discarded: %q", buf.String())
	}
}

func TestNew_PIIMessage_IsDropped(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, slog.LevelInfo).Info("email me at user@example.com")

	if buf.Len() != 0 {
		t.Fatalf("PII record reached the sink: %q", buf.String())
	}
}

func TestNew_LevelBelowThreshold_IsDropped(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, slog.LevelWarn).Info("below threshold")

	if buf.Len() != 0 {
		t.Fatalf("record below level reached the sink: %q", buf.String())
	}
}
