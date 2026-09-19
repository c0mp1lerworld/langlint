package handlers

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/ports/mocks"
	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/services"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/logger"
)

func newRunner(t *testing.T, out *bytes.Buffer) (*Runner, *mocks.MockAnalyticsSourceRepository, *mocks.MockErrorMetricRepository, *mocks.MockRawDataRepository, *mocks.MockDeletionRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)

	source := mocks.NewMockAnalyticsSourceRepository(ctrl)
	metrics := mocks.NewMockErrorMetricRepository(ctrl)
	raw := mocks.NewMockRawDataRepository(ctrl)
	deletions := mocks.NewMockDeletionRepository(ctrl)

	refresh := services.NewRefreshAggregatesService(source, metrics)
	purge := services.NewPurgeRawDataService(raw, time.Hour)
	execute := services.NewExecuteDeletionsService(deletions, time.Hour)

	log := logger.New(out, slog.LevelInfo)
	return NewRunner(refresh, purge, execute, log), source, metrics, raw, deletions
}

func TestRunner_Commands_ReturnsAllSubcommands(t *testing.T) {
	runner, _, _, _, _ := newRunner(t, &bytes.Buffer{})

	want := []string{CommandRefreshAggregates, CommandPurgeRawData, CommandExecuteDeletions}
	got := runner.Commands()
	if len(got) != len(want) {
		t.Fatalf("Commands() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Commands()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRunner_Run_UnknownCommand_ReturnsError(t *testing.T) {
	runner, _, _, _, _ := newRunner(t, &bytes.Buffer{})

	if err := runner.Run(context.Background(), "drop-everything"); err == nil {
		t.Fatal("Run() with unknown subcommand: want error, got nil")
	}
}

func TestRunner_Run_RefreshAggregates_DispatchesAndReports(t *testing.T) {
	out := &bytes.Buffer{}
	runner, source, metrics, _, _ := newRunner(t, out)

	source.EXPECT().ListCompletedAnalyses(gomock.Any()).Return(nil, nil)
	metrics.EXPECT().ReplaceAll(gomock.Any(), gomock.Any()).Return(nil)

	if err := runner.Run(context.Background(), CommandRefreshAggregates); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "refresh-aggregates") {
		t.Fatalf("log = %q, want it to mention the job", out.String())
	}
}

func TestRunner_Run_PurgeRawData_DispatchesAndReportsCount(t *testing.T) {
	out := &bytes.Buffer{}
	runner, _, _, raw, _ := newRunner(t, out)

	raw.EXPECT().PurgePracticesDeletedBefore(gomock.Any(), gomock.Any()).Return(3, nil)

	if err := runner.Run(context.Background(), CommandPurgeRawData); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), `"affected":3`) {
		t.Fatalf("log = %q, want affected=3", out.String())
	}
}

func TestRunner_Run_ExecuteDeletions_Dispatches(t *testing.T) {
	out := &bytes.Buffer{}
	runner, _, _, _, deletions := newRunner(t, out)

	deletions.EXPECT().ListPending(gomock.Any()).Return(nil, nil)

	if err := runner.Run(context.Background(), CommandExecuteDeletions); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "execute-deletions") {
		t.Fatalf("log = %q, want it to mention the job", out.String())
	}
}

func TestRunner_Run_JobError_IsReturned(t *testing.T) {
	out := &bytes.Buffer{}
	runner, _, _, raw, _ := newRunner(t, out)

	raw.EXPECT().PurgePracticesDeletedBefore(gomock.Any(), gomock.Any()).Return(0, errors.New("delete failed"))

	if err := runner.Run(context.Background(), CommandPurgeRawData); err == nil {
		t.Fatal("Run() with failing job: want error, got nil")
	}
	if !strings.Contains(out.String(), "job failed") {
		t.Fatalf("log = %q, want a failure record", out.String())
	}
}
