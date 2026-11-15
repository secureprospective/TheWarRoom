// Package lineup presents the native lineup judge for a league still run on MFL: the rules come from
// MFL's starters block through the import's own translation, the verdict from nativecheck. It holds
// no lineup rule of its own.
package lineup

import (
	"errors"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// Rules are the league's lineup limits as the app reads them, in the order MFL lists the positions.
type Rules struct {
	Native leaguerules.LineupRules
	// StarterCount is a full lineup's size: MFL's exact starter total, 0 when MFL names none.
	StarterCount int
	order        []domain.Position
}

// ParseRules reads the league's starting-lineup settings; anything MFL holds that the app cannot
// read is an error naming it, never a guess.
func ParseRules(config league.RawConfig) (Rules, error) {
	native, problems := mflsource.LineupRules(config)
	if len(problems) > 0 {
		return Rules{}, errors.New(strings.Join(problems, "; "))
	}
	if native.Positions.State != leaguerules.IsKnown {
		return Rules{}, errors.New("MFL names no starting position limits")
	}
	r := Rules{Native: native}
	if total, ok := native.Groups.Value[leaguerules.Total]; ok && total.Min.State == leaguerules.IsKnown {
		r.StarterCount = total.Min.Value
	}
	for _, p := range config.Starters.Positions {
		if pos, _ := normalize.PositionFromMFL(p.Name); pos != "" {
			r.order = append(r.order, pos)
		}
	}
	return r, nil
}

// Order places a position by MFL's own listing; positions it does not list come last.
func (r Rules) Order(pos domain.Position) int {
	for i, p := range r.order {
		if p == pos {
			return i
		}
	}
	return len(r.order)
}

// minimum is a position's fewest starters, 0 when the limits leave it open.
func (r Rules) minimum(pos domain.Position) int {
	b, ok := r.Native.Positions.Value[native(pos)]
	if !ok || b.Min.State != leaguerules.IsKnown {
		return 0
	}
	return b.Min.Value
}

// native names a held position the way the league's rules do: MFL's kicker is PK there.
func native(pos domain.Position) leaguerules.Position {
	if pos == domain.PosK {
		return leaguerules.PK
	}
	return leaguerules.Position(pos)
}

// Problem kinds: a partial lineup below a minimum is short (MFL accepts it); over exceeds a
// maximum (MFL refuses it); unknown is a starter whose position the rules cannot place.
const (
	KindShort   = string(nativecheck.ProblemShort)
	KindOver    = string(nativecheck.ProblemOver)
	KindUnknown = string(nativecheck.ProblemUnknown)
)

// Problem names what it is about (a position or a group) and says what is wrong.
type Problem struct {
	Subject string `json:"subject"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// Result: Legal means MFL would accept the lineup (nothing over a limit or unplaceable); Full means
// the native judge finds it valid. Problems keep the judge's order and words.
type Result struct {
	Full     bool      `json:"full"`
	Legal    bool      `json:"legal"`
	Problems []Problem `json:"problems"`
}

// Check judges saved starters by position with the native judge; an empty position is a starter
// the directory lacks.
func Check(rules Rules, starters []domain.Position) Result {
	in := make([]nativecheck.Starter, len(starters))
	for i, pos := range starters {
		in[i] = nativecheck.Starter{Position: native(pos)}
	}
	v := nativecheck.JudgeLineup(rules.Native, true, in)
	r := Result{Full: v.Status == nativecheck.LineupValid, Legal: true, Problems: make([]Problem, len(v.Problems))}
	for i, p := range v.Problems {
		r.Problems[i] = Problem{Subject: p.Subject, Kind: string(p.Kind), Message: p.Words}
		if p.Kind != nativecheck.ProblemShort {
			r.Legal = false
		}
	}
	return r
}
