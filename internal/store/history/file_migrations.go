package history

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// League numbering continues from the last combined schema so older servers still refuse
// future league files. Only the league's own steps follow this baseline.
const leagueMigrationBase = 13

func LeagueSchemaVersion() int    { return leagueMigrationBase + len(leagueMigrations()) }
func ReferenceSchemaVersion() int { return len(referenceMigrations()) }

func leagueMigrations() []db.Migration    { return []db.Migration{createLeagueFile, addLeagueIdentity} }
func referenceMigrations() []db.Migration { return []db.Migration{createReferenceFile} }

func createLeagueFile(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, eventsDDL+appendOnlyDDL("league_event")); err != nil {
		return fmt.Errorf("create league events: %w", err)
	}
	for _, step := range []db.Migration{guardLeagueJSON, addClosedBooks, initializeLedgerAudit,
		addInactiveImports, addTransferAuthority, db.CreateServerRelease, db.CreateLeagueRecovery, db.CreateLeagueOwnership} {
		if err := step(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

func guardLeagueJSON(ctx context.Context, tx *sql.Tx) error {
	if err := db.GuardJSON(ctx, tx, db.JSONColumn{Table: "league_event", Column: "payload", Versioned: true}); err != nil {
		return fmt.Errorf("guard league payload: %w", err)
	}
	return nil
}

func createReferenceFile(ctx context.Context, tx *sql.Tx) error {
	var ddl strings.Builder
	ddl.WriteString(historyDDL)
	ddl.WriteString(registryDDL)
	ddl.WriteString(quarantineDDL)
	ddl.WriteString(featureViewsDDL())
	for _, table := range appendOnly() {
		if table != "league_event" {
			ddl.WriteString(appendOnlyDDL(table))
		}
	}
	for _, table := range []string{"quarantine_model_runs", "quarantine_model_scores"} {
		ddl.WriteString(appendOnlyDDL(table))
	}
	if _, err := tx.ExecContext(ctx, ddl.String()); err != nil {
		return fmt.Errorf("create reference schema: %w", err)
	}
	for _, step := range []db.Migration{db.CreateReferenceIdentity, guardReferenceJSON, db.GuardPlayerEvidence,
		db.CreatePlayerRecord, db.AddPlayerCareer, addLiveEvidence, db.CreateServerRelease} {
		if err := step(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}
