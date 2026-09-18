package db_test

import (
	"context"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/shared/db"
)

func TestNewPool_InvalidURL_ReturnsError(t *testing.T) {
	if _, err := db.NewPool(context.Background(), "://not-a-dsn"); err == nil {
		t.Fatal("NewPool(invalid) error = nil, want error")
	}
}
