package analytics_test

import (
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

func TestAllWindows_ListsKnownWindowsInCanonicalOrder(t *testing.T) {
	want := []domain.Window{domain.WindowDay, domain.WindowWeek, domain.WindowMonth}

	if len(analytics.AllWindows) != len(want) {
		t.Fatalf("len(AllWindows) = %d, want %d", len(analytics.AllWindows), len(want))
	}
	for i, window := range want {
		if analytics.AllWindows[i] != window {
			t.Fatalf("AllWindows[%d] = %q, want %q", i, analytics.AllWindows[i], window)
		}
	}
}

func TestWindowAlias_ResolvesToRootDomain(t *testing.T) {
	var w analytics.Window = domain.WindowWeek
	if !w.IsValid() {
		t.Fatalf("analytics.Window(%q).IsValid() = false, want true", w)
	}
}
