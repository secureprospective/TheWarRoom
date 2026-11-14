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

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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
	commits chan string
	pools   *db.Pools
	reg     *measures.Registry
	now     func() time.Time

	// evidence feeds native results (the zero value reads the held archive); lineups say who counts for a
	// franchise (the default is the league's own ledger); lineupEnv supplies the football facts a lineup
	// is judged against and is installed at startup.
	evidence  func(context.Context, source.RawNativeScope) (source.NativeEvidence, error)
	statWeeks func(context.Context, int) ([]commissioner.StatWeek, error)
	lineups   LineupSource
	lineupEnv LineupEnvironment
	// bidEnv says which week of the NFL season it is and who is in the playoffs when a bid is judged.
	bidEnv BidEnvironmentSource
	// aliases is the desktop's linked-identity map, read when a caller supplies none.
	aliases VerifiedAliasSource

	leaguesMu sync.Mutex
	leagues   nativeLeagueRevision
	recordsMu sync.Mutex
	records   map[string]recordRevision
	eventsMu  sync.Mutex
	events    map[eventStreamKey]eventRevision
	weeklyMu  sync.Mutex
	weekly    map[string]weeklyFile
	// seasons holds the latest built held season per season, keyed on its archive and identity revision.
	seasons heldSeasonCache
	// flights joins concurrent identical readings; it holds nothing once a reading completes.
	flights readingFlights
	// projMu guards the one scored projections book, keyed by league, rules digest and files.
	projMu   sync.Mutex
	projKey  string
	projBook projectionBook
	// liveMu guards the live stat lines held for the NFL week being played.
	liveMu        sync.Mutex
	live          liveStatBook
	liveHeld      map[liveStatKey]source.HeldWeek
	liveRevisions map[liveStatKey]*liveRevisionState
	// liveEpoch and liveWeeks name the live lines as held, so a score read from them is reused only while
	// they stand: liveEpoch moves when the book is restored, liveWeeks when one week's lines change.
	liveEpoch uint64
	liveWeeks map[[2]int]uint64
	// scores is the season score cache: each franchise-week scored once per revision.
	scores seasonScoreCache
	// statPrices keeps statistics queue prices across readings (commissioner.StatPriceCache).
	poolMu     sync.Mutex
	poolKey    string
	poolPoints map[string]leaguerules.Decimal
	statPrices statPrices
	wmu        sync.Mutex
	// batchFault, set only by tests, fails a batch after the given step to prove nothing is kept.
	batchFault func(step int) error
}

// New returns an unprepared store over history.db's pools, using reg as the registry.
func New(pools *db.Pools, reg *measures.Registry) *Store {
	return &Store{commits: make(chan string, 1), pools: pools, reg: reg, now: time.Now, lineups: ledgerLineups{}}
}

// signalCommissionerCommit names the committed league to the due worker through the resolver.
// Coalesced: a full channel already guarantees one wake, and every wake rereads every league.
func (s *Store) signalCommissionerCommit(league string) {
	select {
	case s.commits <- league:
	default:
	}
}

// LeagueResolver finds the store holding a league. Leagues is the directory of native leagues;
// Stores is every file holding league history (each verifies its own audit chain); Commits names the
// league of each native ledger commit, coalesced. Per-league files arrive behind this seam.
type LeagueResolver interface {
	Leagues(ctx context.Context) ([]commissioner.NativeLeague, error)
	League(ctx context.Context, league string) (*Store, error)
	Stores(ctx context.Context) ([]*Store, error)
	Commits() <-chan string
}

// OneFile resolves every league to the single file both the hosted server and the desktop open today.
type OneFile struct{ store *Store }

func NewOneFile(store *Store) OneFile { return OneFile{store: store} }

func (o OneFile) Leagues(ctx context.Context) ([]commissioner.NativeLeague, error) {
	return o.store.NativeLeagues(ctx)
}

func (o OneFile) League(context.Context, string) (*Store, error) { return o.store, nil }

func (o OneFile) Stores(context.Context) ([]*Store, error) { return []*Store{o.store}, nil }

func (o OneFile) Commits() <-chan string { return o.store.commits }

// Now returns the same clock used to record ledger acts.
func (s *Store) Now() time.Time { return s.now() }

// SetClock replaces the store's clock. Tests use it to write corrections at known times.
func (s *Store) SetClock(now func() time.Time) { s.now = now }

// VerifiedAliasSource supplies identities a trusted account check linked to one account.
type VerifiedAliasSource func(ctx context.Context, league string) ([]governance.VerifiedAlias, error)

// SetVerifiedAliases installs the desktop's linked identities before serving requests.
func (s *Store) SetVerifiedAliases(source VerifiedAliasSource) { s.aliases = source }

// Registry returns the registry the store was built with.
func (s *Store) Registry() *measures.Registry { return s.reg }

