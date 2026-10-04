// Package history owns history.db: everything the app cannot rebuild from MFL. It holds every
// fetched body (raw_archive, fetch_log), every fact under its measure (observations), the
// registries loaded from internal/measures, and the scoring runs with their scores.
//
// The engine and UI read facts only through Features, which picks one row per player, period
// and measure: the latest as of the date asked, from the highest-priority source. History is
// append-only. Triggers abort any UPDATE or DELETE on the history tables; only the registry
// mirrors, the player id map and the unresolved queue change in place.
package history

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

// timeLayout is fixed-width so stored times sort as text.
const timeLayout = "2006-01-02T15:04:05.000000Z"

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("history: stored time %q: %w", s, err)
	}
	return t, nil
}

// Store is the history store. Construct with New, prepare with Initialize. Writes serialize
// under wmu.
type Store struct {
	pools *db.Pools
	reg   *measures.Registry
	now   func() time.Time

	wmu sync.Mutex
}

// New returns an unprepared store over history.db's pools, using reg as the registry.
func New(pools *db.Pools, reg *measures.Registry) *Store {
	return &Store{pools: pools, reg: reg, now: time.Now}
}

// SetClock replaces the store's clock. Tests use it to write corrections at known times.
func (s *Store) SetClock(now func() time.Time) { s.now = now }

// Registry returns the registry the store was built with.
func (s *Store) Registry() *measures.Registry { return s.reg }

// appendOnly lists the tables whose rows never change once written.
func appendOnly() []string {
	return []string{"raw_archive", "fetch_log", "loads", "observations", "param_sets", "scoring_runs", "run_scores"}
}

const historyDDL = `
CREATE TABLE IF NOT EXISTS raw_archive (
	sha256 TEXT PRIMARY KEY,
	size   INTEGER NOT NULL,
	body   BLOB NOT NULL
);
CREATE TABLE IF NOT EXISTS fetch_log (
	fetch_id   INTEGER PRIMARY KEY,
	source     TEXT NOT NULL,
	url        TEXT NOT NULL,
	status     INTEGER NOT NULL,
	sha256     TEXT REFERENCES raw_archive(sha256),
	error      TEXT NOT NULL DEFAULT '',
	fetched_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS loads (
	load_id    INTEGER PRIMARY KEY,
	source     TEXT NOT NULL,
	sha256     TEXT NOT NULL DEFAULT '',
	loaded_at  TEXT NOT NULL,
	facts      INTEGER NOT NULL,
	added      INTEGER NOT NULL,
	unresolved INTEGER NOT NULL,
	error      TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS loads_by_source ON loads (source, loaded_at);
CREATE TABLE IF NOT EXISTS observations (
	player_id TEXT NOT NULL,
	season    INTEGER NOT NULL,
	week      INTEGER NOT NULL,
	measure   TEXT NOT NULL,
	source    TEXT NOT NULL,
	as_of     TEXT NOT NULL,
	value     REAL,
	text      TEXT,
	load_id   INTEGER NOT NULL REFERENCES loads(load_id),
	CHECK ((value IS NULL) <> (text IS NULL)),
	PRIMARY KEY (player_id, season, week, measure, source, as_of)
);
CREATE INDEX IF NOT EXISTS observations_by_measure ON observations (measure, season, week);
CREATE INDEX IF NOT EXISTS observations_by_source ON observations (source, season);
CREATE TABLE IF NOT EXISTS unresolved_observations (
	source   TEXT NOT NULL,
	id_type  TEXT NOT NULL,
	id_value TEXT NOT NULL,
	season   INTEGER NOT NULL,
	week     INTEGER NOT NULL,
	measure  TEXT NOT NULL,
	as_of    TEXT NOT NULL,
	value    REAL,
	text     TEXT,
	load_id  INTEGER NOT NULL REFERENCES loads(load_id),
	CHECK ((value IS NULL) <> (text IS NULL)),
	PRIMARY KEY (source, id_type, id_value, season, week, measure, as_of)
);
CREATE INDEX IF NOT EXISTS unresolved_by_source ON unresolved_observations (source, season);
CREATE TABLE IF NOT EXISTS player_ids (
	id_type   TEXT NOT NULL,
	id_value  TEXT NOT NULL,
	player_id TEXT NOT NULL,
	PRIMARY KEY (id_type, id_value)
);
CREATE TABLE IF NOT EXISTS param_sets (
	param_set_id TEXT PRIMARY KEY,
	measures     TEXT NOT NULL,
	params       TEXT NOT NULL,
	created_at   TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS scoring_runs (
	run_id           INTEGER PRIMARY KEY,
	kind             TEXT NOT NULL CHECK (kind IN ('board', 'rebalance')),
	season           INTEGER NOT NULL,
	as_of            TEXT NOT NULL,
	param_set_id     TEXT NOT NULL REFERENCES param_sets(param_set_id),
	engine           TEXT NOT NULL,
	inputs_hash      TEXT NOT NULL,
	scores_hash      TEXT NOT NULL,
	missing_measures TEXT NOT NULL,
	created_at       TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS run_scores (
	run_id             INTEGER NOT NULL REFERENCES scoring_runs(run_id),
	mfl_id             TEXT NOT NULL,
	base_points        REAL NOT NULL,
	age_pull           REAL NOT NULL,
	film_effective     REAL NOT NULL,
	film_raw           REAL NOT NULL,
	ras_effective      REAL NOT NULL,
	breakout_effective REAL NOT NULL,
	combined           REAL NOT NULL,
	scouting_adjusted  REAL NOT NULL,
	cap_multiplier     REAL NOT NULL,
	cap_tier           TEXT NOT NULL,
	adjusted_score     REAL NOT NULL,
	tb_is_veteran      INTEGER NOT NULL,
	tb_ras             REAL NOT NULL,
	tb_scarcity_rank   INTEGER NOT NULL,
	PRIMARY KEY (run_id, mfl_id)
);`

