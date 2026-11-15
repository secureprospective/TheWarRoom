package domain

import (
	"testing"
)

func TestParseMoneyMillions_Exact(t *testing.T) {
	tests := []struct {
		raw     string
		want    Money // exact cents
		wantErr bool
	}{
		{"92233720368.54775807", Money(9223372036854775807), false},
		{"92233720368.54775808", 0, true},
		{"92233720369", 0, true},
		{"9223372036854775807", 0, true},
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
