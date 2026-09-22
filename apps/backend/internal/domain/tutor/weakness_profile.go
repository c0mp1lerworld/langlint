package tutor

import (
	"sort"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// WeaknessEntry is one aggregated historical error pattern for a learner
// (PRODUCT_DOMAIN §12.1). It groups the frequency of a single error code, so the
// tutor consumes aggregated analytics, never raw practice text (8.1.4).
type WeaknessEntry struct {
	Code       domain.ErrorPatternCode     `json:"code"`
	Severity   domain.ErrorPatternSeverity `json:"severity"`
	Count      int                         `json:"count"`
	LastSeenAt time.Time                   `json:"last_seen_at"`
}

// NewWeaknessEntry validates code, severity, count and lastSeenAt and returns an
// entry with its timestamp normalized to UTC.
func NewWeaknessEntry(code domain.ErrorPatternCode, severity domain.ErrorPatternSeverity, count int, lastSeenAt time.Time) (WeaknessEntry, error) {
	entry := WeaknessEntry{Code: code, Severity: severity, Count: count, LastSeenAt: lastSeenAt.UTC()}
	if err := entry.validate(); err != nil {
		return WeaknessEntry{}, err
	}
	return entry, nil
}

// validate checks the invariants of a single entry. NewWeaknessProfile reuses it
// so entries passed as struct literals cannot skip validation.
func (e WeaknessEntry) validate() error {
	if !e.Code.IsValid() {
		return &domain.ValidationError{Field: "weakness.code", Message: "must be a known error pattern code"}
	}
	if !e.Severity.IsValid() {
		return &domain.ValidationError{Field: "weakness.severity", Message: "must be a known error pattern severity"}
	}
	if e.Count <= 0 {
		return &domain.ValidationError{Field: "weakness.count", Message: "must be positive"}
	}
	if e.LastSeenAt.IsZero() {
		return &domain.ValidationError{Field: "weakness.last_seen_at", Message: "must not be zero"}
	}
	return nil
}

// WeaknessProfile groups the learner's historical error patterns into the
// aggregated input the tutor consumes (checklist 8.1.3). It is a value object:
// it never holds raw practice or analysis data (8.1.4).
type WeaknessProfile struct {
	Entries []WeaknessEntry `json:"entries"`
}

// NewWeaknessProfile validates that entries is non-empty and every entry is
// valid, and returns a profile sorted by descending frequency so Weakest is
// deterministic.
func NewWeaknessProfile(entries []WeaknessEntry) (WeaknessProfile, error) {
	profile := WeaknessProfile{Entries: make([]WeaknessEntry, len(entries))}
	copy(profile.Entries, entries)
	if err := profile.validate(); err != nil {
		return WeaknessProfile{}, err
	}
	sort.SliceStable(profile.Entries, func(i, j int) bool {
		return profile.Entries[i].Count > profile.Entries[j].Count
	})
	return profile, nil
}

// validate checks the invariants of the profile as a whole.
func (p WeaknessProfile) validate() error {
	if len(p.Entries) == 0 {
		return &domain.ValidationError{Field: "weakness_profile.entries", Message: "must not be empty"}
	}
	for _, entry := range p.Entries {
		if err := entry.validate(); err != nil {
			return err
		}
	}
	return nil
}

// Weakest returns the entry with the highest frequency. It is safe because the
// profile is non-empty and sorted by descending frequency by construction.
func (p WeaknessProfile) Weakest() WeaknessEntry {
	return p.Entries[0]
}
