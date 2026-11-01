package appvalue

import (
	"context"
	"fmt"
	"maps"
	"math"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/modelrun"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

const (
	season   = 2026
	lastWeek = 17
	regulars = 25 // enough WR regulars for the position's points scale (modelrun's minRegulars)
)

// legacySettings are the Legacy NFL's own settings (MFL league 14432) as the app records them,
// read through the same translation an import uses.
func legacySettings(t *testing.T) leaguerules.RuleSet {
	t.Helper()
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	base, err := leaguemigration.Base(leaguemigration.DefaultPreset(len(export.Config.Franchises)))
	if err != nil {
		t.Fatal(err)
	}
	rules, problems, _ := mflsource.Settings(export.Config, base, export.DraftType)
	if len(problems) > 0 {
		t.Fatalf("the held 14432 settings do not translate: %v", problems)
	}
	return rules
}

func mflID(i int) string  { return fmt.Sprintf("%04d", 200+i) }
func gsisOf(i int) string { return fmt.Sprintf("00-000%04d", i) }

// receivingWeek is one held NFL week: yards receiving, nothing else.
func receivingWeek(gsis string, year, week int, yards int64) source.PlayerWeek {
	return source.PlayerWeek{GSIS: gsis, Position: "WR", Season: year, Week: week, SeasonType: "REG",
		Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{
			"receiving_yards": leaguerules.Decimal(yards * 10000)}}}
}

// heldWeeks are 25 receivers' 2025, each at a different yardage, and nothing in 2026 yet.
func heldWeeks() map[int][]source.PlayerWeek {
	out := map[int][]source.PlayerWeek{season - 1: {}}
	for i := range regulars {
		for w := 1; w <= 16; w++ {
			out[season-1] = append(out[season-1], receivingWeek(gsisOf(i), season-1, w, int64(20+4*i)))
		}
	}
	// A playoff week and a week past the league's last scoring week must never be counted.
	post := receivingWeek(gsisOf(0), season-1, 19, 900)
	post.SeasonType = "POST"
	out[season-1] = append(out[season-1], post, receivingWeek(gsisOf(0), season-1, lastWeek+1, 900))
	return out
}

type listing map[string]normalize.PlayerFacts

func (l listing) Facts(id string) (normalize.PlayerFacts, bool) {
	f, ok := l[id]
	return f, ok
}
func (l listing) IDs() []string { return slices.Sorted(maps.Keys(l)) }

// feats serves the record's facts: who each player is, and (for the mirror) his season points.
type feats struct {
	rows    map[int][]history.Feature
	written []history.NewModelRun
}

func (f *feats) Features(_ context.Context, q history.FeatureQuery) ([]history.Feature, error) {
	return f.rows[q.Season], nil
}
func (f *feats) WriteModelRun(_ context.Context, nr history.NewModelRun) (history.Run, bool, error) {
	f.written = append(f.written, nr)
	return history.Run{ID: 1, Kind: history.RunModel}, true, nil
}

// identities is the record's side: 25 receivers, all the same age and draft slot, so only their
// league points can separate them.
func identities() (listing, *feats) {
	l := listing{}
	f := &feats{rows: map[int][]history.Feature{}}
	for i := range regulars {
		id := mflID(i)
		l[id] = normalize.PlayerFacts{Name: fmt.Sprintf("Receiver %02d", i), Position: domain.PosWR}
		f.rows[0] = append(f.rows[0],
			history.Feature{PlayerID: id, Measure: "context.nfl_position", Text: "WR"},
			history.Feature{PlayerID: id, Measure: "context.birth_date", Text: "2000-03-01"},
			history.Feature{PlayerID: id, Measure: "context.rookie_season", Value: 2022})
	}
	return l, f
}

// inputs is the league as the valuation sees it: its own settings, every receiver rostered one
// per franchise in blocks of five, and the held weeks.
func inputs(t *testing.T, league string, rules leaguerules.RuleSet) Inputs {
	t.Helper()
	in := Inputs{League: league, Season: season, AsOf: migrationtest.ReadAt, LastWeek: lastWeek,
		Scoring: rules.Scoring, Lineup: rules.Lineup, Owner: map[string]string{},
		Weeks: heldWeeks(), GSIS: map[string]string{}, Notes: []string{}}
	for i := range regulars {
		in.Owner[mflID(i)] = fmt.Sprintf("%04d", 1+i/5)
		in.GSIS[mflID(i)] = gsisOf(i)
	}
	return in
}

