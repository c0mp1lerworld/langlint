package domain

// Window is the aggregation time window for analytics metrics (PRODUCT_DOMAIN
// §4.3). It lives in the root domain package because it is shared by the
// analytics and tutor bounded contexts: the tutor consumes it through the
// WeaknessDetected event (A3). analytics re-exports it as a type alias.
type Window string

// Known time windows. The values are the wire representation.
const (
	WindowDay   Window = "day"
	WindowWeek  Window = "week"
	WindowMonth Window = "month"
)

// IsValid reports whether the window is one of the known values.
func (w Window) IsValid() bool {
	switch w {
	case WindowDay, WindowWeek, WindowMonth:
		return true
	default:
		return false
	}
}

// String returns the wire representation of the window.
func (w Window) String() string {
	return string(w)
}
