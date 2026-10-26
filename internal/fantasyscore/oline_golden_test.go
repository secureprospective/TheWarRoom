package fantasyscore

import (
	"bytes"
	"encoding/csv"
	"os"
	"strconv"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Every played 2023–2025 regular-season team-game, weeks 1–13: the config study's frozen O-Line
// components (part 3 components_w1-13.csv, built from nflverse play-by-play and PFR advanced rushing)
// beside its Set B score (oline_scores_setB.csv). The study's nosack column is checked, not supplied:
// the rules derive it from sacks.
func TestDefault32LineUnitMatchesStudyTeamGames(t *testing.T) {
	body, err := os.ReadFile("testdata/oline_setB_2023_2025.csv")
	if err != nil {
		t.Fatal(err)
	}
	table, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil || len(table) < 1000 {
		t.Fatalf("fixture: %v (%d rows)", err, len(table))
	}
	components := map[int]leaguerules.StatID{4: "line_rb_yards_before_contact", 5: "line_rushes_10_plus",
		6: "line_rushing_tds", 7: "line_short_conversions", 8: "line_tackled_for_loss", 9: "line_sacks", 10: "line_qb_hits"}
	rules := leaguerules.Default32Scoring()
	clamped := 0
	for _, row := range table[1:] {
		line := UnitLine{Played: true, Quantities: map[leaguerules.StatID]leaguerules.Decimal{}}
		for column, id := range components {
			line.Quantities[id] = parseTenThousandths(t, row[column])
		}
		points, items, err := ScoreUnit(rules, line)
		if err != nil || points.State != leaguerules.IsKnown || points.Value != parseTenThousandths(t, row[12]) {
			t.Fatalf("%s %s: %+v %v, study %s", row[2], row[3], points, err, row[12])
		}
		noSack := false
		for _, item := range items {
			noSack = noSack || item.Stat == "line_no_sack"
			if item.Stat == "line_clamp" {
				clamped++
			}
		}
		if noSack != (row[11] == "1.0") {
			t.Fatalf("%s %s: no-sack bonus %v, study %s", row[2], row[3], noSack, row[11])
		}
	}
	// Part 3 reports the clamp binding in 3, 6 and 7 games for 2023, 2024 and 2025.
	if clamped != 16 {
		t.Fatalf("clamp bound %d times, the study reports 16", clamped)
	}
}

func parseTenThousandths(t *testing.T, raw string) leaguerules.Decimal {
	t.Helper()
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		t.Fatal(err)
	}
	return tenThousandths(v)
}
