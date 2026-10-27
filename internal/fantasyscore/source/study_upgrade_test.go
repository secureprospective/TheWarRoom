package source_test

import (
	"compress/gzip"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// studyPositions maps an nflverse roster position to the league position it scores as.
func studyPositions() map[string]leaguerules.Position {
	return map[string]leaguerules.Position{"QB": leaguerules.QB, "RB": leaguerules.RB, "FB": leaguerules.RB,
		"WR": leaguerules.WR, "TE": leaguerules.TE, "K": leaguerules.PK, "DT": leaguerules.DT, "NT": leaguerules.DT,
		"DE": leaguerules.DE, "LB": leaguerules.LB, "OLB": leaguerules.LB, "ILB": leaguerules.LB, "MLB": leaguerules.LB,
		"CB": leaguerules.CB, "DB": leaguerules.CB, "S": leaguerules.S, "FS": leaguerules.S, "SS": leaguerules.S, "SAF": leaguerules.S}
}

// The Legacy seed upgrade (version 2) over every held 2023–25 study week, weeks 1–13: a league
// seeded before the scoring kinds scores the same after the upgrade except on the kick-return
// touchdowns the new rows price. Opt in with TWR_STUDY_CACHE=<config-study p3 cache directory>.
func TestStudyWeeksUnderTheLegacyUpgrade(t *testing.T) {
	dir := os.Getenv("TWR_STUDY_CACHE")
	if dir == "" {
		t.Skip("set TWR_STUDY_CACHE to the config study's held nflverse cache")
	}
	after := leaguerules.LegacyNFL().Scoring
	before := after
	old := maps.Clone(after.Catalogue.Value)
	for _, id := range []leaguerules.StatID{"other_return_td", "blocked_fg_td", "blocked_punt_td"} {
		delete(old, id)
	}
	before.Catalogue = leaguerules.Known(old)
	before.Ranges = leaguerules.Known(map[leaguerules.StatID]leaguerules.StatRange{})
	positions := studyPositions()
	scored, changed := 0, 0
	for _, season := range []int{2023, 2024, 2025} {
		weeks := readGzip(t, filepath.Join(dir, fmt.Sprintf("stats_player_week_%d.csv.gz", season)), source.ReadWeeks)
		plays := readGzip(t, filepath.Join(dir, fmt.Sprintf("play_by_play_%d.csv.gz", season)), source.ReadPlays)
		var regular []source.PlayerWeek
		for _, w := range weeks {
			if w.SeasonType == "REG" {
				regular = append(regular, w)
			}
		}
		if err := source.AttachPlays(regular, plays); err != nil {
			t.Fatal(err)
		}
		for _, w := range regular {
			position, ok := positions[w.Position]
			if !ok || w.Week > 13 {
				continue
			}
			was, _, err := fantasyscore.Score(before, position, w.Line)
			now, items, nerr := fantasyscore.Score(after, position, w.Line)
			if err != nil || nerr != nil || was.State != leaguerules.IsKnown || now.State != leaguerules.IsKnown {
				t.Fatalf("%s %d/%d: %v %v", w.GSIS, season, w.Week, err, nerr)
			}
			scored++
			if was.Value == now.Value {
				continue
			}
			changed++
			var why []string
			for _, item := range items {
				switch item.Stat {
				case "other_return_td", "blocked_fg_td", "blocked_punt_td":
					if item.Points.Value != 0 {
						why = append(why, string(item.Stat))
					}
				}
			}
			if len(why) == 0 {
				t.Errorf("%s %d week %d (%s) changed %d → %d with no kick-return touchdown", w.GSIS, season, w.Week, position, was.Value, now.Value)
			}
			t.Logf("CHANGED %s %d week %d %s (%s): %d → %d %v", w.GSIS, season, w.Week, position, w.Position, was.Value, now.Value, why)
		}
	}
	t.Logf("SCORED %d player-weeks; %d changed", scored, changed)
}

func readGzip[T any](t *testing.T, path string, read func(r io.Reader) ([]T, error)) []T {
	t.Helper()
	f, err := os.Open(path) //nolint:gosec // the operator's own study cache, opted in by TWR_STUDY_CACHE
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := read(z)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return rows
}
