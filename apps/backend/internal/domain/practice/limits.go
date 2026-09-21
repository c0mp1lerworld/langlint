package practice

// Practice limits keep a session focused on active production instead of an
// unbounded list of rules or a very long text (PRODUCT_DOMAIN §1.1). They are
// enforced by the domain, not only by the UI (A5).
const (
	// maxTextRunes caps the source and draft length in Unicode code points.
	maxTextRunes = 2000
	// maxTargetRules caps how many grammar goals a practice can aim at.
	maxTargetRules = 5
)
