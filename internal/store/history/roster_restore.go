package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
)

func addRosterRestorePoints(ctx context.Context, tx *sql.Tx) error {
	if err := db.CreateLeagueRecovery(ctx, tx); err != nil {
		return fmt.Errorf("prepare league roster recovery: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT league FROM league_event WHERE stream=? ORDER BY league`, commissioner.Stream)
	if err != nil {
		return fmt.Errorf("find leagues for roster recovery: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var leagues []string
	for rows.Next() {
		var league string
		if err := rows.Scan(&league); err != nil {
			return fmt.Errorf("read recovery league: %w", err)
		}
		leagues = append(leagues, league)
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return fmt.Errorf("read recovery leagues: %w", err)
	}
	store := &Store{}
	for _, league := range leagues {
		if err := store.captureRosterRestorePoint(ctx, tx, league); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) captureRosterRestorePoint(ctx context.Context, tx *sql.Tx, league string) error {
	records, err := s.eventsInTransaction(ctx, tx, league)
	if err != nil {
		return err
	}
	return saveRosterRestorePoint(ctx, tx, league, records)
}

func saveRosterRestorePoint(ctx context.Context, tx *sql.Tx, league string, records []commissioner.Record) error {
	point, seeded, err := commissioner.RosterRestorePoint(league, records)
	if err != nil {
		return fmt.Errorf("capture roster restore point: %w", err)
	}
	if !seeded {
		return nil
	}
	if err := saveRosterIdentity(ctx, tx, &point); err != nil {
		return err
	}
	body, err := json.Marshal(point)
	if err != nil {
		return fmt.Errorf("encode roster restore point: %w", err)
	}
	var unchanged bool
	err = tx.QueryRowContext(ctx, `SELECT json_extract(payload,'$.snapshot')=json_extract(?1,'$.snapshot') AND json_extract(payload,'$.identities') IS json_extract(?1,'$.identities')
 FROM league_roster_restore WHERE league=?`, string(body), league).Scan(&unchanged)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("compare league roster restore point: %w", err)
	}
	if unchanged {
		return nil
	}
	if err := db.SaveLeagueRestorePoint(ctx, tx, league, point.Revision, point.CapturedAt.Format(time.RFC3339Nano), string(body)); err != nil {
		return fmt.Errorf("persist roster recovery: %w", err)
	}
	return nil
}

// RosterRestorePoint reads validated private recovery data. Transport callers must
// authorize league access before exposing it, just as for transfer documents.
func (s *Store) RosterRestorePoint(ctx context.Context, league string) (commissioner.RawRosterRestorePoint, error) {
	var point commissioner.RawRosterRestorePoint
	var body string
	var revision int64
	if err := s.pools.Read().QueryRowContext(ctx, `SELECT revision,payload FROM league_roster_restore WHERE league=?`, league).Scan(&revision, &body); err != nil {
		return point, fmt.Errorf("read league roster restore point: %w", err)
	}
	if err := json.Unmarshal([]byte(body), &point); err != nil {
		return point, fmt.Errorf("decode league roster restore point: %w", err)
	}
	if err := point.Validate(); err != nil {
		return point, fmt.Errorf("validate stored roster restore point: %w", err)
	}
	if point.League != league || point.Revision != revision {
		return point, fmt.Errorf("league roster restore point does not match its saved identity")
	}
	return point, nil
}

func saveRosterIdentity(ctx context.Context, tx *sql.Tx, point *commissioner.RawRosterRestorePoint) error {
	var installed bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE name='canonical_player')`).Scan(&installed); err != nil {
		return fmt.Errorf("check player identity schema: %w", err)
	}
	if !installed {
		return nil
	}
	owners := make([]db.RosterOwner, 0, len(point.Snapshot.Players))
	for _, player := range point.Snapshot.Players {
		owners = append(owners, db.RosterOwner{MFL: player.ID, Side: point.Snapshot.SideOf(player.Franchise),
			Franchise: player.Franchise})
	}
	if err := db.ReplaceRosterOwnership(ctx, tx, point.League, owners); err != nil {
		return fmt.Errorf("save league ownership: %w", err)
	}
	identities, err := db.RosterIdentities(ctx, tx, point.League)
	if err != nil {
		return fmt.Errorf("capture league identities: %w", err)
	}
	point.Identities = identities
	if err := point.Validate(); err != nil {
		return fmt.Errorf("validate captured identities: %w", err)
	}
	return nil
}
