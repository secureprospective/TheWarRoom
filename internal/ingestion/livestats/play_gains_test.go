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

// Week 10 2026 play text: carries after eligibility and direct-snap sentences or two bracketed notes are
// still carries, and a play wiped out by offsetting penalties is not a catch (the box leaves it out).
func TestLongGainsReadAnnouncedCarriesAndSkipOffsettingPlays(t *testing.T) {
	kiner := gainLine("Corey Kiner", "NE", map[leaguerules.StatID]leaguerules.Decimal{"rushing_attempt": 30000, "rushing_yards": 90000})
	williams := gainLine("Isaiah Williams", "NYJ", map[leaguerules.StatID]leaguerules.Decimal{"reception": 10000, "receiving_yards": 130000})
	lines := map[string]*Line{"1": kiner, "2": williams}
	game := livegame.Game{Plays: []livegame.Play{
		{Text: "C.Lomu reported in as eligible.  C.Kiner right tackle to NE 49 for 4 yards (M.Rodriguez; B.Branch)."},
		{Text: "Direct snap to C.Kiner.  C.Kiner up the middle to NE 38 for 2 yards (D.Williams)."},
		{Text: "(1:43) (No Huddle, Shotgun) C.Kiner left end to NE 41 for 3 yards (A.Hooker)."},
		{Text: "(1:43) (No Huddle, Shotgun) G.Smith pass short right to I.Williams to BUF 24 for 45 yards (A.Hooker).\nPenalty on NYJ-M.Taylor, Offensive Pass Interference, offsetting, enforced at NYJ 31."},
		{Text: "(Shotgun) G.Smith pass short left to I.Williams to NYJ 44 for 13 yards (C.Bishop)."},
	}}
	attachLongGains(game, lines)
	if got := kiner.Stats.Events["long_rush"]; !slices.Equal(got, []leaguerules.Decimal{40000, 20000, 30000}) {
		t.Fatalf("announced carries: %v", got)
	}
	if got := williams.Stats.Events["long_reception"]; !slices.Equal(got, []leaguerules.Decimal{130000}) {
		t.Fatalf("offsetting catch: %v", got)
	}
}
