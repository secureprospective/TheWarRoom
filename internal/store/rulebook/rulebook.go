// Package rulebook is the B3b League Rulebook: the first Layer-2 CONFIG store. It
// stores and serves the league's MFL-sourced rules (scoring config, roster rules,
// salary-cap amount, settings) with a commissioner override layer. It is PURE DATA
// ACCESS — it computes no scores, tag prices, or dead cap (that is engine/B7 work
// that READS from here). It sets the store template B3c/B4 clone.
//
// Stability model (Christopher's call): MFL config is stored as IMMUTABLE versioned
// snapshots with one explicit ACTIVE pointer. Reload SIDE-LOADS — it fetches, stores
// a new candidate version, and reports a ChangeSet, but does NOT promote: reads keep
// serving the stable active version until a human confirms. Promote repoints active
// to any version, so "update to current" and "roll back bad data" are the same one
// operation. Commissioner deltas are SEPARATE override records layered at read time.
//
// Writes are an admin-only path (AD-05): this store validates its own Set and is
// NEVER routed through B7 (B7 is the sole writer of league STATE, not config).
package rulebook

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
)

// Store is the league rulebook config store. Construct with New; seed with
// Initialize. It is safe for concurrent reads (Wails IPC goroutines) via an RWMutex
// guarding the in-memory active snapshot + override map.
type Store struct {
	pools *db.Pools
	src   league.Source

	// wmu serializes admin MUTATIONS (Initialize seed, Promote, SetOverride) end to
	// end, so the SQLite write and the subsequent in-memory loadActive are one
	// atomic step. Without it, two concurrent admin writers can interleave their
	// loadActive (read active version, then swap memory) such that a stale swap wins
	// last and reads serve a version other than the committed active pointer. mu
	// (below) is the finer reader/snapshot lock; wmu is always the outer lock.
	wmu sync.Mutex

	mu        sync.RWMutex
	active    league.RawConfig
	activeVer int
	overrides map[string]Override // keyed scope+"\x00"+key
}

// New constructs an unseeded store over the given SQLite pools. Call Initialize
// before any read.
func New(pools *db.Pools) *Store {
	return &Store{pools: pools, overrides: map[string]Override{}}
}

// Initialize ensures the schema, remembers the source for later Reloads, and loads
// the active config into memory. On a fresh database (no version yet) it seeds the
// first version from src and promotes it; on an existing database it simply loads
// the current active version without a network call (stability across restarts).
func (s *Store) Initialize(ctx context.Context, src league.Source) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	s.src = src
	if err := s.initSchema(ctx); err != nil {
		return err
	}
	ver, err := s.activeVersion(ctx)
	if err != nil {
		return err
	}
	if ver == 0 {
		cfg, ferr := src.Fetch(ctx)
		if ferr != nil {
			return fmt.Errorf("rulebook: initial fetch: %w", ferr)
		}
		ver, err = s.insertVersion(ctx, cfg)
		if err != nil {
			return err
		}
		if err = s.setActive(ctx, ver); err != nil {
			return err
		}
	}
	return s.loadActive(ctx)
}

// Reload re-fetches the live config, stores it as a NEW candidate version, and
// returns the ChangeSet versus the active config. It does NOT promote — the active
// version stays live until Promote is called. A non-empty ChangeSet is the
// commissioner gate's signal.
func (s *Store) Reload(ctx context.Context) (ChangeSet, error) {
	if s.src == nil {
		return ChangeSet{}, fmt.Errorf("rulebook: Reload before Initialize")
	}
	cfg, err := s.src.Fetch(ctx)
	if err != nil {
		return ChangeSet{}, fmt.Errorf("rulebook: reload fetch: %w", err)
	}
	cand, err := s.insertVersion(ctx, cfg)
	if err != nil {
		return ChangeSet{}, err
	}
	s.mu.RLock()
	from, active := s.activeVer, s.active
	s.mu.RUnlock()

	return ChangeSet{
		FromVersion: from,
		ToVersion:   cand,
		Deltas:      diffConfigs(active, cfg),
	}, nil
}

