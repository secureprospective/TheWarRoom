package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// Money is an exact monetary amount in integer CENTS of US dollars. All league money
// — salary, adjusted salary, derived cap usage, dead cap — is Money, never float64 and
// never a JS number for math (OQ-014: exactness-by-construction, since cap flows into
// the output store's exact-equality tiebreak). Float appears ONLY at two edges, and only
// by explicit conversion: the engine's dimensionless L5 cap-ratio (Millions) and
// display/IPC payloads (Millions/String). Money never round-trips through float.
type Money int64

// centsPerMillion converts a value denominated in millions of dollars to cents:
// $1M = 1,000,000 dollars × 100 cents.
const centsPerMillion = 100_000_000

// maxMoneyFracDigits is the most fractional digits (in millions) representable as exact
// cents: the 8th decimal of a million is $0.01 = 1 cent. A 9th would be sub-cent and
// cannot be an exact integer of cents, so it is rejected rather than silently truncated.
const maxMoneyFracDigits = 8

// ParseMoneyMillions converts an MFL money string denominated in MILLIONS of dollars
// ("7", "1.30", "0.1155") into exact cents using integer string-math — no float64 ever
// touches the value. An empty string is a legitimate $0 (e.g. an unsigned slot). A
// non-numeric, negative, or sub-cent-precision value is schema drift and fails loud.
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
	// Right-pad the fraction to 8 digits so it reads directly as cents: the k-th
	// fractional digit of a million is worth 10^(8-k) cents.
	frac8 := fracPart + strings.Repeat("0", maxMoneyFracDigits-len(fracPart))
	fracCents, err := atoiOrZero(frac8)
	if err != nil {
		return 0, fmt.Errorf("domain: money %q fractional part: %w", raw, err)
	}
	return Money(millions*centsPerMillion + fracCents), nil
}

// centsPer10k is $10,000 expressed in cents: 10,000 dollars × 100 cents = 1,000,000 cents.
// It is the league's universal money granularity — every salary, owner-directed move, and
// derived charge snaps to a multiple of it (rulebook §1, Christopher 2026-07-04: one flat
// rule, applied everywhere, never individualized per op).
const centsPer10k = 1_000_000

// RoundToNearest10k snaps a Money amount to the nearest $10,000, half-up (a value exactly
// halfway rounds AWAY from zero). This is the SINGLE rounding implementation the whole
// league shares — every op (tag, extension, restructure, dead cap, retirement, buyout,
// seed) applies it AFTER its exact-cents math, never before and never reversed. Keeping it
// here (one function, DRY / Agent Codex) is what stops each op re-deriving the rule and
// drifting. Flat math: no per-op scope, no salaries-vs-charges split.
func RoundToNearest10k(m Money) Money {
	c := int64(m)
	half := int64(centsPer10k / 2)
	if c < 0 {
		return Money(-(((-c) + half) / centsPer10k * centsPer10k))
	}
	return Money((c + half) / centsPer10k * centsPer10k)
}

// Millions returns the amount as a float64 number of millions of dollars. It is for the
// engine's dimensionless cap-ratio (L5) and for display/IPC edges ONLY — never for money
// math or storage, both of which stay in integer cents.
func (m Money) Millions() float64 { return float64(m) / centsPerMillion }

// Cents returns the raw integer cents — the storage and IPC-safe transport form
// (league totals are ~1e12 cents, well under both int64 and 2^53).
func (m Money) Cents() int64 { return int64(m) }

// String renders the amount as "$X,XXX,XXX.XX" for logs and human-facing display.
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

// allDigits reports whether s is empty or all ASCII digits (empty is a valid absent
// integer/fraction part, handled as zero by the caller).
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
