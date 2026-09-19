package httpx_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/httpx"
)

func TestUserResolver_InjectsUser(t *testing.T) {
	want := domain.MustNewID()
	var got domain.ID

	handler := httpx.UserResolver(want)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		id, ok := httpx.UserFrom(r.Context())
		if !ok {
			t.Fatal("UserFrom() ok = false, want true")
		}
		got = id
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if got != want {
		t.Fatalf("UserFrom() = %v, want %v", got, want)
	}
}

func TestUserFrom_MissingUser_ReturnsFalse(t *testing.T) {
	if _, ok := httpx.UserFrom(context.Background()); ok {
		t.Fatal("UserFrom(empty) ok = true, want false")
	}
}

func TestNewRouter_Request_IsLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	router := httpx.NewRouter(logger)
	router.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping?secret=user@example.com", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := buf.String()
	if !strings.Contains(body, "http request") {
		t.Fatalf("request was not logged: %q", body)
	}
	if strings.Contains(body, "secret") || strings.Contains(body, "user@example.com") {
		t.Fatalf("log leaked query string: %q", body)
	}
}
