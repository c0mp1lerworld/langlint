package analytics

import "github.com/c0mp1lerworld/langlint/backend/internal/domain"

// Window is the aggregation time window for analytics metrics (PRODUCT_DOMAIN
// §4.3). It is defined in the root domain package because the tutor bounded
// context also consumes it through the WeaknessDetected event (A3); analytics
// re-exports it as a type alias for convenience.
type Window = domain.Window

// Known time windows, aliased from the root domain package.
const (
	WindowDay   = domain.WindowDay
	WindowWeek  = domain.WindowWeek
	WindowMonth = domain.WindowMonth
)

// AllWindows lists the known windows in canonical order. It is the single
// source of the windows to aggregate over.
var AllWindows = []Window{WindowDay, WindowWeek, WindowMonth}
