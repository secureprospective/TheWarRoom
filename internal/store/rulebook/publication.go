package rulebook

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
)

// CapPercents is the staged rulebook's discount view, including commissioner overrides.
type CapPercents struct{ taxi, ir float64 }

func (c CapPercents) TaxiCapPercent() float64 { return c.taxi }
func (c CapPercents) IRCapPercent() float64   { return c.ir }

// StageSync holds the writer lock until the prepared write is committed or released.
func (s *Store) StageSync(cfg league.RawConfig) (db.PreparedWrite, CapPercents, bool, error) {
	s.wmu.Lock()
	prepared, caps, changed, err := s.stageSync(cfg)
	if err != nil {
		s.wmu.Unlock()
		return db.PreparedWrite{}, CapPercents{}, false, err
	}
	return prepared, caps, changed, nil
}

func (s *Store) stageSync(cfg league.RawConfig) (db.PreparedWrite, CapPercents, bool, error) {
	if err := validateConfigPercents(cfg); err != nil {
		return db.PreparedWrite{}, CapPercents{}, false, err
	}
	s.mu.RLock()
	active, version := s.active, s.activeVer
	taxi, ir := cfg.IncludeTaxiWithSalary, cfg.IncludeIRWithSalary
	if o, ok := s.overrides[ovKey(scopeSetting, "includeTaxiWithSalary")]; ok {
		taxi = o.Value
	}
	if o, ok := s.overrides[ovKey(scopeSetting, "includeIRWithSalary")]; ok {
		ir = o.Value
	}
	s.mu.RUnlock()
	same, err := sameConfig(active, cfg)
	if err != nil {
		return db.PreparedWrite{}, CapPercents{}, false, err
	}
	if same {
		cfg = active
	}
	cfg = cloneConfig(cfg)
	payload, err := json.Marshal(cfg)
	if err != nil {
		return db.PreparedWrite{}, CapPercents{}, false, fmt.Errorf("rulebook: stage config: %w", err)
	}
	prepared := db.PreparedWrite{
		Persist: func(ctx context.Context, tx *sql.Tx) error {
			if same {
				return nil
			}
			var err error
			version, err = persistConfig(ctx, tx, cfg.Source, string(payload))
			return err
		},
		Lock: s.mu.Lock, Unlock: s.mu.Unlock, Release: s.wmu.Unlock,
		Publish: func() { s.active, s.activeVer = cfg, version },
	}
	return prepared, CapPercents{taxi: capPercent(taxi), ir: capPercent(ir)}, !same, nil
}

func persistConfig(ctx context.Context, tx *sql.Tx, source, payload string) (int, error) {
	version, err := insertConfig(ctx, tx, source, payload)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO rulebook_active (singleton, version) VALUES (1, ?) ON CONFLICT(singleton) DO UPDATE SET version = excluded.version`, version); err != nil {
		return 0, fmt.Errorf("rulebook: set active: %w", err)
	}
	return version, nil
}

func insertConfig(ctx context.Context, tx *sql.Tx, source, payload string) (int, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO rulebook_versions (source, payload, created_at) VALUES (?, ?, ?)`, source, payload, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("rulebook: insert version: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("rulebook: version id: %w", err)
	}
	return int(id), nil
}

func validateConfigPercents(cfg league.RawConfig) error {
	for _, raw := range []string{cfg.IncludeTaxiWithSalary, cfg.IncludeIRWithSalary} {
		if strings.TrimSpace(raw) != "" {
			if err := validateCapPercent(raw); err != nil {
				return err
			}
		}
	}
	return nil
}
