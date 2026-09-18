// Package logger provides the shared structured logging helpers of the backend
// (A8). It is built on the standard library log/slog; no third-party logger.
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"unicode/utf8"
)

// maxFreeTextLength is the threshold above which a string value is considered
// free text that may carry PII (names, prompts, user content) and is therefore
// dropped (A8).
const maxFreeTextLength = 100

var (
	emailPattern = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	phonePattern = regexp.MustCompile(
		`(?:\+\d{1,3}(?:[\s.\-]?\(?\d{1,4}\)?){2,})` +
			`|(?:\(\d{1,4}\)[\s.\-]?\d{2,4}(?:[\s.\-]?\d{2,4})?)` +
			`|(?:\b\d{2,3}[\s.\-]\d{2,4}(?:[\s.\-]?\d{2,4})?\b)`,
	)
	apiKeyPattern = regexp.MustCompile(`\b(?:sk|pk)-[A-Za-z0-9_\-]{16,}\b|\bAKIA[0-9A-Z]{16}\b`)
	jwtPattern    = regexp.MustCompile(`\beyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+`)
)

// PIIHandler is a slog.Handler decorator that discards any record whose message
// or attributes carry PII: free text longer than maxFreeTextLength runes,
// emails, phone numbers, API keys or JWTs (A8). It is the runtime guard that
// keeps user content out of the logs.
type PIIHandler struct {
	inner slog.Handler
}

var _ slog.Handler = (*PIIHandler)(nil)

// NewPIIHandler wraps inner so PII-bearing records are dropped.
func NewPIIHandler(inner slog.Handler) *PIIHandler {
	return &PIIHandler{inner: inner}
}

// Enabled reports whether the inner handler is enabled for level.
func (h *PIIHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle drops the record when it carries PII and delegates otherwise.
func (h *PIIHandler) Handle(ctx context.Context, record slog.Record) error {
	if containsPII(record) {
		return nil
	}
	return h.inner.Handle(ctx, record)
}

// WithAttrs strips PII-bearing attributes before they are attached, so a value
// bound with Logger.With can never leak through a later record.
func (h *PIIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		if !attrIsSensitive(attr) {
			clean = append(clean, attr)
		}
	}
	return &PIIHandler{inner: h.inner.WithAttrs(clean)}
}

// WithGroup returns a handler scoped to the named group.
func (h *PIIHandler) WithGroup(name string) slog.Handler {
	return &PIIHandler{inner: h.inner.WithGroup(name)}
}

// containsPII reports whether the message or any attribute carries PII.
func containsPII(record slog.Record) bool {
	if isSensitive(record.Message) {
		return true
	}

	found := false
	record.Attrs(func(attr slog.Attr) bool {
		if attrIsSensitive(attr) {
			found = true
			return false
		}
		return true
	})
	return found
}

// attrIsSensitive inspects an attribute value, recursing into groups.
func attrIsSensitive(attr slog.Attr) bool {
	value := attr.Value.Resolve()

	switch value.Kind() {
	case slog.KindString:
		return isSensitive(value.String())
	case slog.KindGroup:
		for _, nested := range value.Group() {
			if attrIsSensitive(nested) {
				return true
			}
		}
	case slog.KindAny:
		return isSensitive(fmt.Sprint(value.Any()))
	}
	return false
}

// isSensitive reports whether a string value looks like PII.
func isSensitive(value string) bool {
	if utf8.RuneCountInString(value) > maxFreeTextLength {
		return true
	}
	return emailPattern.MatchString(value) ||
		phonePattern.MatchString(value) ||
		apiKeyPattern.MatchString(value) ||
		jwtPattern.MatchString(value)
}
