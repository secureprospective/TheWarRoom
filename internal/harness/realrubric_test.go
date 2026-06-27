package harness

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/defense"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/offense"
)

// realRegistry mirrors harness_app.rubrics(): the real QB + DT rubrics plus the identity K
// placeholder. Keeping it here lets the harness tests prove the close-gate claim against
// the SAME registry the app wires, without a package-main test.
func realRegistry() RubricRegistry {
	return RubricRegistry{
		domain.PosQB: offense.NewQB(),
		domain.PosDT: defense.NewDT(),
		domain.PosK:  engine.IdentityLayer4(),
	}
}

// TestRealQBRegistryFlips3C is the B5b-QB close gate: with the real QB rubric and the K
// placeholder registered, case 3C (SL-020) flips PENDING → PASS and the suite has ZERO
// failures (the remaining cases stay PENDING for their later B5b blocks).
func TestRealQBRegistryFlips3C(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	if r := find(t, results, "3C"); r.State != StatePass {
		t.Fatalf("3C should PASS with the real QB rubric + K placeholder, got %s (%s)", r.State, r.Detail)
	}
	if s := Summarize(results); s.Fail != 0 {
		t.Fatalf("real QB registry must produce zero FAIL, got %d", s.Fail)
	}
	// 3A stays PENDING — it also gates on WR (B5b-WR), which is not registered this session.
	if r := find(t, results, "3A"); r.State != StatePending {
		t.Fatalf("3A should stay PENDING (needs WR rubric), got %s", r.State)
	}
}

// TestRealDTRegistryFlips3F is the B5b-DT close gate: with the DT rubric registered, case
// 3F (SL-021 cushion guard) flips PENDING → PASS, and the co-gated cases 3D and 3G stay
// PENDING because their other positions (LB/WR, DE) are not registered this session — the
// three-state model holding correctly.
func TestRealDTRegistryFlips3F(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	if r := find(t, results, "3F"); r.State != StatePass {
		t.Fatalf("3F should PASS with the real DT rubric, got %s (%s)", r.State, r.Detail)
	}
	for _, id := range []string{"3D", "3G"} {
		if r := find(t, results, id); r.State != StatePending {
			t.Fatalf("%s should stay PENDING (co-gated positions absent), got %s", id, r.State)
		}
	}
	if s := Summarize(results); s.Fail != 0 {
		t.Fatalf("real QB+DT registry must produce zero FAIL, got %d", s.Fail)
	}
}

// TestRealDTRankingDifferentiates proves the DT rubric separates two DTs on its ACTIVE
// components: DT Alpha (elite RAS + early breakout + dominant share) must out-score DT Bravo
// on both the RAS and breakout Layer-4 components — DT differentiates on more than identity
// would give, unlike QB where SL-020 zeroes the RAS component.
func TestRealDTRankingDifferentiates(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), realRegistry())
	var alpha, bravo RookieRow
	for _, r := range rows {
		switch r.MFLID {
		case "0701":
			alpha = r
		case "0702":
			bravo = r
		}
	}
	if alpha.Err != "" || bravo.Err != "" {
		t.Fatalf("DT rows errored: alpha=%q bravo=%q", alpha.Err, bravo.Err)
	}
	if !(alpha.Result.Layer4Output.RASEffective > bravo.Result.Layer4Output.RASEffective) {
		t.Fatalf("DT Alpha RASEffective %v should exceed Bravo %v (active RAS component)",
			alpha.Result.Layer4Output.RASEffective, bravo.Result.Layer4Output.RASEffective)
	}
	if !(alpha.Result.Layer4Output.BreakoutEffective > bravo.Result.Layer4Output.BreakoutEffective) {
		t.Fatalf("DT Alpha breakout %v should exceed Bravo %v",
			alpha.Result.Layer4Output.BreakoutEffective, bravo.Result.Layer4Output.BreakoutEffective)
	}
}

// TestRealQBRankingDifferentiates proves Module 1 no longer treats QBs as identical under
// the real rubric: QB Alpha (elite breakout) must out-score QB Bravo (weaker breakout) by
// the L4 breakout component, beyond what BasePoints/age/cap alone would give.
func TestRealQBRankingDifferentiates(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), realRegistry())
	var alpha, bravo RookieRow
	for _, r := range rows {
		switch r.MFLID {
		case "0101":
			alpha = r
		case "0102":
			bravo = r
		}
	}
	if alpha.Err != "" || bravo.Err != "" {
		t.Fatalf("QB rows errored: alpha=%q bravo=%q", alpha.Err, bravo.Err)
	}
	if !(alpha.Result.Layer4Output.BreakoutEffective > bravo.Result.Layer4Output.BreakoutEffective) {
		t.Fatalf("QB Alpha breakout %v should exceed Bravo %v",
			alpha.Result.Layer4Output.BreakoutEffective, bravo.Result.Layer4Output.BreakoutEffective)
	}
	// SL-020 still holds end-to-end: both QBs have RASEffective exactly 1.000.
	if alpha.Result.Layer4Output.RASEffective != 1.0 || bravo.Result.Layer4Output.RASEffective != 1.0 {
		t.Fatalf("QB RASEffective must be 1.000 (SL-020); got alpha=%v bravo=%v",
			alpha.Result.Layer4Output.RASEffective, bravo.Result.Layer4Output.RASEffective)
	}
}
