package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestStudyEdgeConversions(t *testing.T) {
	dir := os.Getenv("TWR_STUDY_CACHE")
	if dir == "" {
		t.Skip("set TWR_STUDY_CACHE to the config study's held nflverse cache")
	}
	weeks := readGzip(t, filepath.Join(dir, "stats_player_week_2025.csv.gz"), source.ReadWeeks)
	plays := readGzip(t, filepath.Join(dir, "play_by_play_2025.csv.gz"), source.ReadPlays)
	var selected []source.PlayerWeek
	for _, w := range weeks {
		if w.SeasonType == "REG" && w.Week <= 13 && (w.GSIS == "00-0033923" || w.GSIS == "00-0033280") {
			selected = append(selected, w)
		}
	}
	if err := source.AttachPlays(selected, plays); err != nil {
		t.Fatal(err)
	}
	totals := map[string][2]leaguerules.Decimal{}
	for _, w := range selected {
		a := totals[w.GSIS]
		a[0] += w.Line.Quantities["third_down_conversion"]
		a[1] += w.Line.Quantities["fourth_down_conversion"]
		totals[w.GSIS] = a
	}
	for _, tc := range []struct {
		name, id string
		want     [2]leaguerules.Decimal
	}{{"Hunt", "00-0033923", [2]leaguerules.Decimal{170000, 110000}}, {"McCaffrey", "00-0033280", [2]leaguerules.Decimal{290000, 10000}}} {
		t.Logf("%s: third=%d fourth=%d", tc.name, totals[tc.id][0]/10000, totals[tc.id][1]/10000)
		if totals[tc.id] != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, totals[tc.id], tc.want)
		}
	}
}

func TestOldPBPBodyKeepsFingerprint(t *testing.T) {
	weeks := heldWeeks(t)
	if got := source.LineDigest(weeks[0]); got != "8960fbdbe7fd68f6bd3a2551efc4c899ef55d87c14b2e8a527c84c6a1f204857" {
		t.Fatalf("old held line digest changed: %s", got)
	}
}
