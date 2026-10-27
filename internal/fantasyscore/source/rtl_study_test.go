package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// Run with TWR_STUDY_CACHE set and go test -run TestRTLStudy -v ./internal/fantasyscore/source.
func TestRTLStudy(t *testing.T) {
	dir := os.Getenv("TWR_STUDY_CACHE")
	if dir == "" {
		t.Skip("set TWR_STUDY_CACHE to the held 2025 nflverse cache")
	}
	weeks := readGzip(t, filepath.Join(dir, "stats_player_week_2025.csv.gz"), source.ReadWeeks)
	plays := readGzip(t, filepath.Join(dir, "play_by_play_2025.csv.gz"), source.ReadPlays)
	var reg []source.PlayerWeek
	maxWeek := 0
	games := map[string]bool{}
	for _, play := range plays {
		if play.Season == "2025" && play.SeasonType == "REG" {
			if !play.EdgeAvailable {
				t.Fatal("play-by-play lacks edge columns")
			}
			games[play.GameID] = true
		}
	}
	if len(games) != 272 {
		t.Fatalf("2025 regular season incomplete: %d games", len(games))
	}
	for _, w := range weeks {
		if w.Season == 2025 && w.SeasonType == "REG" {
			reg = append(reg, w)
			if w.Week > maxWeek {
				maxWeek = w.Week
			}
		}
	}
	if maxWeek != 18 {
		t.Fatalf("2025 regular season incomplete: last week %d", maxWeek)
	}
	if err := source.AttachPlays(reg, plays); err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{13, 18} {
		for _, pos := range []string{"QB", "RB", "WR", "TE"} {
			var carries, losses int64
			for _, w := range reg {
				if w.Week <= limit && (w.Position == pos || (pos == "RB" && w.Position == "FB")) {
					carries += int64(w.Line.Quantities["rushing_attempt"]) / 10000
					losses += int64(w.Line.Quantities["rushing_tfl"]) / 10000
				}
			}
			if carries == 0 {
				t.Fatalf("no carries for %s", pos)
			}
			t.Logf("weeks 1-%d %s TFL=%d carries=%d r=%.8f extra=%.4f", limit, pos, losses, carries, float64(losses)/float64(carries), float64(int(3124*float64(losses)/float64(carries)+0.5))/10000)
		}
	}
}
