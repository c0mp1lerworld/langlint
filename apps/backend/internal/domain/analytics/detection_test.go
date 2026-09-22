package analytics_test

import (
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
)

func TestDetectWeakness_EmptyFrequencies_ReturnsEmpty(t *testing.T) {
	got := analytics.DetectWeakness(nil, analytics.WeaknessThreshold)
	if len(got) != 0 {
		t.Fatalf("DetectWeakness(nil) = %v, want empty", got)
	}
}

func TestDetectWeakness_BelowThreshold_ReturnsEmpty(t *testing.T) {
	freq := map[domain.ErrorPatternCode]int{
		domain.ErrorPatternCodeWordOrder: 4,
	}
	got := analytics.DetectWeakness(freq, analytics.WeaknessThreshold)
	if len(got) != 0 {
		t.Fatalf("DetectWeakness(below) = %v, want empty", got)
	}
}

func TestDetectWeakness_AtThreshold_ReturnsCode(t *testing.T) {
	freq := map[domain.ErrorPatternCode]int{
		domain.ErrorPatternCodeWordOrder: analytics.WeaknessThreshold,
	}
	got := analytics.DetectWeakness(freq, analytics.WeaknessThreshold)
	if len(got) != 1 || got[0] != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("DetectWeakness(at) = %v, want [word_order]", got)
	}
}

func TestDetectWeakness_AboveThreshold_ReturnsCode(t *testing.T) {
	freq := map[domain.ErrorPatternCode]int{
		domain.ErrorPatternCodeWordOrder: analytics.WeaknessThreshold + 2,
	}
	got := analytics.DetectWeakness(freq, analytics.WeaknessThreshold)
	if len(got) != 1 || got[0] != domain.ErrorPatternCodeWordOrder {
		t.Fatalf("DetectWeakness(above) = %v, want [word_order]", got)
	}
}

func TestDetectWeakness_MultipleCodes_ReturnsSortedByCode(t *testing.T) {
	freq := map[domain.ErrorPatternCode]int{
		domain.ErrorPatternCodeWordOrder:             10,
		domain.ErrorPatternCodePrepositionInfinitive: 7,
		domain.ErrorPatternCodeFalseFriend:           2,
	}
	got := analytics.DetectWeakness(freq, analytics.WeaknessThreshold)

	want := []domain.ErrorPatternCode{
		domain.ErrorPatternCodePrepositionInfinitive,
		domain.ErrorPatternCodeWordOrder,
	}
	if len(got) != len(want) {
		t.Fatalf("DetectWeakness() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DetectWeakness()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