// Promote repoints the active version to ver after confirming it exists, then
// reloads the in-memory snapshot. This is the commissioner gate's apply step and
// the rollback path (promote a prior version). Admin-only (AD-05).
func (s *Store) Promote(ctx context.Context, ver int) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	var exists int
	row := s.pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(1) FROM rulebook_versions WHERE version = ?`, ver)
	if err := row.Scan(&exists); err != nil {
		return fmt.Errorf("rulebook: promote lookup: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("rulebook: promote: version %d does not exist", ver)
	}
	if err := s.setActive(ctx, ver); err != nil {
		return err
	}
	return s.loadActive(ctx)
}

// SetOverride upserts a commissioner override and refreshes the in-memory layer.
// It validates the override (the data-integrity half of the commissioner gate):
// known scope, non-empty key/value, and a cap value that parses as a number. This
// is the guard that stops bad data going live. Admin-only (AD-05).
func (s *Store) SetOverride(ctx context.Context, scope, key, value, note string) error {
	if err := validateOverride(scope, key, value); err != nil {
		return err
	}
	s.wmu.Lock()
	defer s.wmu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.pools.Write().ExecContext(ctx,
		`INSERT INTO rulebook_overrides (scope, rule_key, value, note, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(scope, rule_key) DO UPDATE SET value = excluded.value,
		   note = excluded.note, created_at = excluded.created_at`,
		scope, key, value, note, now)
	if err != nil {
		return fmt.Errorf("rulebook: set override: %w", err)
	}
	return s.loadActive(ctx)
}

// validateOverride enforces the supported override surface and value sanity.
func validateOverride(scope, key, value string) error {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
		return fmt.Errorf("rulebook: override requires non-empty key and value")
	}
	switch scope {
	case scopeCap:
		if _, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err != nil {
			return fmt.Errorf("rulebook: cap override %q is not numeric: %w", value, err)
		}
	case scopeSetting:
		// any non-empty scalar is acceptable; the value stays a raw string
	default:
		return fmt.Errorf("rulebook: unknown override scope %q", scope)
	}
	return nil
}

// --- Reads -----------------------------------------------------------------

// GetSalaryCap returns the active salary-cap AMOUNT (AD-21), with a cap override
// applied if one is set.
func (s *Store) GetSalaryCap() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if o, ok := s.overrides[ovKey(scopeCap, capKey)]; ok {
		return o.Value
	}
	return s.active.SalaryCapAmount
}

// GetScoringRules returns the active position-additive scoring config (raw MFL
// values). Scoring changes flow through MFL Reload + Promote, not overrides.
func (s *Store) GetScoringRules() []league.PositionRuleSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneScoring(s.active.ScoringRules)
}

// GetSetting returns a scalar league setting by key (e.g. "rosterSize",
// "taxiSquad"), applying a setting override if present. ok is false for an unknown
// key.
func (s *Store) GetSetting(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if o, ok := s.overrides[ovKey(scopeSetting, key)]; ok {
		return o.Value, true
	}
	v, ok := settingsMap(s.active)[key]
	return v, ok
}

// ActiveConfig returns a copy of the active raw config (no scalar overrides applied
// — use the typed getters for override-aware reads). Intended for the engine's
// bulk read of the rulebook.
func (s *Store) ActiveConfig() league.RawConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.active)
}

// Versions lists stored config versions newest-first for the audit/gate surface.
func (s *Store) Versions(ctx context.Context) ([]VersionMeta, error) {
	active, err := s.activeVersion(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pools.Read().QueryContext(ctx,
		`SELECT version, source, created_at FROM rulebook_versions ORDER BY version DESC`)
	if err != nil {
		return nil, fmt.Errorf("rulebook: list versions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []VersionMeta
	for rows.Next() {
		var m VersionMeta
		if err := rows.Scan(&m.Version, &m.Source, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("rulebook: scan version: %w", err)
		}
		m.Active = m.Version == active
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rulebook: iterate versions: %w", err)
	}
	return out, nil
}

// --- Persistence helpers ---------------------------------------------------

// initSchema creates the store's tables if absent. Versions are immutable; one
// single-row active pointer; overrides are unique per (scope, key).
func (s *Store) initSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS rulebook_versions (
	version    INTEGER PRIMARY KEY AUTOINCREMENT,
	source     TEXT NOT NULL,
	payload    TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS rulebook_active (
	singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
	version   INTEGER NOT NULL REFERENCES rulebook_versions(version)
);
CREATE TABLE IF NOT EXISTS rulebook_overrides (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	scope      TEXT NOT NULL,
	rule_key   TEXT NOT NULL,
	value      TEXT NOT NULL,
	note       TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	UNIQUE (scope, rule_key)
);`
	if _, err := s.pools.Write().ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("rulebook: init schema: %w", err)
	}
	return nil
}

