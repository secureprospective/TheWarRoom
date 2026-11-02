package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

func addLiveEvidence(ctx context.Context, tx *sql.Tx) error {
	const ddl = `
CREATE TABLE live_evidence (
 id INTEGER PRIMARY KEY, player TEXT NOT NULL, season INTEGER NOT NULL, week INTEGER NOT NULL,
 game TEXT NOT NULL, source TEXT NOT NULL CHECK(source IN ('espn','sleeper')), digest TEXT NOT NULL, fetched_at TEXT NOT NULL,
 fields TEXT NOT NULL CHECK(json_valid(fields))
);
CREATE INDEX live_evidence_scope ON live_evidence(player, season, week, id);
CREATE TABLE live_chronology (
 player TEXT NOT NULL, season INTEGER NOT NULL, week INTEGER NOT NULL, seq INTEGER NOT NULL,
 origin_seq INTEGER NOT NULL, revision TEXT NOT NULL CHECK(json_valid(revision)),
 PRIMARY KEY(player, season, week, seq)
);
CREATE TABLE live_board (player TEXT PRIMARY KEY, evidence_id INTEGER NOT NULL REFERENCES live_evidence(id));
`
	if _, err := tx.ExecContext(ctx, ddl+appendOnlyDDL("live_evidence")+appendOnlyDDL("live_chronology")); err != nil {
		return fmt.Errorf("create live evidence: %w", err)
	}
	return nil
}

// RestoreLiveStats restores the held book and the exact chronology before scoring starts.
// It is also called lazily by evidence readers used outside the desktop startup path.
func (s *Store) RestoreLiveStats(ctx context.Context) error {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	return s.restoreLiveStats(ctx)
}

func (s *Store) restoreLiveStats(ctx context.Context) error {
	if s.liveRevisions != nil || s.pools == nil {
		return nil
	}
	tx, err := s.pools.Read().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("begin live restore: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `SELECT e.player, e.fields, b.player IS NOT NULL FROM live_evidence e LEFT JOIN live_board b ON b.evidence_id=e.id WHERE e.id IN (SELECT MAX(id) FROM live_evidence GROUP BY player, season, week) ORDER BY e.id`)
	if err != nil {
		return fmt.Errorf("restore live evidence: %w", err)
	}
	defer func() { _ = rows.Close() }()
	held := map[liveStatKey]source.HeldWeek{}
	book := liveStatBook{lines: map[string]source.PlayerWeek{}}
	for rows.Next() {
		var player, body string
		var active bool
		var line source.HeldWeek
		if err := rows.Scan(&player, &body, &active); err != nil {
			return fmt.Errorf("read live evidence: %w", err)
		}
		if err := json.Unmarshal([]byte(body), &line); err != nil {
			return fmt.Errorf("decode live evidence: %w", err)
		}
		if line.Week == nil {
			return fmt.Errorf("live evidence for %s has no week", player)
		}
		held[liveStatKey{player, line.Week.Season, line.Week.Week}] = line
		if active {
			book.lines[player] = *line.Week
			book.season, book.week, book.at, book.authority, book.missing = line.Week.Season, line.Week.Week, line.FetchedAt, line.Authority, line.Missing
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read live evidence: %w", err)
	}
	states, err := s.restoreLiveChronology(ctx, tx)
	if err != nil {
		return err
	}
	s.liveHeld, s.live, s.liveRevisions = held, book, states
	return nil
}

func (s *Store) restoreLiveChronology(ctx context.Context, tx *sql.Tx) (map[liveStatKey]*liveRevisionState, error) {
	rows, err := tx.QueryContext(ctx, `SELECT player, season, week, origin_seq, revision FROM live_chronology ORDER BY player, season, week, seq`)
	if err != nil {
		return nil, fmt.Errorf("restore live chronology: %w", err)
	}
	defer func() { _ = rows.Close() }()
	states := map[liveStatKey]*liveRevisionState{}
	for rows.Next() {
		var key liveStatKey
		var origin int
		var body string
		var version source.LineRevision
		if err := rows.Scan(&key.player, &key.season, &key.week, &origin, &body); err != nil {
			return nil, fmt.Errorf("read live chronology: %w", err)
		}
		if err := json.Unmarshal([]byte(body), &version); err != nil {
			return nil, fmt.Errorf("decode live chronology: %w", err)
		}
		state := states[key]
		if state == nil {
			state = &liveRevisionState{archived: map[liveRevisionOrigin]bool{}}
			states[key] = state
		}
		if version.Seq != len(state.versions) || version.Held.Week == nil {
			return nil, fmt.Errorf("invalid live chronology for %s", key.player)
		}
		state.versions = append(state.versions, version)
		if origin >= 0 {
			state.archived[liveRevisionOrigin{origin, version.Digest, version.Held.Authority}] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read live chronology: %w", err)
	}
	return states, nil
}

// The book replacement and changed lines commit together; a failed write leaves the held book intact.
func (s *Store) storeLiveBook(ctx context.Context, book liveStatBook) error {
	if s.pools == nil {
		return nil
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("begin live capture: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM live_board`); err != nil {
		return fmt.Errorf("replace live board: %w", err)
	}
	for player, line := range book.lines {
		held := heldLiveLine(line, book.authority, book.at, book.missing)
		body, err := json.Marshal(held)
		if err != nil {
			return fmt.Errorf("encode live evidence: %w", err)
		}
		provider, err := liveProvider(held.Authority)
		if err != nil {
			return err
		}
		digest := held.SHA256
		if _, err := tx.ExecContext(ctx, `INSERT INTO live_evidence(player, season, week, game, source, digest, fetched_at, fields)
SELECT ?,?,?,?,?,?,?,? WHERE NOT EXISTS (
 SELECT 1 FROM live_evidence WHERE id=(SELECT MAX(id) FROM live_evidence WHERE player=? AND season=? AND week=?) AND digest=?)`,
			player, line.Season, line.Week, line.GameID, provider, digest, book.at.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), string(body), player, line.Season, line.Week, digest); err != nil {
			return fmt.Errorf("capture live evidence: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO live_board SELECT ?, MAX(id) FROM live_evidence WHERE player=? AND season=? AND week=?`, player, player, line.Season, line.Week); err != nil {
			return fmt.Errorf("capture live board: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit live capture: %w", err)
	}
	return nil
}

func (s *Store) storeLiveVersions(ctx context.Context, key liveStatKey, versions []source.LineRevision, origins map[int]int) error {
	if s.pools == nil || len(versions) == 0 {
		return nil
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("begin live chronology: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, version := range versions {
		body, err := json.Marshal(version)
		if err != nil {
			return fmt.Errorf("encode live revision: %w", err)
		}
		origin := -1
		if seq, ok := origins[version.Seq]; ok {
			origin = seq
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO live_chronology VALUES (?,?,?,?,?,?)`, key.player, key.season, key.week, version.Seq, origin, string(body)); err != nil {
			return fmt.Errorf("capture live chronology: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit live chronology: %w", err)
	}
	return nil
}

func liveProvider(authority string) (string, error) {
	switch authority {
	case source.ESPNLive, source.ESPNLive + " + " + source.Sleeper:
		return "espn", nil
	case source.Sleeper:
		return "sleeper", nil
	default:
		return "", fmt.Errorf("unsupported live evidence authority %q", authority)
	}
}
