package state

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Mirror is the league as MFL states it: the season, every rostered player's contract and the
// salary adjustments. Refresh replaces it whole with Replace, and nothing else writes it (MFL
// wins, R2). It serves the same Reader as the what-if Store, and seeds that Store as its Source.
type Mirror struct {
	pools     *db.Pools
	discounts CapDiscounts
	wmu       sync.Mutex
	leagueView
	snap MirrorSnapshot
	hash string
}

// MirrorSnapshot is one refresh's view of the league.
type MirrorSnapshot struct {
	Season      int
	Players     []MirrorPlayer // any order; Replace sorts
	Adjustments []Adjustment
}

// MirrorPlayer is one rostered player's contract as MFL states it.
type MirrorPlayer struct {
	MFLID          string
	FranchiseID    string
	RosterStatus   domain.RosterStatus
	Salary         domain.Money
	ContractYear   int
	ContractStatus domain.ContractStatus
	ContractInfo   string
}

// Adjustment is one MFL salary adjustment: a dead-cap charge, or a credit when negative.
type Adjustment struct {
	ID          string
	FranchiseID string
	Amount      domain.Money
	Description string
}

const mirrorDDL = `
CREATE TABLE IF NOT EXISTS league_mirror (
	id           INTEGER PRIMARY KEY CHECK (id = 1),
	season       INTEGER NOT NULL,
	sha256       TEXT NOT NULL,
	snapshot     TEXT NOT NULL,
	refreshed_at TEXT NOT NULL
);`

// NewMirror returns an unloaded mirror over pools. discounts supplies the taxi and IR cap
// percentages.
func NewMirror(pools *db.Pools, discounts CapDiscounts) *Mirror {
	return &Mirror{pools: pools, discounts: discounts, leagueView: newLeagueView()}
}

// Initialize creates the table and loads what it holds. An empty mirror is legal: the first
// refresh fills it.
func (m *Mirror) Initialize(ctx context.Context) error {
	if _, err := m.pools.Write().ExecContext(ctx, mirrorDDL); err != nil {
		return fmt.Errorf("state: mirror schema: %w", err)
	}
	var raw string
	err := m.pools.Read().QueryRowContext(ctx, `SELECT snapshot FROM league_mirror WHERE id = 1`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("state: mirror load: %w", err)
	}
	var snap MirrorSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return fmt.Errorf("state: mirror decode: %w", err)
	}
	m.wmu.Lock()
	defer m.wmu.Unlock()
	_, err = m.install(snap)
	return err
}

// Replace makes snap the mirror. It writes only when snap differs from what is held, and
// reports whether it did; either way it rebuilds the view, so new cap percentages apply.
func (m *Mirror) Replace(ctx context.Context, snap MirrorSnapshot) (bool, error) {
	if snap.Season == 0 || len(snap.Players) == 0 {
		return false, fmt.Errorf("state: mirror refresh for season %d with %d players refused: an empty league would wipe the mirror", snap.Season, len(snap.Players))
	}
	m.wmu.Lock()
	defer m.wmu.Unlock()
	prev := m.hash
	enc, err := m.install(snap)
	if err != nil || m.hash == prev {
		return false, err
	}
	if _, err := m.pools.Write().ExecContext(ctx, `
INSERT INTO league_mirror (id, season, sha256, snapshot, refreshed_at) VALUES (1, ?, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET season = excluded.season, sha256 = excluded.sha256,
	snapshot = excluded.snapshot, refreshed_at = excluded.refreshed_at`,
		snap.Season, m.hash, string(enc), time.Now().UTC().Format(time.RFC3339)); err != nil {
		return false, fmt.Errorf("state: mirror write: %w", err)
	}
	return true, nil
}

// install sorts and validates snap, builds its view and swaps it in. It returns the canonical
// encoding, whose sha256 becomes the mirror's hash. The caller holds wmu.
func (m *Mirror) install(snap MirrorSnapshot) ([]byte, error) {
	sort.Slice(snap.Players, func(i, j int) bool { return snap.Players[i].MFLID < snap.Players[j].MFLID })
	sort.Slice(snap.Adjustments, func(i, j int) bool { return snap.Adjustments[i].ID < snap.Adjustments[j].ID })
	fr := map[string]*FranchiseState{}
	idx := make(map[string]string, len(snap.Players))
	franchise := func(id string) *FranchiseState {
		if fr[id] == nil {
			fr[id] = &FranchiseState{FranchiseID: id}
		}
		return fr[id]
	}
	for _, p := range snap.Players {
		if _, dup := idx[p.MFLID]; dup {
			return nil, fmt.Errorf("state: mirror lists player %s twice", p.MFLID)
		}
		idx[p.MFLID] = p.FranchiseID
		f := franchise(p.FranchiseID)
		f.Players = append(f.Players, PlayerState{
			MFLID: p.MFLID, FranchiseID: p.FranchiseID, RosterStatus: p.RosterStatus,
			Salary: p.Salary, CapSalary: p.Salary, ExpirationYear: p.ContractYear, ContractStatus: p.ContractStatus,
		})
		f.CapUsed += domain.RoundToNearest10k(capContribution(p.Salary, p.RosterStatus, m.discounts))
	}
	for _, a := range snap.Adjustments {
		franchise(a.FranchiseID).CapUsed += a.Amount
	}
	enc, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("state: mirror encode: %w", err)
	}
	sum := sha256.Sum256(enc)
	m.mu.Lock()
	m.franchises, m.byPlayer = fr, idx
	m.mu.Unlock()
	m.snap, m.hash = snap, hex.EncodeToString(sum[:])
	return enc, nil
}

// Reader returns the read surface.
func (m *Mirror) Reader() Reader { return &m.leagueView }

// Season is the season the mirror holds; 0 before the first refresh.
func (m *Mirror) Season() int {
	m.wmu.Lock()
	defer m.wmu.Unlock()
	return m.snap.Season
}

// DeadCap is each franchise's net salary adjustments: the dead cap MFL charges, less any credits.
func (m *Mirror) DeadCap() map[string]domain.Money {
	m.wmu.Lock()
	defer m.wmu.Unlock()
	out := map[string]domain.Money{}
	for _, a := range m.snap.Adjustments {
		out[a.FranchiseID] += a.Amount
	}
	return out
}

// Rosters returns the mirror as per-franchise records, so it can seed a what-if Store.
func (m *Mirror) Rosters(_ context.Context) ([]domain.Roster, error) {
	m.wmu.Lock()
	defer m.wmu.Unlock()
	byFr := map[string][]domain.PlayerRecord{}
	for _, p := range m.snap.Players {
		id, err := playerid.New(p.MFLID)
		if err != nil {
			return nil, fmt.Errorf("state: mirror player id: %w", err)
		}
		byFr[p.FranchiseID] = append(byFr[p.FranchiseID], domain.PlayerRecord{
			MFLID: id, FranchiseID: p.FranchiseID, RosterStatus: p.RosterStatus, Salary: p.Salary,
			ContractYear: p.ContractYear, ContractStatus: p.ContractStatus, ContractInfo: p.ContractInfo,
		})
	}
	out := make([]domain.Roster, 0, len(byFr))
	for _, fid := range slices.Sorted(maps.Keys(byFr)) {
		out = append(out, domain.Roster{FranchiseID: fid, Players: byFr[fid]})
	}
	return out, nil
}
