//go:build leaguebench

package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type rawBigLeaguePreview commissioner.RawNativeSnapshot

// Only the season, franchise identity and player contracts cross into the preview baseline.
func (r rawBigLeaguePreview) Validate() error {
	if r.Season < 2013 || r.Season > 9999 || len(r.Franchises) != 32 || len(r.Players) == 0 {
		return fmt.Errorf("invalid preview fixture scope")
	}
	teams, players := map[string]bool{}, map[string]bool{}
	for _, team := range r.Franchises {
		if team.ID == "" || teams[team.ID] {
			return fmt.Errorf("invalid fixture franchise")
		}
		teams[team.ID] = true
	}
	for _, p := range r.Players {
		id, err := playerid.New(p.ID)
		if err != nil || id.String() != p.ID || players[p.ID] || !teams[p.Franchise] || p.Salary < 0 || p.Salary > 100000000000000 || p.ExpirationYear > 9999 {
			return fmt.Errorf("invalid fixture player %q", p.ID)
		}
		if p.Status != domain.RosterActive && p.Status != domain.RosterTaxi && p.Status != domain.RosterIR {
			return fmt.Errorf("invalid fixture roster status")
		}
		players[p.ID] = true
	}
	charges := map[string]bool{}
	for _, c := range r.Adjustments {
		if c.Season != 0 && c.Season != r.Season {
			continue
		}
		if c.ID == "" || charges[c.ID] || !teams[c.Franchise] || c.Amount < -100000000000000 || c.Amount > 100000000000000 {
			return fmt.Errorf("invalid fixture adjustment")
		}
		charges[c.ID] = true
	}

	return nil
}

func readBigLeaguePreview(b *testing.B, tier string) rawBigLeaguePreview {
	b.Helper()
	dir := os.Getenv("BENCH_LEAGUE_SCRATCH")
	if dir == "" {
		b.Fatal("run make bench-league to generate native inputs first")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			b.Error(err)
		}
	}()
	body, err := root.ReadFile(tier + ".json")
	if err != nil {
		b.Fatal(err)
	}
	var raw rawBigLeaguePreview
	if err := json.Unmarshal(body, &raw); err != nil {
		b.Fatal(err)
	}
	if err := raw.Validate(); err != nil {
		b.Fatal(err)
	}
	return raw
}