// appendOnly lists the tables whose rows never change once written.
func appendOnly() []string {
	return []string{"raw_archive", "fetch_log", "loads", "observations", "param_sets", "scoring_runs", "run_scores",
		"model_runs", "model_scores", "input_manifests", "compiled_snapshots", "snapshot_checks", "league_event"}
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
);
CREATE TABLE IF NOT EXISTS model_runs (
	run_id           INTEGER PRIMARY KEY,
	kind             TEXT NOT NULL CHECK (kind = 'model'),
	season           INTEGER NOT NULL,
	as_of            TEXT NOT NULL,
	param_set_id     TEXT NOT NULL REFERENCES param_sets(param_set_id),
	engine           TEXT NOT NULL,
	inputs_hash      TEXT NOT NULL,
	scores_hash      TEXT NOT NULL,
	missing_measures TEXT NOT NULL,
	created_at       TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS model_scores (
	run_id       INTEGER NOT NULL REFERENCES model_runs(run_id),
	mfl_id       TEXT NOT NULL,
	position     TEXT NOT NULL,
	now          REAL NOT NULL,
	now_ppg      REAL NOT NULL,
	dynasty      REAL NOT NULL,
	dynasty_ppg  REAL NOT NULL,
	prior        REAL NOT NULL,
	past_games   REAL NOT NULL,
	season_games REAL NOT NULL,
	z_past       REAL NOT NULL,
	z_now        REAL NOT NULL,
	on_field     REAL NOT NULL,
	inputs       TEXT NOT NULL,
	PRIMARY KEY (run_id, mfl_id)
);
CREATE TABLE IF NOT EXISTS input_manifests (
	manifest_sha TEXT PRIMARY KEY,
	name         TEXT NOT NULL,
	entries      TEXT NOT NULL,
	published_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS compiled_snapshots (
	snapshot_id  INTEGER PRIMARY KEY,
	manifest_sha TEXT NOT NULL REFERENCES input_manifests(manifest_sha),
	compiler     TEXT NOT NULL,
	rules_hash   TEXT NOT NULL,
	result_sha   TEXT,
	error        TEXT NOT NULL DEFAULT '',
	built_at     TEXT NOT NULL,
	CHECK ((result_sha IS NULL) <> (error = ''))
);
CREATE TABLE IF NOT EXISTS snapshot_checks (
	snapshot_id INTEGER PRIMARY KEY REFERENCES compiled_snapshots(snapshot_id),
	checks      TEXT NOT NULL
);`

// registryDDL creates registry mirrors; Initialize refreshes their rows without DDL.
const registryDDL = `
CREATE TABLE IF NOT EXISTS measures (
	measure   TEXT PRIMARY KEY,
	family    TEXT NOT NULL,
	grain     TEXT NOT NULL,
	unit      TEXT NOT NULL,
	positions TEXT NOT NULL,
	meaning   TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sources (
	source       TEXT PRIMARY KEY,
	name         TEXT NOT NULL,
	status       TEXT NOT NULL,
	max_age_days INTEGER NOT NULL,
	host         TEXT NOT NULL,
	path_prefix  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS source_fields (
	source   TEXT NOT NULL,
	field    TEXT NOT NULL,
	measure  TEXT NOT NULL,
	priority INTEGER NOT NULL,
	PRIMARY KEY (source, field)
);`

// Initialize migrates the file, then refreshes the rebuildable registry mirrors.
func (s *Store) Initialize(ctx context.Context) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if _, err := s.pools.Migrate(ctx, historyMigrations()...); err != nil {
		return fmt.Errorf("history: migrate schema: %w", err)
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("history: initialize: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM measures; DELETE FROM sources; DELETE FROM source_fields`); err != nil {
		return fmt.Errorf("history: clear registry mirrors: %w", err)
	}
	if _, err := db.VerifyAudit(ctx, tx, ledgerAuditSource, ledgerAuditQuery); err != nil {
		return fmt.Errorf("history: verify audit: %w", err)
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

// LeagueState is one league as its own file records it: every league with a create event, whatever
// its transfer state, and when it was created. The server's league directory is copied from this.
type LeagueState struct {
	League, State, Created string
}

// LeagueStates reads every league this file holds; it is the truth the server directory follows.
func (s *Store) LeagueStates(ctx context.Context) ([]LeagueState, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT e.league, COALESCE((SELECT state FROM transfer_current t WHERE t.league=e.league),'active'), MIN(e.recorded_at)
 FROM league_event e WHERE e.stream=? AND e.kind=? GROUP BY e.league ORDER BY e.league`, commissioner.Stream, commissioner.Create)
	if err != nil {
		return nil, fmt.Errorf("read league states: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []LeagueState
	for rows.Next() {
		var l LeagueState
		if err := rows.Scan(&l.League, &l.State, &l.Created); err != nil {
			return nil, fmt.Errorf("read league state: %w", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read league states: %w", err)
	}
	return out, nil
}

// NamedPrincipals lists every account principal pools' league file names: as the actor of an entry
// or anywhere in an entry's payload (seats, commissioners, invitations). A restore checks each is in
// the sign-in file restored beside it.
func NamedPrincipals(ctx context.Context, pools *db.Pools) ([]string, error) {
	rows, err := pools.Read().QueryContext(ctx, `SELECT actor FROM league_event WHERE actor LIKE 'acct.%'
 UNION SELECT j.value FROM league_event e, json_tree(e.payload) j WHERE j.type='text' AND j.value LIKE 'acct.%' ORDER BY 1`)
	if err != nil {
		return nil, fmt.Errorf("read named principals: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("read named principals: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read named principals: %w", err)
	}
	return out, nil
}
