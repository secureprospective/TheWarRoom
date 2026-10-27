package source_test

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// e5Rate is the owner-ruled prior full-season rate: 2024 play-by-play rushes tackled for loss over
// 2024 official carries, priced at the loss rule and rounded to the four decimals the engine keeps.
type e5Rate struct{ losses, carries leaguerules.Decimal }

// e5Study is what scripts/e5_study.py computes for 2025 weeks 1-13 from the same held sources,
// counting the play-derived stats independently in Python. Never copied from the engine.
func e5Study() map[string]leaguerules.Decimal {
	return map[string]leaguerules.Decimal{
		"Hunt": 1634400, "McCaffrey": 4196000, "Hurts": 3265300, "Allen": 3569000, "Barkley": 2302800,
	}
}

func TestE5Golden(t *testing.T) {
	weeks := readGzip(t, "testdata/e5_weeks.csv.gz", source.ReadWeeks)
	plays := readGzip(t, "testdata/e5_plays.csv.gz", source.ReadPlays)
	if err := source.AttachPlays(weeks, plays); err != nil {
		t.Fatal(err)
	}
	sleeper := e5Sleeper(t)
	rules := leaguerules.Default32Scoring()
	rates := map[leaguerules.Position]e5Rate{
		leaguerules.QB: {78, 2352}, leaguerules.RB: {1106, 11832},
		leaguerules.WR: {62, 430}, leaguerules.TE: {7, 62},
	}
	for i, extra := range rules.Extras.Value["rushing_attempt"] {
		rate, ok := rates[extra.Positions[0]]
		if !ok || len(extra.Positions) != 1 {
			t.Fatalf("no ruled rate for %v", extra.Positions)
		}
		rules.Extras.Value["rushing_attempt"][i].Points = (3124*rate.losses + rate.carries/2) / rate.carries
	}
	players := map[string]struct {
		name, sleeper string
		position      leaguerules.Position
	}{
		"00-0033923": {"Hunt", "4098", leaguerules.RB}, "00-0033280": {"McCaffrey", "4034", leaguerules.RB},
		"00-0036389": {"Hurts", "6904", leaguerules.QB}, "00-0034857": {"Allen", "4984", leaguerules.QB},
		"00-0034844": {"Barkley", "4866", leaguerules.RB},
	}
	want := e5Study()
	totals := map[string]leaguerules.Decimal{}
	for _, week := range weeks {
		player, ok := players[week.GSIS]
		if !ok {
			t.Fatalf("unexpected player %s", week.GSIS)
		}
		raw, ok := sleeper[week.Week][player.sleeper]
		if !ok {
			t.Fatalf("missing Sleeper row for %s week %d", player.name, week.Week)
		}
		line, err := raw.Convert()
		if err != nil {
			t.Fatal(err)
		}
		for stat, quantity := range line.Quantities {
			week.Line.Quantities[stat] = quantity
		}
		points, _, err := fantasyscore.Score(rules, player.position, week.Line)
		if err != nil {
			t.Fatal(err)
		}
		if points.State != leaguerules.IsKnown {
			t.Fatalf("%s week %d is not fully known: %+v", player.name, week.Week, points)
		}
		totals[player.name] += points.Value
	}
	if len(totals) != len(want) {
		t.Fatalf("scored %d players, want %d", len(totals), len(want))
	}
	for name, study := range want {
		got := totals[name]
		difference := got - study
		if difference < 0 {
			difference = -difference
		}
		// One cent per player-season, the tolerance E5 was set: not widened.
		if difference > 100 {
			t.Errorf("%s: engine %.4f, study %.4f, off by %.4f", name,
				float64(got)/10000, float64(study)/10000, float64(difference)/10000)
			continue
		}
		t.Logf("%s: engine %.4f, study %.4f", name, float64(got)/10000, float64(study)/10000)
	}
}

func e5Sleeper(t *testing.T) map[int]map[string]source.RawSleeperStats {
	t.Helper()
	f, err := os.Open("testdata/e5_sleeper.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var byWeek map[string]map[string]source.RawSleeperStats
	if err = json.NewDecoder(z).Decode(&byWeek); err != nil {
		t.Fatal(err)
	}
	rows := make(map[int]map[string]source.RawSleeperStats, len(byWeek))
	for week := 1; week <= 13; week++ {
		body, ok := byWeek[strconv.Itoa(week)]
		if !ok {
			t.Fatalf("pinned Sleeper fixture is missing week %d", week)
		}
		rows[week] = body
	}
	return rows
}
