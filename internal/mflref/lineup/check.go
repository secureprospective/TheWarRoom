package lineup

import (
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

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
