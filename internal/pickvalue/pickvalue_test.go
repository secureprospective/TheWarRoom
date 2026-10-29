package pickvalue

import (
	"math"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
)

// worth is a made-up league's native value by NFL capital: the 1st overall pick is a superstar
// (16.5 PPG), the 25th a solid starter (11.7), the 150th a bench player (2.4), undrafted 0.8.
func worth(overall int) float64 { return 16*math.Exp(-float64(overall)/70) + 0.5 }

const (
	superstar = 16.5
	starter   = 11.7
	bench     = 2.4
)

// league is four NFL classes of 260 picks and 100 undrafted each; one in five drafted players
// past the 100th pick is out of the league (not valued).
func league() []Player {
	var out []Player
	for year := 2023; year <= 2026; year++ {
		for o := 1; o <= 260; o++ {
			out = append(out, Player{Draft: playerrecord.Draft{Status: playerrecord.Drafted, Year: year, Overall: o},
				Value: worth(o), Valued: o <= 100 || o%5 != 0})
		}
		for range 100 {
			out = append(out, Player{Draft: playerrecord.Draft{Status: playerrecord.Undrafted, Year: year}, Value: 0.8, Valued: true})
		}
	}
	// An older class and a far-off one never count.
	out = append(out, Player{Draft: playerrecord.Draft{Status: playerrecord.Drafted, Year: 2015, Overall: 200}, Value: 40, Valued: true})
	return out
}

func fit(t *testing.T) Model {
	t.Helper()
	h, err := History()
	if err != nil {
		t.Fatal(err)
	}
	m, err := Fit(h, league(), 2026, 0.75)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// The scale: an early 1st lands near a solid starter, not a superstar; a late 3rd near a bench
// player. Numbers are logged for the report.
func TestScaleEarlyFirstIsAStarterLateThirdIsBench(t *testing.T) {
	m := fit(t)
	for _, c := range []struct {
		name     string
		round    int
		slot     int
		low, top float64
	}{
		{"early 1st (1.03)", 1, 3, 0.8 * starter, 0.8 * superstar},
		{"mid 1st (1.16)", 1, 16, 0.5 * starter, starter},
		{"late 3rd (3.30)", 3, 30, 0.6 * bench, 1.9 * bench},
		{"late 5th (5.30)", 5, 30, 0, 1.5 * bench},
	} {
		v, err := m.Pick(2026, c.round, c.slot, 32)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%-17s %.2f PPG · NFL band %.0f–%.0f, %.0f%% undrafted, %d players", c.name, v.PPG, v.BandLow, v.BandHigh,
			100*v.Undrafted, v.Players)
		if v.PPG < c.low || v.PPG > c.top {
			t.Errorf("%s = %.2f PPG, want %.2f–%.2f (starter %.1f, superstar %.1f, bench %.1f)", c.name, v.PPG, c.low, c.top,
				starter, superstar, bench)
		}
	}
	first, _ := m.Pick(2026, 1, 3, 32)
	third, _ := m.Pick(2026, 3, 30, 32)
	if !(first.PPG > 2*third.PPG) {
		t.Errorf("an early 1st (%.2f) must be worth well over a late 3rd (%.2f)", first.PPG, third.PPG)
	}
}

// A pick in a later draft is the same pick discounted once per year out; a slot not yet known is
// the round's mean.
func TestYearsOutDiscountAndUnknownSlot(t *testing.T) {
	m := fit(t)
	now, _ := m.Pick(2026, 2, 10, 32)
	later, _ := m.Pick(2028, 2, 10, 32)
	if math.Abs(later.PPG-now.PPG*0.75*0.75) > 1e-9 || later.YearsOut != 2 {
		t.Errorf("two years out = %.4f, want %.4f", later.PPG, now.PPG*0.5625)
	}
	round, _ := m.Pick(2027, 2, 0, 32)
	var sum float64
	for s := 1; s <= 32; s++ {
		v, _ := m.Pick(2027, 2, s, 32)
		sum += v.PPG
	}
	if math.Abs(round.PPG-sum/32) > 1e-9 {
		t.Errorf("round mean = %.4f, want %.4f", round.PPG, sum/32)
	}
	if _, err := m.Pick(2027, 0, 1, 32); err == nil {
		t.Error("round 0 is no pick")
	}
	if _, err := m.Pick(2027, 1, 33, 32); err == nil {
		t.Error("slot 33 of 32 is no pick")
	}
}

// Later slots are worth less: each round's mean below the one before.
func TestRoundsDescend(t *testing.T) {
	m := fit(t)
	prev := math.Inf(1)
	for r := 1; r <= 5; r++ {
		v, _ := m.Pick(2026, r, 0, 32)
		t.Logf("round %d: %.2f PPG", r, v.PPG)
		if v.PPG >= prev {
			t.Errorf("round %d (%.2f) not below round %d (%.2f)", r, v.PPG, r-1, prev)
		}
		prev = v.PPG
	}
}

func TestHistoryIsTheLeaguesDrafts(t *testing.T) {
	h, err := History()
	if err != nil {
		t.Fatal(err)
	}
	years := map[int]bool{}
	for _, p := range h {
		years[p.Year] = true
	}
	if len(h) != 1689 || len(years) != 10 || !years[2017] || !years[2026] {
		t.Errorf("%d picks over %d drafts", len(h), len(years))
	}
	if _, err := Fit(h, nil, 2026, 0.75); err == nil {
		t.Error("no drafted player must refuse the fit")
	}
}

