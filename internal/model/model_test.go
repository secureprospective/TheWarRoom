package model

import (
	"math"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestScaleCountsTiesHalfAndStaysInside(t *testing.T) {
	s := NewScale([]float64{1, 2, 2, 3})
	for _, c := range []struct{ x, want float64 }{{2, 0.5}, {2.5, 0.75}, {0, 0.125}, {9, 0.875}, {math.NaN(), 0.5}} {
		if got := s.Pct(c.x); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("Pct(%v) = %v, want %v", c.x, got, c.want)
		}
	}
}

func TestBuildCountsWeeksPlayedAndLeaguePoints(t *testing.T) {
	obs := []Obs{
		{Player: "1", Measure: mPosition, Text: "WR"},
		{Player: "1", Measure: mBirth, Text: "2000-09-01"},
		{Player: "1", Measure: mRookie, Value: 2022},
		{Player: "1", Season: 2024, Week: 1, Measure: mOffSnaps, Value: 40},
		{Player: "1", Season: 2024, Week: 1, Measure: mWeekPoints, Value: 12},
		{Player: "1", Season: 2024, Week: 2, Measure: mOffSnaps, Value: 30},  // played, scored nothing
		{Player: "1", Season: 2024, Week: 3, Measure: mWeekPoints, Value: 6}, // scored without a snap row
		{Player: "1", Season: 2024, Week: 18, Measure: mOffSnaps, Value: 50}, // after the league's season
		{Player: "1", Season: 2021, Measure: "prior.college_receiving_yards", Value: 900},
	}
	d := Build(obs, map[int]int{2024: 17})
	s := d.Seasons["1"][2024]
	if s.Games() != 3 || s.PPG() != 6 {
		t.Fatalf("games %d ppg %v, want 3 and 6", s.Games(), s.PPG())
	}
	p := d.Players["1"]
	if p.Position != domain.PosWR || p.Experience(2024) != 3 || math.Abs(p.AgeAt(2024)-24) > 0.01 {
		t.Errorf("player = %+v", p)
	}
	if p.College[2021]["receiving_yards"] != 900 {
		t.Errorf("college = %v", p.College)
	}
}

func TestPriorInputsReadDraftAgeCombineAndCollege(t *testing.T) {
	p := &Player{Position: domain.PosWR, DraftPick: 20, Rookie: 2022, Combine: map[string]float64{"forty": 4.4},
		College: map[int]map[string]float64{2021: {"receiving_yards": 900, "team_receiving_yards": 3000}}}
	x, known := p.PriorInputs()
	names := PriorFeatures(domain.PosWR)
	got := map[string]float64{}
	for i, n := range names {
		if known[i] {
			got[n] = x[i]
		}
	}
	if math.Abs(got["draft"]-math.Log(20)) > 1e-12 || got["forty"] != 4.4 || math.Abs(got["college"]-0.3) > 1e-12 {
		t.Errorf("inputs = %v", got)
	}
	if _, ok := got["entry_age"]; ok {
		t.Error("entry age with no birth date must be unknown")
	}
}

func TestPriorInputsAtDTReadEachDefensiveShare(t *testing.T) {
	p := &Player{Position: domain.PosDT, Rookie: 2024, College: map[int]map[string]float64{
		2022: {"sacks": 9, "team_sacks": 30},
		2023: {"sacks": 6, "team_sacks": 24, "tackles_for_loss": 10, "team_tackles_for_loss": 80,
			"qb_hurries": 12, "team_qb_hurries": 48, "total_tackles": 40},
	}}
	got := inputs(p)
	want := map[string]float64{"college_sacks": 0.25, "college_tackles_for_loss": 0.125, "college_qb_hurries": 0.25}
	for name, v := range want {
		if math.Abs(got[name]-v) > 1e-12 {
			t.Errorf("%s = %v, want %v (the last college season)", name, got[name], v)
		}
	}
	if _, ok := got["college_total_tackles"]; ok {
		t.Error("a share without the team's total must be unknown")
	}
	if _, ok := got["college"]; ok {
		t.Error("DT reads each share, not the single college number")
	}
}

func TestPriorInputsAtDEAddTheBestSeason(t *testing.T) {
	p := &Player{Position: domain.PosDE, Rookie: 2025, College: map[int]map[string]float64{
		2022: {"sacks": 12, "team_sacks": 30, "tackles_for_loss": 10, "team_tackles_for_loss": 100},
		2023: {"sacks": 3, "team_sacks": 30, "tackles_for_loss": 20, "team_tackles_for_loss": 100},
		2024: {"sacks": 6, "team_sacks": 30, "tackles_for_loss": 10, "team_tackles_for_loss": 100},
		2025: {"sacks": 30, "team_sacks": 30}, // an NFL-year row is not college
	}}
	got := inputs(p)
	if math.Abs(got["college"]-0.15) > 1e-12 {
		t.Errorf("college = %v, want 0.15 (the last season, 2024)", got["college"])
	}
	if math.Abs(got["college_best"]-0.3) > 1e-12 || got["college_seasons"] != 3 {
		t.Errorf("best %v over %v seasons, want 0.3 (best sacks 0.4, best TFL 0.2) over 3", got["college_best"], got["college_seasons"])
	}
}

