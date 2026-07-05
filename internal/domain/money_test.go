package domain

import (
	"strconv"
	"testing"
)

func TestParseMoneyMillions_Exact(t *testing.T) {
	tests := []struct {
		raw     string
		want    Money // exact cents
		wantErr bool
	}{
		{"7", 700_000_000, false},
		{"1.30", 130_000_000, false},
		{"17.70", 1_770_000_000, false},
		{"0.1155", 11_550_000, false}, // $115,500 — a 4-decimal adjustment value
		{"0.00000001", 1, false},      // 8th decimal of a million = exactly 1 cent
		{"", 0, false},                // legitimate $0
		{"  2.5 ", 250_000_000, false},
		{".5", 50_000_000, false},  // absent integer part
		{"5.", 500_000_000, false}, // absent fractional part
		{"free", 0, true},
		{"-3", 0, true},          // negative money is drift
		{"1.2.3", 0, true},       // malformed
		{"0.000000001", 0, true}, // 9th decimal = sub-cent, cannot be exact
	}
	for _, tt := range tests {
		got, err := ParseMoneyMillions(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Fatalf("ParseMoneyMillions(%q) err=%v, wantErr=%v", tt.raw, err, tt.wantErr)
		}
		if err == nil && got != tt.want {
			t.Fatalf("ParseMoneyMillions(%q) = %d cents, want %d", tt.raw, got, tt.want)
		}
	}
}

// TestMoney_MillionsRoundTrip proves the display edge recovers the millions value
// exactly for realistic $10k-granular salaries (the only float conversion allowed).
func TestMoney_MillionsRoundTrip(t *testing.T) {
	for _, raw := range []string{"7", "1.30", "17.70", "0.53", "125", "0.1155"} {
		m, err := ParseMoneyMillions(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		back, err := ParseMoneyMillions(formatMillions(m.Millions()))
		if err != nil {
			t.Fatalf("reparse %q: %v", raw, err)
		}
		if back != m {
			t.Fatalf("%q round-trip: %d → %f → %d", raw, m, m.Millions(), back)
		}
	}
}

func TestMoney_String(t *testing.T) {
	cases := map[Money]string{
		0:             "$0.00",
		1:             "$0.01",
		700_000_000:   "$7,000,000.00",
		1_770_000_000: "$17,700,000.00",
		11_550_000:    "$115,500.00",
	}
	for m, want := range cases {
		if got := m.String(); got != want {
			t.Errorf("Money(%d).String() = %q, want %q", int64(m), got, want)
		}
	}
}

// formatMillions renders a float millions value at 1-cent (8-decimal) resolution so it
// re-parses exactly — a test helper, not production money math.
func formatMillions(f float64) string {
	return strconv.FormatFloat(f, 'f', 8, 64)
}

func TestRoundToNearest10k(t *testing.T) {
	tests := []struct {
		name string
		in   Money // exact cents
		want Money // snapped cents
	}{
		{"exact 10k multiple stays", 2_000_000, 2_000_000}, // $20,000 → $20,000
		{"round down below half", 20_499_999, 20_000_000},  // $204,999.99 → $200,000
		{"round up at half (away)", 500_000, 1_000_000},    // $5,000 → $10,000
		{"round up above half", 700_000, 1_000_000},        // $7,000 → $10,000
		{"zero", 0, 0}, // $0 → $0
		{"already snapped large", 125_000_000_000, 125_000_000_000}, // $1.25M... cap-scale multiple
		{"just under a 10k step rounds down", 499_999, 0},           // $4,999.99 → $0
		{"negative half rounds away", -500_000, -1_000_000},         // symmetric
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoundToNearest10k(tt.in); got != tt.want {
				t.Fatalf("RoundToNearest10k(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
