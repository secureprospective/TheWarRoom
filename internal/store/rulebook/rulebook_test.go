package rulebook

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
)

// fakeSource is a controllable league.Source: tests mutate cfg and re-Reload to
// simulate MFL config drift, with no network.
type fakeSource struct {
	cfg league.RawConfig
	err error
}

func (f *fakeSource) Fetch(_ context.Context) (league.RawConfig, error) {
	return f.cfg, f.err
}

func baseConfig() league.RawConfig {
	return league.RawConfig{
		Source:          "test:v1",
		SalaryCapAmount: "120",
		RosterSize:      "80",
		TaxiSquad:       "8",
		ScoringRules: []league.PositionRuleSet{{
			Positions: "CB|S",
			Rules: []league.ScoringRule{
				{Event: "TK", Points: "*0.5", Range: "0-99"},
				{Event: "PD", Points: "*0.5", Range: "0-99"},
			},
		}},
		RosterLimits: []league.PositionLimit{{Name: "QB", Limit: "0-0"}},
	}
}

// newStore opens a fresh temp SQLite store seeded from src.
func newStore(t *testing.T, src league.Source) *Store {
	t.Helper()
	ctx := context.Background()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "rulebook_test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = pools.Close() })

	s := New(pools)
	if err := s.Initialize(ctx, src); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return s
}

func TestInitializeAndReads(t *testing.T) {
	s := newStore(t, &fakeSource{cfg: baseConfig()})

	if got := s.GetSalaryCap(); got != "120" {
		t.Errorf("cap = %q, want 120", got)
	}
	if got := s.GetScoringRules(); len(got) != 1 || got[0].Positions != "CB|S" {
		t.Errorf("scoring = %+v, want one CB|S block", got)
	}
	if v, ok := s.GetSetting("rosterSize"); !ok || v != "80" {
		t.Errorf("rosterSize = %q,%v, want 80,true", v, ok)
	}
	if _, ok := s.GetSetting("nope"); ok {
		t.Error("unknown setting reported ok")
	}
}

// TestCapPercent_DefaultsTo100WhenUnset confirms a config predating the
// includeTaxiWithSalary/includeIRWithSalary fields (empty string, e.g. an old stored
// version) is treated as 100% — the historical no-discount behavior — not an error.
func TestCapPercent_DefaultsTo100WhenUnset(t *testing.T) {
	s := newStore(t, &fakeSource{cfg: baseConfig()})
	if got := s.TaxiCapPercent(); got != 100 {
		t.Errorf("TaxiCapPercent = %v, want 100", got)
	}
	if got := s.IRCapPercent(); got != 100 {
		t.Errorf("IRCapPercent = %v, want 100", got)
	}
}

// TestCapPercent_ReadsMFLValueAndOverride confirms TaxiCapPercent/IRCapPercent read
// the MFL-sourced value, and that a commissioner SetOverride (the toggle-off /
// slot-count mechanism, scopeSetting) takes precedence — the same override
// machinery GetSetting already uses, no new code path.
func TestCapPercent_ReadsMFLValueAndOverride(t *testing.T) {
	cfg := baseConfig()
	cfg.IncludeTaxiWithSalary = "100"
	cfg.IncludeIRWithSalary = "100"
	s := newStore(t, &fakeSource{cfg: cfg})

	if got := s.TaxiCapPercent(); got != 100 {
		t.Errorf("TaxiCapPercent = %v, want 100", got)
	}

	if err := s.SetOverride(context.Background(), scopeSetting, "includeTaxiWithSalary", "50", "commissioner discount"); err != nil {
		t.Fatalf("SetOverride: %v", err)
	}
	if got := s.TaxiCapPercent(); got != 50 {
		t.Errorf("TaxiCapPercent after override = %v, want 50", got)
	}
	if got := s.IRCapPercent(); got != 100 {
		t.Errorf("IRCapPercent (no override set) = %v, want 100", got)
	}
}

func TestSetOverride_AppliesAndValidates(t *testing.T) {
	ctx := context.Background()
	s := newStore(t, &fakeSource{cfg: baseConfig()})

	if err := s.SetOverride(ctx, scopeCap, capKey, "150", "commissioner vote"); err != nil {
		t.Fatalf("SetOverride: %v", err)
	}
	if got := s.GetSalaryCap(); got != "150" {
		t.Errorf("cap after override = %q, want 150", got)
	}

	// Planted failures: the data-integrity gate must reject bad overrides.
	if err := s.SetOverride(ctx, scopeCap, capKey, "lots", ""); err == nil {
		t.Error("non-numeric cap override accepted, want rejection")
	}
	if err := s.SetOverride(ctx, "bogus", "x", "1", ""); err == nil {
		t.Error("unknown scope accepted, want rejection")
	}
	if err := s.SetOverride(ctx, scopeSetting, "rosterSize", "", ""); err == nil {
		t.Error("empty value accepted, want rejection")
	}
}

