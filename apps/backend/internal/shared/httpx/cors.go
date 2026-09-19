package httpx

import "net/http"

// corsAllowMethods lists the HTTP methods exposed by the API contract.
const corsAllowMethods = "GET, POST, PATCH, PUT, DELETE, OPTIONS"

// corsAllowHeaders lists the request headers the browser may send.
const corsAllowHeaders = "Content-Type, Authorization"

// CORS returns a middleware that allows cross-origin requests from the given
// origins (F3: the frontend talks to the backend over HTTP from another origin).
// It answers preflight (OPTIONS) requests directly so they never reach the
// router. An empty allow-list disables cross-origin access.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; origin == "" || !ok {
				next.ServeHTTP(w, r)
				return
			}

			header := w.Header()
			header.Set("Access-Control-Allow-Origin", origin)
			header.Add("Vary", "Origin")

			if r.Method == http.MethodOptions {
				header.Set("Access-Control-Allow-Methods", corsAllowMethods)
				header.Set("Access-Control-Allow-Headers", corsAllowHeaders)
				header.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
