package livestats

import (
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func gainLine(name, team string, q map[leaguerules.StatID]leaguerules.Decimal) *Line {
	return &Line{Name: name, Team: team, Stats: fantasyscore.StatLine{Quantities: q}}
}

// Week 10 2026: Isaiah Likely's 66-yard touchdown earns C20 and C40; the box alone states neither.
func TestLongGainsReconcileWithTheBox(t *testing.T) {
	likely := gainLine("Isaiah Likely", "NYG", map[leaguerules.StatID]leaguerules.Decimal{"reception": 20000, "receiving_yards": 750000})
	etienne := gainLine("Travis Etienne Jr.", "JAX", map[leaguerules.StatID]leaguerules.Decimal{"rushing_attempt": 30000, "rushing_yards": 230000})
	short := gainLine("Chris Olave", "NO", map[leaguerules.StatID]leaguerules.Decimal{"reception": 20000, "receiving_yards": 150000})
	lines := map[string]*Line{"1": likely, "2": etienne, "3": short}
	game := livegame.Game{Plays: []livegame.Play{
		{Text: "(Shotgun) J.McCarthy pass short middle to I.Likely to NYG 46 for 9 yards (S.Styles; J.Reaves)."},
		{Text: "(Shotgun) J.McCarthy pass incomplete short right to I.Likely."},
		{Text: "(Shotgun) J.McCarthy pass deep middle to I.Likely for 66 yards, TOUCHDOWN. D.Zvada extra point is GOOD."},
		{Text: " (Shotgun) T.Etienne left tackle to JAX 30 for 21 yards (A.Key)."},
		{Text: "T.Etienne up the middle to JAX 32 for no gain (A.Key)."},
		{Text: "(No Huddle, Shotgun) T.Etienne right end ran ob at TEN 40 for 2 yards."},
		{Text: "PENALTY on NO, offensive holding. T.Etienne up the middle for 9 yards. NO PLAY."},
		{Text: "D.Carr pass short left to C.Olave to NO 40 for 15 yards (J.Doe)."},
	}}
	attachLongGains(game, lines)
	if got := likely.Stats.Events["long_reception"]; !slices.Equal(got, []leaguerules.Decimal{90000, 660000}) {
		t.Fatalf("receptions: %v", got)
	}
	if got := etienne.Stats.Events["long_rush"]; !slices.Equal(got, []leaguerules.Decimal{210000, 0, 20000}) {
		t.Fatalf("suffixed rusher: %v", got)
	}
	// Two catches in the box, one in the plays: no distance is invented for the missing one.
	if _, ok := short.Stats.Events["long_reception"]; ok {
		t.Fatal("unreconciled catches became distances")
	}
}
