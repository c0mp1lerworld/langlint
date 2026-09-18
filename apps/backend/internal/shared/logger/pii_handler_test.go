package logger

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func newPIILogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(NewPIIHandler(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
}

func TestPIIHandler_CleanRecord_IsLogged(t *testing.T) {
	var buf bytes.Buffer
	newPIILogger(&buf).Info("practice created", "practice_id", "abc")

	if !strings.Contains(buf.String(), "practice created") {
		t.Fatalf("clean record was discarded: %q", buf.String())
	}
}

func TestPIIHandler_LongMessage_IsDiscarded(t *testing.T) {
	var buf bytes.Buffer
	newPIILogger(&buf).Info(strings.Repeat("a", 101))

	if buf.Len() != 0 {
		t.Fatalf("long free-text message was logged: %q", buf.String())
	}
}

func TestPIIHandler_LongFreeTextAttr_IsDiscarded(t *testing.T) {
	var buf bytes.Buffer
	newPIILogger(&buf).Info("ok", "note", strings.Repeat("a", 101))

	if buf.Len() != 0 {
		t.Fatalf("long free-text attr was logged: %q", buf.String())
	}
}

func TestPIIHandler_EmailMessage_IsDiscarded(t *testing.T) {
	var buf bytes.Buffer
	newPIILogger(&buf).Info("contact user@example.com")

	if buf.Len() != 0 {
		t.Fatalf("email message was logged: %q", buf.String())
	}
}

func TestPIIHandler_PhoneMessage_IsDiscarded(t *testing.T) {
	var buf bytes.Buffer
	newPIILogger(&buf).Info("call +34 600 123 456")

	if buf.Len() != 0 {
		t.Fatalf("phone message was logged: %q", buf.String())
	}
}

func TestPIIHandler_APIKeyAttr_IsDiscarded(t *testing.T) {
	var buf bytes.Buffer
	newPIILogger(&buf).Info("ok", "key", "sk-abcdefghijklmnop1234")

	if buf.Len() != 0 {
		t.Fatalf("api key attr was logged: %q", buf.String())
	}
}

func TestPIIHandler_JWTAttr_IsDiscarded(t *testing.T) {
	var buf bytes.Buffer
	newPIILogger(&buf).Info("ok", "token", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature")

	if buf.Len() != 0 {
		t.Fatalf("jwt attr was logged: %q", buf.String())
	}
}

func TestPIIHandler_GroupWithPII_IsDiscarded(t *testing.T) {
	var buf bytes.Buffer
	newPIILogger(&buf).Info("ok", slog.Group("user", "email", "user@example.com"))

	if buf.Len() != 0 {
		t.Fatalf("group with PII was logged: %q", buf.String())
	}
}

func TestPIIHandler_AnyAttrWithPII_IsDiscarded(t *testing.T) {
	var buf bytes.Buffer
	newPIILogger(&buf).Info("ok", slog.Any("payload", struct{ Email string }{Email: "user@example.com"}))

	if buf.Len() != 0 {
		t.Fatalf("any attr with PII was logged: %q", buf.String())
	}
}

func TestPIIHandler_WithAttrs_StripsSensitiveAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := newPIILogger(&buf).With("prompt", strings.Repeat("x", 120))
	logger.Info("clean")

	out := buf.String()
	if !strings.Contains(out, "clean") {
		t.Fatalf("clean record was discarded: %q", out)
	}
	if strings.Contains(out, strings.Repeat("x", 120)) {
		t.Fatalf("sensitive attr bound with With was logged: %q", out)
	}
}

func TestPIIHandler_WithAttrs_KeepsCleanAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := newPIILogger(&buf).With("model", "gpt-4o-mini")
	logger.Info("clean")

	if !strings.Contains(buf.String(), "gpt-4o-mini") {
		t.Fatalf("clean attr bound with With was dropped: %q", buf.String())
	}
}

func TestPIIHandler_WithGroup_ScopesRecords(t *testing.T) {
	var buf bytes.Buffer
	newPIILogger(&buf).WithGroup("api").Info("ok", "status", 200)

	if !strings.Contains(buf.String(), "status") {
		t.Fatalf("grouped clean record was discarded: %q", buf.String())
	}
}

func TestPIIHandler_Enabled_Delegates(t *testing.T) {
	handler := NewPIIHandler(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn}))

	if handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("Enabled(Info) = true, want false at warn level")
	}
	if !handler.Enabled(context.Background(), slog.LevelError) {
		t.Fatal("Enabled(Error) = false, want true at warn level")
	}
}
