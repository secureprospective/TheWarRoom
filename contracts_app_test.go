package main

import (
	"fmt"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestContractReadingStatesAndCantCut(t *testing.T) {
	r := contractReading(123, "2026-10-10T00:00:00Z", "MFL", "Integer cents", 1, nil)
	if r.Status != ReadingOK || r.Value != 123 || r.HowCounted == "" || r.Cutoff != r.Provenance.Freshness.FetchedAt {
		t.Fatalf("reading: %+v", r)
	}
	failed := failedContracts(fmt.Errorf("mirror unavailable"))
	if failed.Table.Status != ReadingFailed || failed.Cap.Status != ReadingFailed || failed.Adjustments.Status != ReadingFailed {
		t.Fatal("failure became figures")
	}
	a := &App{}
	if a.TargetContracts(snapshot.RawContractScope{Sort: "bad"}).Table.Status != ReadingFailed {
		t.Fatal("invalid scope accepted")
	}
	if a.TargetPlayerContract("bad").Status != ReadingFailed {
		t.Fatal("invalid player accepted")
	}
}

func TestTargetContractsAndPlayerReadOnlyMirror(t *testing.T) {
	a := targetTestApp(t)
	raw := snapshot.RawContractScope{Mine: "0001", Sort: "salary"}
	got := a.TargetContracts(raw)
	if got.Table.Status != ReadingOK || len(got.Table.Value.Rows) != 1 || got.Cap.Status != ReadingOK || got.Cap.Value.Cap != nil || got.Cap.Value.Room != nil || got.Adjustments.Status != ReadingOK {
		t.Fatalf("dashboard: %+v", got)
	}
	player := a.TargetPlayerContract("0042")
	if player.Status != ReadingOK || player.Value.Player.ID != "0042" || len(player.Value.Years) != 1 || player.Value.Years[0].Inferred {
		t.Fatalf("player: %+v", player)
	}
	if a.TargetPlayerContract("9999").Status != ReadingFailed {
		t.Fatal("unrostered player invented")
	}
	raw.Position = "WR"
	if len(a.TargetContracts(raw).Table.Value.Rows) != 0 {
		t.Fatal("filter failed")
	}
	raw.Mine = "9999"
	if a.TargetContracts(raw).Cap.Status != ReadingFailed {
		t.Fatal("unknown team cap shown")
	}
}

func TestCapSourceUsesGMSettingsWords(t *testing.T) {
	a := whatIfTestApp(t)
	if err := a.rulebook.Initialize(a.ctx, lineupRulesSource{cfg: league.RawConfig{SalaryCapAmount: "120"}}); err != nil {
		t.Fatal(err)
	}
	read := func() string {
		r := a.TargetContracts(snapshot.RawContractScope{Mine: "0001", Sort: "name"})
		if r.Cap.Status != ReadingOK {
			t.Fatalf("cap reading: %+v", r.Cap)
		}
		return r.Cap.Value.CapSource
	}
	if got := read(); got != "League salary cap: $120,000,000 (MFL league settings)" {
		t.Fatal(got)
	}
	if err := a.rulebook.SetOverride(a.ctx, "cap", "salaryCapAmount", "150", "test"); err != nil {
		t.Fatal(err)
	}
	if got := read(); got != "League salary cap: $150,000,000 (set by the commissioner)" {
		t.Fatal(got)
	}
}
