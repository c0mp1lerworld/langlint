package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestWindow_IsValid_KnownValues_ReturnsTrue(t *testing.T) {
	for _, window := range []domain.Window{
		domain.WindowDay,
		domain.WindowWeek,
		domain.WindowMonth,
	} {
		if !window.IsValid() {
			t.Fatalf("IsValid(%q) = false, want true", window)
		}
	}
}

func TestWindow_IsValid_Unknown_ReturnsFalse(t *testing.T) {
	if domain.Window("year").IsValid() {
		t.Fatal(`IsValid("year") = true, want false`)
	}
}

func TestWindow_String_ReturnsWireValue(t *testing.T) {
	if got := domain.WindowDay.String(); got != "day" {
		t.Fatalf("String() = %q, want %q", got, "day")
	}
}

func TestWindow_MarshalJSON_UsesLowercaseValue(t *testing.T) {
	raw, err := json.Marshal(domain.WindowWeek)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(raw) != `"week"` {
		t.Fatalf("Marshal() = %s, want %q", raw, "week")
	}
}
