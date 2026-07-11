package freeagency

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// TestMinSalaryFloor pins the §6 minimum-salary table at every band boundary (the fenceposts),
// so a shifted threshold or a transcribed figure is caught. Figures are exact dollars → cents.
func TestMinSalaryFloor(t *testing.T) {
	cases := []struct {
		exp    int
		want   domain.Money
		reason string
	}{
		{-3, dollars(330_000), "negative treated as rookie"},
		{0, dollars(330_000), "rookie (0 years)"},
		{1, dollars(380_000), "1 year"},
		{2, dollars(430_000), "2 years"},
		{3, dollars(480_000), "3 years"},
		{4, dollars(530_000), "4–6 low boundary"},
		{6, dollars(530_000), "4–6 high boundary"},
		{7, dollars(580_000), "7–9 low boundary"},
		{9, dollars(580_000), "7–9 high boundary"},
		{10, dollars(630_000), "10+ low boundary"},
		{25, dollars(630_000), "10+ far"},
	}
	for _, c := range cases {
		if got := MinSalaryFloor(c.exp); got != c.want {
			t.Errorf("MinSalaryFloor(%d) = %s, want %s (%s)", c.exp, got, c.want, c.reason)
		}
	}
}

// TestDollarsIsCents pins the whole-dollar → cents conversion (domain.Money is exact cents), so
// the §6 table is stored at the right magnitude ($330,000 = 33,000,000 cents, not 330,000).
func TestDollarsIsCents(t *testing.T) {
	if got := dollars(330_000); got != domain.Money(33_000_000) {
		t.Fatalf("dollars(330_000) = %d cents, want 33_000_000", int64(got))
	}
}
