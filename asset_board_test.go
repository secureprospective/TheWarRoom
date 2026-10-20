package main

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

type fakeAssets map[string]dynastyValue

func (f fakeAssets) Player(pid string, _ time.Time) dynastyValue { return f[pid] }
func (fakeAssets) Pick(int, int, string, time.Time) dynastyValue { return dynastyValue{} }

func TestAssetBoardOnFieldNowFromHeldWeeks(t *testing.T) {
	t.Parallel()
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
