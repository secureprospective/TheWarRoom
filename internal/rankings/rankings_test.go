package rankings

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/output"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// --- fakes -------------------------------------------------------------------

type fakeState struct {
	franchises map[string][]state.PlayerState
}

func (f fakeState) Franchises() []string {
	out := make([]string, 0, len(f.franchises))
	for _, id := range []string{"0001", "0002", "0003"} { // deterministic order like the store
		if _, ok := f.franchises[id]; ok {
			out = append(out, id)
		}
	}
	return out
}
func (f fakeState) Roster(id string) ([]state.PlayerState, bool) {
	r, ok := f.franchises[id]
	return r, ok
}
func (f fakeState) FranchiseState(id string) (state.FranchiseState, bool) {
	r, ok := f.franchises[id]
	return state.FranchiseState{FranchiseID: id, Players: r}, ok
}
func (f fakeState) CapUsed(string) (domain.Money, bool) { return 0, false }
func (f fakeState) Player(mflID string) (state.PlayerState, bool) {
	for _, r := range f.franchises {
		for _, p := range r {
			if p.MFLID == mflID {
				return p, true
			}
		}
	}
	return state.PlayerState{}, false
}

type fakeDir struct {
	facts map[string]normalize.PlayerFacts
}

func (f fakeDir) Facts(id string) (normalize.PlayerFacts, bool) {
	pf, ok := f.facts[id]
	return pf, ok
}

type fakeCfg struct {
	ver int
	err error
}

func (f fakeCfg) ActiveVersion(context.Context) (int, error) { return f.ver, f.err }

// fakeOut records the single Write it receives and serves preset existing scores.
type fakeOut struct {
	existing []output.SeasonScore
	writeErr error

	gotSeason, gotVer int
	gotRecs           []output.ScoreRecord
	writes            int
}

func (f *fakeOut) Scores(_ context.Context, _, _ int) ([]output.SeasonScore, error) {
	return f.existing, nil
}
func (f *fakeOut) Score(context.Context, int, int, string) (output.SeasonScore, bool, error) {
	return output.SeasonScore{}, false, nil
}
func (f *fakeOut) Write(_ context.Context, season, ver int, recs []output.ScoreRecord) error {
	f.writes++
	f.gotSeason, f.gotVer, f.gotRecs = season, ver, recs
	return f.writeErr
}

type fakeParams struct{}

func (fakeParams) GetCapTiers() (params.CapTiers, error) {
	return params.CapTiers{ColdCeiling: 1.2, HotFloor: 4.8}, nil
}
func (fakeParams) GetGlobal(key string) (float64, error) {
	switch key {
	case params.KeyLayer3DecayRate:
		return 0.03, nil
	default:
		return 0, fmt.Errorf("fakeParams: unknown key %q", key)
	}
}

type fakeCap struct{}

func (fakeCap) GetSalaryCap() string { return "125" }

// --- fixture ------------------------------------------------------------------

// asOf pins the deterministic scoring instant every test uses (gochecknoglobals
// forbids a package var; a func is the sanctioned constant-Time idiom).
func asOf() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) }

// birth returns an epoch for a player aged exactly `years` at asOf.
func birth(years float64) int64 {
	return asOf().Add(-time.Duration(years * 365.2425 * 24 * float64(time.Hour))).Unix()
}

