// Package handlers holds the CLI subcommands of the provisioner entry point
// (PRODUCT_DOMAIN §7.2). It is the provisioner counterpart of the api handlers:
// it dispatches a subcommand to its batch service and never touches SQL.
package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/c0mp1lerworld/langlint/backend/internal/provisioner/services"
)

// Subcommand names of the provisioner batch jobs.
const (
	CommandRefreshAggregates = "refresh-aggregates"
	CommandPurgeRawData      = "purge-raw-data"
	CommandExecuteDeletions  = "execute-deletions"
)

// CommandNames returns the subcommand names the provisioner understands.
func CommandNames() []string {
	return []string{CommandRefreshAggregates, CommandPurgeRawData, CommandExecuteDeletions}
}

// Runner dispatches a provisioner subcommand to its batch service.
type Runner struct {
	refresh *services.RefreshAggregatesService
	purge   *services.PurgeRawDataService
	execute *services.ExecuteDeletionsService
	log     *slog.Logger
}

// NewRunner wires the subcommands through the batch services.
func NewRunner(
	refresh *services.RefreshAggregatesService,
	purge *services.PurgeRawDataService,
	execute *services.ExecuteDeletionsService,
	log *slog.Logger,
) *Runner {
	return &Runner{refresh: refresh, purge: purge, execute: execute, log: log}
}

// Commands returns the known subcommands.
func (r *Runner) Commands() []string {
	return CommandNames()
}

// Run executes the given subcommand and reports how many rows it affected. An
// unknown subcommand is rejected before any job runs.
func (r *Runner) Run(ctx context.Context, command string) error {
	switch command {
	case CommandRefreshAggregates:
		affected, err := r.refresh.Refresh(ctx)
		return r.report(command, affected, err)
	case CommandPurgeRawData:
		affected, err := r.purge.Purge(ctx)
		return r.report(command, affected, err)
	case CommandExecuteDeletions:
		affected, err := r.execute.Run(ctx)
		return r.report(command, affected, err)
	default:
		return fmt.Errorf("unknown subcommand %q (want %v)", command, r.Commands())
	}
}

// report logs the affected count, or returns the job error unchanged.
func (r *Runner) report(command string, affected int, err error) error {
	if err != nil {
		r.log.Error("job failed", "job", command, "error", err)
		return err
	}
	r.log.Info("job completed", "job", command, "affected", affected)
	return nil
}
