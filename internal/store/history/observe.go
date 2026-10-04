package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// execer is what the write helpers need from a transaction.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
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
// value appends a correction with a later as_of. A week-grain zero is written only as a
// correction: an unwritten week count reads as zero.
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

	ids, err := directory(ctx, tx, facts)
	if err != nil {
		return LoadReport{}, err
	}
	held, err := latestHeld(ctx, tx, b.Source, seasonsOf(facts, b.Scope))
	if err != nil {
		return LoadReport{}, err
	}
	for i := range facts {
		if facts[i].playerID == "" {
			facts[i].playerID = ids[[2]string{facts[i].idType, facts[i].idValue}]
		}
	}
	facts = append(facts, zeroedByScope(facts, held, b.Scope)...)
	rep, changed := changes(facts, held)

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
	if err := insertFacts(ctx, tx, b.Source, asOf, rep.LoadID, changed); err != nil {
		return LoadReport{}, err
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
	case !m.ValidPeriod(in.Season, in.Week):
		return fact{}, fmt.Errorf("history: %s is %s-grain; %s reports season %d week %d",
			m.Name, m.Grain, in.ID, in.Season, in.Week)
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

// key identifies the value a fact would replace: resolved facts by player, waiting ones by the
// source's id.
func (f fact) key() string {
	who := f.playerID
	if who == "" {
		who = f.idType + "\x00" + f.idValue
	}
	return strings.Join([]string{who, strconv.Itoa(f.season), strconv.Itoa(f.week), f.measure}, "\x00")
}

// directory reads the player_ids rows for every id type in facts.
func directory(ctx context.Context, tx *sql.Tx, facts []fact) (map[[2]string]string, error) {
	types := map[string]bool{}
	for _, f := range facts {
		if f.playerID == "" {
			types[f.idType] = true
		}
	}
	out := map[[2]string]string{}
	for t := range types {
		if err := readIDs(ctx, tx, t, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func readIDs(ctx context.Context, tx *sql.Tx, idType string, out map[[2]string]string) error {
	rows, err := tx.QueryContext(ctx, `SELECT id_value, player_id FROM player_ids WHERE id_type = ?`, idType)
	if err != nil {
		return fmt.Errorf("history: read %s ids: %w", idType, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var v, id string
		if err := rows.Scan(&v, &id); err != nil {
			return fmt.Errorf("history: scan %s id: %w", idType, err)
		}
		out[[2]string{idType, v}] = id
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("history: read %s ids: %w", idType, err)
	}
	return nil
}

// changes picks the facts that differ from what is held. A week-grain zero with nothing held is
// left out: an unwritten week count reads as zero.
func changes(facts []fact, held map[string]fact) (LoadReport, []fact) {
	rep := LoadReport{Facts: len(facts)}
	var changed []fact
	for _, f := range facts {
		prev, ok := held[f.key()]
		if ok && prev.value == f.value && prev.text == f.text {
			continue
		}
		if !ok && f.week > 0 && f.value.Valid && f.value.Float64 == 0 {
			continue
		}
		if f.playerID == "" {
			rep.Unresolved++
		} else {
			rep.Added++
		}
		changed = append(changed, f)
	}
	return rep, changed
}

// seasonsOf lists the seasons a batch touches: its facts' and its scope's.
func seasonsOf(facts []fact, scope *measures.Scope) map[int]bool {
	seasons := map[int]bool{}
	for _, f := range facts {
		seasons[f.season] = true
	}
	if scope != nil {
		for _, season := range scope.Seasons {
			seasons[season] = true
		}
	}
	return seasons
}

// zeroedByScope returns a zero for every non-zero week value held in scope that the batch no
// longer reports.
func zeroedByScope(facts []fact, held map[string]fact, scope *measures.Scope) []fact {
	if scope == nil {
		return nil
	}
	reported := make(map[string]bool, len(facts))
	for _, f := range facts {
		reported[f.key()] = true
	}
	var out []fact
	for k, h := range held {
		if reported[k] || h.week == 0 || !h.value.Valid || h.value.Float64 == 0 ||
			!slices.Contains(scope.Seasons, h.season) || !slices.Contains(scope.Measures, h.measure) {
			continue
		}
		h.value.Float64 = 0
		out = append(out, h)
	}
	slices.SortFunc(out, func(a, b fact) int { return strings.Compare(a.key(), b.key()) })
	return out
}

// latestHeld reads the latest value source holds for every key in seasons, both resolved and
// waiting, keyed as fact.key.
func latestHeld(ctx context.Context, tx *sql.Tx, source string, seasons map[int]bool) (map[string]fact, error) {
	out := map[string]fact{}
	for season := range seasons {
		if err := readHeld(ctx, tx, source, season, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func readHeld(ctx context.Context, tx *sql.Tx, source string, season int, out map[string]fact) error {
	rows, err := tx.QueryContext(ctx, `
SELECT player_id, '', '', week, measure, value, text FROM (
	SELECT *, ROW_NUMBER() OVER (PARTITION BY player_id, week, measure ORDER BY as_of DESC) AS newest
	FROM observations WHERE source = ?1 AND season = ?2) WHERE newest = 1
UNION ALL
SELECT '', id_type, id_value, week, measure, value, text FROM (
	SELECT *, ROW_NUMBER() OVER (PARTITION BY id_type, id_value, week, measure ORDER BY as_of DESC) AS newest
	FROM unresolved_observations WHERE source = ?1 AND season = ?2) WHERE newest = 1`, source, season)
	if err != nil {
		return fmt.Errorf("history: latest %s values for %d: %w", source, season, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		f := fact{season: season}
		if err := rows.Scan(&f.playerID, &f.idType, &f.idValue, &f.week, &f.measure, &f.value, &f.text); err != nil {
			return fmt.Errorf("history: scan latest %s value: %w", source, err)
		}
		out[f.key()] = f
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("history: latest %s values for %d: %w", source, season, err)
	}
	return nil
}

func insertFacts(ctx context.Context, tx *sql.Tx, source, asOf string, loadID int64, facts []fact) error {
	resolved, err := tx.PrepareContext(ctx, `
INSERT INTO observations (player_id, season, week, measure, source, as_of, value, text, load_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("history: prepare observation insert: %w", err)
	}
	defer func() { _ = resolved.Close() }()
	waiting, err := tx.PrepareContext(ctx, `
INSERT INTO unresolved_observations (source, id_type, id_value, season, week, measure, as_of, value, text, load_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("history: prepare unresolved insert: %w", err)
	}
	defer func() { _ = waiting.Close() }()
	for _, f := range facts {
		if f.playerID != "" {
			_, err = resolved.ExecContext(ctx, f.playerID, f.season, f.week, f.measure, source, asOf, f.value, f.text, loadID)
		} else {
			_, err = waiting.ExecContext(ctx, source, f.idType, f.idValue, f.season, f.week, f.measure, asOf, f.value, f.text, loadID)
		}
		if err != nil {
			return fmt.Errorf("history: write %s for %s: %w", f.measure, f.idValue, err)
		}
	}
	return nil
}

// PlayerIDLink maps one source id onto an MFL player id.
type PlayerIDLink struct {
	IDType, IDValue, PlayerID string
}

// LinkPlayerIDs adds or corrects source-id mappings from one source's load, then promotes every
// waiting observation that now resolves, keeping its as_of. The load is recorded against source,
// so source health covers the directory too. It returns how many observations were promoted.
func (s *Store) LinkPlayerIDs(ctx context.Context, source string, links []PlayerIDLink) (int, error) {
	if _, ok := s.source(source); !ok {
		return 0, fmt.Errorf("history: links from unregistered source %q", source)
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("history: link player ids: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO player_ids (id_type, id_value, player_id) VALUES (?, ?, ?)
ON CONFLICT (id_type, id_value) DO UPDATE SET player_id = excluded.player_id`)
	if err != nil {
		return 0, fmt.Errorf("history: link player ids: %w", err)
	}
	defer func() { _ = stmt.Close() }()
	for _, l := range links {
		id, err := playerid.New(l.PlayerID)
		if err != nil {
			return 0, fmt.Errorf("history: link %s %s: %w", l.IDType, l.IDValue, err)
		}
		if l.IDType == "" || l.IDType == measures.IDTypeMFL || l.IDValue == "" {
			return 0, fmt.Errorf("history: cannot link id type %q value %q", l.IDType, l.IDValue)
		}
		if _, err := stmt.ExecContext(ctx, l.IDType, l.IDValue, id.String()); err != nil {
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
	if _, err := tx.ExecContext(ctx, `
INSERT INTO loads (source, loaded_at, facts, added, unresolved) VALUES (?, ?, ?, ?, 0)`,
		source, formatTime(s.now()), len(links), promoted); err != nil {
		return 0, fmt.Errorf("history: record load %s: %w", source, err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("history: link player ids commit: %w", err)
	}
	return int(promoted), nil
}

// LinkedPlayers returns every player with an id of idType in the directory.
func (s *Store) LinkedPlayers(ctx context.Context, idType string) (map[string]bool, error) {
	return s.distinct(ctx, `SELECT DISTINCT player_id FROM player_ids WHERE id_type = ?`, idType)
}

// MeasuresHeld returns every measure source holds a value of in season.
func (s *Store) MeasuresHeld(ctx context.Context, source string, season int) (map[string]bool, error) {
	return s.distinct(ctx, `SELECT DISTINCT measure FROM observations WHERE source = ? AND season = ?`, source, season)
}

// KnownIDs returns every id of idType the directory can resolve.
func (s *Store) KnownIDs(ctx context.Context, idType string) (map[string]bool, error) {
	return s.distinct(ctx, `SELECT id_value FROM player_ids WHERE id_type = ?`, idType)
}

// distinct runs a one-column query and returns its values as a set.
func (s *Store) distinct(ctx context.Context, query string, args ...any) (map[string]bool, error) {
	rows, err := s.pools.Read().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("history: directory read: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("history: directory scan: %w", err)
		}
		out[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: directory read: %w", err)
	}
	return out, nil
}
