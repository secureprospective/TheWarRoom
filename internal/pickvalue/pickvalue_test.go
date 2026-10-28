package pickvalue

import (
	"math"
	"testing"

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
