package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Money is exact US cents. All league money is Money, never float64 (OQ-014), because cap
// feeds the output store's exact-equality tiebreak. Float appears only by explicit conversion
// at two edges: the L5 cap ratio and display/IPC.
type Money int64

const centsPerMillion = 100_000_000

// maxMoneyFracDigits: the 8th decimal of a million is one cent; a 9th would be sub-cent and
// is rejected, not truncated.
const maxMoneyFracDigits = 8

// ParseMoneyMillions converts an MFL amount in millions ("7", "1.30", "0.1155") to exact cents
// with string math, no float. Empty is $0; non-numeric, negative or sub-cent is an error.
func ParseMoneyMillions(raw string) (Money, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, nil
	}
	if strings.HasPrefix(s, "-") {
		return 0, fmt.Errorf("domain: money %q is negative", raw)
	}
	intPart, fracPart := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, fracPart = s[:i], s[i+1:]
	}
	if intPart == "" && fracPart == "" {
		return 0, fmt.Errorf("domain: money %q has no digits", raw)
	}
	if !allDigits(intPart) || !allDigits(fracPart) {
		return 0, fmt.Errorf("domain: money %q is not a decimal number", raw)
	}
	if len(fracPart) > maxMoneyFracDigits {
		return 0, fmt.Errorf("domain: money %q has sub-cent precision (>%d fractional digits)", raw, maxMoneyFracDigits)
	}

	millions, err := atoiOrZero(intPart)
	if err != nil {
		return 0, fmt.Errorf("domain: money %q integer part: %w", raw, err)
	}
	// Right-pad to 8 digits so the fraction reads directly as cents.
	frac8 := fracPart + strings.Repeat("0", maxMoneyFracDigits-len(fracPart))
	fracCents, err := atoiOrZero(frac8)
	if err != nil {
		return 0, fmt.Errorf("domain: money %q fractional part: %w", raw, err)
	}
	return Money(millions*centsPerMillion + fracCents), nil
}

// centsPer10k is $10,000 in cents, the league's money granularity (§1): every salary, move
// and charge snaps to it.
const centsPer10k = 1_000_000

// RoundToNearest10k snaps to the nearest $10,000, halves away from zero. It is the one
// rounding rule every op shares, applied after its exact-cents math.
func RoundToNearest10k(m Money) Money {
	c := int64(m)
	half := int64(centsPer10k / 2)
	if c < 0 {
		return Money(-(((-c) + half) / centsPer10k * centsPer10k))
	}
	return Money((c + half) / centsPer10k * centsPer10k)
}

// Millions returns the amount in millions as a float, for the L5 cap ratio and display only.
func (m Money) Millions() float64 { return float64(m) / centsPerMillion }

// Cents returns the integer cents, the storage and IPC form (totals are far below 2^53).
func (m Money) Cents() int64 { return int64(m) }

// String renders "$X,XXX,XXX.XX".
func (m Money) String() string {
	neg := m < 0
	c := m.Cents()
	if neg {
		c = -c
	}
	dollars, cents := c/100, c%100
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	b.WriteByte('$')
	b.WriteString(groupThousands(dollars))
	fmt.Fprintf(&b, ".%02d", cents)
	return b.String()
}

// allDigits reports whether s is empty or all ASCII digits.
func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// atoiOrZero parses an all-digit string to int64, treating "" as 0.
func atoiOrZero(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("domain: parse int %q: %w", s, err)
	}
	return n, nil
}

// groupThousands formats a non-negative integer with comma thousands separators.
func groupThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