func BenchmarkLeagueWhatIf(b *testing.B) {
	for _, tier := range []string{"legacy", "double"} {
		b.Run(tier+"/D4Season", func(b *testing.B) {
			raw := readBigLeaguePreview(b, tier)
			// Full-season 2026: round(287*1500/809) adds + round(109*1500/809) trades.
			const previews = 532 + 202
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				path := filepath.Join(b.TempDir(), "whatif.db")
				pools, err := db.Open(b.Context(), path, state.SchemaVersion())
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
				for n := range previews {
					if err := seedBigLeaguePreview(b.Context(), pools, raw, n); err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				reportBigLeagueWhatIf(b, pools, previews)
				if err := pools.Close(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func reportBigLeagueWhatIf(b *testing.B, pools *db.Pools, previews int) {
	b.Helper()
	var rosters, contracts, cells, changes, dead, sessions, counts, adjustments int64
	err := pools.Read().QueryRow(`SELECT
		(SELECT count(*) FROM rosters), (SELECT count(*) FROM contracts),
		(SELECT count(*) FROM contract_years), (SELECT count(*) FROM contract_year_changes),
		(SELECT count(*) FROM dead_cap_ledger), (SELECT count(*) FROM whatif_sessions),
		(SELECT count(*) FROM transaction_counts),
		(SELECT count(*) FROM mfl_salary_adjustments)`).Scan(&rosters, &contracts, &cells, &changes, &dead, &sessions, &counts, &adjustments)
	if err != nil {
		b.Fatal(err)
	}
	var generations int
	if err := pools.Read().QueryRow("SELECT count(DISTINCT league_id) FROM rosters").Scan(&generations); err != nil {
		b.Fatal(err)
	}
	if generations != previews || rosters == 0 || rosters != contracts || cells < rosters {
		b.Fatal("previews did not retain one complete franchise generation apiece")
	}
	if _, err := pools.Write().Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		b.Fatal(err)
	}
	var pages, pageSize int64
	if err := pools.Read().QueryRow("PRAGMA page_count").Scan(&pages); err != nil {
		b.Fatal(err)
	}
	if err := pools.Read().QueryRow("PRAGMA page_size").Scan(&pageSize); err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(previews), "previews/op")
	b.ReportMetric(float64(pages*pageSize), "db-bytes/op")
	b.ReportMetric(float64(rosters+contracts+cells+changes+dead+sessions+counts+adjustments), "rows/op")
	b.Logf("D4 N=%d bytes=%d rosters=%d contracts=%d contract_years=%d contract_year_changes=%d dead_cap_ledger=%d whatif_sessions=%d transaction_counts=%d mfl_salary_adjustments=%d", previews, pages*pageSize, rosters, contracts, cells, changes, dead, sessions, counts, adjustments)
}

func seedBigLeaguePreview(ctx context.Context, pools *db.Pools, raw rawBigLeaguePreview, n int) error {
	mine := raw.Franchises[n%32].ID
	snapshot := state.MirrorSnapshot{Season: raw.Season}
	for _, p := range raw.Players {
		if p.Franchise == mine {
			snapshot.Players = append(snapshot.Players, state.MirrorPlayer{MFLID: p.ID, FranchiseID: mine,
				RosterStatus: p.Status, Salary: p.Salary, ContractYear: p.ExpirationYear, ContractStatus: p.ContractStatus})
		}
	}

	for _, c := range raw.Adjustments {
		if c.Season == 0 || c.Season == raw.Season {
			snapshot.Adjustments = append(snapshot.Adjustments, state.Adjustment{ID: c.ID, FranchiseID: c.Franchise, Amount: c.Amount, Description: c.Description})
		}
	}
	if len(snapshot.Players) == 0 {
		return fmt.Errorf("preview franchise has no players")
	}
	session := state.WhatIfSession{Mine: mine, Snapshot: snapshot, Generation: fmt.Sprintf("contract-preview:%s:%06d", mine, n), TaxiPercent: 1, IRPercent: 1}
	store, err := LoadWhatIf(ctx, pools, session)
	if err != nil {
		return err
	}
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		return fmt.Errorf("preview coordinator: %w", err)
	}
	player := snapshot.Players[0]
	if err := coord.ProjectContract(ctx, player.MFLID, []contractacts.Cell{{Season: raw.Season, Salary: player.Salary + 1000000}}); err != nil {
		return fmt.Errorf("project preview contract: %w", err)
	}
	return nil
}

func TestBigLeaguePreviewRetainsGenerations(t *testing.T) {
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "whatif.db"), state.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	raw := rawBigLeaguePreview{Season: 2026}
	for n := range 32 {
		team := fmt.Sprintf("franchise.%02d", n+1)
		raw.Franchises = append(raw.Franchises, commissioner.NativeFranchise{ID: team})
		raw.Players = append(raw.Players, commissioner.NativePlayer{ID: fmt.Sprintf("%04d", n+1000), Franchise: team,
			Status: domain.RosterActive, Salary: 100000000, ExpirationYear: 2027, ContractStatus: domain.CStatusUFA})
	}
	raw.Adjustments = []commissioner.NativeCharge{{ID: "current", Franchise: raw.Franchises[31].ID, Season: 2026, Amount: 1000000},
		{ID: "past", Franchise: raw.Franchises[0].ID, Season: 2025, Amount: 2000000}}
	if err := raw.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{0, 32} {
		if err := seedBigLeaguePreview(t.Context(), pools, raw, n); err != nil {
			t.Fatal(err)
		}
	}
	var rosters, cells, changes int
	if err := pools.Read().QueryRow(`SELECT (SELECT count(*) FROM rosters), (SELECT count(*) FROM contract_years), (SELECT count(*) FROM contract_year_changes)`).Scan(&rosters, &cells, &changes); err != nil {
		t.Fatal(err)
	}
	var adjustments int
	if err := pools.Read().QueryRow("SELECT count(*) FROM mfl_salary_adjustments").Scan(&adjustments); err != nil {
		t.Fatal(err)
	}
	if adjustments != 2 {
		t.Fatalf("current league adjustments=%d, want one per generation", adjustments)
	}
	if rosters != 2 || cells != 6 || changes != 8 {
		t.Fatalf("retained rows: rosters=%d cells=%d changes=%d", rosters, cells, changes)
	}
}
