package modelrun

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// kickers is 25 K regulars in 2025 scoring 60, 65, … 180 league points; two are rostered: the
// best, aged 36 (past the kicker peak, so the age pull bites), and the 11th, aged 27.
func kickers() (*fakeHistory, fakeState, fakeDir, map[string]float64) {
	h := &fakeHistory{feats: map[int][]history.Feature{}}
	add := func(id string, season, week int, measure string, v float64, text string) {
		h.feats[season] = append(h.feats[season], history.Feature{PlayerID: id, Season: season, Week: week,
			Measure: measure, Value: v, Text: text})
	}
	totals := map[string]float64{}
	for i := range 25 {
		id := fmt.Sprintf("%04d", 400+i)
		birth := "1999-03-01"
		if i == 24 {
			birth = "1989-03-01"
		}
		add(id, 0, 0, "context.nfl_position", 0, "K")
		add(id, 0, 0, "context.birth_date", 0, birth)
		add(id, 0, 0, "context.rookie_season", 2021, "")
		for w := 1; w <= 16; w++ {
			add(id, 2025, w, "exposure.offense_snaps", 5, "")
		}
		totals[id] = float64(60 + 5*i)
		add(id, 2025, 0, "outcome.fantasy_points", totals[id], "")
	}
	dir := fakeDir{"0424": {Name: "Old Leg", Position: domain.PosK}, "0410": {Name: "Mid Leg", Position: domain.PosK}}
	return h, fakeState{roster: []state.PlayerState{{MFLID: "0424"}, {MFLID: "0410"}}}, dir, totals
}

func kickerScores(t *testing.T, set params.Set) []history.ModelScore {
	t.Helper()
	h, st, dir, _ := kickers()
	r, err := New(st, dir, h, "v-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(context.Background(), Spec{Season: 2026, AsOf: time.Now(), Params: set, LastWeek: 17}); err != nil {
		t.Fatal(err)
	}
	return h.written[0].Scores
}

// The fit found the model ranks kickers worse than today's board on both talent and value
// (Fit_Report "Against today's board"), so the shipped params fall back at K: each kicker's
// value is last season's total league points times the engine's age pull, per game of a season.
func TestKickerValueFallsBackToTodaysBoardRule(t *testing.T) {
	_, _, dir, totals := kickers()
	set := params.DefaultSet()
	if kept, err := set.GetPosition("model.board_kept", string(domain.PosK)); err != nil || kept != 0 {
		t.Fatalf("shipped kicker params keep the model: %v %v", kept, err)
	}
	decay, err := set.GetGlobal(params.KeyLayer3DecayRate)
	if err != nil {
		t.Fatal(err)
	}
	for _, sc := range kickerScores(t, set) {
		year := map[string]int{"0424": 1989, "0410": 1999}[sc.MFLID]
		kicker := model.Player{Birth: time.Date(year, 3, 1, 0, 0, 0, 0, time.UTC)}
		pull, err := AgePull(dir[sc.MFLID].Position, kicker.AgeAt(2026), decay)
		if err != nil {
			t.Fatal(err)
		}
		want := totals[sc.MFLID] * pull / 17
		if math.Abs(sc.NowPPG-want) > 1e-9 || math.Abs(sc.DynastyPPG-want) > 1e-9 {
			t.Errorf("%s: now %.3f dynasty %.3f, want today's rule %.3f (total %.0f × pull %.3f / 17)",
				sc.MFLID, sc.NowPPG, sc.DynastyPPG, want, totals[sc.MFLID], pull)
		}
	}
	// Keeping the model at K values the same kickers differently.
	values := set.Values()
	values["model.board_kept@K"] = 1
	for i, sc := range kickerScores(t, params.SetOf(values)) {
		if fallback := kickerScores(t, set)[i]; sc.DynastyPPG == fallback.DynastyPPG {
			t.Errorf("%s: the model and today's rule agree, so the test proves nothing", sc.MFLID)
		}
	}
}

// Every other position beat today's board on at least one ranking and keeps the model.
func TestOnlyKickersFallBackToTodaysBoardRule(t *testing.T) {
	set := params.DefaultSet()
	for _, pos := range []domain.Position{domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosDT,
		domain.PosDE, domain.PosLB, domain.PosCB, domain.PosS} {
		if kept, err := set.GetPosition("model.board_kept", string(pos)); err != nil || kept != 1 {
			t.Errorf("%s: board_kept %v %v", pos, kept, err)
		}
	}
}

// A kicker with no games last season is valued on this season's pace, not 0: Zvada's 38 points
// in 2026 with no 2025 season read 0 on the board before.
func TestKickerWithoutLastSeasonIsValuedOnThisSeasonsPace(t *testing.T) {
	h, st, dir, _ := kickers()
	add := func(season, week int, measure string, v float64, text string) {
		h.feats[season] = append(h.feats[season], history.Feature{PlayerID: "0430", Season: season, Week: week,
			Measure: measure, Value: v, Text: text})
	}
	add(0, 0, "context.nfl_position", 0, "K")
	add(0, 0, "context.birth_date", 0, "1999-03-01")
	add(0, 0, "context.rookie_season", 2026, "")
	for w := 1; w <= 6; w++ {
		add(2026, w, "exposure.offense_snaps", 5, "")
	}
	add(2026, 0, "outcome.fantasy_points", 48, "")
	dir["0430"] = normalize.PlayerFacts{Name: "New Leg", Position: domain.PosK}
	st.roster = append(st.roster, state.PlayerState{MFLID: "0430"})
	set := params.DefaultSet()
	r, err := New(st, dir, h, "v-test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(context.Background(), Spec{Season: 2026, AsOf: time.Now(), Params: set, LastWeek: 17}); err != nil {
		t.Fatal(err)
	}
	decay, err := set.GetGlobal(params.KeyLayer3DecayRate)
	if err != nil {
		t.Fatal(err)
	}
	pull, err := AgePull(domain.PosK, model.Player{Birth: time.Date(1999, 3, 1, 0, 0, 0, 0, time.UTC)}.AgeAt(2026), decay)
	if err != nil {
		t.Fatal(err)
	}
	for _, sc := range h.written[0].Scores {
		if sc.MFLID != "0430" {
			continue
		}
		if want := 48.0 / 6 * pull; math.Abs(sc.DynastyPPG-want) > 1e-9 || sc.DynastyPPG == 0 {
			t.Fatalf("dynasty %.3f, want this season's pace %.3f (8 a game × pull %.3f)", sc.DynastyPPG, want, pull)
		}
		return
	}
	t.Fatal("the new kicker was not valued")
}
