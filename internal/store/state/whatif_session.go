package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

type MirrorStamps struct {
	Rosters  string `json:"rosters"`
	Contract string `json:"contract"`
	DeadCap  string `json:"deadCap"`
}

// Stamps uses the mirror's recorded revision, not a content hash. Legacy ledger tables
// are optional: the current mirror stores contracts and adjustments inside its snapshot.
func (m *Mirror) Stamps(ctx context.Context) (MirrorStamps, error) {
	out := MirrorStamps{Rosters: m.AsOf()}
	for _, item := range []struct {
		table, query string
		value        *string
	}{
		{"contract_years", `SELECT COALESCE(MAX(last_updated), '') FROM contract_years`, &out.Contract},
		{"dead_cap_ledger", `SELECT COALESCE(MAX(created_at), '') FROM dead_cap_ledger`, &out.DeadCap},
	} {
		var n int
		if err := m.pools.Read().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, item.table).Scan(&n); err != nil {
			return out, fmt.Errorf("mirror stamps: table: %w", err)
		}
		if n == 0 {
			continue
		}
		if err := m.pools.Read().QueryRowContext(ctx, item.query).Scan(item.value); err != nil {
			return out, fmt.Errorf("mirror stamps: %s: %w", item.table, err)
		}
	}
	return out, nil
}

func (s MirrorStamps) NewerThan(old MirrorStamps) bool {
	return stampNewer(s.Rosters, old.Rosters) || stampNewer(s.Contract, old.Contract) || stampNewer(s.DeadCap, old.DeadCap)
}

// SnapshotSource keeps seeding pinned to the stored baseline during replay and restart.
type SnapshotSource struct{ Snapshot MirrorSnapshot }

func (s SnapshotSource) Rosters(ctx context.Context) ([]domain.Roster, error) {
	m := &Mirror{snap: s.Snapshot}
	return m.Rosters(ctx)
}
func (s SnapshotSource) Adjustments(context.Context) ([]Adjustment, error) {
	return s.Snapshot.Adjustments, nil
}

func stampNewer(current, baseline string) bool {
	if current == "" {
		return false
	}
	if baseline == "" {
		return true
	}
	now, nowErr := time.Parse(time.RFC3339Nano, current)
	old, oldErr := time.Parse(time.RFC3339Nano, baseline)
	if nowErr == nil && oldErr == nil {
		return now.After(old)
	}
	return current > baseline
}

// ReleaseStamps reads only the subject contract and owner's ledger rows. Rosters still use
// the mirror revision; literal player and adjustment fields distinguish unrelated refreshes.
func (m *Mirror) ReleaseStamps(ctx context.Context, player, mine string) (MirrorStamps, error) {
	out := MirrorStamps{Rosters: m.AsOf()}
	for _, item := range []struct {
		table, query string
		value        *string
		scope        string
	}{
		{"contract_years", `SELECT COALESCE(MAX(last_updated),'') FROM contract_years WHERE mfl_id=?`, &out.Contract, player},
		{"dead_cap_ledger", `SELECT COALESCE(MAX(created_at),'') FROM dead_cap_ledger WHERE franchise_id=?`, &out.DeadCap, mine},
	} {
		var n int
		if err := m.pools.Read().QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, item.table).Scan(&n); err != nil {
			return out, fmt.Errorf("release stamps table: %w", err)
		}
		if n == 0 {
			continue
		}
		if err := m.pools.Read().QueryRowContext(ctx, item.query, item.scope).Scan(item.value); err != nil {
			return out, fmt.Errorf("release stamps %s: %w", item.table, err)
		}
	}
	return out, nil
}

const whatIfSessionsDDL = `CREATE TABLE IF NOT EXISTS whatif_sessions (franchise_id TEXT PRIMARY KEY, session TEXT NOT NULL);`

// WhatIfRecords stores session JSON under the shared whatif.db migration sequence.
type WhatIfRecords struct{ pools *db.Pools }

func NewWhatIfRecords(pools *db.Pools) *WhatIfRecords { return &WhatIfRecords{pools: pools} }

func (s *WhatIfRecords) Load(ctx context.Context, mine string) (string, bool, error) {
	var raw string
	err := s.pools.Read().QueryRowContext(ctx, `SELECT session FROM whatif_sessions WHERE franchise_id = ?`, mine).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("session read: %w", err)
	}
	return raw, true, nil
}

func (s *WhatIfRecords) Save(ctx context.Context, mine, raw string) error {
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO whatif_sessions (franchise_id, session) VALUES (?, ?) ON CONFLICT(franchise_id) DO UPDATE SET session = excluded.session`, mine, raw); err != nil {
		return fmt.Errorf("session write: %w", err)
	}
	return nil
}
