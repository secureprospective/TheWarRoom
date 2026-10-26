package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

// StandUpRosters is an offline recovery operation, not a hosted permission. The
// recovered league is fenced read-only: a roster document cannot grant authority.
func (s *Store) StandUpRosters(ctx context.Context, point commissioner.RawRosterRestorePoint) (string, error) {
	if err := point.Validate(); err != nil {
		return "", fmt.Errorf("check roster recovery: %w", err)
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return "", fmt.Errorf("begin roster recovery: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var occupied bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM league_event UNION ALL SELECT 1 FROM authority_policy UNION ALL SELECT 1 FROM league_import UNION ALL SELECT 1 FROM league_roster_restore)`).Scan(&occupied); err != nil {
		return "", fmt.Errorf("check fresh recovery database: %w", err)
	}
	if occupied {
		return "", fmt.Errorf("roster recovery requires a fresh database")
	}
	if err := db.InstallRosterIdentities(ctx, tx, point.Identities); err != nil {
		return "", fmt.Errorf("install recovered player identities: %w", err)
	}
	records, err := s.standUpRecords(ctx, tx, point)
	if err != nil {
		return "", err
	}
	restored, seeded, err := commissioner.RosterRestorePoint(point.League, records)
	if err != nil {
		return "", fmt.Errorf("verify recovered rosters: %w", err)
	}
	if !seeded || !reflect.DeepEqual(restored.Snapshot, point.Snapshot) {
		return "", fmt.Errorf("recovered rosters and contracts differ from the restore point")
	}
	body, err := json.Marshal(point)
	if err != nil {
		return "", fmt.Errorf("encode recovery source: %w", err)
	}
	const message = "Rosters and contracts restored and checked exactly. Historical results were not restored. This copy is read-only until rules and authority are recovered through the full server restore procedure."
	if _, err := tx.ExecContext(ctx, `INSERT INTO league_import VALUES(?,?,?,?,?)`, point.League, "roster-recovery", "roster-recovery", point.CapturedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), body); err != nil {
		return "", fmt.Errorf("fence recovered league: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO league_recovery_log(league,revision,recorded_at,message) VALUES(?,?,?,?)`, point.League, point.Revision, point.CapturedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), message); err != nil {
		return "", fmt.Errorf("record roster recovery: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit roster recovery: %w", err)
	}
	return message, nil
}

func (s *Store) standUpRecords(ctx context.Context, tx *sql.Tx, point commissioner.RawRosterRestorePoint) ([]commissioner.Record, error) {
	var franchises []string
	for _, f := range point.Snapshot.Franchises {
		franchises = append(franchises, f.ID)
	}
	commands := []commissioner.RawCommand{
		{League: point.League, Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Recovered league", Preset: commissioner.PresetLegacy, Season: point.Snapshot.Season, FranchiseCount: len(franchises), Franchises: franchises}, Request: "recovery-create", Reason: "Recover rosters and contracts", Effective: "now", Visibility: "league"},
		{League: point.League, Command: commissioner.NativeSeed, Args: commissioner.RawArgs{Native: &point.Snapshot}, Expected: 1, Request: "recovery-seed", Reason: "Recover rosters and contracts", Effective: "now", Visibility: "league"},
	}
	var records []commissioner.Record
	for _, command := range commands {
		raw, canonical, err := preparedCommand(command, "offline.recovery", true)
		if err != nil {
			return nil, err
		}
		role, err := commandRole(raw, "offline.recovery", nil, records)
		if err != nil {
			return nil, err
		}
		if err := commissioner.CheckNativeAppend(raw, records); err != nil {
			return nil, fmt.Errorf("check recovery seed: %w", err)
		}
		if _, err := s.appendCommissioner(ctx, tx, appendRequest{raw: raw, principal: "offline.recovery", role: role, canonical: canonical, at: point.CapturedAt, records: records}); err != nil {
			return nil, err
		}
		records, err = s.eventsInTransaction(ctx, tx, point.League)
		if err != nil {
			return nil, err
		}
	}
	return records, nil
}