func read(t *testing.T, franchise string, in Inputs) Reading {
	t.Helper()
	l, f := identities()
	got, err := Value(context.Background(), modelrun.Held{Listing: l, Features: f}, params.DefaultSet(),
		franchise, in)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// The correctness proof the brief asks for. The mirror values a player from MFL's held season
// total; this read values him from his held NFL weeks scored under the app league's own settings.
// Where the two agree on the points — which is what "the same scoring settings" means — every
// value must be identical, player for player. A second evaluation, a refit or a different scale
// anywhere in the adapter would separate them.
func TestAppLeagueValuesMatchTheMirrorForTheSamePoints(t *testing.T) {
	rules := legacySettings(t)
	in := inputs(t, "native.legacy", rules)
	app := read(t, "0001", in)

	// The mirror: the same players, the same franchise, and each season total set to exactly the
	// points this league's scoring settings produce.
	l, f := identities()
	totals := map[string]float64{}
	for _, weeks := range in.Weeks {
		for _, w := range weeks {
			if w.SeasonType != "REG" || w.Week > lastWeek {
				continue
			}
			points, _, err := fantasyscore.Score(rules.Scoring, leaguerules.WR, w.Line)
			if err != nil {
				t.Fatal(err)
			}
			totals[w.GSIS] += float64(points.Value) / 10000
		}
	}
	var roster []state.PlayerState
	for i := range regulars {
		f.rows[season-1] = append(f.rows[season-1], history.Feature{PlayerID: mflID(i), Season: season - 1,
			Measure: "outcome.fantasy_points", Value: totals[gsisOf(i)]})
		for w := 1; w <= 16; w++ {
			f.rows[season-1] = append(f.rows[season-1], history.Feature{PlayerID: mflID(i), Season: season - 1,
				Week: w, Measure: "exposure.offense_snaps", Value: 40})
		}
		roster = append(roster, state.PlayerState{MFLID: mflID(i)})
	}
	runner, err := modelrun.New(mirror{roster}, modelrun.Held{Listing: l, Features: f}, f, "v-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), modelrun.Spec{Season: season, AsOf: migrationtest.ReadAt,
		Params: params.DefaultSet(), LastWeek: lastWeek}); err != nil {
		t.Fatal(err)
	}
	mirrorValue := map[string]history.ModelScore{}
	for _, sc := range f.written[0].Scores {
		mirrorValue[sc.MFLID] = sc
	}
	if len(mirrorValue) != regulars {
		t.Fatalf("the mirror valued %d of %d players", len(mirrorValue), regulars)
	}
	checked := 0
	for _, p := range read(t, "", in).Players {
		want, ok := mirrorValue[p.Player]
		if !ok {
			t.Fatalf("the mirror did not value %s", p.Player)
		}
		if !p.Available || want.Unavailable {
			t.Fatalf("%s: app available %t, mirror unavailable %t (%s / %s)", p.Player, p.Available,
				want.Unavailable, p.Words, want.Reason)
		}
		if p.NowPPG != want.NowPPG || p.DynastyPP != want.DynastyPPG || p.Position != want.Position {
			t.Errorf("%s: app %s now %v dynasty %v; mirror %s now %v dynasty %v", p.Player, p.Position,
				p.NowPPG, p.DynastyPP, want.Position, want.NowPPG, want.DynastyPPG)
		}
		checked++
	}
	if checked != regulars {
		t.Fatalf("compared %d of %d players", checked, regulars)
	}
	if app.Best.StarterN == 0 || !app.Best.Available || app.Best.Value <= 0 {
		t.Fatalf("best N = %+v", app.Best)
	}
}

type mirror struct{ roster []state.PlayerState }