// insertVersion stores cfg as a new immutable version and returns its number.
func (s *Store) insertVersion(ctx context.Context, cfg league.RawConfig) (int, error) {
	payload, err := json.Marshal(cfg)
	if err != nil {
		return 0, fmt.Errorf("rulebook: marshal config: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.pools.Write().ExecContext(ctx,
		`INSERT INTO rulebook_versions (source, payload, created_at) VALUES (?, ?, ?)`,
		cfg.Source, string(payload), now)
	if err != nil {
		return 0, fmt.Errorf("rulebook: insert version: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("rulebook: version id: %w", err)
	}
	return int(id), nil
}

// setActive repoints the single active-version row to ver.
func (s *Store) setActive(ctx context.Context, ver int) error {
	_, err := s.pools.Write().ExecContext(ctx,
		`INSERT INTO rulebook_active (singleton, version) VALUES (1, ?)
		 ON CONFLICT(singleton) DO UPDATE SET version = excluded.version`, ver)
	if err != nil {
		return fmt.Errorf("rulebook: set active: %w", err)
	}
	return nil
}

// activeVersion returns the active version number, or 0 when none is set yet.
func (s *Store) activeVersion(ctx context.Context) (int, error) {
	var ver int
	row := s.pools.Read().QueryRowContext(ctx, `SELECT version FROM rulebook_active WHERE singleton = 1`)
	switch err := row.Scan(&ver); err {
	case nil:
		return ver, nil
	case sql.ErrNoRows:
		return 0, nil
	default:
		return 0, fmt.Errorf("rulebook: read active version: %w", err)
	}
}

// loadActive reads the active version's payload and override layer into memory
// under the write lock, so concurrent readers see a consistent snapshot.
func (s *Store) loadActive(ctx context.Context) error {
	ver, err := s.activeVersion(ctx)
	if err != nil {
		return err
	}
	var payload string
	row := s.pools.Read().QueryRowContext(ctx,
		`SELECT payload FROM rulebook_versions WHERE version = ?`, ver)
	if err := row.Scan(&payload); err != nil {
		return fmt.Errorf("rulebook: load active payload: %w", err)
	}
	var cfg league.RawConfig
	if err := json.Unmarshal([]byte(payload), &cfg); err != nil {
		return fmt.Errorf("rulebook: decode active payload: %w", err)
	}
	ovs, err := s.loadOverrides(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.active, s.activeVer, s.overrides = cfg, ver, ovs
	s.mu.Unlock()
	return nil
}

// loadOverrides reads the override layer keyed by scope+key.
func (s *Store) loadOverrides(ctx context.Context) (map[string]Override, error) {
	rows, err := s.pools.Read().QueryContext(ctx,
		`SELECT scope, rule_key, value, note, created_at FROM rulebook_overrides`)
	if err != nil {
		return nil, fmt.Errorf("rulebook: load overrides: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]Override{}
	for rows.Next() {
		var o Override
		if err := rows.Scan(&o.Scope, &o.Key, &o.Value, &o.Note, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("rulebook: scan override: %w", err)
		}
		out[ovKey(o.Scope, o.Key)] = o
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rulebook: iterate overrides: %w", err)
	}
	return out, nil
}

// ovKey is the in-memory override map key.
func ovKey(scope, key string) string { return scope + "\x00" + key }