// registryDDL recreates the registry mirrors. They hold whatever the CSV files say, so their
// shape can change with the files.
const registryDDL = `
DROP VIEW IF EXISTS season_features;
DROP VIEW IF EXISTS week_features;
DROP TABLE IF EXISTS measures;
DROP TABLE IF EXISTS sources;
DROP TABLE IF EXISTS source_fields;
CREATE TABLE measures (
	measure   TEXT PRIMARY KEY,
	family    TEXT NOT NULL,
	grain     TEXT NOT NULL,
	unit      TEXT NOT NULL,
	positions TEXT NOT NULL,
	meaning   TEXT NOT NULL
);
CREATE TABLE sources (
	source       TEXT PRIMARY KEY,
	name         TEXT NOT NULL,
	status       TEXT NOT NULL,
	max_age_days INTEGER NOT NULL,
	host         TEXT NOT NULL,
	path_prefix  TEXT NOT NULL
);
CREATE TABLE source_fields (
	source   TEXT NOT NULL,
	field    TEXT NOT NULL,
	measure  TEXT NOT NULL,
	priority INTEGER NOT NULL,
	PRIMARY KEY (source, field)
);`

// Initialize creates the history tables and their append-only triggers, then reloads the
// registry mirrors and the features views in one transaction.
func (s *Store) Initialize(ctx context.Context) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("history: initialize: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var ddl strings.Builder
	ddl.WriteString(historyDDL)
	for _, t := range appendOnly() {
		for _, op := range []string{"UPDATE", "DELETE"} {
			fmt.Fprintf(&ddl, `
CREATE TRIGGER IF NOT EXISTS %[1]s_no_%[2]s BEFORE %[2]s ON %[1]s
BEGIN SELECT RAISE(ABORT, '%[1]s is append-only: %[2]s forbidden'); END;`, t, strings.ToLower(op))
		}
	}
	ddl.WriteString(registryDDL)
	ddl.WriteString(featureViewsDDL())
	if _, err := tx.ExecContext(ctx, ddl.String()); err != nil {
		return fmt.Errorf("history: create schema: %w", err)
	}
	if err := s.loadRegistry(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: initialize commit: %w", err)
	}
	return nil
}

// loadRegistry copies the registry into its mirror tables.
func (s *Store) loadRegistry(ctx context.Context, tx execer) error {
	for _, m := range s.reg.Measures {
		positions := "all"
		if m.Positions != nil {
			ps := make([]string, len(m.Positions))
			for i, p := range m.Positions {
				ps[i] = string(p)
			}
			positions = strings.Join(ps, " ")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO measures VALUES (?, ?, ?, ?, ?, ?)`,
			m.Name, string(m.Family), string(m.Grain), m.Unit, positions, m.Meaning); err != nil {
			return fmt.Errorf("history: load measure %s: %w", m.Name, err)
		}
	}
	for _, src := range s.reg.Sources {
		if _, err := tx.ExecContext(ctx, `INSERT INTO sources VALUES (?, ?, ?, ?, ?, ?)`,
			src.ID, src.Name, string(src.Status), int(src.MaxAge.Hours()/24), src.Host, src.PathPrefix); err != nil {
			return fmt.Errorf("history: load source %s: %w", src.ID, err)
		}
	}
	for _, f := range s.reg.Fields {
		if _, err := tx.ExecContext(ctx, `INSERT INTO source_fields VALUES (?, ?, ?, ?)`,
			f.Source, f.Field, f.Measure, f.Priority); err != nil {
			return fmt.Errorf("history: load source field %s.%s: %w", f.Source, f.Field, err)
		}
	}
	return nil
}
