// Package httpx holds the shared HTTP primitives of the backend: the base chi
// router, global middlewares and the single-user resolver of the MVP. It has no
// knowledge of the API routes (the generated handler registers them).
package httpx

import (
	"context"
	"net/http"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// userIDKey is the private context key used to carry the resolved user.
type userIDKey struct{}

// UserResolver injects the configured single-user identity into every request
// context. The MVP wire contract has no authentication (PRODUCT_DOMAIN §3.2),
// so the current user comes from configuration instead of a token.
func UserResolver(userID domain.ID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), userIDKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFrom returns the identity injected by UserResolver. ok is false when the
// middleware did not run.
func UserFrom(ctx context.Context) (domain.ID, bool) {
	id, ok := ctx.Value(userIDKey{}).(domain.ID)
	return id, ok
}
