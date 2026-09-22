package analytics

import (
	"sort"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// WeaknessThreshold is the default frequency threshold that flags an error
// pattern as a weakness to tutor (checklist 8.2.3). A pattern is detected when
// its materialized count reaches this value.
const WeaknessThreshold = 5

// DetectWeakness returns the error codes whose frequency reached the threshold,
// ordered by code for determinism. It is the pure frequency rule of checklist
// 8.2.3: it takes the aggregated totals for a window and a threshold and yields
// the codes that must be flagged to the tutor.
func DetectWeakness(frequencies map[domain.ErrorPatternCode]int, threshold int) []domain.ErrorPatternCode {
	codes := make([]domain.ErrorPatternCode, 0)
	for code, count := range frequencies {
		if count >= threshold {
			codes = append(codes, code)
		}
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes
}
