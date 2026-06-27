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
		domain.PosRB: offense.NewRB(),
		domain.PosWR: offense.NewWR(),
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
	// 3A now flips PASS — WR (B5b-WR) is registered alongside QB in realRegistry.
	if r := find(t, results, "3A"); r.State != StatePass {
		t.Fatalf("3A should PASS (WR + QB registered), got %s (%s)", r.State, r.Detail)
	}
}

// TestRealRBRegistryFlips3B is the B5b-RB close gate: with the RB rubric registered, case 3B
// (the Herbert pattern — L4 pulls below 1.000 for a thin-profile vet) flips PENDING → PASS,
// the suite has ZERO failures, and 3A stays PENDING (it also gates on WR, not registered).
func TestRealRBRegistryFlips3B(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	if r := find(t, results, "3B"); r.State != StatePass {
		t.Fatalf("3B should PASS with the real RB rubric, got %s (%s)", r.State, r.Detail)
	}
	if s := Summarize(results); s.Fail != 0 {
		t.Fatalf("real registry must produce zero FAIL, got %d", s.Fail)
	}
	if r := find(t, results, "3A"); r.State != StatePass {
		t.Fatalf("3A should PASS (WR registered alongside QB), got %s (%s)", r.State, r.Detail)
	}
}

// TestRealRBRankingDifferentiates proves the RB rubric separates two RBs on its ACTIVE
// components: RB Alpha (elite RAS + early breakout + dominant workload) out-scores RB Bravo
// (thin profile) on the Layer-4 Combined, and Bravo's thin profile pulls his Combined below
// 1.000 (the §7 Herbert pattern, breakout-driven while film is neutral).
func TestRealRBRankingDifferentiates(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), realRegistry())
	var alpha, bravo RookieRow
	for _, r := range rows {
		switch r.MFLID {
		case "0201":
			alpha = r
		case "0202":
			bravo = r
		}
	}
	if alpha.Err != "" || bravo.Err != "" {
		t.Fatalf("RB rows errored: alpha=%q bravo=%q", alpha.Err, bravo.Err)
	}
	if !(alpha.Result.Layer4Output.Combined > bravo.Result.Layer4Output.Combined) {
		t.Fatalf("RB Alpha L4 %v should exceed Bravo %v",
			alpha.Result.Layer4Output.Combined, bravo.Result.Layer4Output.Combined)
	}
	if !(bravo.Result.Layer4Output.Combined < 1.0) {
		t.Fatalf("RB Bravo (thin profile) L4 Combined %v should pull below 1.000", bravo.Result.Layer4Output.Combined)
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

// TestRealWRRegistryFlips3AAnd3K is the B5b-WR close gate: with the WR rubric registered,
// case 3A (the Lockett pattern — L4 stays non-negative for declining elite-draft vets) and
// case 3K (S-curve boundary clamp) both flip PENDING → PASS, the suite has ZERO failures, and
// 3D stays PENDING because its co-gate LB is still unregistered (three-state holding).
func TestRealWRRegistryFlips3AAnd3K(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	for _, id := range []string{"3A", "3K"} {
		if r := find(t, results, id); r.State != StatePass {
			t.Fatalf("%s should PASS with the real WR rubric, got %s (%s)", id, r.State, r.Detail)
		}
	}
	if r := find(t, results, "3D"); r.State != StatePending {
		t.Fatalf("3D should stay PENDING (co-gate LB absent), got %s", r.State)
	}
	if s := Summarize(results); s.Fail != 0 {
		t.Fatalf("real registry must produce zero FAIL, got %d", s.Fail)
	}
}

// TestRealWRRankingDifferentiates proves the WR rubric separates the three sample WRs on its
// ACTIVE components: WR Alpha (elite RAS + true-freshman breakout + dominant usage) out-scores
// both Bravo and Charlie on the Layer-4 Combined. Unlike RB Bravo, WR Charlie's thin-but-real
// draft profile keeps his Combined from collapsing the way the Herbert pattern does — the §5
// Lockett structural point, visible end to end through Module 1.
func TestRealWRRankingDifferentiates(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), realRegistry())
	byID := map[string]RookieRow{}
	for _, r := range rows {
		byID[r.MFLID] = r
	}
	alpha, bravo, charlie := byID["0301"], byID["0302"], byID["0303"]
	if alpha.Err != "" || bravo.Err != "" || charlie.Err != "" {
		t.Fatalf("WR rows errored: alpha=%q bravo=%q charlie=%q", alpha.Err, bravo.Err, charlie.Err)
	}
	if !(alpha.Result.Layer4Output.Combined > bravo.Result.Layer4Output.Combined) {
		t.Fatalf("WR Alpha L4 %v should exceed Bravo %v",
			alpha.Result.Layer4Output.Combined, bravo.Result.Layer4Output.Combined)
	}
	if !(alpha.Result.Layer4Output.Combined > charlie.Result.Layer4Output.Combined) {
		t.Fatalf("WR Alpha L4 %v should exceed Charlie %v",
			alpha.Result.Layer4Output.Combined, charlie.Result.Layer4Output.Combined)
	}
	// WR Bravo's RAS is absent → Data-Parity neutral RAS (exactly 1.000), never forced or zeroed.
	if bravo.Result.Layer4Output.RASEffective != 1.0 {
		t.Fatalf("WR Bravo absent RAS must be Data-Parity neutral 1.000, got %v", bravo.Result.Layer4Output.RASEffective)
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
