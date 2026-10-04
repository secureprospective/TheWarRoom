package model

import (
	"math"
	"testing"

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
	names := PriorFeatures()
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

func TestParamsRoundTripThroughValues(t *testing.T) {
	n := len(PriorFeatures())
	p := Params{Intercept: 0.4, Weight: make([]float64, n), Missing: make([]float64, n), KNow: 3, KDynasty: 9,
		Exponential: true, Recency: [2]float64{0.5, 0.2}, Arc: [4]float64{0.01, -0.02, -0.001, 0.05},
		Survival: [6]float64{1, -0.1, -0.01, -0.3, 2, 1}}
	p.Weight[0], p.Missing[n-1] = -0.1, 0.02
	values := p.Values()
	back, err := ParamsFrom(func(k string) (float64, error) { return values[k], nil })
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
