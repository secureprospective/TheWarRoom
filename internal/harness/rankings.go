package harness

import (
	"sort"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// RookieRow is one player's full scored output for the Module 1 rankings table. It
// embeds the engine.Result so EVERY intermediate (AgePull, the three Layer-4 effs,
// CapMultiplier, CapTier, AdjustedScore) is inspectable in the UI — the debuggability
// bar. RASImputed surfaces L1 hygiene ("RAS" vs "Imputed") for the table.
type RookieRow struct {
	MFLID      string        `json:"mflID"`
	Name       string        `json:"name"`
	Position   string        `json:"position"`
	Age        float64       `json:"age"`
	RASImputed bool          `json:"rasImputed"`
	Result     engine.Result `json:"result"`
	Err        string        `json:"err"` // non-empty if this player could not be assembled/scored
}

// RankRookies assembles and scores every spec, then sorts by AdjustedScore (descending),
// breaking ties with the L6 tiebreaker. It scores through the registered rubric for each
// player's position when one exists, falling back to identity L4 otherwise — so the same
// call produces the scouting-baseline board today and the real board once B5b lands.
// A player that fails to assemble or score is not dropped silently: it appears with Err
// set so the sandbox shows the failure rather than hiding it.
func RankRookies(a *composition.Assembler, specs []composition.PlayerSpec, reg RubricRegistry) []RookieRow {
	rows := make([]RookieRow, 0, len(specs))
	for _, s := range specs {
		row := RookieRow{MFLID: s.MFLID, Name: s.Name, Position: string(s.Position), Age: s.Age, RASImputed: !s.HasRAS}
		in, sc, cal, err := a.Assemble(s)
		if err != nil {
			row.Err = err.Error()
			rows = append(rows, row)
			continue
		}
		res, err := engine.NewPipeline(reg[s.Position]).Score(in, sc, cal)
		if err != nil {
			row.Err = err.Error()
			rows = append(rows, row)
			continue
		}
		row.Result = res
		rows = append(rows, row)
	}
	sortRows(rows)
	return rows
}

// sortRows orders scored rows by AdjustedScore desc, then the L6 tiebreaker; errored rows
// sink to the bottom (they have a zero Result) so the ranked board stays readable.
func sortRows(rows []RookieRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if (a.Err == "") != (b.Err == "") {
			return a.Err == "" // non-errored first
		}
		if a.Result.AdjustedScore != b.Result.AdjustedScore {
			return a.Result.AdjustedScore > b.Result.AdjustedScore
		}
		return a.Result.Tiebreaker.RanksAbove(b.Result.Tiebreaker)
	})
}
