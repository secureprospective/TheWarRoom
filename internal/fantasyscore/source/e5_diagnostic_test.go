package source_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestE5Diagnostic(t *testing.T) {
	dir := os.Getenv("TWR_STUDY_CACHE")
	sleeper := os.Getenv("TWR_SLEEPER_STUDY_CACHE")
	if dir == "" || sleeper == "" {
		t.Skip("set TWR_STUDY_CACHE and TWR_SLEEPER_STUDY_CACHE for the E5 discrepancy diagnostic")
	}
	prior := readGzip(t, filepath.Join(dir, "stats_player_week_2024.csv.gz"), source.ReadWeeks)
	plays24 := readGzip(t, filepath.Join(dir, "play_by_play_2024.csv.gz"), source.ReadPlays)
	var regular []source.PlayerWeek
	for _, w := range prior {
		if w.SeasonType == "REG" && w.Season == 2024 {
			regular = append(regular, w)
		}
	}
	if err := source.AttachPlays(regular, plays24); err != nil {
		t.Fatal(err)
	}
	counts := map[string][2]int64{}
	for _, w := range regular {
		p := w.Position
		if p == "FB" {
			p = "RB"
		}
		a := counts[p]
		a[0] += int64(w.Line.Quantities["rushing_tfl"]) / 10000
		a[1] += int64(w.Line.Quantities["rushing_attempt"]) / 10000
		counts[p] = a
	}
	rules := leaguerules.Default32Scoring()
	for i, extra := range rules.Extras.Value["rushing_attempt"] {
		a := counts[string(extra.Positions[0])]
		if a[1] == 0 {
			t.Fatal("missing rate")
		}
		rules.Extras.Value["rushing_attempt"][i].Points = leaguerules.Decimal((3124*a[0] + a[1]/2) / a[1])
		t.Logf("rate %s %d/%d extra=%d", extra.Positions[0], a[0], a[1], rules.Extras.Value["rushing_attempt"][i].Points)
	}
	weeks := readGzip(t, filepath.Join(dir, "stats_player_week_2025.csv.gz"), source.ReadWeeks)
	plays := readGzip(t, filepath.Join(dir, "play_by_play_2025.csv.gz"), source.ReadPlays)
	players := map[string]struct {
		name, sid string
		pos       leaguerules.Position
	}{"00-0033923": {"Hunt", "4098", leaguerules.RB}, "00-0033280": {"McCaffrey", "4034", leaguerules.RB}, "00-0036389": {"Hurts", "6904", leaguerules.QB}, "00-0034857": {"Allen", "4984", leaguerules.QB}, "00-0034844": {"Barkley", "4866", leaguerules.RB}}
	var selected []source.PlayerWeek
	for _, w := range weeks {
		if _, ok := players[w.GSIS]; ok && w.SeasonType == "REG" && w.Week <= 13 {
			selected = append(selected, w)
		}
	}
	if err := source.AttachPlays(selected, plays); err != nil {
		t.Fatal(err)
	}
	bodies := map[int]map[string]source.RawSleeperStats{}
	for i := 1; i <= 13; i++ {
		b, err := os.ReadFile(filepath.Join(sleeper, fmt.Sprintf("sleeper_2025_%d.json", i))) //nolint:gosec // operator-selected opt-in study cache
		if err != nil {
			t.Fatal(err)
		}
		var rows map[string]source.RawSleeperStats
		if err = json.Unmarshal(b, &rows); err != nil {
			t.Fatal(err)
		}
		bodies[i] = rows
	}
	total := map[string]leaguerules.Decimal{}
	breakdown := map[string]map[string]leaguerules.Decimal{}
	for _, w := range selected {
		p := players[w.GSIS]
		raw, ok := bodies[w.Week][p.sid]
		if !ok {
			t.Fatalf("missing %s week %d", p.name, w.Week)
		}
		line, err := raw.Convert()
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range line.Quantities {
			w.Line.Quantities[k] = v
		}
		points, items, err := fantasyscore.Score(rules, p.pos, w.Line)
		if err != nil {
			t.Fatal(err)
		}
		if points.State != leaguerules.IsKnown {
			t.Fatalf("%s %+v", p.name, points)
		}
		total[p.name] += points.Value
		if breakdown[p.name] == nil {
			breakdown[p.name] = map[string]leaguerules.Decimal{}
		}
		for _, it := range items {
			if !it.Unused && it.Points.State == leaguerules.IsKnown {
				key := string(it.Stat)
				if it.Extra {
					key += "_extra"
				}
				breakdown[p.name][key] += it.Points.Value
			}
		}
	}
	for _, p := range players {
		t.Logf("%s total=%.4f", p.name, float64(total[p.name])/10000)
		var keys []string
		for k := range breakdown[p.name] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := breakdown[p.name][k]
			if v != 0 {
				t.Logf("%s %s %.4f", p.name, k, float64(v)/10000)
			}
		}
	}
}