func (m mirror) Franchises() []string { return []string{"0001"} }
func (m mirror) Roster(string) ([]state.PlayerState, bool) {
	return m.roster, true
}
func (m mirror) FranchiseState(string) (state.FranchiseState, bool) {
	return state.FranchiseState{}, false
}
func (m mirror) CapUsed(string) (domain.Money, bool)     { return 0, false }
func (m mirror) Player(string) (state.PlayerState, bool) { return state.PlayerState{}, false }

// Two leagues whose scoring settings are equal read identical values, whatever their ids: the
// read may not branch on where a league came from.
func TestEqualScoringSettingsGiveEqualValuesWhateverTheLeague(t *testing.T) {
	rules := legacySettings(t)
	imported := read(t, "0001", inputs(t, "native.imported-from-mfl", rules))
	byHand := read(t, "0001", inputs(t, "native.built-by-hand", rules))
	if fmt.Sprint(imported.Players) != fmt.Sprint(byHand.Players) {
		t.Errorf("the same settings read different values:\n%v\n%v", imported.Players, byHand.Players)
	}
	if imported.Best.Value != byHand.Best.Value || imported.Best.StarterN != byHand.Best.StarterN {
		t.Errorf("best N differs: %+v vs %+v", imported.Best, byHand.Best)
	}
}

// Changing one scoring setting by hand must move the values: that is what makes them the league's.
func TestChangingOneScoringSettingMovesTheValues(t *testing.T) {
	rules := legacySettings(t)
	before := read(t, "0001", inputs(t, "native.legacy", rules))
	changed := legacySettings(t)
	in := inputs(t, "native.legacy", changed)
	in.Scoring = doubleReceivingYards(t, changed.Scoring)
	after := read(t, "0001", in)
	if before.Best.Value <= 0 || after.Best.Value <= before.Best.Value {
		t.Errorf("doubling the points for a receiving yard must raise the board: %v then %v",
			before.Best.Value, after.Best.Value)
	}
}

// Age bands hold the roster's value, and only players whose age is known.
func TestValueByAgeHoldsTheRostersValueInItsOwnBands(t *testing.T) {
	got := read(t, "0001", inputs(t, "native.legacy", legacySettings(t)))
	var banded float64
	for _, b := range got.Age {
		banded += b.Value
	}
	var roster float64
	for _, p := range got.Players {
		if p.Available {
			roster += p.DynastyPP
		}
	}
	if len(got.Age) != len(bandFrom()) || math.Abs(banded-roster) > 1e-9 {
		t.Errorf("bands %+v hold %v of the roster's %v", got.Age, banded, roster)
	}
	// Every receiver is 26 in 2026, so one band holds them all.
	held := 0
	for _, b := range got.Age {
		if b.Players > 0 {
			held++
			if b.From != 26 {
				t.Errorf("age-26 receivers landed in the band from %d", b.From)
			}
		}
	}
	if held != 1 {
		t.Errorf("%d bands hold players: %+v", held, got.Age)
	}
}

// doubleReceivingYards changes exactly one of the league's own scoring settings, as the
// rule-change flow would: what a receiving yard is worth. Nothing else moves.
func doubleReceivingYards(t *testing.T, rules leaguerules.ScoringRules) leaguerules.ScoringRules {
	t.Helper()
	held := rules.Rules.Value["receiving_yards"]
	if rules.Rules.State != leaguerules.IsKnown || len(held) == 0 || held[0].Points == 0 {
		t.Fatalf("the held 14432 settings score no receiving yards: %+v", held)
	}
	out := rules
	out.Rules.Value = maps.Clone(rules.Rules.Value)
	changed := slices.Clone(held)
	changed[0].Points *= 2
	out.Rules.Value["receiving_yards"] = changed
	return out
}

func TestMissingWeeksRefusesValuesInsteadOfFabricatingZero(t *testing.T) {
	in := inputs(t, "native.test", legacySettings(t))
	in.Weeks = nil
	got := read(t, "0001", in)
	if got.Best.Available || len(got.Notes) == 0 {
		t.Fatalf("missing weeks looked valued: %+v", got)
	}
	for _, p := range got.Players {
		if p.Available || p.Words == "" {
			t.Fatalf("missing production silently valued %s: %+v", p.Player, p)
		}
	}
}
