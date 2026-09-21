package analytics

import (
	"sort"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// ProgressMetric is one point of the temporal progress series for a user
// (PRODUCT_DOMAIN §4.2.4). It aggregates a single period (period_start) within
// the requested window; Accuracy is derived from the raw counts.
type ProgressMetric struct {
	UserID         domain.ID `json:"user_id"`
	Window         Window    `json:"window"`
	PeriodStart    time.Time `json:"period_start"`
	TotalFragments int       `json:"total_fragments"`
	ErrorCount     int       `json:"error_count"`
	Accuracy       float64   `json:"accuracy"`
}

// ProgressSample is a raw, per-analysis contribution to the progress series: the
// number of fragments and errors the analysis produced at a given time. It is
// pure input for BuildProgressSeries, which buckets it into a window.
type ProgressSample struct {
	CompletedAt    time.Time
	TotalFragments int
	ErrorCount     int
}

// NewProgressMetric creates a progress metric for one period, deriving Accuracy
// from the counts.
func NewProgressMetric(userID domain.ID, window Window, periodStart time.Time, totalFragments, errorCount int) (*ProgressMetric, error) {
	if !window.IsValid() {
		return nil, &domain.ValidationError{Field: "window", Message: "must be a known window"}
	}
	if periodStart.IsZero() {
		return nil, &domain.ValidationError{Field: "period_start", Message: "must not be zero"}
	}
	if totalFragments < 0 {
		return nil, &domain.ValidationError{Field: "total_fragments", Message: "must not be negative"}
	}
	if errorCount < 0 {
		return nil, &domain.ValidationError{Field: "error_count", Message: "must not be negative"}
	}
	return &ProgressMetric{
		UserID:         userID,
		Window:         window,
		PeriodStart:    periodStart.UTC(),
		TotalFragments: totalFragments,
		ErrorCount:     errorCount,
		Accuracy:       accuracy(totalFragments, errorCount),
	}, nil
}

// BucketPeriod truncates a timestamp to the start of its period for the given
// window: the calendar day for day, the Monday of the ISO week for week and the
// first of the month for month. All buckets are computed in UTC so the series is
// deterministic regardless of the caller's location.
func BucketPeriod(t time.Time, window Window) time.Time {
	t = t.UTC()
	year, month, day := t.Date()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	switch window {
	case WindowWeek:
		offset := (int(startOfDay.Weekday()) + 6) % 7 // Monday == 0
		return startOfDay.AddDate(0, 0, -offset)
	case WindowMonth:
		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	default:
		return startOfDay
	}
}

// BuildProgressSeries buckets the samples into the given window and aggregates
// them per period, ordered from oldest to newest. Only periods that have at
// least one sample are emitted.
func BuildProgressSeries(userID domain.ID, window Window, samples []ProgressSample) ([]ProgressMetric, error) {
	if !window.IsValid() {
		return nil, &domain.ValidationError{Field: "window", Message: "must be a known window"}
	}

	type totals struct {
		totalFragments int
		errorCount     int
	}
	buckets := make(map[time.Time]*totals)
	periods := make([]time.Time, 0)
	for _, sample := range samples {
		period := BucketPeriod(sample.CompletedAt, window)
		aggregate, ok := buckets[period]
		if !ok {
			aggregate = &totals{}
			buckets[period] = aggregate
			periods = append(periods, period)
		}
		aggregate.totalFragments += sample.TotalFragments
		aggregate.errorCount += sample.ErrorCount
	}

	sort.Slice(periods, func(i, j int) bool { return periods[i].Before(periods[j]) })

	metrics := make([]ProgressMetric, 0, len(periods))
	for _, period := range periods {
		aggregate := buckets[period]
		metric, err := NewProgressMetric(userID, window, period, aggregate.totalFragments, aggregate.errorCount)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, *metric)
	}
	return metrics, nil
}

// accuracy returns 1 - errorCount/totalFragments clamped to [0, 1]. With no
// fragments there is nothing to be wrong about, so accuracy is 1.
func accuracy(totalFragments, errorCount int) float64 {
	if totalFragments == 0 {
		return 1
	}
	value := 1 - float64(errorCount)/float64(totalFragments)
	if value < 0 {
		return 0
	}
	return value
}
