// Package accesslog records the A9 access audit: every authenticated request is
// appended as an immutable AccessEvent (A4, A9). It lives in the API entry point
// because it depends on an API storage port, so it is deliberately not part of
// shared/httpx.
package accesslog

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/storage"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/httpx"
)

// Middleware appends one access event per authenticated request.
type Middleware struct {
	repo storage.AccessLogRepository
	log  *slog.Logger
	now  func() time.Time
}

// NewMiddleware wires the audit recorder through its port.
func NewMiddleware(repo storage.AccessLogRepository, log *slog.Logger) *Middleware {
	return &Middleware{repo: repo, log: log, now: time.Now}
}

// Wrap records the event after the handler runs, so the response is never
// delayed by the audit write. A failed append is logged but never fails the
// request: the audit is best-effort and the response is already sent.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		userID, ok := httpx.UserFrom(r.Context())
		if !ok {
			return
		}

		event, err := identity.NewAccessEvent(userID, r.Method, r.URL.Path, "", m.now())
		if err != nil {
			m.log.Warn("access log: cannot build event", "error", err)
			return
		}
		if err := m.repo.Append(r.Context(), event); err != nil {
			m.log.Warn("access log: cannot append event", "error", err)
		}
	})
}
