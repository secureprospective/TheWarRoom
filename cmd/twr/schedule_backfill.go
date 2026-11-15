package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/spf13/cobra"
)

// scheduleBackfillCommand is an explicit offline-profile migration, not an application startup write.
func (e env) scheduleBackfillCommand() *cobra.Command {
	var database, league, source, principal, reason string
	var season int
	command := &cobra.Command{Use: "schedule-backfill", Short: "Read MFL's schedule into an existing app league (close the app first)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateScheduleBackfill(database, league, source, principal, reason, season); err != nil {
				return err
			}
			info, err := os.Lstat(database)
			if err != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("choose an existing league database, not a link or missing file")
			}
			client, err := mfl.New("", 1)
			if err != nil {
				return fmt.Errorf("open read-only MFL client: %w", err)
			}
			weeks, err := leagueschedule.Fetch(cmd.Context(), client, strconv.Itoa(season), source)
			if err != nil {
				return fmt.Errorf("read schedule before changing league: %w", err)
			}
			pools, err := db.Open(cmd.Context(), database, history.SchemaVersion())
			if err != nil {
				return fmt.Errorf("open schedule destination: %w", err)
			}
			defer func() { _ = pools.Close() }()
			registry, err := measures.Embedded()
			if err != nil {
				return fmt.Errorf("load measures: %w", err)
			}
			store := history.New(pools, registry)
			receipts, err := mflsource.BackfillSchedule(cmd.Context(), leagueservice.MigrationLedger{Store: store}, league, reason, principal, season, weeks)
			if err != nil {
				return fmt.Errorf("backfill existing league: %w", err)
			}
			if _, err = fmt.Fprintf(e.out, "Schedule %s: %d weeks read; %d ledger changes (0 means unchanged).\n", league, len(weeks), len(receipts)); err != nil {
				return fmt.Errorf("report schedule: %w", err)
			}
			return nil
		}}
	command.Flags().StringVar(&database, "db", "", "existing history.db (use a profile copy first)")
	command.Flags().StringVar(&league, "league", "", "destination app league ID")
	command.Flags().StringVar(&source, "mfl-league", "", "source MFL league number")
	command.Flags().IntVar(&season, "season", 0, "source season")
	command.Flags().StringVar(&principal, "principal", "", "recorded commissioner identity")
	command.Flags().StringVar(&reason, "reason", "", "reason recorded in the league ledger")
	return command
}

func validateScheduleBackfill(database, league, source, principal, reason string, season int) error {
	if database == "" || league == "" || source == "" || principal == "" || reason == "" || season < 1920 || season > 9999 {
		return fmt.Errorf("supply --db, --league, --mfl-league, --season, --principal and --reason")
	}
	return nil
}
