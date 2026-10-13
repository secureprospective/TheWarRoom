package main

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/labcompile"
)

type fakeAssets map[string]dynastyValue

func (f fakeAssets) player(pid string, _ time.Time) dynastyValue { return f[pid] }

func TestAssetBoardOnFieldNowFromHeldWeeks(t *testing.T) {
	in := realResearch(t) // week 5: weeks 1-4 are finished
	v := researchView{researchInput: in}
	b := buildAssetBoard(v, nil, time.Now(), []string{"99999"})
	if !strings.Contains(b.NowSource, "4 of 4 finished weeks held") {
		t.Fatal(b.NowSource)
	}
	// Recount one player by hand from the weekly feeds: points per game over the weeks he scored.
	const id = "16162"
	sum, games := 0.0, 0
	for w := 1; w <= 4; w++ {
		for _, e := range researchRows(v.feed("playerScores", strconv.Itoa(w))) {
			if e.ID.String() == id {
				sum, games = sum+e.Value, games+1
			}
		}
	}
	if games == 0 {
		t.Fatal("fixture player has no weekly scores")
	}
	cells := map[string]AssetValue{}
	for _, value := range b.Values {
		cells[value.ID] = value
	}
	got := cells[id].Now
	position := v.player(id).Position
	ppg := strconv.FormatFloat(sum/float64(games), 'f', 2, 64) + " PPG"
	if !got.Held || got.Text != ppg || !strings.HasPrefix(got.Sub, position+" ") || !strings.Contains(got.Sub, "· top ") ||
		cells[id].NowValue != sum/float64(games) ||
		!strings.Contains(got.Detail, strconv.Itoa(games)+" scored week") {
		t.Fatalf("%s: %+v, want %q", id, got, ppg)
	}
	// Ranks run 1..n at each position, the best points per game first.
	firsts := 0
	for _, value := range cells {
		if f := strings.Fields(value.Now.Sub); value.Now.Held && len(f) == 5 && strings.HasPrefix(f[1], "1/") {
			firsts++
		}
	}
	if firsts < 8 {
		t.Fatal("expected a leader at every position", firsts)
	}
	if c := cells["99999"].Now; c.Held || c.Text != "No score" || c.Sub != "this season" {
		t.Fatal("unscored rostered player", c)
	}
	if c := cells[id].Dynasty; c.Held || c.Text != "Not valued yet" || c.Sub != "Lab model not compiled" || b.DynastySource == "" {
		t.Fatal("no model", c, b.DynastySource)
	}
}

func TestAssetBoardDynastyCellsInWords(t *testing.T) {
	price, low, high := 3.1, 2.4, 3.9
	filler, fillerLow, fillerHigh := -0.03, -0.04, -0.01
	model := fakeAssets{
		"1": {Valued: true, Discounted: 2.314, Seasons: 5, ClassLabel: "Wide receiver, 25 to 27", Price: &price, Low: &low, High: &high},
		"2": {Valued: false, Reason: "The player has no known position or birth date"},
		"3": {Valued: true, Discounted: 0.5, Seasons: 5, ClassLabel: "Kicker, 28 or older"},
		"4": {Valued: true, Discounted: 0.05, Seasons: 5, ClassLabel: "Defensive back, 28 or older", Price: &filler, Low: &fillerLow, High: &fillerHigh},
	}
	cells := map[string]AssetValue{}
	for _, value := range buildAssetBoard(researchView{}, model, time.Now(), []string{"1", "2", "3", "4"}).Values {
		cells[value.ID] = value
	}
	if d, m := cells["1"].Dynasty, cells["1"].Market; d.Text != "2.31 wins" || d.Sub != "next 5 seasons" ||
		!strings.HasPrefix(d.Detail, "Wide receiver, 25 to 27 · ") || m.Text != "3.10" || m.Sub != "win-equivalents · 95% 2.40 to 3.90" || strings.Contains(m.Detail, "Below zero") || !m.Held {
		t.Fatal(d, m)
	}
	if cells["1"].DynastyValue != 2.314 || cells["2"].DynastyValue != 0 {
		t.Fatal("dynasty sort keys", cells["1"].DynastyValue, cells["2"].DynastyValue)
	}
	if d := cells["2"].Dynasty; d.Held || d.Text != "Not valued" || d.Sub != "The player has no known position or birth date" {
		t.Fatal(d)
	}
	if m := cells["4"].Market; m.Text != "−0.03" || m.Sub != "win-equivalents · 95% −0.04 to −0.01" || !strings.Contains(m.Detail, "Below zero") {
		t.Fatal(m)
	}
	if m := cells["3"].Market; m.Held || m.Text != "No price" || m.Sub != "no market for Kicker, 28 or older" {
		t.Fatal(m)
	}
}

func TestSignedPricesNeverReadMinusZero(t *testing.T) {
	for v, want := range map[float64]string{-0.03: "−0.03", -0.001: "0.00", 0: "0.00", 0.114: "0.11", -1.5: "−1.50"} {
		if got := signed(v); got != want {
			t.Errorf("signed(%v) = %q, want %q", v, got, want)
		}
	}
}

func TestTopPercentRoundsUp(t *testing.T) {
	for _, c := range [][3]int{{1, 50, 2}, {14, 54, 26}, {54, 54, 100}, {1, 1, 100}, {3, 200, 2}} {
		if got := topPercent(c[0], c[1]); got != c[2] {
			t.Errorf("topPercent(%d, %d) = %d, want %d", c[0], c[1], got, c[2])
		}
	}
}

func TestDynastyFromKeepsTheLabAnswer(t *testing.T) {
	price := 3.1
	d := dynastyFrom(labcompile.AssetValue{Forecast: []float64{1, 1, 1, 1, 1}, Discounted: 2.314, Class: "WR:prime",
		Label: "Wide receiver, 25 to 27", Price: &price, Range: &labcompile.Interval{Low: 2.4, High: 3.9}, Valued: true})
	if !d.Valued || d.Seasons != 5 || d.ClassLabel != "Wide receiver, 25 to 27" || *d.Price != 3.1 || *d.Low != 2.4 || *d.High != 3.9 {
		t.Fatalf("%+v", d)
	}
	if d := dynastyFrom(labcompile.AssetValue{Reason: "No production or rookie forecast is available"}); d.Valued ||
		d.Low != nil || d.Reason != "No production or rookie forecast is available" {
		t.Fatalf("%+v", d)
	}
	if d := (labAssets{}).player("13604", time.Now()); d.Valued || d.Reason == "" {
		t.Fatalf("a missing model must say so: %+v", d)
	}
}

func TestDynastyDisplayNeverShowsMinusZero(t *testing.T) {
	for _, v := range []float64{-0.0001, -0.004, 0} {
		cell, _, raw := dynastyCells(fakeAssets{"1": {Valued: true, Discounted: v, Seasons: 5}}, "1", time.Now())
		if cell.Text != "0.00 wins" || raw != v {
			t.Fatalf("display rounded zero or raw value changed: %+v, %v", cell, raw)
		}
	}
	cell, _, _ := dynastyCells(fakeAssets{"1": {Valued: true, Discounted: -0.01}}, "1", time.Now())
	if cell.Text != "−0.01 wins" {
		t.Fatal(cell.Text)
	}
}
