package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// execer is what the write helpers need from a transaction.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// LoadReport says what one load wrote. A reload of unchanged data adds nothing.
type LoadReport struct {
	LoadID     int64
	Facts      int
	Added      int // new or changed values written to observations
	Unresolved int // new or changed values waiting for a player match
}

// fact is a Fact mapped onto its measure and parsed. playerID is empty while unresolved.
type fact struct {
	playerID        string
	idType, idValue string
	season, week    int
	measure         string
	value           sql.NullFloat64
	text            sql.NullString
}

// Ingest writes one batch. Every fact must map through source_fields and parse for its measure,
// or nothing is written and the failure is recorded as a failed load. A value equal to the latest
// one held for its key is skipped, so loading the same data twice changes nothing; a different
// value appends a correction with a later as_of.
func (s *Store) Ingest(ctx context.Context, b measures.Batch) (LoadReport, error) {
	facts, err := s.mapBatch(b)
	if err != nil {
		return LoadReport{}, errors.Join(err, s.LoadFailed(ctx, b.Source, err))
	}

	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return LoadReport{}, fmt.Errorf("history: ingest %s: %w", b.Source, err)
	}
	defer func() { _ = tx.Rollback() }()

	rep := LoadReport{Facts: len(facts)}
	var changed []fact
	for _, f := range facts {
		if f.playerID == "" {
			id, ok, err := lookupPlayer(ctx, tx, f.idType, f.idValue)
			if err != nil {
				return LoadReport{}, err
			}
			if ok {
				f.playerID = id
			}
		}
		same, err := sameAsLatest(ctx, tx, b.Source, f)
		if err != nil {
			return LoadReport{}, err
		}
		if same {
			continue
		}
		if f.playerID == "" {
			rep.Unresolved++
		} else {
			rep.Added++
		}
		changed = append(changed, f)
	}

	asOf := formatTime(s.now())
	res, err := tx.ExecContext(ctx, `
INSERT INTO loads (source, sha256, loaded_at, facts, added, unresolved) VALUES (?, ?, ?, ?, ?, ?)`,
		b.Source, b.BodySHA256, asOf, rep.Facts, rep.Added, rep.Unresolved)
	if err != nil {
		return LoadReport{}, fmt.Errorf("history: record load %s: %w", b.Source, err)
	}
	if rep.LoadID, err = res.LastInsertId(); err != nil {
		return LoadReport{}, fmt.Errorf("history: load id: %w", err)
	}
	for _, f := range changed {
		if err := insertFact(ctx, tx, b.Source, asOf, rep.LoadID, f); err != nil {
			return LoadReport{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return LoadReport{}, fmt.Errorf("history: ingest commit: %w", err)
	}
	return rep, nil
}

// LoadFailed records a load attempt that produced nothing: the fetch failed, or its body could
// not be read. Source health counts these.
func (s *Store) LoadFailed(ctx context.Context, source string, cause error) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if _, err := s.pools.Write().ExecContext(ctx, `
INSERT INTO loads (source, loaded_at, facts, added, unresolved, error) VALUES (?, ?, 0, 0, 0, ?)`,
		source, formatTime(s.now()), cause.Error()); err != nil {
		return fmt.Errorf("history: record failed load %s: %w", source, err)
	}
	return nil
}

// mapBatch maps every fact through the registry and parses its value, rejecting the batch on the
// first problem and on any key repeated within it.
func (s *Store) mapBatch(b measures.Batch) ([]fact, error) {
	if _, ok := s.source(b.Source); !ok {
		return nil, fmt.Errorf("history: batch from unregistered source %q", b.Source)
	}
	out := make([]fact, 0, len(b.Facts))
	seen := make(map[string]bool, len(b.Facts))
	for _, in := range b.Facts {
		f, err := s.mapFact(b.Source, in)
		if err != nil {
			return nil, err
		}
		key := strings.Join([]string{f.idType, f.idValue, strconv.Itoa(f.season), strconv.Itoa(f.week), f.measure}, "\x00")
		if seen[key] {
			return nil, fmt.Errorf("history: %s reports %s %s %s twice in one load", b.Source, in.ID, f.measure, period(f))
		}
		seen[key] = true
		out = append(out, f)
	}
	return out, nil
}

func (s *Store) mapFact(source string, in measures.Fact) (fact, error) {
	sf, ok := s.reg.Field(source, in.Field)
	if !ok {
		return fact{}, fmt.Errorf("history: %s field %q has no row in source_fields.csv", source, in.Field)
	}
	m, _ := s.reg.Measure(sf.Measure) // the registry validated every mapping's measure
	f := fact{idType: in.IDType, idValue: in.ID, season: in.Season, week: in.Week, measure: m.Name}
	switch {
	case in.IDType == "" || in.ID == "":
		return fact{}, fmt.Errorf("history: %s %s fact has no player id", source, in.Field)
	case in.Season < 1900:
		return fact{}, fmt.Errorf("history: %s %s for %s has season %d", source, in.Field, in.ID, in.Season)
	case !m.ValidWeek(in.Week):
		return fact{}, fmt.Errorf("history: %s is %s-grain; %s reports week %d", m.Name, m.Grain, in.ID, in.Week)
	}
	if in.IDType == measures.IDTypeMFL {
		id, err := playerid.New(in.ID)
		if err != nil {
			return fact{}, fmt.Errorf("history: %s %s: %w", source, in.Field, err)
		}
		f.playerID, f.idValue = id.String(), id.String()
	}
	raw := strings.TrimSpace(in.Raw)
	if m.IsText() {
		if raw == "" {
			return fact{}, fmt.Errorf("history: %s for %s is empty", m.Name, in.ID)
		}
		f.text = sql.NullString{String: raw, Valid: true}
		return f, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return fact{}, fmt.Errorf("history: %s for %s is %q, not a finite number", m.Name, in.ID, in.Raw)
	}
	f.value = sql.NullFloat64{Float64: v, Valid: true}
	return f, nil
}

