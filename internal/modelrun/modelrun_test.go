package modelrun

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type fakeState struct{ roster []state.PlayerState }

func (f fakeState) Franchises() []string { return []string{"0001"} }
func (f fakeState) Roster(string) ([]state.PlayerState, bool) {
	return f.roster, true
}
func (f fakeState) FranchiseState(string) (state.FranchiseState, bool) {
	return state.FranchiseState{}, false
}
func (f fakeState) CapUsed(string) (domain.Money, bool)     { return 0, false }
func (f fakeState) Player(string) (state.PlayerState, bool) { return state.PlayerState{}, false }

type fakeDir map[string]normalize.PlayerFacts

func (f fakeDir) Facts(id string) (normalize.PlayerFacts, bool) {
	pf, ok := f[id]
	return pf, ok
}

// fakeHistory serves features by season and keeps the model run written.
type fakeHistory struct {
	feats   map[int][]history.Feature
	written []history.NewModelRun
}

func (f *fakeHistory) Features(_ context.Context, q history.FeatureQuery) ([]history.Feature, error) {
	return f.feats[q.Season], nil
}

func (f *fakeHistory) WriteModelRun(_ context.Context, nr history.NewModelRun) (history.Run, bool, error) {
	f.written = append(f.written, nr)
	return history.Run{ID: int64(len(f.written)), Kind: history.RunModel}, true, nil
}

// league is 25 WR regulars in 2025, the first of them rostered, plus a rostered rookie with no
// history and a rostered id the players database does not know.
func league() (*fakeHistory, fakeState, fakeDir) {
	h := &fakeHistory{feats: map[int][]history.Feature{}}
	add := func(id string, season, week int, measure string, v float64, text string) {
		h.feats[season] = append(h.feats[season], history.Feature{PlayerID: id, Season: season, Week: week,
			Measure: measure, Value: v, Text: text})
	}
	for i := range 25 {
		id := fmt.Sprintf("%04d", 200+i)
		add(id, 0, 0, "context.nfl_position", 0, "WR")
		add(id, 0, 0, "context.birth_date", 0, "1999-03-01")
		add(id, 0, 0, "context.rookie_season", 2022, "")
		for w := 1; w <= 16; w++ {
			add(id, 2025, w, "exposure.offense_snaps", 40, "")
		}
		add(id, 2025, 0, "outcome.fantasy_points", float64(50+10*i), "")
	}
	birth := time.Date(2004, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	dir := fakeDir{
		"0200": {Name: "Veteran", Position: domain.PosWR},
		"0300": {Name: "Rookie", Position: domain.PosWR, IsRookie: true, Birthdate: birth, HasBirthdate: true},
	}
	st := fakeState{roster: []state.PlayerState{{MFLID: "0200"}, {MFLID: "0300"}, {MFLID: "0999"}}}
	return h, st, dir
}

func TestRunValuesTheRosterAndRecordsInputs(t *testing.T) {
	h, st, dir := league()
	r, err := New(st, dir, h, "v-test")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := r.Run(context.Background(), Spec{Season: 2026, AsOf: time.Now(), Params: params.DefaultSet(), LastWeek: 17})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Scored != 2 || rep.Rookies != 1 || len(rep.Excluded) != 1 || rep.Excluded[0].MFLID != "0999" {
		t.Fatalf("report = %+v", rep)
	}
	run := h.written[0]
	for k := range run.Params.Params {
		if !params.IsModel(k) {
			t.Errorf("model run records board param %q", k)
		}
	}
	vet, rookie := run.Scores[0], run.Scores[1]
	if vet.SeasonGames != 0 || vet.PastGames != 16 || vet.Now >= 0.5 || vet.NowPPG <= 0 {
		t.Errorf("the league's lowest-scoring regular: %+v", vet)
	}
	if want := []string{"entry_age", "age", "season.2025"}; fmt.Sprint(vet.Inputs) != fmt.Sprint(want) {
		t.Errorf("veteran inputs = %v, want %v", vet.Inputs, want)
	}
	if rookie.Now <= 0 || rookie.Dynasty <= 0 || fmt.Sprint(rookie.Inputs) != "[age]" {
		t.Errorf("rookie = %+v", rookie)
	}
}

// A history record whose first NFL season is far from MFL's draft year belongs to another player:
// the rookie is valued on MFL's birth date, not the old record's, and the run names him.
func TestRunSetsAsideAnotherPlayersRecord(t *testing.T) {
	h, st, dir := league()
	add := func(measure string, v float64, text string) {
		h.feats[0] = append(h.feats[0], history.Feature{PlayerID: "0300", Measure: measure, Value: v, Text: text})
	}
	add("context.birth_date", 0, "1953-01-09")
	add("context.rookie_season", 1978, "")
	rookie := dir["0300"]
	rookie.DraftYear, rookie.HasDraftYear = 2026, true
	dir["0300"] = rookie
	r, err := New(st, dir, h, "v-test")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := r.Run(context.Background(), Spec{Season: 2026, AsOf: time.Now(), Params: params.DefaultSet(), LastWeek: 17})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Mislinked) != 1 || rep.Mislinked[0].MFLID != "0300" {
		t.Fatalf("mislinked = %+v, want the rookie", rep.Mislinked)
	}
	got := h.written[0].Scores[1]
	if got.MFLID != "0300" || got.Now <= 0 || got.Dynasty <= 0 {
		t.Errorf("the rookie must be valued on MFL's facts, not a 73-year-old's: %+v", got)
	}
}

func TestAtLeaguePositionsUsesTheLeaguesPosition(t *testing.T) {
	d := model.Data{Players: map[string]*model.Player{
		"edge":    {ID: "edge", Position: domain.PosLB}, // nflverse LB, the league's DE
		"unknown": {ID: "unknown", Position: domain.PosDE},
		"flagged": {ID: "flagged", Position: domain.PosCB}, // the league's code is not one the model scores
		"db":      {ID: "db"},                              // nflverse's generic DB, which the model leaves out
		"same":    {ID: "same", Position: domain.PosDT},
	}}
	dir := fakeDir{
		"edge":    {Position: domain.PosDE},
		"flagged": {Position: domain.PosFlag},
		"db":      {Position: domain.PosS},
		"same":    {Position: domain.PosDT},
	}
	if moved := AtLeaguePositions(d, dir); moved != 2 {
		t.Fatalf("moved %d players, want 2", moved)
	}
	want := map[string]domain.Position{"edge": domain.PosDE, "unknown": domain.PosDE, "flagged": domain.PosCB,
		"db": domain.PosS, "same": domain.PosDT}
	for id, pos := range want {
		if got := d.Players[id].Position; got != pos {
			t.Errorf("%s at %q, want %q", id, got, pos)
		}
	}
}