// inputs is the player's known prior inputs by name.
func inputs(p *Player) map[string]float64 {
	x, known := p.PriorInputs()
	out := map[string]float64{}
	for i, name := range PriorFeatures(p.Position) {
		if known[i] {
			out[name] = x[i]
		}
	}
	return out
}

func TestParamsRoundTripThroughValues(t *testing.T) {
	n := len(PriorFeatures(domain.PosDT))
	p := Params{Position: domain.PosDT, Intercept: 0.4, Weight: make([]float64, n), Missing: make([]float64, n), KNow: 3, KDynasty: 9,
		Exponential: true, Recency: [2]float64{0.5, 0.2}, Arc: [4]float64{0.01, -0.02, -0.001, 0.05},
		Survival: [6]float64{1, -0.1, -0.01, -0.3, 2, 1}}
	p.Weight[0], p.Missing[n-1] = -0.1, 0.02
	values := p.Values()
	back, err := ParamsFrom(domain.PosDT, func(k string) (float64, error) { return values[k], nil })
	if err != nil {
		t.Fatal(err)
	}
	if back.Intercept != p.Intercept || back.Weight[0] != -0.1 || back.Missing[n-1] != 0.02 || !back.Exponential ||
		back.Arc != p.Arc || back.Survival != p.Survival || back.Recency != p.Recency || back.KDynasty != 9 {
		t.Errorf("round trip = %+v", back)
	}
}

func TestZShapes(t *testing.T) {
	p := Params{}
	if got := p.Z(10, 10); got != 0.5 {
		t.Errorf("credibility Z = %v", got)
	}
	p.Exponential = true
	if got := p.Z(10, 10); math.Abs(got-(1-math.Exp(-1))) > 1e-12 {
		t.Errorf("exponential Z = %v", got)
	}
	if p.Z(0, 10) != 0 {
		t.Error("no evidence must give production no weight")
	}
}

func testParams() Params {
	n := len(PriorFeatures(domain.PosWR))
	return Params{Position: domain.PosWR, Intercept: 0.4, Weight: make([]float64, n), Missing: make([]float64, n), KNow: 3, KDynasty: 8,
		Recency: [2]float64{0.5, 0.25}, Arc: [4]float64{-0.02, -0.01, 0, 0.05},
		Survival: [6]float64{2, -0.1, -0.01, 0, 1, 1}}
}

func TestProjectOneSeasonIsTheSeasonPairPrediction(t *testing.T) {
	p := testParams()
	pl := &Player{Position: domain.PosWR, Birth: time.Date(1998, 9, 1, 0, 0, 0, 0, time.UTC), Rookie: 2021}
	got, e := p.Project(pl, []Past{{Year: 2024, Games: 12, Pct: 0.8}}, 2025)
	z := 12.0 / 20
	want := z*0.8 + (1-z)*0.4 + p.ArcStep(pl.AgeAt(2024), 4)
	if math.Abs(got-want) > 1e-12 || e != 12 {
		t.Errorf("Project = %v (e %v), want %v (e 12)", got, e, want)
	}
}

func TestRookieIsHisPriorAndNeverZero(t *testing.T) {
	p := testParams()
	pl := &Player{Position: domain.PosRB, Birth: time.Date(2004, 1, 1, 0, 0, 0, 0, time.UTC), Rookie: 2026}
	v := p.Value(pl, nil, Past{Year: 2026}, Horizon{Seasons: 5, Discount: 0.85}, NewScale([]float64{5, 10, 15, 20}))
	if v.Now != 0.4 || v.Dynasty <= 0 || v.NowPPG <= 0 || v.ZPast != 0 || v.ZNow != 0 {
		t.Errorf("rookie value = %+v", v)
	}
}

func TestThisSeasonsGamesMoveNowByKNow(t *testing.T) {
	p := testParams()
	pl := &Player{Position: domain.PosWR, Birth: time.Date(1998, 9, 1, 0, 0, 0, 0, time.UTC), Rookie: 2021}
	past := []Past{{Year: 2025, Games: 16, Pct: 0.5}}
	before := p.Value(pl, past, Past{Year: 2026}, Horizon{Seasons: 1, Discount: 1}, Scale{})
	after := p.Value(pl, past, Past{Year: 2026, Games: 3, Pct: 0.9}, Horizon{Seasons: 1, Discount: 1}, Scale{})
	if want := 0.5*0.9 + 0.5*before.Now; math.Abs(after.Now-want) > 1e-12 || after.ZNow != 0.5 {
		t.Errorf("now = %v (Z %v), want %v (Z 0.5)", after.Now, after.ZNow, want)
	}
}

func TestScalePPGInvertsPct(t *testing.T) {
	s := NewScale([]float64{2, 4, 6, 8, 10})
	for _, x := range []float64{2, 5, 8, 10} {
		if got := s.PPG(s.Pct(x)); math.Abs(got-x) > 1e-9 {
			t.Errorf("PPG(Pct(%v)) = %v", x, got)
		}
	}
	if !math.IsNaN((Scale{}).PPG(0.5)) {
		t.Error("an empty scale has no points")
	}
}
