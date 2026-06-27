// Package harness is the testing-sandbox logic behind the Wails app: the Module 3
// architectural validation suite (the 12 cases 3A..3L) and the rookie ranking flow.
// It is NOT the production engine; it imports the pure engine + the composition boundary
// and exercises them so each rule can be inspected. It carries no Wails coupling — the
// app layer wraps it.
//
// THREE-STATE MODEL (GLM second-opinion, non-negotiable): a case is PASS (ran, assertion
// held), FAIL (ran, assertion violated — a code error), or PENDING (the per-position
// Layer-4 rubric it tests is not registered yet, OR the Layer4Input does not yet carry
// the sub-signals it specifies). PENDING is an ENCODED SPEC awaiting its B5b block, not a
// failure: it must never be counted as red. As each B5b rubric lands it registers in the
// RubricRegistry and its cases auto-flip from PENDING to PASS/FAIL — the suite IS the
// B5b acceptance gate. Today only 3L runs; the rest are PENDING by design.
package harness

import (
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// CaseState is a validation case's terminal state. PENDING is first-class: it is not a
// failure and is reported separately from FAIL so a dashboard never shows red for an
// unimplemented-but-encoded rule.
type CaseState string

const (
	StatePass    CaseState = "PASS"
	StateFail    CaseState = "FAIL"
	StatePending CaseState = "PENDING"
)

// RubricRegistry maps a position to its real B5b Layer-4 rubric. It is EMPTY today
// (identity L4 only). When a B5b block ships it adds its position here and the cases
// that test it begin evaluating instead of reporting PENDING. The runner never special-
// cases "identity ⇒ skip"; a case is PENDING purely because its rubric is absent.
type RubricRegistry map[domain.Position]engine.Layer4

// CaseResult is one validation case's outcome for the UI board.
type CaseResult struct {
	ID       string    `json:"id"`       // "3A" .. "3L"
	Name     string    `json:"name"`     // human label
	State    CaseState `json:"state"`    // PASS / FAIL / PENDING
	Detail   string    `json:"detail"`   // actual-vs-expected, or why it is pending
	B5bBlock string    `json:"b5bBlock"` // the block that turns this case green
}

// case3 is one encoded validation case: its identity plus an eval closure that, given
// the current rubric registry, returns a state and a detail string. Encoding the cases
// as data (not 12 test functions) is what makes them cheap to write now while the rules
// are crisp and lets the same set run on page load in the app.
type case3 struct {
	id       string
	name     string
	b5bBlock string
	eval     func(reg RubricRegistry) (CaseState, string)
}

// RunValidationSuite evaluates all 12 cases against the given registry and returns their
// results in stable 3A..3L order. Called on page load by the app and by the unit tests.
func RunValidationSuite(reg RubricRegistry) []CaseResult {
	cases := validationCases()
	out := make([]CaseResult, 0, len(cases))
	for _, c := range cases {
		state, detail := c.eval(reg)
		out = append(out, CaseResult{
			ID: c.id, Name: c.name, State: state, Detail: detail, B5bBlock: c.b5bBlock,
		})
	}
	return out
}

// Summary counts the suite by state for the dashboard header. PENDING is reported on its
// own so "0 failures" is true even while most of the suite awaits its B5b block.
type Summary struct {
	Pass    int `json:"pass"`
	Fail    int `json:"fail"`
	Pending int `json:"pending"`
	Total   int `json:"total"`
}

// Summarize tallies a result set.
func Summarize(results []CaseResult) Summary {
	s := Summary{Total: len(results)}
	for _, r := range results {
		switch r.State {
		case StatePass:
			s.Pass++
		case StateFail:
			s.Fail++
		case StatePending:
			s.Pending++
		}
	}
	return s
}
