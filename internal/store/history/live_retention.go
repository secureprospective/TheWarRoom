package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// Once a final counts, discard only covered, superseded captures that were never exposed
// as revisions. All exposed chronology is retained: even undone approvals can be replayed.
func (s *Store) compactLiveStats(ctx context.Context, league string, records []commissioner.Record, weeks []commissioner.StatWeek, season int) error {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	for _, week := range weeks {
		key := liveStatKey{week.Player, season, week.Week}
		if _, ok := s.liveHeld[key]; !ok {
			continue
		}
		final := commissioner.FeedVersion(week.Evidence.Versions())
		if final == nil || final.Held.Authority != source.NFLVerse {
			continue
		}
		scope := source.RawNativeScope{Player: week.Player, Season: season, Week: week.Week}
		counted, err := commissioner.NativePlayerScoreReading(league, scope, records, week.Evidence, s.now())
		if err != nil {
			return fmt.Errorf("check counted final for retention: %w", err)
		}
		if counted.SHA256 != final.Held.SHA256 || counted.Points == nil {
			continue
		}
		if err := s.compactLivePlayer(ctx, key, *final.Held.Week); err != nil {
			return err
		}
		s.bumpLiveWeekLocked(season, week.Week)
	}
	return nil
}

func (s *Store) compactLivePlayer(ctx context.Context, key liveStatKey, final source.PlayerWeek) error {
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("begin live retention: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	ids, err := liveRetentionCandidates(ctx, tx, key, final)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	// The exception is transaction-scoped: rollback restores both the rows and their guard.
	if _, err := tx.ExecContext(ctx, `DROP TRIGGER live_evidence_no_delete`); err != nil {
		return fmt.Errorf("prepare live retention: %w", err)
	}
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `DELETE FROM live_evidence WHERE id=?`, id); err != nil {
			return fmt.Errorf("compact live evidence: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, appendOnlyDDL("live_evidence")); err != nil {
		return fmt.Errorf("restore live evidence guard: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit live retention: %w", err)
	}
	return nil
}

func finalCoversLive(final, live source.PlayerWeek) bool {
	for id := range live.Line.Quantities {
		if _, ok := final.Line.Quantities[id]; !ok {
			return false
		}
		if owner := final.FieldSources[id]; owner != "" && owner != source.NFLVerse {
			return false
		}
	}
	for id := range live.Line.Events {
		if _, ok := final.Line.Events[id]; !ok {
			return false
		}
		if owner := final.FieldSources[id]; owner != "" && owner != source.NFLVerse {
			return false
		}
	}
	return true
}

func liveRetentionCandidates(ctx context.Context, tx *sql.Tx, key liveStatKey, final source.PlayerWeek) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT e.id, e.fields FROM live_evidence e
WHERE e.player=? AND e.season=? AND e.week=?
AND e.id < (SELECT MAX(id) FROM live_evidence WHERE player=e.player AND season=e.season AND week=e.week)
AND NOT EXISTS (SELECT 1 FROM live_board b WHERE b.evidence_id=e.id)
AND NOT EXISTS (SELECT 1 FROM live_chronology c WHERE c.player=e.player AND c.season=e.season AND c.week=e.week
 AND json_extract(c.revision,'$.Held.SHA256')=e.digest)`, key.player, key.season, key.week)
	if err != nil {
		return nil, fmt.Errorf("find live retention candidates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var ids []int64
	for rows.Next() {
		var id int64
		var body string
		var held source.HeldWeek
		if err := rows.Scan(&id, &body); err != nil {
			return nil, fmt.Errorf("read live retention candidate: %w", err)
		}
		if err := json.Unmarshal([]byte(body), &held); err != nil {
			return nil, fmt.Errorf("decode live retention candidate: %w", err)
		}
		if held.Week != nil && finalCoversLive(final, *held.Week) {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read live retention candidates: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close live retention candidates: %w", err)
	}
	return ids, nil
}
