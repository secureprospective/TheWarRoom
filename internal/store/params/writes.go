package params

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/numeric"
)

// SetOverride range-checks and upserts an admin override, then reloads memory. An unknown
// parameter or an out-of-range value never reaches the database.
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

// ClearOverride removes an admin override, so the parameter follows its shipped default again,
// a later release's included. Clearing a parameter with no override is a no-op.
func (s *Store) ClearOverride(ctx context.Context, key, position string) error {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	s.mu.RLock()
	_, ok := s.defs[defKey(key, position)]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("params: reset targets unknown parameter %q (position %q)", key, position)
	}
	if _, err := s.pools.Write().ExecContext(ctx,
		`DELETE FROM param_overrides WHERE param_key = ? AND position = ?`, key, position); err != nil {
		return fmt.Errorf("params: clear override %q: %w", key, err)
	}
	return s.load(ctx)
}

// validateOverride rejects an unknown parameter or a value outside its [Min,Max]; it never
// clamps.
func (s *Store) validateOverride(key, position string, value float64) error {
	s.mu.RLock()
	def, ok := s.defs[defKey(key, position)]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("params: override targets unknown parameter %q (position %q)", key, position)
	}
	// NaN passes both range comparisons, so non-finite values are rejected first.
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("params: override %q = %v is not a finite number", key, value)
	}
	if value < def.Min || value > def.Max {
		return fmt.Errorf("params: override %q = %g is outside [%g, %g]", key, value, def.Min, def.Max)
	}
	return nil
}

// initSchema creates param_defaults and param_overrides, both keyed by (param_key, position).
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

// seedDefaults writes every shipped default in one transaction. A parameter added or refitted
// in a later release therefore reaches an existing database; admin overrides are a separate
// table and are left alone. Scoring runs record the values they used, so replacing a default
// rewrites no history.
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
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT(param_key, position) DO UPDATE SET value_type = excluded.value_type,
			   default_val = excluded.default_val, min_val = excluded.min_val, max_val = excluded.max_val,
			   is_calibrated = excluded.is_calibrated, description = excluded.description`,
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

// load reads both tables into memory under the write lock.
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