// The whole curve is non-increasing: a later pick is never worth more than an earlier one. v1's
// band means rose at 61 of the 159 steps from 1.01 to 5.32.
func TestCurveIsNonIncreasing(t *testing.T) {
	m := fit(t)
	prev, clamped := math.Inf(1), 0
	for n := 1; n <= curveLength; n++ {
		v := m.Selection(n, 0)
		if v.PPG > prev {
			t.Fatalf("selection %d (%.4f) is worth more than selection %d (%.4f)", n, v.PPG, n-1, prev)
		}
		if v.Clamped {
			clamped++
		}
		prev = v.PPG
	}
	t.Logf("%d of %d selections held down to the curve's running minimum", clamped, curveLength)
	// The same holds slot by slot through Pick, in the league's own rounds.
	prev = math.Inf(1)
	for r := 1; r <= 5; r++ {
		for s := 1; s <= 32; s++ {
			v, err := m.Pick(2026, r, s, 32)
			if err != nil {
				t.Fatal(err)
			}
			if v.PPG > prev {
				t.Fatalf("%d.%02d (%.4f) is worth more than the pick before it (%.4f)", r, s, v.PPG, prev)
			}
			prev = v.PPG
		}
	}
	first, _ := m.Pick(2026, 1, 1, 32)
	last, _ := m.Pick(2026, 1, 32, 32)
	if !(first.PPG > last.PPG) {
		t.Errorf("1.01 (%.2f) must be worth more than 1.32 (%.2f)", first.PPG, last.PPG)
	}
}

// flat is four classes of 260 drafted players all worth the same, keeping only the picks keep
// says: a pick with no record is a player out of the league.
func flat(keep func(overall int) bool) []Player {
	var out []Player
	for year := 2023; year <= 2026; year++ {
		for o := 1; o <= 260; o++ {
			if !keep(o) {
				continue
			}
			out = append(out, Player{Draft: playerrecord.Draft{Status: playerrecord.Drafted, Year: year, Overall: o},
				Value: 10, Valued: true})
		}
	}
	return out
}

// The denominator is every NFL pick the band holds across the classes fitted, not the records
// that survive: a drafted player the league no longer lists counts as zero. Measured on the band
// means themselves, before the curve is held down. v1 divided by the survivors and read about
// 30% high at the top of the first round.
func TestMissingRecordsCountAsZero(t *testing.T) {
	h, err := History()
	if err != nil {
		t.Fatal(err)
	}
	all, err := Fit(h, flat(func(int) bool { return true }), 2026, 0.75)
	if err != nil {
		t.Fatal(err)
	}
	half, err := Fit(h, flat(func(o int) bool { return o%2 == 1 }), 2026, 0.75)
	if err != nil {
		t.Fatal(err)
	}
	// Selections 1, 12, 38 and 102 are 1.01, 1.12, 2.06 and 4.06 of a 32-team draft.
	for _, n := range []int{1, 12, 38, 102} {
		full, thin := all.rawSelection(n), half.rawSelection(n)
		if thin.Expected != full.Expected {
			t.Errorf("selection %d: the denominator moved with the records, %d against %d", n,
				thin.Expected, full.Expected)
		}
		if thin.Players >= full.Players {
			t.Errorf("selection %d: dropping half the records left %d of %d", n, thin.Players, full.Players)
		}
		undrafted := 1 - full.Undrafted
		wantFull := 10 * undrafted * float64(full.Players) / float64(full.Expected)
		wantThin := 10 * undrafted * float64(thin.Players) / float64(thin.Expected)
		t.Logf("selection %d: %.3f with every record (%d of %d), %.3f with half of them (%d of %d)", n,
			full.PPG, full.Players, full.Expected, thin.PPG, thin.Players, thin.Expected)
		if math.Abs(full.PPG-wantFull) > 1e-9 || math.Abs(thin.PPG-wantThin) > 1e-9 {
			t.Errorf("selection %d = %.6f and %.6f, want %.6f and %.6f: the missing must count as zero",
				n, full.PPG, thin.PPG, wantFull, wantThin)
		}
	}
}

func TestPickBandsRetainMissingSeasonReasons(t *testing.T) {
	h, err := History()
	if err != nil {
		t.Fatal(err)
	}
	for _, allBad := range []bool{false, true} {
		t.Run(map[bool]string{false: "reduced", true: "unavailable"}[allBad], func(t *testing.T) {
			players := flat(func(int) bool { return true })
			why := "league points for 2024 are not ready"
			for i := range players {
				players[i].InputStatus = model.InputStatus{ReducedInputs: !allBad, Unavailable: allBad, Reason: why}
				players[i].Valued = !allBad
			}
			m, err := Fit(h, players, 2026, .75)
			if err != nil {
				t.Fatal(err)
			}
			for _, slot := range []int{0, 1} {
				v, err := m.Pick(2027, 1, slot, 32)
				if err != nil || !v.ReducedInputs || v.Unavailable != allBad || !strings.Contains(v.Reason, why) {
					t.Fatal(v, err)
				}
				if !allBad && v.PPG <= 0 {
					t.Fatal("usable values lost", v)
				}
			}
		})
	}
}
