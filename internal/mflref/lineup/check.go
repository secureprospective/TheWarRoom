package lineup

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// Problem kinds: a partial lineup below a minimum is short (MFL accepts it); over exceeds a
// maximum or cap (MFL refuses it); unknown is a starter whose position the rules cannot place.
const (
	KindShort   = "short"
	KindOver    = "over"
	KindUnknown = "unknown"
)

// Problem names what it is about (a position, a cap, or the rules) and says what is wrong.
type Problem struct {
	Subject string `json:"subject"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// Result: Legal means MFL would accept the lineup; Full means every minimum is met and the
// starter count is exact. Problems are in the rules' position order, then unknowns, then caps.
type Result struct {
	Full     bool      `json:"full"`
	Legal    bool      `json:"legal"`
	Problems []Problem `json:"problems"`
}

// Check judges saved starters by position; an empty position is a starter the directory lacks.
func Check(rules Rules, starters []domain.Position) Result {
	r := Result{Full: true, Legal: true, Problems: []Problem{}}
	counts := make(map[domain.Position]int)
	for _, pos := range starters {
		counts[pos]++
	}
	for _, p := range rules.Positions {
		r.bound(p.Name, counts[p.Position], p.Min, p.Max)
	}
	unknown := []domain.Position{}
	offense, defense := 0, 0
	for pos, n := range counts {
		if rules.Order(pos) == len(rules.Positions) {
			unknown = append(unknown, pos)
		}
		// MFL's IOP/IDP split: these are fixed football facts, not league settings.
		switch pos {
		case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE:
			offense += n
		case domain.PosDT, domain.PosDE, domain.PosLB, domain.PosCB, domain.PosS:
			defense += n
		case domain.PosK, domain.PosFlag:
		}
	}
	slices.Sort(unknown)
	for _, pos := range unknown {
		r.Problems = append(r.Problems, unknownProblem(pos, counts[pos]))
		r.Full, r.Legal = false, false
	}
	// MFL's offense cap excludes PK; the settings page, not the export's misleading label, defines it.
	r.bound("QB/RB/WR/TE", offense, 0, rules.OffenseCap)
	r.bound("Defense", defense, rules.DefenseTotal, rules.DefenseTotal)
	r.bound("Total", len(starters), rules.StarterCount, rules.StarterCount)
	return r
}

func unknownProblem(pos domain.Position, n int) Problem {
	if pos == "" {
		return Problem{Subject: "Position", Kind: KindUnknown,
			Message: fmt.Sprintf("Position unknown: %d starting", n)}
	}
	return Problem{Subject: string(pos), Kind: KindUnknown,
		Message: fmt.Sprintf("%s: %d starting, not a lineup position", pos, n)}
}

func (r *Result) bound(name string, count, minimum, maximum int) {
	kind, message := "", ""
	switch {
	case count < minimum:
		kind = KindShort
		message = fmt.Sprintf("%s: %d starting, needs at least %d", name, count, minimum)
		r.Full = false
	case count > maximum:
		kind = KindOver
		message = fmt.Sprintf("%s: %d starting, at most %d", name, count, maximum)
		r.Full, r.Legal = false, false
	}
	if kind != "" {
		r.Problems = append(r.Problems, Problem{Subject: name, Kind: kind, Message: message})
	}
}