func newRunner(t *testing.T, st fakeState, dir fakeDir, base map[string]float64, cfg fakeCfg, out *fakeOut) *Runner {
	t.Helper()
	r, err := New(st, dir, MapScoutingDirectory{}, base, cfg, out, composition.New(fakeParams{}, fakeCap{}), Registry{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func healthyFixture() (fakeState, fakeDir, map[string]float64) {
	st := fakeState{franchises: map[string][]state.PlayerState{
		"0001": {
			{MFLID: "1001", FranchiseID: "0001", Salary: 20},
			{MFLID: "1002", FranchiseID: "0001", Salary: 5},
		},
		"0002": {
			{MFLID: "2001", FranchiseID: "0002", Salary: 1},
		},
	}}
	dir := fakeDir{facts: map[string]normalize.PlayerFacts{
		"1001": {Name: "Vet, Good", Position: domain.PosQB, Birthdate: birth(28), HasBirthdate: true},
		"1002": {Name: "Rook, Zero", Position: domain.PosWR, IsRookie: true, Birthdate: birth(22), HasBirthdate: true},
		"2001": {Name: "Other, Guy", Position: domain.PosRB, Birthdate: birth(25), HasBirthdate: true},
	}}
	base := map[string]float64{"1001": 400.5, "2001": 250}
	return st, dir, base
}

// --- tests ---------------------------------------------------------------------

func TestNew_NilDependencyFails(t *testing.T) {
	st, dir, base := healthyFixture()
	if _, err := New(nil, dir, MapScoutingDirectory{}, base, fakeCfg{ver: 1}, &fakeOut{}, composition.New(fakeParams{}, fakeCap{}), Registry{}); err == nil {
		t.Fatal("New with nil state should error")
	}
	if _, err := New(st, dir, MapScoutingDirectory{}, nil, fakeCfg{ver: 1}, &fakeOut{}, composition.New(fakeParams{}, fakeCap{}), Registry{}); err == nil {
		t.Fatal("New with nil base map should error")
	}
	// The GLM M1 planted gate: a NIL registry reads as identity-L4 for every
	// position without panicking — a silently wrong-rubric league. Construction
	// must refuse it; an explicitly-empty Registry{} stays legal.
	if _, err := New(st, dir, MapScoutingDirectory{}, base, fakeCfg{ver: 1}, &fakeOut{}, composition.New(fakeParams{}, fakeCap{}), nil); err == nil {
		t.Fatal("New with nil registry should error")
	}
	// S-Phase 0: a NIL scouting directory is a wiring error (MapScoutingDirectory
	// is the empty-but-legal state); a nil one silently neutralizes every
	// scouting signal — surface it at construction like the other nil deps.
	if _, err := New(st, dir, nil, base, fakeCfg{ver: 1}, &fakeOut{}, composition.New(fakeParams{}, fakeCap{}), Registry{}); err == nil {
		t.Fatal("New with nil scouting directory should error")
	}
}

// TestRun_NoActiveConfigRefusesToScore is the DECISION-010 provenance gate: a
// zero version must never be stamped onto B6 rows.
func TestRun_NoActiveConfigRefusesToScore(t *testing.T) {
	st, dir, base := healthyFixture()
	out := &fakeOut{}
	r := newRunner(t, st, dir, base, fakeCfg{ver: 0}, out)
	if _, err := r.Run(context.Background(), 2026, asOf()); err == nil {
		t.Fatal("Run with active version 0 should error")
	}
	if out.writes != 0 {
		t.Fatalf("nothing may be written under version 0; got %d writes", out.writes)
	}
}

// TestRun_SkipIfPresent is the gate-checked re-score UX: an already-scored
// (season, config) is reported, never re-written.
func TestRun_SkipIfPresent(t *testing.T) {
	st, dir, base := healthyFixture()
	out := &fakeOut{existing: []output.SeasonScore{{Season: 2026, ScoringConfigID: 3, MFLID: "1001"}}}
	r := newRunner(t, st, dir, base, fakeCfg{ver: 3}, out)

	rep, err := r.Run(context.Background(), 2026, asOf())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Scored must be 0 — nothing was scored THIS pass; the persisted count rides
	// in Existing (GLM M1 review).
	if !rep.SkippedExisting || rep.Scored != 0 || rep.Existing != 1 {
		t.Fatalf("want SkippedExisting with Scored=0/Existing=1, got %+v", rep)
	}
	if out.writes != 0 {
		t.Fatalf("skip-if-present must not write; got %d writes", out.writes)
	}
}

// TestRun_ScoresStampsAndReports is the happy-path spine: every scorable player
// is written in one batch stamped with the ACTIVE version, and the zero-base
// placeholder path is counted, not hidden.
func TestRun_ScoresStampsAndReports(t *testing.T) {
	st, dir, base := healthyFixture()
	out := &fakeOut{}
	r := newRunner(t, st, dir, base, fakeCfg{ver: 7}, out)

	rep, err := r.Run(context.Background(), 2026, asOf())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.writes != 1 || out.gotSeason != 2026 || out.gotVer != 7 {
		t.Fatalf("want one write stamped (2026, 7); got writes=%d season=%d ver=%d", out.writes, out.gotSeason, out.gotVer)
	}
	if rep.Scored != 3 || len(out.gotRecs) != 3 {
		t.Fatalf("want 3 scored, got report %+v recs %d", rep, len(out.gotRecs))
	}
	if rep.ZeroBase != 1 { // 1002 has no YTD record
		t.Fatalf("want ZeroBase 1 (rookie without a proxy score), got %d", rep.ZeroBase)
	}
	if len(rep.Excluded) != 0 {
		t.Fatalf("healthy fixture should exclude nobody, got %+v", rep.Excluded)
	}
	for _, rec := range out.gotRecs {
		if rec.MFLID == "1001" && math.Abs(rec.Result.BasePoints-400.5) > 1e-12 {
			t.Fatalf("BasePoints proxy not threaded through: %+v", rec.Result)
		}
	}
}

// TestRun_ExclusionPolicies plants one player per gate-checked exclusion reason
// and asserts each lands in the report with its reason while the rest score.
func TestRun_ExclusionPolicies(t *testing.T) {
	st, dir, base := healthyFixture()
	st.franchises["0003"] = []state.PlayerState{
		{MFLID: "3001", FranchiseID: "0003", Salary: 1}, // not in directory
		{MFLID: "3002", FranchiseID: "0003", Salary: 1}, // FLAG position
		{MFLID: "3003", FranchiseID: "0003", Salary: 1}, // missing birthdate
		{MFLID: "3004", FranchiseID: "0003", Salary: 1}, // birthdate in the future → implausible age
	}
	dir.facts["3002"] = normalize.PlayerFacts{Name: "Flagged, Man", Position: domain.PosFlag, Birthdate: birth(24), HasBirthdate: true}
	dir.facts["3003"] = normalize.PlayerFacts{Name: "Created, Comm", Position: domain.PosWR}
	dir.facts["3004"] = normalize.PlayerFacts{Name: "Future, Kid", Position: domain.PosWR, Birthdate: asOf().Add(24 * time.Hour).Unix(), HasBirthdate: true}

	out := &fakeOut{}
	r := newRunner(t, st, dir, base, fakeCfg{ver: 2}, out)
	rep, err := r.Run(context.Background(), 2026, asOf())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Scored != 3 || len(rep.Excluded) != 4 {
		t.Fatalf("want 3 scored / 4 excluded, got %+v", rep)
	}
	wantReason := map[string]string{
		"3001": "not in the players database",
		"3002": "unclassified position",
		"3003": "missing birthdate",
		"3004": "implausible age",
	}
	for _, e := range rep.Excluded {
		want, ok := wantReason[e.MFLID]
		if !ok {
			t.Errorf("unexpected exclusion %+v", e)
			continue
		}
		if !strings.Contains(e.Reason, want) {
			t.Errorf("exclusion %s reason %q missing %q", e.MFLID, e.Reason, want)
		}
		if e.FranchiseID != "0003" {
			t.Errorf("exclusion %s franchise %q, want 0003", e.MFLID, e.FranchiseID)
		}
	}
}

// TestRun_NegativeProxyFloorsToZero: the engine rejects a negative base as
// poisoned, so the placeholder floors it to 0 and counts it — scored, visible,
// at the bottom; never an exclusion and never a raw negative into the engine.
func TestRun_NegativeProxyFloorsToZero(t *testing.T) {
	st, dir, base := healthyFixture()
	base["2001"] = -3.5
	out := &fakeOut{}
	r := newRunner(t, st, dir, base, fakeCfg{ver: 2}, out)

	rep, err := r.Run(context.Background(), 2026, asOf())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The two zero populations are DISTINCT (GLM M1 review): 1002 has no proxy
	// record (ZeroBase), 2001's negative total was floored (NegativeBase).
	if rep.Scored != 3 || rep.ZeroBase != 1 || rep.NegativeBase != 1 {
		t.Fatalf("want 3 scored / ZeroBase 1 / NegativeBase 1, got %+v", rep)
	}
	for _, rec := range out.gotRecs {
		if rec.MFLID == "2001" && rec.Result.BasePoints != 0 {
			t.Fatalf("negative proxy must floor to 0, got %v", rec.Result.BasePoints)
		}
	}
}

// TestRun_EmptyLeagueFailsLoud: a pass that can score NOBODY is a broken input
// set (dead directory, empty state), never a legitimate empty persist.
func TestRun_EmptyLeagueFailsLoud(t *testing.T) {
	st := fakeState{franchises: map[string][]state.PlayerState{
		"0001": {{MFLID: "1001", FranchiseID: "0001"}},
	}}
	dir := fakeDir{facts: map[string]normalize.PlayerFacts{}} // resolves nobody
	out := &fakeOut{}
	r := newRunner(t, st, dir, map[string]float64{}, fakeCfg{ver: 2}, out)
	if _, err := r.Run(context.Background(), 2026, asOf()); err == nil {
		t.Fatal("Run with zero scorable players should error")
	}
	if out.writes != 0 {
		t.Fatalf("an empty league must never be persisted; got %d writes", out.writes)
	}
}

// TestRun_WriteFailurePropagates: a B6 rejection (drift, poisoned value) must
// surface, not be swallowed into a half-reported success.
func TestRun_WriteFailurePropagates(t *testing.T) {
	st, dir, base := healthyFixture()
	out := &fakeOut{writeErr: fmt.Errorf("planted: %w", output.ErrDuplicate)}
	r := newRunner(t, st, dir, base, fakeCfg{ver: 2}, out)
	if _, err := r.Run(context.Background(), 2026, asOf()); err == nil {
		t.Fatal("Run should propagate the Write error")
	}
}

// TestYearsBetween pins the age derivation the exclusion gate and L3 decay ride on.
func TestYearsBetween(t *testing.T) {
	b := time.Date(2000, 7, 2, 0, 0, 0, 0, time.UTC)
	got := yearsBetween(b, asOf())
	if math.Abs(got-26.0) > 0.02 {
		t.Fatalf("yearsBetween(2000-07-02, 2026-07-02) = %v, want ~26.0", got)
	}
}

// --- S-Phase 0 scouting directory -------------------------------------------

// TestRun_ScoutingDirectoryPopulatesRAS proves the ScoutingDirectory port
// injects RAS through scorePlayer → spec.RAS/HasRAS → composition.Assembler →
// the engine. Two rostered players: 1002 has a profile with HasRAS=true (RAS 8.0
// threaded through — per-field presence, not directory presence), 1001 does not
// (HasRAS=false → L1 imputes DefaultRASFallback, ApplyHygiene zeroes the raw RAS). The
// engine's TiebreakerKey carries the cleaned RAS, which is the surface that
// distinguishes the two wiring paths — 8.0 for the profile-present player, 0.0
// (hygiene-zeroed) for the absent-profile player.
func TestRun_ScoutingDirectoryPopulatesRAS(t *testing.T) {
	st, dir, base := healthyFixture()
	// Give 1002 (Rook, Zero, WR) a RAS profile; leave 1001 and 2001 absent.
	id1002, _ := playerid.New("1002")
	scout := NewMapScoutingDirectory(map[playerid.PlayerID]scouting.Profile{
		id1002: {MFLID: id1002, RAS: 8.0, HasRAS: true},
	})

	out := &fakeOut{}
	r, err := New(st, dir, scout, base, fakeCfg{ver: 7}, out, composition.New(fakeParams{}, fakeCap{}), Registry{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := r.Run(context.Background(), 2026, asOf()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	find := func(mflID string) *output.ScoreRecord {
		for i := range out.gotRecs {
			if out.gotRecs[i].MFLID == mflID {
				return &out.gotRecs[i]
			}
		}
		t.Fatalf("%s not in persisted records", mflID)
		return nil
	}

	// 1002: profile present → spec.HasRAS=true → RAS threaded through to the
	// engine's Tiebreaker (ApplyHygiene keeps a finite RAS when HasRAS is true).
	got1002 := find("1002")
	if math.Abs(got1002.Result.Tiebreaker.RAS-8.0) > 1e-12 {
		t.Fatalf("1002: expected Tiebreaker.RAS = 8.0 (profile present, RAS threaded), got %v",
			got1002.Result.Tiebreaker.RAS)
	}

	// 1001: absent profile → spec.HasRAS=false → L1 imputes the
	// DefaultRASFallback (5.00) so the Tiebreaker rides on the fallback value,
	// NOT a real measured RAS. 8.0-vs-5.0 is the wiring signal: the profile-
	// present player's real RAS thread through; the absent-profile player
	// falls through to the L1 imputation unchanged.
	got1001 := find("1001")
	if math.Abs(got1001.Result.Tiebreaker.RAS-composition.DefaultRASFallback) > 1e-12 {
		t.Fatalf("1001: expected Tiebreaker.RAS = DefaultRASFallback (%v) (no profile → L1 imputes), got %v",
			composition.DefaultRASFallback, got1001.Result.Tiebreaker.RAS)
	}
}

// TestRun_EmptyScoutingDirectoryIsLegal pins the "explicitly-empty map is a
// legal condition" half of the nil-guard rule. A real ScoreLeague pass might
// run when the scouting fetch returned nothing for the whole league (network
// outage, every player missed); that is NOT a wiring error, and the pass must
// complete with every player scored against the L1-imputed RAS fallback.
func TestRun_EmptyScoutingDirectoryIsLegal(t *testing.T) {
	st, dir, base := healthyFixture()
	out := &fakeOut{}
	// Empty (non-nil) MapScoutingDirectory — every player misses cleanly.
	r, err := New(st, dir, MapScoutingDirectory{}, base, fakeCfg{ver: 7}, out, composition.New(fakeParams{}, fakeCap{}), Registry{})
	if err != nil {
		t.Fatalf("New with empty (non-nil) scouting dir should succeed: %v", err)
	}
	rep, err := r.Run(context.Background(), 2026, asOf())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Scored != 3 {
		t.Fatalf("want 3 scored with empty scouting dir, got %d", rep.Scored)
	}
}
