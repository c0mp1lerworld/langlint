package accesslog

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/httpx"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func okHandler(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func TestMiddleware_Wrap_RecordsAuthenticatedRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAccessLogRepository(ctrl)
	userID := domain.MustNewID()
	now := time.Date(2026, time.September, 19, 12, 0, 0, 0, time.UTC)

	m := NewMiddleware(repo, discardLogger())
	m.now = func() time.Time { return now }

	repo.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, event *identity.AccessEvent) error {
			if event.UserID != userID {
				t.Fatalf("UserID = %s, want %s", event.UserID, userID)
			}
			if event.Action != http.MethodGet {
				t.Fatalf("Action = %q, want GET", event.Action)
			}
			if event.ResourceType != "/practices" {
				t.Fatalf("ResourceType = %q, want /practices", event.ResourceType)
			}
			if !event.OccurredAt.Equal(now) {
				t.Fatalf("OccurredAt = %v, want %v", event.OccurredAt, now)
			}
			return nil
		},
	)

	handler := httpx.UserResolver(userID)(m.Wrap(http.HandlerFunc(okHandler)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/practices", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestMiddleware_Wrap_NoUser_DoesNotRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAccessLogRepository(ctrl)
	// No Append expected.

	m := NewMiddleware(repo, discardLogger())
	rec := httptest.NewRecorder()
	m.Wrap(http.HandlerFunc(okHandler)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/practices", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestMiddleware_Wrap_ZeroUser_DoesNotRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAccessLogRepository(ctrl)
	// A zero user id makes NewAccessEvent fail, so no Append is expected.

	m := NewMiddleware(repo, discardLogger())
	handler := httpx.UserResolver(domain.ID{})(m.Wrap(http.HandlerFunc(okHandler)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/practices", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestMiddleware_Wrap_AppendError_DoesNotFailRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAccessLogRepository(ctrl)
	userID := domain.MustNewID()

	repo.EXPECT().Append(gomock.Any(), gomock.Any()).Return(errors.New("db down"))

	m := NewMiddleware(repo, discardLogger())
	handler := httpx.UserResolver(userID)(m.Wrap(http.HandlerFunc(okHandler)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/practices", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (best-effort audit)", rec.Code)
	}
}
