//go:build integration

package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	httpapi "github.com/c0mp1lerworld/langlint/backend/internal/api/handlers"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestMeEndpoints_ExportDeleteAccessLog(t *testing.T) {
	pool := startPool(t)
	userID := domain.MustNewID()
	seedPractice(t, pool, userID, nil)

	api := httptest.NewServer(newRouter(t, pool, userID))
	defer api.Close()

	// [1] GET /me/data/export returns the configured identity and practices.
	resp, err := http.Get(api.URL + "/me/data/export")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var export httpapi.DataExport
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&export))
	resp.Body.Close()

	require.Equal(t, "student@example.com", string(export.Email))
	require.Equal(t, userID.String(), export.UserId.String())
	require.Len(t, export.Practices, 1)
	require.False(t, export.GeneratedAt.IsZero())

	// [2] The access log records the export request.
	resp, err = http.Get(api.URL + "/me/access-log")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var log httpapi.AccessLog
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&log))
	resp.Body.Close()

	require.GreaterOrEqual(t, log.Total, 1)
	foundExport := false
	for _, item := range log.Items {
		if item.ResourceType != nil && *item.ResourceType == "/me/data/export" {
			foundExport = true
		}
	}
	require.True(t, foundExport, "access log must include the export access")

	// [3] DELETE /me/data is idempotent: two calls, one pending request.
	for i := 0; i < 2; i++ {
		req, err := http.NewRequest(http.MethodDelete, api.URL+"/me/data", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, resp.StatusCode)
		resp.Body.Close()
	}
	require.Equal(t, 1, countRows(t, pool,
		`SELECT count(*) FROM deletion_requests WHERE user_id = $1 AND executed_at IS NULL`, userID.String()))
}
