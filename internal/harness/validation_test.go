package harness

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// fakeRubric is a stand-in B5b Layer-4 rubric: it ignores its input and returns a canned
// output, which is all the runner machinery needs to be exercised before real rubrics
// exist.
type fakeRubric struct{ out engine.Layer4Output }

func (f fakeRubric) Apply(engine.Layer4Input) engine.Layer4Output { return f.out }

func find(t *testing.T, results []CaseResult, id string) CaseResult {
	t.Helper()
	for _, r := range results {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("case %s not found in results", id)
	return CaseResult{}
}

// TestSuiteDefaultStateIsHonest: with an empty registry (today), exactly one case (3L)
// passes, none fail, and the rest are PENDING — the suite tells the truth instead of
// showing red for unimplemented rules.
func TestSuiteDefaultStateIsHonest(t *testing.T) {
	results := RunValidationSuite(RubricRegistry{})
	if len(results) != 13 {
		t.Fatalf("expected 13 cases, got %d", len(results))
	}
	s := Summarize(results)
	if s.Fail != 0 {
		t.Fatalf("PENDING must never count as FAIL; got %d failures", s.Fail)
	}
	if s.Pass != 1 {
		t.Fatalf("only 3L should pass today; got %d passes", s.Pass)
	}
	if s.Pending != 12 {
		t.Fatalf("expected 12 pending, got %d", s.Pending)
	}
	if r := find(t, results, "3L"); r.State != StatePass {
		t.Fatalf("3L should PASS now, got %s (%s)", r.State, r.Detail)
	}
	if r := find(t, results, "3C"); r.State != StatePending {
		t.Fatalf("3C should be PENDING with no QB/K rubric, got %s", r.State)
	}
}

// TestCase3CAutoFlipsToPass proves the rubric-pluggable contract: registering a correct
// QB/K rubric flips 3C from PENDING to PASS with no change to the runner.
func TestCase3CAutoFlipsToPass(t *testing.T) {
	good := fakeRubric{out: engine.Layer4Output{
		FilmEffective: 1, RASEffective: 1, BreakoutEffective: 1, Combined: 1,
	}}
	reg := RubricRegistry{domain.PosQB: good, domain.PosK: good}
	r := find(t, RunValidationSuite(reg), "3C")
	if r.State != StatePass {
		t.Fatalf("3C should PASS with SL-020-correct rubric, got %s (%s)", r.State, r.Detail)
	}
}

// TestCase3CFailsOnViolation proves FAIL is real: a QB rubric that lets RAS influence the
// output (RASEffective != 1.000) must fail SL-020, not silently pass or read as pending.
func TestCase3CFailsOnViolation(t *testing.T) {
	bad := fakeRubric{out: engine.Layer4Output{
		FilmEffective: 1, RASEffective: 0.95, BreakoutEffective: 1, Combined: 0.95,
	}}
	ok := fakeRubric{out: engine.Layer4Output{
		FilmEffective: 1, RASEffective: 1, BreakoutEffective: 1, Combined: 1,
	}}
	reg := RubricRegistry{domain.PosQB: bad, domain.PosK: ok}
	r := find(t, RunValidationSuite(reg), "3C")
	if r.State != StateFail {
		t.Fatalf("3C should FAIL when QB RAS influences output, got %s (%s)", r.State, r.Detail)
	}
}

// TestPartialRegistryStillPending: a case requiring two positions stays PENDING until
// BOTH register (3C needs QB and K).
func TestPartialRegistryStillPending(t *testing.T) {
	good := fakeRubric{out: engine.Layer4Output{RASEffective: 1, Combined: 1}}
	reg := RubricRegistry{domain.PosQB: good} // K missing
	r := find(t, RunValidationSuite(reg), "3C")
	if r.State != StatePending {
		t.Fatalf("3C should stay PENDING with K rubric missing, got %s", r.State)
	}
}
