package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/shared/httpx"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORS_AllowedOrigin_SetsAllowOrigin(t *testing.T) {
	handler := httpx.CORS([]string{"http://localhost:3000"})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/practices", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want origin", got)
	}
}

func TestCORS_DisallowedOrigin_DoesNotSetAllowOrigin(t *testing.T) {
	handler := httpx.CORS([]string{"http://localhost:3000"})(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/practices", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestCORS_Preflight_ReturnsNoContentWithHeaderList(t *testing.T) {
	var reached bool
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })
	handler := httpx.CORS([]string{"http://localhost:3000"})(next)

	req := httptest.NewRequest(http.MethodOptions, "/practices", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if reached {
		t.Fatal("preflight reached the next handler")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, PATCH, PUT, DELETE, OPTIONS" {
		t.Fatalf("Access-Control-Allow-Methods = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("Access-Control-Allow-Headers is empty")
	}
}

func TestCORS_NoOrigin_PassesThroughWithoutHeaders(t *testing.T) {
	handler := httpx.CORS([]string{"http://localhost:3000"})(okHandler())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/practices", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestCORS_EmptyAllowList_DoesNotSetAllowOrigin(t *testing.T) {
	handler := httpx.CORS(nil)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/practices", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}