func period(f fact) string {
	if f.week == 0 {
		return fmt.Sprintf("season %d", f.season)
	}
	return fmt.Sprintf("season %d week %d", f.season, f.week)
}

// lookupPlayer resolves a source id through player_ids.
func lookupPlayer(ctx context.Context, tx execer, idType, idValue string) (string, bool, error) {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT player_id FROM player_ids WHERE id_type = ? AND id_value = ?`,
		idType, idValue).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("history: resolve %s %s: %w", idType, idValue, err)
	}
	return id, true, nil
}

// sameAsLatest reports whether the latest value held for f's key equals f's value.
func sameAsLatest(ctx context.Context, tx execer, source string, f fact) (bool, error) {
	var row *sql.Row
	if f.playerID != "" {
		row = tx.QueryRowContext(ctx, `
SELECT value, text FROM observations
WHERE player_id = ? AND season = ? AND week = ? AND measure = ? AND source = ?
ORDER BY as_of DESC LIMIT 1`, f.playerID, f.season, f.week, f.measure, source)
	} else {
		row = tx.QueryRowContext(ctx, `
SELECT value, text FROM unresolved_observations
WHERE source = ? AND id_type = ? AND id_value = ? AND season = ? AND week = ? AND measure = ?
ORDER BY as_of DESC LIMIT 1`, source, f.idType, f.idValue, f.season, f.week, f.measure)
	}
	var v sql.NullFloat64
	var t sql.NullString
	err := row.Scan(&v, &t)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("history: latest %s for %s: %w", f.measure, f.idValue, err)
	}
	return v == f.value && t == f.text, nil
}

func insertFact(ctx context.Context, tx execer, source, asOf string, loadID int64, f fact) error {
	var err error
	if f.playerID != "" {
		_, err = tx.ExecContext(ctx, `
INSERT INTO observations (player_id, season, week, measure, source, as_of, value, text, load_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, f.playerID, f.season, f.week, f.measure, source, asOf, f.value, f.text, loadID)
	} else {
		_, err = tx.ExecContext(ctx, `
INSERT INTO unresolved_observations (source, id_type, id_value, season, week, measure, as_of, value, text, load_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, source, f.idType, f.idValue, f.season, f.week, f.measure, asOf, f.value, f.text, loadID)
	}
	if err != nil {
		return fmt.Errorf("history: write %s for %s: %w", f.measure, f.idValue, err)
	}
	return nil
}

// PlayerIDLink maps one source id onto an MFL player id.
type PlayerIDLink struct {
	IDType, IDValue, PlayerID string
}

// LinkPlayerIDs adds or corrects source-id mappings, then promotes every waiting observation
// that now resolves, keeping its as_of. It returns how many were promoted.
func (s *Store) LinkPlayerIDs(ctx context.Context, links []PlayerIDLink) (int, error) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("history: link player ids: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, l := range links {
		id, err := playerid.New(l.PlayerID)
		if err != nil {
			return 0, fmt.Errorf("history: link %s %s: %w", l.IDType, l.IDValue, err)
		}
		if l.IDType == "" || l.IDType == measures.IDTypeMFL || l.IDValue == "" {
			return 0, fmt.Errorf("history: cannot link id type %q value %q", l.IDType, l.IDValue)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO player_ids (id_type, id_value, player_id) VALUES (?, ?, ?)
ON CONFLICT (id_type, id_value) DO UPDATE SET player_id = excluded.player_id`,
			l.IDType, l.IDValue, id.String()); err != nil {
			return 0, fmt.Errorf("history: link %s %s: %w", l.IDType, l.IDValue, err)
		}
	}
	const matched = `FROM unresolved_observations u JOIN player_ids p ON p.id_type = u.id_type AND p.id_value = u.id_value`
	res, err := tx.ExecContext(ctx, `
INSERT INTO observations (player_id, season, week, measure, source, as_of, value, text, load_id)
SELECT p.player_id, u.season, u.week, u.measure, u.source, u.as_of, u.value, u.text, u.load_id `+matched)
	if err != nil {
		return 0, fmt.Errorf("history: promote resolved observations: %w", err)
	}
	promoted, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("history: promoted count: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
DELETE FROM unresolved_observations WHERE EXISTS (SELECT 1 FROM player_ids p
WHERE p.id_type = unresolved_observations.id_type AND p.id_value = unresolved_observations.id_value)`); err != nil {
		return 0, fmt.Errorf("history: clear resolved observations: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("history: link player ids commit: %w", err)
	}
	return int(promoted), nil
}