func TestReload_SideLoadsWithoutPromoting(t *testing.T) {
	ctx := context.Background()
	src := &fakeSource{cfg: baseConfig()}
	s := newStore(t, src)

	// MFL config drifts: cap 120->130 and the TK award changes (an exploit fix).
	drift := baseConfig()
	drift.Source = "test:v2"
	drift.SalaryCapAmount = "130"
	drift.ScoringRules[0].Rules[0].Points = "*0.75"
	src.cfg = drift

	cs, err := s.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if cs.Empty() {
		t.Fatal("ChangeSet empty, want cap + scoring deltas")
	}
	if !hasDelta(cs, "salaryCapAmount", KindChanged, "120", "130") {
		t.Errorf("missing cap delta in %+v", cs.Deltas)
	}
	if !hasDelta(cs, "scoring:CB|S/TK[0-99]", KindChanged, "*0.5", "*0.75") {
		t.Errorf("missing scoring delta in %+v", cs.Deltas)
	}

	// STABILITY: reads still serve the active (v1) config — no auto-promote.
	if got := s.GetSalaryCap(); got != "120" {
		t.Errorf("cap after Reload = %q, want stable 120 (not promoted)", got)
	}

	// The gate confirms: promote the candidate -> reads update.
	if err := s.Promote(ctx, cs.ToVersion); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if got := s.GetSalaryCap(); got != "130" {
		t.Errorf("cap after Promote = %q, want 130", got)
	}
}

func TestReload_NoChangeEmptyChangeSet(t *testing.T) {
	src := &fakeSource{cfg: baseConfig()}
	s := newStore(t, src)

	cs, err := s.Reload(context.Background())
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if !cs.Empty() {
		t.Errorf("ChangeSet = %+v, want empty for identical config", cs.Deltas)
	}
}

func TestPromote_Rollback(t *testing.T) {
	ctx := context.Background()
	src := &fakeSource{cfg: baseConfig()}
	s := newStore(t, src)

	drift := baseConfig()
	drift.SalaryCapAmount = "130"
	src.cfg = drift
	cs, err := s.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if err := s.Promote(ctx, cs.ToVersion); err != nil {
		t.Fatalf("Promote v2: %v", err)
	}

	// Roll back to v1: same operation, prior version.
	if err := s.Promote(ctx, cs.FromVersion); err != nil {
		t.Fatalf("Promote rollback: %v", err)
	}
	if got := s.GetSalaryCap(); got != "120" {
		t.Errorf("cap after rollback = %q, want 120", got)
	}
}

func TestPromote_UnknownVersionFailsLoud(t *testing.T) {
	s := newStore(t, &fakeSource{cfg: baseConfig()})
	if err := s.Promote(context.Background(), 999); err == nil {
		t.Error("Promote to nonexistent version accepted, want error")
	}
}

// TestOverrideSurvivesReloadPromote proves a commissioner override is NOT clobbered
// by a re-pull of MFL defaults — it is a separate record layered at read time.
func TestOverrideSurvivesReloadPromote(t *testing.T) {
	ctx := context.Background()
	src := &fakeSource{cfg: baseConfig()}
	s := newStore(t, src)

	if err := s.SetOverride(ctx, scopeCap, capKey, "999", "vote"); err != nil {
		t.Fatalf("SetOverride: %v", err)
	}

	drift := baseConfig()
	drift.SalaryCapAmount = "130"
	src.cfg = drift
	cs, err := s.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if err := s.Promote(ctx, cs.ToVersion); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	if got := s.GetSalaryCap(); got != "999" {
		t.Errorf("cap = %q, want 999 (override survives MFL re-pull)", got)
	}
}

// TestGetScoringRules_NoAlias proves a returned slice shares no backing array with
// the store's active snapshot: mutating it must not corrupt in-memory state.
func TestGetScoringRules_NoAlias(t *testing.T) {
	s := newStore(t, &fakeSource{cfg: baseConfig()})

	got := s.GetScoringRules()
	got[0].Rules[0].Points = "MUTATED"
	got[0].Positions = "WRECKED"

	again := s.GetScoringRules()
	if again[0].Rules[0].Points != "*0.5" || again[0].Positions != "CB|S" {
		t.Errorf("active config corrupted via returned slice: %+v", again[0])
	}
	if cfg := s.ActiveConfig(); cfg.ScoringRules[0].Rules[0].Points != "*0.5" {
		t.Errorf("ActiveConfig leaked an alias: %+v", cfg.ScoringRules[0])
	}
}

// TestConcurrentAdminAndReads exercises the wmu serialization + reader locks under
// the race detector: concurrent Promotes, SetOverrides, and reads must not race and
// must leave memory consistent with the committed active pointer.
func TestConcurrentAdminAndReads(t *testing.T) {
	ctx := context.Background()
	src := &fakeSource{cfg: baseConfig()}
	s := newStore(t, src)

	drift := baseConfig()
	drift.SalaryCapAmount = "130"
	src.cfg = drift
	cs, err := s.Reload(ctx)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			switch n % 4 {
			case 0:
				_ = s.Promote(ctx, cs.ToVersion)
			case 1:
				_ = s.Promote(ctx, cs.FromVersion)
			case 2:
				_ = s.SetOverride(ctx, scopeSetting, "taxiSquad", "9", "")
			default:
				_ = s.GetSalaryCap()
				_ = s.GetScoringRules()
				_, _ = s.GetSetting("taxiSquad")
			}
		}(i)
	}
	wg.Wait()

	// Whatever interleaving won, memory must match the committed active pointer.
	active, err := s.activeVersion(ctx)
	if err != nil {
		t.Fatalf("activeVersion: %v", err)
	}
	if err := s.loadActive(ctx); err != nil {
		t.Fatalf("loadActive: %v", err)
	}
	if s.activeVer != active {
		t.Errorf("in-memory active %d != committed %d", s.activeVer, active)
	}
}

func hasDelta(cs ChangeSet, field string, kind ChangeKind, old, newVal string) bool {
	for _, d := range cs.Deltas {
		if d.Field == field && d.Kind == kind && d.Old == old && d.New == newVal {
			return true
		}
	}
	return false
}
