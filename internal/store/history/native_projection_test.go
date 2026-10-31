package history

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func receivingWeek(gsis, pos string, season, week int, yards int64, kind string) source.PlayerWeek {
	return source.PlayerWeek{GSIS: gsis, Position: pos, Season: season, Week: week, SeasonType: kind,
		Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{
			"receiving_yards": leaguerules.Decimal(yards * 10000)}}}
}

func TestProjectionBook(t *testing.T) {
	rules := leaguerules.Default32Scoring()
	var weeks []source.PlayerWeek
	for w := 1; w <= 9; w++ {
		weeks = append(weeks, receivingWeek("00-0000001", "WR", 2026, w, 100, "REG"))
		weeks = append(weeks, receivingWeek("00-0000002", "WR", 2026, w, 20, "REG"))
	}
	weeks = append(weeks, receivingWeek("00-0000001", "WR", 2026, 10, 900, "REG"))  // the projected week itself
	weeks = append(weeks, receivingWeek("00-0000002", "WR", 2025, 19, 900, "POST")) // playoffs are not counted
	weeks = append(weeks, receivingWeek("00-0000003", "OT", 2026, 1, 50, "REG"))    // not a scoring position
	book := scoreProjectionBook(rules, map[int][]source.PlayerWeek{2026: weeks})
	ids := map[string]string{"0001": "00-0000001", "0002": "00-0000002", "0003": "00-0000003", "0004": ""}
	got := book.project([]string{"0001", "0002", "0003", "0004", "0005"}, ids, 2026, 10)
	byID := map[string]NativeProjection{}
	for _, r := range got {
		byID[r.Player] = r
	}
	tests := []struct {
		name, player string
		available    bool
		words        string
	}{
		{"star receiver projects", "0001", true, "From 9 of his own weeks"},
		{"depth receiver projects", "0002", true, "From 9 of his own weeks"},
		{"non-scoring position has no weeks", "0003", false, "No NFL weeks held"},
		{"blank identity", "0004", false, "No NFL statistics identity"},
		{"unknown player", "0005", false, "No NFL statistics identity"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := byID[tt.player]
			if r.Available != tt.available || !strings.HasPrefix(r.Words, tt.words) {
				t.Fatalf("got %+v", r)
			}
			if r.Available && (r.Floor > r.Median || r.Median > r.Ceiling) {
				t.Fatalf("range out of order: %+v", r)
			}
		})
	}
	if len(got) != 5 || got[0].Player != "0001" {
		t.Fatalf("players out of order: %+v", got)
	}
	if byID["0001"].Median <= byID["0002"].Median {
		t.Fatalf("the 100-yard receiver must project above the 20-yard one: %+v vs %+v", byID["0001"], byID["0002"])
	}
	if byID["0001"].Ceiling >= 90 {
		t.Fatalf("the projected week's own 900 yards leaked into the range: %+v", byID["0001"])
	}
}
