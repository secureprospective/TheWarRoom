// Package rulebook stores the league's MFL-sourced rules (scoring, roster limits, salary cap,
// settings) with a commissioner override layer. It computes nothing; the engine and the
// transaction coordinator read from it.
//
// MFL config is stored as immutable versions with one active pointer. Reload stores a new
// candidate and reports what changed but does not promote it; Promote repoints the active
// version, so "update to current" and "roll back" are the same operation. Writes are admin-only
// and never go through the transaction coordinator, which writes league state, not config.
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

// Store is the rulebook. Construct with New, seed with Initialize. Reads are safe from
// concurrent Wails IPC goroutines.
type Store struct {
	pools *db.Pools
	src   league.Source

	// wmu serializes admin writes so the SQLite write and the in-memory reload are one step;
	// otherwise two writers could leave memory on a version other than the committed pointer. It
	// is always taken before mu.
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

// Reload fetches the live config, stores it as a new candidate version and returns what
// changed versus the active one. It never promotes.
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

// Promote makes ver the active version. It is both the apply step and the rollback path.
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

// SetOverride validates and upserts a commissioner override: a known scope, a non-empty key
// and value, and a numeric cap.
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
		// any non-empty value is accepted
	default:
		return fmt.Errorf("rulebook: unknown override scope %q", scope)
	}
	return nil
}

// GetSalaryCap returns the active salary-cap amount, override applied.
func (s *Store) GetSalaryCap() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if o, ok := s.overrides[ovKey(scopeCap, capKey)]; ok {
		return o.Value
	}
	return s.active.SalaryCapAmount
}

// GetScoringRules returns the active scoring rules as raw MFL values.
func (s *Store) GetScoringRules() []league.PositionRuleSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneScoring(s.active.ScoringRules)
}

// GetSetting returns a scalar league setting (e.g. "rosterSize"), override applied. ok is
// false for an unknown key.
func (s *Store) GetSetting(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if o, ok := s.overrides[ovKey(scopeSetting, key)]; ok {
		return o.Value, true
	}
	v, ok := settingsMap(s.active)[key]
	return v, ok
}

// ActiveConfig returns a copy of the active config without overrides; use the typed getters
// for override-aware reads.
func (s *Store) ActiveConfig() league.RawConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneConfig(s.active)
}

// Versions lists stored config versions, newest first.
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

// initSchema creates the version, active-pointer and override tables if absent.
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

// ActiveVersion returns the active config version, or 0 before Initialize. Scored output is
// stamped with it; a caller must refuse to score on 0.
func (s *Store) ActiveVersion(ctx context.Context) (int, error) {
	return s.activeVersion(ctx)
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

// loadActive reads the active version and the overrides into memory under the write lock.
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
