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
