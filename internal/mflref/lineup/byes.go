package lineup

import (
	"cmp"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

type ByePlayer struct {
	Position domain.Position
	Team     string
	Active   bool
}
type ByeShortfall struct {
	Week      int
	Position  domain.Position
	Available int
	Required  int
}

// ByeShortfalls counts active players left at each position against the held starter minimum.
// Unknown team byes are excluded: an incomplete feed must not promise a legal lineup.
func ByeShortfalls(rules Rules, players []ByePlayer, byes map[string]int) []ByeShortfall {
	weeks := map[int]bool{}
	for _, w := range byes {
		weeks[w] = true
	}
	out := []ByeShortfall{}
	for week := range weeks {
		counts := map[domain.Position]int{}
		for _, p := range players {
			bye, known := byes[p.Team]
			if p.Active && known && bye != week {
				counts[p.Position]++
			}
		}
		for _, rule := range rules.Positions {
			available := counts[rule.Position]
			if available < rule.Min {
				out = append(out, ByeShortfall{Week: week, Position: rule.Position, Available: available, Required: rule.Min})
			}
		}
	}
	slices.SortFunc(out, func(a, b ByeShortfall) int {
		if order := cmp.Compare(a.Week, b.Week); order != 0 {
			return order
		}
		return cmp.Compare(a.Position, b.Position)
	})
	return out
}
