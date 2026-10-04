package rankings

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
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

type fakeCap struct{}

func (fakeCap) GetSalaryCap() string { return "125" }

// --- fixture ------------------------------------------------------------------

// asOf is the scoring instant every test uses: midnight UTC, so ages are whole-day exact.
func asOf() time.Time { return time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC) }

// birth returns an epoch for a player aged exactly `years` at asOf.
func birth(years float64) int64 {
	return asOf().Add(-time.Duration(years * 365.2425 * 24 * float64(time.Hour))).Unix()
}

// newHistory is a real history store on the shipped registry, its clock at asOf.
func newHistory(t *testing.T) (*history.Store, *time.Time) {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	h := history.New(pools, reg)
	now := asOf()
	h.SetClock(func() time.Time { return now })
	if err := h.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	return h, &now
}

// loadBase loads last season's fantasy points the way the app does: an MFL batch.
func loadBase(t *testing.T, h *history.Store, points map[string]string) {
	t.Helper()
	b := measures.Batch{Source: "mfl"}
	for id, v := range points {
		b.Facts = append(b.Facts, measures.Fact{IDType: measures.IDTypeMFL, ID: id, Season: 2025,
			Field: "playerScores.score", Raw: v})
	}
	if _, err := h.Ingest(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func testParams(decay float64) params.Set {
	values := params.DefaultSet().Values()
	values[params.KeyLayer3DecayRate] = decay
	return params.SetOf(values)
}

func boardSpec(p params.Set) RunSpec {
	return RunSpec{Kind: history.RunBoard, Season: 2026, AsOf: asOf(), Params: p, Measures: BoardMeasures()}
}

func newRunner(t *testing.T, st fakeState, dir fakeDir, scout ScoutingDirectory, h *history.Store) *Runner {
	t.Helper()
	r, err := New(st, dir, scout, fakeCap{}, h, "v-test")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func healthyFixture() (fakeState, fakeDir) {
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
	return st, dir
}

func scoresOf(t *testing.T, h *history.Store, runID int64) map[string]history.Score {
	t.Helper()
	got, err := h.RunScores(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]history.Score{}
	for _, s := range got {
		out[s.MFLID] = s
	}
	return out
}

// --- tests ---------------------------------------------------------------------

func TestNewRefusesMissingDependencies(t *testing.T) {
	st, dir := healthyFixture()
	h, _ := newHistory(t)
	cases := map[string]func() (*Runner, error){
		"state":    func() (*Runner, error) { return New(nil, dir, MapScoutingDirectory{}, fakeCap{}, h, "v") },
		"scouting": func() (*Runner, error) { return New(st, dir, nil, fakeCap{}, h, "v") },
		"history":  func() (*Runner, error) { return New(st, dir, MapScoutingDirectory{}, fakeCap{}, nil, "v") },
		"engine":   func() (*Runner, error) { return New(st, dir, MapScoutingDirectory{}, fakeCap{}, h, "") },
	}
	for name, build := range cases {
		if _, err := build(); err == nil {
			t.Errorf("New without %s succeeded", name)
		}
	}
}

func TestRunScoresTheBoardFromHistory(t *testing.T) {
	st, dir := healthyFixture()
	h, _ := newHistory(t)
	loadBase(t, h, map[string]string{"1001": "400.5", "2001": "250"})
	rep, err := newRunner(t, st, dir, MapScoutingDirectory{}, h).Run(context.Background(), boardSpec(testParams(0.03)))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Scored != 3 || rep.ZeroBase != 1 || rep.Unchanged || len(rep.Excluded) != 0 || len(rep.MissingMeasures) != 0 {
		t.Fatalf("report = %+v, want 3 scored, 1 without base points, a new run on the full set", rep)
	}
	got := scoresOf(t, h, rep.RunID)
	if got["1001"].BasePoints != 400.5 || got["1002"].BasePoints != 0 {
		t.Fatalf("base points not read from history: %+v", got)
	}
}

func TestRerunIsUnchangedUntilAFactChanges(t *testing.T) {
	st, dir := healthyFixture()
	h, now := newHistory(t)
	loadBase(t, h, map[string]string{"1001": "400.5", "2001": "250"})
	r := newRunner(t, st, dir, MapScoutingDirectory{}, h)
	first, err := r.Run(context.Background(), boardSpec(testParams(0.03)))
	if err != nil {
		t.Fatal(err)
	}
	later := boardSpec(testParams(0.03))
	later.AsOf = asOf().Add(9 * time.Hour) // same day: same ages
	again, err := r.Run(context.Background(), later)
	if err != nil || !again.Unchanged || again.RunID != first.RunID {
		t.Fatalf("same-day rerun = %+v, %v; want unchanged run %d", again, err, first.RunID)
	}

	*now = asOf().Add(10 * time.Hour)
	loadBase(t, h, map[string]string{"1001": "402", "2001": "250"}) // MFL corrected a stat
	later.AsOf = asOf().Add(11 * time.Hour)
	corrected, err := r.Run(context.Background(), later)
	if err != nil || corrected.Unchanged {
		t.Fatalf("after a correction = %+v, %v; want a new run", corrected, err)
	}
	if scoresOf(t, h, corrected.RunID)["1001"].BasePoints != 402 || scoresOf(t, h, first.RunID)["1001"].BasePoints != 400.5 {
		t.Fatal("the corrected run and the first run must each keep their own base points")
	}
}

// TestChangedParamsMakeASecondReadableBoard is the Stage 1 gate: a param change scores a new
// board, and the old board stays readable.
func TestChangedParamsMakeASecondReadableBoard(t *testing.T) {
	st, dir := healthyFixture()
	dir.facts["2001"] = normalize.PlayerFacts{Name: "Old, Back", Position: domain.PosRB, Birthdate: birth(30), HasBirthdate: true}
	h, _ := newHistory(t)
	loadBase(t, h, map[string]string{"1001": "400.5", "2001": "250"})
	r := newRunner(t, st, dir, MapScoutingDirectory{}, h)
	first, err := r.Run(context.Background(), boardSpec(testParams(0.03)))
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.Run(context.Background(), boardSpec(testParams(0.10)))
	if err != nil || second.Unchanged || second.RunID == first.RunID {
		t.Fatalf("changed params = %+v, %v; want a second run", second, err)
	}
	a, b := scoresOf(t, h, first.RunID)["2001"], scoresOf(t, h, second.RunID)["2001"]
	if b.AgePull >= a.AgePull {
		t.Fatalf("a faster decay left the 30-year-old back's age pull at %v (was %v)", b.AgePull, a.AgePull)
	}
}

// TestSourceLossDrill is the Stage 1 gate: MFL is lost, the board still runs on the facts it
// has and says it is running on a reduced set, and a rebalance proposal sits beside the board
// without changing it.
func TestSourceLossDrill(t *testing.T) {
	st, dir := healthyFixture()
	h, now := newHistory(t)
	ctx := context.Background()
	loadBase(t, h, map[string]string{"1001": "400.5", "2001": "250"})
	r := newRunner(t, st, dir, MapScoutingDirectory{}, h)
	before, err := r.Run(ctx, boardSpec(testParams(0.03)))
	if err != nil {
		t.Fatal(err)
	}

	*now = asOf().Add(10 * 24 * time.Hour)
	if err := h.LoadFailed(ctx, "mfl", fmt.Errorf("MFL answered 503")); err != nil {
		t.Fatal(err)
	}
	spec := boardSpec(testParams(0.03))
	spec.AsOf = *now
	reduced, err := r.Run(ctx, spec)
	if err != nil {
		t.Fatalf("the board stopped when its source was lost: %v", err)
	}
	if fmt.Sprint(reduced.MissingMeasures) != "["+BaseMeasure+"]" {
		t.Fatalf("missing = %v, want the reduced-set label for %s", reduced.MissingMeasures, BaseMeasure)
	}
	if scoresOf(t, h, reduced.RunID)["1001"].BasePoints != 400.5 {
		t.Fatal("the reduced board dropped the history it still holds")
	}

	proposal := spec
	proposal.Kind, proposal.Measures = history.RunRebalance, nil
	prop, err := r.Run(ctx, proposal)
	if err != nil || prop.Unchanged || len(prop.MissingMeasures) != 0 {
		t.Fatalf("rebalance proposal = %+v, %v", prop, err)
	}
	board, ok, err := h.LatestRun(ctx, 2026, history.RunBoard)
	if err != nil || !ok || board.ID != reduced.RunID {
		t.Fatalf("board = %+v, want run %d: a proposal never changes the board", board, reduced.RunID)
	}
	if before.RunID == reduced.RunID || scoresOf(t, h, before.RunID)["1001"].BasePoints != 400.5 {
		t.Fatal("the board before the loss must stay readable")
	}
}

func TestRun_ExclusionPolicies(t *testing.T) {
	st, dir := healthyFixture()
	st.franchises["0003"] = []state.PlayerState{
		{MFLID: "3001", FranchiseID: "0003", Salary: 1}, // not in directory
		{MFLID: "3002", FranchiseID: "0003", Salary: 1}, // FLAG position
		{MFLID: "3003", FranchiseID: "0003", Salary: 1}, // missing birthdate
		{MFLID: "3004", FranchiseID: "0003", Salary: 1}, // birthdate in the future → implausible age
	}
	dir.facts["3002"] = normalize.PlayerFacts{Name: "Flagged, Man", Position: domain.PosFlag, Birthdate: birth(24), HasBirthdate: true}
	dir.facts["3003"] = normalize.PlayerFacts{Name: "Created, Comm", Position: domain.PosWR}
	dir.facts["3004"] = normalize.PlayerFacts{Name: "Future, Kid", Position: domain.PosWR, Birthdate: asOf().Add(24 * time.Hour).Unix(), HasBirthdate: true}

	h, _ := newHistory(t)
	rep, err := newRunner(t, st, dir, MapScoutingDirectory{}, h).Run(context.Background(), boardSpec(testParams(0.03)))
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
		if !strings.Contains(e.Reason, wantReason[e.MFLID]) || e.FranchiseID != "0003" {
			t.Errorf("exclusion %+v, want reason %q in franchise 0003", e, wantReason[e.MFLID])
		}
	}
}

// TestRun_NegativeBaseFloorsToZero: the engine rejects a negative base, so it floors to 0 and
// is counted apart from a missing one.
func TestRun_NegativeBaseFloorsToZero(t *testing.T) {
	st, dir := healthyFixture()
	h, _ := newHistory(t)
	loadBase(t, h, map[string]string{"1001": "400.5", "2001": "-3.5"})
	rep, err := newRunner(t, st, dir, MapScoutingDirectory{}, h).Run(context.Background(), boardSpec(testParams(0.03)))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Scored != 3 || rep.ZeroBase != 1 || rep.NegativeBase != 1 {
		t.Fatalf("want 3 scored / ZeroBase 1 / NegativeBase 1, got %+v", rep)
	}
	if scoresOf(t, h, rep.RunID)["2001"].BasePoints != 0 {
		t.Fatal("a negative base must floor to 0")
	}
}

// TestRun_EmptyLeagueFailsLoud: a pass that can score nobody is broken input, never an empty
// board.
func TestRun_EmptyLeagueFailsLoud(t *testing.T) {
	st := fakeState{franchises: map[string][]state.PlayerState{
		"0001": {{MFLID: "1001", FranchiseID: "0001"}},
	}}
	h, _ := newHistory(t)
	r := newRunner(t, st, fakeDir{facts: map[string]normalize.PlayerFacts{}}, MapScoutingDirectory{}, h)
	if _, err := r.Run(context.Background(), boardSpec(testParams(0.03))); err == nil {
		t.Fatal("Run with zero scorable players should error")
	}
	if runs, _ := h.Runs(context.Background(), 2026); len(runs) != 0 {
		t.Fatalf("an empty league was written: %d runs", len(runs))
	}
}

// TestRun_ScoutingDirectoryPopulatesRAS: a profile's RAS reaches the engine (the tiebreaker
// carries the cleaned RAS); a player without one gets the L1 fallback.
func TestRun_ScoutingDirectoryPopulatesRAS(t *testing.T) {
	st, dir := healthyFixture()
	id1002, _ := playerid.New("1002")
	scout := NewMapScoutingDirectory(map[playerid.PlayerID]scouting.Profile{
		id1002: {MFLID: id1002, RAS: 8.0, HasRAS: true},
	})
	h, _ := newHistory(t)
	rep, err := newRunner(t, st, dir, scout, h).Run(context.Background(), boardSpec(testParams(0.03)))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := scoresOf(t, h, rep.RunID)
	if ras := got["1002"].Tiebreaker.RAS; math.Abs(ras-8.0) > 1e-12 {
		t.Errorf("1002 RAS = %v, want 8.0 from its profile", ras)
	}
	if ras := got["1001"].Tiebreaker.RAS; math.Abs(ras-composition.DefaultRASFallback) > 1e-12 {
		t.Errorf("1001 RAS = %v, want the fallback %v", ras, composition.DefaultRASFallback)
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

// TestApplyScouting_FilmComposite pins the film blend for every seat combination:
//   - CB/S coverage only:   0.20·coverage + 0.80·neutral
//   - IDP Madden only:      0.95·Madden + 0.05·neutral (the NFLProduction seat)
//   - CB/S both:            0.20·coverage + 0.75·Madden + 0.05·neutral
//   - offense:              0.95·Composite + 0.05·neutral
//
// A neutral (0.50) input leaves the composite neutral.
func TestApplyScouting_FilmComposite(t *testing.T) {
	id, _ := playerid.New("1001")
	cov := func(v float64) *scouting.NGSCoverage { return &scouting.NGSCoverage{CoverageMetrics: v} }
	idp := func(v float64) *scouting.IDPFilm { return &scouting.IDPFilm{MaddenComposite: v} }
	off := func(v float64) *scouting.OffenseFilm { return &scouting.OffenseFilm{Composite: v} }
	cases := []struct {
		name    string
		profile scouting.Profile
		want    float64
	}{
		{"coverage neutral", scouting.Profile{Coverage: cov(0.50)}, 0.50},
		{"coverage elite +0.10", scouting.Profile{Coverage: cov(1.00)}, 0.60},
		{"coverage poor -0.10", scouting.Profile{Coverage: cov(0.00)}, 0.40},
		{"IDP Madden neutral", scouting.Profile{IDPFilm: idp(0.50)}, 0.50},
		{"IDP Madden elite", scouting.Profile{IDPFilm: idp(1.00)}, 0.95*1.00 + 0.05*0.50},
		{"IDP Madden poor", scouting.Profile{IDPFilm: idp(0.00)}, 0.95*0.00 + 0.05*0.50},
		{"CB/S coverage and Madden", scouting.Profile{Coverage: cov(0.90), IDPFilm: idp(0.80)}, 0.20*0.90 + 0.75*0.80 + 0.05*0.50},
		{"offense neutral", scouting.Profile{OffenseFilm: off(0.50)}, 0.50},
		{"offense elite", scouting.Profile{OffenseFilm: off(1.00)}, 0.95*1.00 + 0.05*0.50},
		{"offense poor", scouting.Profile{OffenseFilm: off(0.00)}, 0.95*0.00 + 0.05*0.50},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.profile.MFLID = id
			var spec composition.PlayerSpec
			applyScouting(&spec, c.profile)
			if !spec.HasFilm || math.Abs(spec.FilmComposite-c.want) > 1e-12 {
				t.Fatalf("HasFilm=%v FilmComposite=%v, want true and %v", spec.HasFilm, spec.FilmComposite, c.want)
			}
		})
	}
}

// TestApplyScouting_NoFilmSignalsLeavesFilmAbsent: without either a Coverage group or an
// IDPFilm group, HasFilm stays false and the rubric neutralizes film via Data-Parity
// (every offense position, and any IDP player whose Madden record did not resolve).
func TestApplyScouting_NoFilmSignalsLeavesFilmAbsent(t *testing.T) {
	id, _ := playerid.New("1001")
	var spec composition.PlayerSpec
	applyScouting(&spec, scouting.Profile{MFLID: id, RAS: 8.0, HasRAS: true})
	if spec.HasFilm {
		t.Fatalf("no film signal → HasFilm must stay false (got FilmComposite=%v)", spec.FilmComposite)
	}
}
