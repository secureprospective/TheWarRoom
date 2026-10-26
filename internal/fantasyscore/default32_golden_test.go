package fantasyscore

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Real 2024 player-weeks scored by the config study's reference scorer under the default; each row
// cites its nflverse source. The fixture keeps the study's event codes so the mapping is checked here.
type goldenRow struct {
	Source     string               `json:"source"`
	Name       string               `json:"name"`
	Position   leaguerules.Position `json:"position"`
	Codes      map[string]float64   `json:"codes"`
	ChunkPlays []float64            `json:"chunk_plays"`
	Passes40   []float64            `json:"passes_40"`
	FGMade     []float64            `json:"fg_made"`
	FGMissed   []float64            `json:"fg_missed"`
	Expected   float64              `json:"expected"`
}

func goldenCodes() (map[string]leaguerules.StatID, map[string]bool) {
	scored := map[string]leaguerules.StatID{
		"PY": "passing_yards", "#P": "passing_td", "IN": "interception_thrown", "P2": "passing_two_point",
		"RY": "rushing_yards", "RA": "rushing_attempt", "#R": "rushing_td", "R2": "rushing_two_point",
		"CY": "receiving_yards", "CC": "reception", "#C": "receiving_td", "C2": "receiving_two_point",
		"EP": "pat_made", "EM": "pat_missed", "UY": "punt_return_yards", "KY": "kickoff_return_yards",
		"FU": "fumbles_total", "FC": "opponent_fumble_recovery", "FCY": "opponent_fumble_recovery_yards",
		"OFC": "own_fumble_recovery", "FF": "forced_fumble", "IC": "interception_caught",
		"ICY": "interception_return_yards", "PD": "pass_defensed", "BLF": "blocked_fg", "BLP": "blocked_punt",
		"BLE": "blocked_extra_point", "TK": "tackle", "AS": "assist", "SK": "sack", "QH": "qb_hit",
		"TKL": "tackle_for_loss", "SF": "safety", "#KT": "kickoff_return_td", "#UT": "punt_return_td",
		"#DR": "fumble_recovery_tds", "#IR": "interception_return_td", "#BF": "blocked_fg_td", "#BP": "blocked_punt_td",
	}
	// Study measures with no rule in the default; TCH is counted from RA + CC.
	ignored := map[string]bool{"TCH": true, "R10": true, "R20": true, "R40": true, "C10": true, "C20": true,
		"C40": true, "RFD": true, "CFD": true, "PFD": true, "pR20": true, "pR40": true, "pC20": true, "pC40": true}
	return scored, ignored
}

func tenThousandths(x float64) leaguerules.Decimal { return leaguerules.Decimal(math.Round(x * 10000)) }

func distances(xs []float64) []leaguerules.Decimal {
	result := make([]leaguerules.Decimal, len(xs))
	for i, x := range xs {
		result[i] = tenThousandths(x)
	}
	return result
}

func TestDefault32MatchesReferenceOn2024Weeks(t *testing.T) {
	body, err := os.ReadFile("testdata/default32_2024.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct{ Rows []goldenRow }
	if err := json.Unmarshal(body, &fixture); err != nil || len(fixture.Rows) < 20 {
		t.Fatalf("fixture: %v (%d rows)", err, len(fixture.Rows))
	}
	scored, ignored := goldenCodes()
	rules := leaguerules.Default32Scoring()
	for _, row := range fixture.Rows {
		t.Run(row.Source, func(t *testing.T) {
			line := StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"rushing_attempt": 0, "reception": 0},
				Events: map[leaguerules.StatID][]leaguerules.Decimal{"chunk_play": distances(row.ChunkPlays),
					"long_pass": distances(row.Passes40), "fg_made_distance": distances(row.FGMade),
					"fg_missed_distance": distances(row.FGMissed)}}
			for code, value := range row.Codes {
				id, found := scored[code]
				if !found && !ignored[code] {
					t.Fatalf("unmapped study code %s", code)
				}
				if found {
					line.Quantities[id] = tenThousandths(value)
				}
			}
			points, items, err := Score(rules, row.Position, line)
			if err != nil || points.State != leaguerules.IsKnown || points.Value != tenThousandths(row.Expected) {
				t.Fatalf("%s: %+v %v, reference %.2f; items %+v", row.Name, points, err, row.Expected, items)
			}
		})
	}
}
