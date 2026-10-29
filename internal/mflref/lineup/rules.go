// Package lineup checks saved starters against the league's own settings without I/O.
package lineup

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// PositionRule is one position's starter bounds as MFL states them ("1" or "2-4").
type PositionRule struct {
	Position domain.Position
	Name     string
	Min      int
	Max      int
}

// Rules are the league's lineup settings: per-position bounds in MFL's order, the cap on
// QB+RB+WR+TE starters, the exact defensive total and the exact starter count of a full lineup.
type Rules struct {
	Positions    []PositionRule
	OffenseCap   int
	DefenseTotal int
	StarterCount int
}

// ParseRules reads MFL's starters block; anything it cannot read is an error naming the field.
func ParseRules(raw league.Starters) (Rules, error) {
	r := Rules{Positions: make([]PositionRule, 0, len(raw.Positions))}
	for _, field := range []struct {
		name, value string
		dest        *int
	}{
		{"count", raw.Count, &r.StarterCount},
		{"iop_starters", raw.IOPStarters, &r.OffenseCap},
		{"idp_starters", raw.IDPStarters, &r.DefenseTotal},
	} {
		n, err := number(field.value)
		if err != nil {
			return Rules{}, fmt.Errorf("lineup rules %s: %w", field.name, err)
		}
		*field.dest = n
	}
	seen := make(map[domain.Position]bool)
	for _, field := range raw.Positions {
		pos, _ := normalize.PositionFromMFL(field.Name)
		if pos == "" || pos == domain.PosFlag {
			return Rules{}, fmt.Errorf("lineup rules %s: unknown position", field.Name)
		}
		if seen[pos] {
			return Rules{}, fmt.Errorf("lineup rules %s: duplicate position", field.Name)
		}
		seen[pos] = true
		minimum, maximum, err := bounds(field.Limit)
		if err != nil {
			return Rules{}, fmt.Errorf("lineup rules %s: %w", field.Name, err)
		}
		r.Positions = append(r.Positions, PositionRule{Position: pos, Name: field.Name, Min: minimum, Max: maximum})
	}
	if len(r.Positions) == 0 {
		return Rules{}, fmt.Errorf("lineup rules positions: none provided")
	}
	return r, nil
}

func number(raw string) (int, error) {
	if raw == "" || strings.IndexFunc(raw, func(c rune) bool { return c < '0' || c > '9' }) >= 0 {
		return 0, fmt.Errorf("invalid nonnegative count %q", raw)
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("count %q: %w", raw, err)
	}
	return n, nil
}

func bounds(raw string) (int, int, error) {
	parts := strings.Split(raw, "-")
	if len(parts) > 2 {
		return 0, 0, fmt.Errorf("invalid range %q", raw)
	}
	minimum, err := number(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("minimum: %w", err)
	}
	maximum := minimum
	if len(parts) == 2 {
		maximum, err = number(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("maximum: %w", err)
		}
	}
	if maximum < minimum {
		return 0, 0, fmt.Errorf("reversed range %q", raw)
	}
	return minimum, maximum, nil
}

// Order leaves positions absent from the rules last, without guessing their meaning.
func (r Rules) Order(pos domain.Position) int {
	for i, p := range r.Positions {
		if p.Position == pos {
			return i
		}
	}
	return len(r.Positions)
}
