package state

import (
	"context"
	"database/sql"
	"encoding/json"
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

type WhatIfMove struct {
	ID          string       `json:"id"`
	Kind        string       `json:"kind"`
	Player      string       `json:"player"`
	Name        string       `json:"name"`
	Charge      domain.Money `json:"charge"`
	Explanation string       `json:"explanation"`
}

type WhatIfSession struct {
	Generation  string         `json:"generation"`
	Mine        string         `json:"mine"`
	Stamps      MirrorStamps   `json:"stamps"`
	Snapshot    MirrorSnapshot `json:"snapshot"`
	SeedUsed    domain.Money   `json:"seedUsed"`
	TaxiPercent float64        `json:"taxiPercent"`
	IRPercent   float64        `json:"irPercent"`
	Used        domain.Money   `json:"used"`
	Cap         *domain.Money  `json:"cap"`
	Moves       []WhatIfMove   `json:"moves"`
}

// WhatIfSessions publishes one complete session per franchise. Failed replays leave the
// previous session intact; generation state is isolated from the legacy transaction screens.
type WhatIfSessions struct{ pools *db.Pools }

func NewWhatIfSessions(pools *db.Pools) *WhatIfSessions { return &WhatIfSessions{pools: pools} }

func (s *WhatIfSessions) Load(ctx context.Context, mine string) (WhatIfSession, bool, error) {
	if _, err := s.pools.Write().ExecContext(ctx, `CREATE TABLE IF NOT EXISTS whatif_sessions (franchise_id TEXT PRIMARY KEY, session TEXT NOT NULL)`); err != nil {
		return WhatIfSession{}, false, fmt.Errorf("what-if schema: %w", err)
	}
	var raw string
	err := s.pools.Read().QueryRowContext(ctx, `SELECT session FROM whatif_sessions WHERE franchise_id = ?`, mine).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return WhatIfSession{}, false, nil
	}
	if err != nil {
		return WhatIfSession{}, false, fmt.Errorf("what-if load: %w", err)
	}
	var out WhatIfSession
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return out, false, fmt.Errorf("what-if decode: %w", err)
	}
	return out, true, nil
}
func (s *WhatIfSessions) Save(ctx context.Context, session WhatIfSession) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("what-if encode: %w", err)
	}
	if _, err := s.pools.Write().ExecContext(ctx, `INSERT INTO whatif_sessions (franchise_id, session) VALUES (?, ?) ON CONFLICT(franchise_id) DO UPDATE SET session = excluded.session`, session.Mine, string(raw)); err != nil {
		return fmt.Errorf("what-if save: %w", err)
	}
	return nil
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

func (s WhatIfSession) TaxiCapPercent() float64 { return s.TaxiPercent }
func (s WhatIfSession) IRCapPercent() float64   { return s.IRPercent }

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
