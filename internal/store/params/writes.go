package params

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/numeric"
)

// SetOverride upserts an admin override for (key, position) and refreshes the
// in-memory layer. The value is range-checked against the parameter's shipped bounds
// BEFORE the write (the M3 planted-failure gate), so an out-of-range value or an
// unknown parameter never reaches the DB. Admin-only write path — NEVER routed
// through B7 (AD-05).
func (s *Store) SetOverride(ctx context.Context, key, position string, value float64, note string) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	if err := s.validateOverride(key, position, value); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.pools.Write().ExecContext(ctx,
		`INSERT INTO param_overrides (param_key, position, value, note, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(param_key, position) DO UPDATE SET value = excluded.value,
		   note = excluded.note, updated_at = excluded.updated_at`,
		key, position, ftoa(value), note, now)
	if err != nil {
		return fmt.Errorf("params: set override %q: %w", key, err)
	}
	return s.load(ctx)
}

// validateOverride enforces that the target parameter exists and the new value is
// within its shipped [Min,Max] bounds. A value outside the range is REJECTED, never
// silently clamped or stored — the data-integrity gate.
func (s *Store) validateOverride(key, position string, value float64) error {
	s.mu.RLock()
	def, ok := s.defs[defKey(key, position)]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("params: override targets unknown parameter %q (position %q)", key, position)
	}
	// Reject non-finite first: a NaN passes BOTH range comparisons (unordered) and
	// would round-trip silently into engine calibration. ±Inf would fail the range
	// check below, but we reject all non-finite values explicitly and uniformly.
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("params: override %q = %v is not a finite number", key, value)
	}
	if value < def.Min || value > def.Max {
		return fmt.Errorf("params: override %q = %g is outside [%g, %g]", key, value, def.Min, def.Max)
	}
	return nil
}

// initSchema creates the store's two tables if absent: param_defaults (the immutable
// shipped set, seeded once) and param_overrides (the admin write surface). Both are
// keyed by (param_key, position) so a future per-position parameter is just more rows.
func (s *Store) initSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS param_defaults (
	param_key     TEXT    NOT NULL,
	position      TEXT    NOT NULL DEFAULT '',
	value_type    TEXT    NOT NULL,
	default_val   TEXT    NOT NULL,
	min_val       TEXT    NOT NULL,
	max_val       TEXT    NOT NULL,
	is_calibrated INTEGER NOT NULL DEFAULT 0,
	description   TEXT    NOT NULL DEFAULT '',
	PRIMARY KEY (param_key, position)
);
CREATE TABLE IF NOT EXISTS param_overrides (
	param_key  TEXT NOT NULL,
	position   TEXT NOT NULL DEFAULT '',
	value      TEXT NOT NULL,
	note       TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL,
	PRIMARY KEY (param_key, position)
);`
	if _, err := s.pools.Write().ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("params: init schema: %w", err)
	}
	return nil
}

// hasDefaults reports whether the defaults table has already been seeded.
func (s *Store) hasDefaults(ctx context.Context) (bool, error) {
	var n int
	row := s.pools.Read().QueryRowContext(ctx, `SELECT COUNT(1) FROM param_defaults`)
	if err := row.Scan(&n); err != nil {
		return false, fmt.Errorf("params: count defaults: %w", err)
	}
	return n > 0, nil
}

// seedDefaults inserts the shipped defaults in one transaction (all-or-nothing).
func (s *Store) seedDefaults(ctx context.Context) error {
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("params: begin seed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, d := range defaultParams() {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO param_defaults
			   (param_key, position, value_type, default_val, min_val, max_val, is_calibrated, description)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			d.Key, d.Position, string(d.Type),
			ftoa(d.Default), ftoa(d.Min), ftoa(d.Max),
			numeric.BoolToInt(d.IsCalibrated), d.Description); err != nil {
			return fmt.Errorf("params: seed %q: %w", d.Key, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("params: commit seed: %w", err)
	}
	return nil
}

// load reads both tables into memory under the write lock, so concurrent readers see
// a consistent defaults+overrides snapshot.
func (s *Store) load(ctx context.Context) error {
	defs, err := s.loadDefaults(ctx)
	if err != nil {
		return err
	}
	ovs, err := s.loadOverrides(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.defs, s.overrides = defs, ovs
	s.mu.Unlock()
	return nil
}

// loadDefaults reads the shipped defaults keyed by (key, position).
func (s *Store) loadDefaults(ctx context.Context) (map[string]ParamDef, error) {
	rows, err := s.pools.Read().QueryContext(ctx,
		`SELECT param_key, position, value_type, default_val, min_val, max_val, is_calibrated, description
		   FROM param_defaults`)
	if err != nil {
		return nil, fmt.Errorf("params: load defaults: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]ParamDef{}
	for rows.Next() {
		d, derr := scanDef(rows)
		if derr != nil {
			return nil, derr
		}
		out[defKey(d.Key, d.Position)] = d
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("params: iterate defaults: %w", err)
	}
	return out, nil
}

// loadOverrides reads the admin override layer keyed by (key, position).
func (s *Store) loadOverrides(ctx context.Context) (map[string]Override, error) {
	rows, err := s.pools.Read().QueryContext(ctx,
		`SELECT param_key, position, value, note, updated_at FROM param_overrides`)
	if err != nil {
		return nil, fmt.Errorf("params: load overrides: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]Override{}
	for rows.Next() {
		o, oerr := scanOverride(rows)
		if oerr != nil {
			return nil, oerr
		}
		out[defKey(o.Key, o.Position)] = o
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("params: iterate overrides: %w", err)
	}
	return out, nil
}
