// Package contractstest builds the fixtures the contracts package tests and the root cross-feature
// scenarios share: real self-locking stores over temporary databases, the five-player release
// mirror, the league phase and the held Arizona snapshot. It never imports the contracts package,
// so the package's own tests can use it; each side wraps these values in its own host.
package contractstest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Stores are the plan stores over temporary databases.
type Stores struct {
	Ctx         context.Context //nolint:containedctx // the fixture's stores share one test-lifetime context
	Pools       *db.Pools
	Rulebook    *rulebook.Store
	League      *state.Mirror
	Moves       *moves.Store
	WhatifPools *db.Pools
	Whatif      *state.Store
}

// Open is the league database holding one player (0042 on franchise 0001), an empty plan ledger
// and an empty what-if database. Whatif, the league phase store, is not opened.
func Open(t *testing.T) *Stores {
	t.Helper()
	ctx := context.Background()
	s := &Stores{Ctx: ctx, Pools: pools(t, "league.db")}
	s.Rulebook = rulebook.New(s.Pools)
	s.League = state.NewMirror(s.Pools, s.Rulebook)
	if err := s.League.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.League.Replace(ctx, state.MirrorSnapshot{Season: 2026,
		Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001", RosterStatus: domain.RosterActive}}}); err != nil {
		t.Fatal(err)
	}
	s.Moves = moves.New(s.Pools)
	if err := s.Moves.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	s.WhatifPools = pools(t, "whatif.db")
	return s
}

// Release is the release fixture: franchise 0001 holds 0042–0046 at $10M, and the league phase
// store is seeded from that mirror.
func Release(t *testing.T) *Stores {
	t.Helper()
	s := Open(t)
	s.ReplaceMirror(t, 10*100_000_000)
	m, _, _ := s.League.ContractsView()
	s.Whatif = state.New(s.WhatifPools, ingestion.LeagueID, 2026, s.Rulebook)
	if err := s.Whatif.Initialize(s.Ctx, state.SnapshotSource{Snapshot: m}); err != nil {
		t.Fatal(err)
	}
	return s
}

// ReplaceMirror holds 0042–0046 on franchise 0001 at salary, expiring 2028 but for 0045 (2027) and
// 0046 (2026), with one literal MFL adjustment.
func (s *Stores) ReplaceMirror(t *testing.T, salary domain.Money) {
	t.Helper()
	players := []state.MirrorPlayer{}
	for i, id := range []string{"0042", "0043", "0044", "0045", "0046"} {
		expiration := 2028
		if i == 3 {
			expiration = 2027
		}
		if i == 4 {
			expiration = 2026
		}
		players = append(players, state.MirrorPlayer{MFLID: id, FranchiseID: "0001", Salary: salary, ContractYear: expiration, RosterStatus: domain.RosterActive})
	}
	if _, err := s.League.Replace(s.Ctx, state.MirrorSnapshot{Season: 2026, Players: players, Adjustments: []state.Adjustment{{ID: "a", FranchiseID: "0001", Amount: -12345}}}); err != nil {
		t.Fatal(err)
	}
}

// SetPhase moves the 2026 league to phase.
func (s *Stores) SetPhase(t *testing.T, phase domain.Phase) {
	t.Helper()
	_, err := s.WhatifPools.Write().ExecContext(s.Ctx, `INSERT INTO season_phases (league_id,season,from_phase,to_phase,note,at) VALUES (?,2026,'',?,'fixture',?)`, ingestion.LeagueID, string(phase), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
}

// HeldSnapshot is the frontend's held Arizona snapshot fixture.
func HeldSnapshot(t *testing.T) snapshot.Snapshot {
	t.Helper()
	data, err := os.ReadFile(RepoPath("frontend", "src", "app", "data", "fixtures", "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s snapshot.Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

// RepoPath is a path from the repository root, wherever the test runs.
func RepoPath(parts ...string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(append([]string{filepath.Dir(file), "..", "..", "..", ".."}, parts...)...)
}

func pools(t *testing.T, name string) *db.Pools {
	t.Helper()
	p, err := db.Open(context.Background(), filepath.Join(t.TempDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	return p
}
