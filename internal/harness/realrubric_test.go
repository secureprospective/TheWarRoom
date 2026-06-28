package harness

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/defense"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/kicker"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/offense"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// realRegistry mirrors harness_app.rubrics(): all 10 real B5b rubrics (K is the last, the
// Madden-driven kicker). Keeping it here lets the harness tests prove the close-gate claim
// against the SAME registry the app wires, without a package-main test.
func realRegistry() RubricRegistry {
	return RubricRegistry{
		domain.PosQB: offense.NewQB(),
		domain.PosRB: offense.NewRB(),
		domain.PosWR: offense.NewWR(),
		domain.PosTE: offense.NewTE(),
		domain.PosDT: defense.NewDT(),
		domain.PosDE: defense.NewDE(),
		domain.PosLB: defense.NewLB(),
		domain.PosCB: defense.NewCB(),
		domain.PosS:  defense.NewS(),
		domain.PosK:  kicker.NewK(),
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
// 3F (SL-021 cushion guard) flips PENDING → PASS, and the gatedPending case 3G stays PENDING
// (its PFFAlpha-assertion wiring is deferred). 3D no longer stays PENDING here — realRegistry
// now also registers LB (B5b-LB) alongside DT+WR, so 3D flips; its gate is asserted in
// TestRealLBRegistryFlips3DAnd3J.
func TestRealDTRegistryFlips3F(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	if r := find(t, results, "3F"); r.State != StatePass {
		t.Fatalf("3F should PASS with the real DT rubric, got %s (%s)", r.State, r.Detail)
	}
	if r := find(t, results, "3G"); r.State != StatePending {
		t.Fatalf("3G should stay PENDING (assertion-wiring deferred), got %s", r.State)
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
// case 3K (S-curve boundary clamp) both flip PENDING → PASS and the suite has ZERO failures.
// (3D used to gate on WR as the ±5% control while LB was absent; as of B5b-LB realRegistry
// registers LB+DT+WR, so 3D now flips — its gate moved to TestRealLBRegistryFlips3DAnd3J.)
func TestRealWRRegistryFlips3AAnd3K(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	for _, id := range []string{"3A", "3K"} {
		if r := find(t, results, id); r.State != StatePass {
			t.Fatalf("%s should PASS with the real WR rubric, got %s (%s)", id, r.State, r.Detail)
		}
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

// TestRealTERegistryFlips3M is the B5b-TE close gate: with the TE rubric registered, case 3M
// (SL-019 RAS-modulator lifts breakout with athletic profile) flips PENDING → PASS and the
// suite has ZERO failures. (3E — TE does not gate it; it is DE's canonical instance — now also
// PASSES because realRegistry registers DE alongside TE as of B5b-DE; see
// TestRealDERegistryFlips3E for that gate. 3M's contract is that TE alone flips 3M.)
func TestRealTERegistryFlips3M(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	if r := find(t, results, "3M"); r.State != StatePass {
		t.Fatalf("3M should PASS with the real TE rubric, got %s (%s)", r.State, r.Detail)
	}
	if s := Summarize(results); s.Fail != 0 {
		t.Fatalf("real registry must produce zero FAIL, got %d", s.Fail)
	}
}

// TestRealTERankingDifferentiates is the §5 Higbee/Henry structural finding through Module 1:
// two aging TEs with similar age but different athletic-and-draft profile strength land apart.
// TE Alpha (athletic vet — high RAS + strong static profile) out-scores TE Bravo (non-athletic
// vet — low RAS + weak static profile) on both the breakout component (where SL-019 adds
// athletic credit) and the Layer-4 Combined.
func TestRealTERankingDifferentiates(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), realRegistry())
	byID := map[string]RookieRow{}
	for _, r := range rows {
		byID[r.MFLID] = r
	}
	alpha, bravo := byID["0401"], byID["0402"]
	if alpha.Err != "" || bravo.Err != "" {
		t.Fatalf("TE rows errored: alpha=%q bravo=%q", alpha.Err, bravo.Err)
	}
	if !(alpha.Result.Layer4Output.BreakoutEffective > bravo.Result.Layer4Output.BreakoutEffective) {
		t.Fatalf("TE Alpha breakout %v should exceed Bravo %v (SL-019 + static profile)",
			alpha.Result.Layer4Output.BreakoutEffective, bravo.Result.Layer4Output.BreakoutEffective)
	}
	if !(alpha.Result.Layer4Output.Combined > bravo.Result.Layer4Output.Combined) {
		t.Fatalf("TE Alpha L4 %v should exceed Bravo %v",
			alpha.Result.Layer4Output.Combined, bravo.Result.Layer4Output.Combined)
	}
}

// TestRealDERegistryFlips3E is the B5b-DE close gate: with the DE rubric registered, case 3E
// (SL-019 RAS-modulator lifts breakout — the SECOND SL-019 instance, reusing curve.SL019) flips
// PENDING → PASS, the suite has ZERO failures, and 3G stays PENDING — DE is registered but 3G's
// PFFAlpha-assertion wiring is deferred (it is a gatedPending, never auto-passing), the
// three-state model holding.
func TestRealDERegistryFlips3E(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	if r := find(t, results, "3E"); r.State != StatePass {
		t.Fatalf("3E should PASS with the real DE rubric, got %s (%s)", r.State, r.Detail)
	}
	if r := find(t, results, "3G"); r.State != StatePending {
		t.Fatalf("3G should stay PENDING (assertion-wiring deferred), got %s", r.State)
	}
	if s := Summarize(results); s.Fail != 0 {
		t.Fatalf("real registry must produce zero FAIL, got %d", s.Fail)
	}
}

// TestRealDERankingDifferentiates is the §5 Garrett/Lawrence structural finding through Module 1:
// two DEs with different athletic-and-draft profile strength land apart. DE Alpha (athletic —
// high RAS + early breakout + strong static profile) out-scores DE Bravo (non-athletic — low RAS
// + late breakout + weak static profile) on both the breakout component (where SL-019 adds
// athletic credit) and the Layer-4 Combined.
func TestRealDERankingDifferentiates(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), realRegistry())
	byID := map[string]RookieRow{}
	for _, r := range rows {
		byID[r.MFLID] = r
	}
	alpha, bravo := byID["0501"], byID["0502"]
	if alpha.Err != "" || bravo.Err != "" {
		t.Fatalf("DE rows errored: alpha=%q bravo=%q", alpha.Err, bravo.Err)
	}
	if !(alpha.Result.Layer4Output.BreakoutEffective > bravo.Result.Layer4Output.BreakoutEffective) {
		t.Fatalf("DE Alpha breakout %v should exceed Bravo %v (SL-019 + static profile)",
			alpha.Result.Layer4Output.BreakoutEffective, bravo.Result.Layer4Output.BreakoutEffective)
	}
	if !(alpha.Result.Layer4Output.Combined > bravo.Result.Layer4Output.Combined) {
		t.Fatalf("DE Alpha L4 %v should exceed Bravo %v",
			alpha.Result.Layer4Output.Combined, bravo.Result.Layer4Output.Combined)
	}
}

// TestRealLBRegistryFlips3DAnd3J is the B5b-LB close gate: with the LB rubric registered
// alongside DT+WR (3D) and DE (3J), case 3D (SL-005 film compression cross-position) and case 3J
// (EDGE classification routing) both flip PENDING → PASS, and the suite has ZERO failures. 3G
// stays PENDING (PFFAlpha-assertion wiring still deferred) — the three-state model holding.
func TestRealLBRegistryFlips3DAnd3J(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	for _, id := range []string{"3D", "3J"} {
		if r := find(t, results, id); r.State != StatePass {
			t.Fatalf("%s should PASS with the real LB rubric registered, got %s (%s)", id, r.State, r.Detail)
		}
	}
	if r := find(t, results, "3G"); r.State != StatePending {
		t.Fatalf("3G should stay PENDING (assertion-wiring deferred), got %s", r.State)
	}
	if s := Summarize(results); s.Fail != 0 {
		t.Fatalf("real registry must produce zero FAIL, got %d", s.Fail)
	}
}

// TestRealLBRankingDifferentiates proves the LB rubric separates two LBs through Module 1. LB is
// scheme-dependent and Medium-tier, so the separation is driven more by breakout + College
// Production Share (0.40, the highest weight of any position) than by the residual Medium RAS:
// LB Alpha (early breakout + dominant tackle/sack/TFL share + Power Four) out-scores LB Bravo
// (late breakout + thin share + Group of Five) on both the breakout component and the Combined.
func TestRealLBRankingDifferentiates(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), realRegistry())
	byID := map[string]RookieRow{}
	for _, r := range rows {
		byID[r.MFLID] = r
	}
	alpha, bravo := byID["0801"], byID["0802"]
	if alpha.Err != "" || bravo.Err != "" {
		t.Fatalf("LB rows errored: alpha=%q bravo=%q", alpha.Err, bravo.Err)
	}
	if !(alpha.Result.Layer4Output.BreakoutEffective > bravo.Result.Layer4Output.BreakoutEffective) {
		t.Fatalf("LB Alpha breakout %v should exceed Bravo %v (College Production Share dominant)",
			alpha.Result.Layer4Output.BreakoutEffective, bravo.Result.Layer4Output.BreakoutEffective)
	}
	if !(alpha.Result.Layer4Output.Combined > bravo.Result.Layer4Output.Combined) {
		t.Fatalf("LB Alpha L4 %v should exceed Bravo %v",
			alpha.Result.Layer4Output.Combined, bravo.Result.Layer4Output.Combined)
	}
}

// TestRealEDGERoutingThroughRankings proves the 3J dispatch is LIVE in Module 1, not just in the
// validation suite: a pass-rush-primary LB-tagged defender is scored through the DE rubric (its
// row Position reads "DE"), while an off-ball LB-tagged defender stays "LB". The two share an
// identical profile and differ only in pass-rush snap share, so the routing is the sole cause.
func TestRealEDGERoutingThroughRankings(t *testing.T) {
	base := composition.PlayerSpec{
		Name: "Edge Hybrid", Position: domain.PosLB, BasePoints: 170, Age: 23, RAS: 8.5, HasRAS: true, Salary: 6,
		BreakoutAge: 20, HasBreakoutAge: true, SchoolTier: scouting.SchoolPowerFour, CollegeShare: 0.22, HasCollegeShare: true,
	}
	rusher := base
	rusher.MFLID, rusher.PassRushSnapShare, rusher.HasPassRushSnapShare = "0901", 0.80, true
	offBall := base
	offBall.MFLID, offBall.PassRushSnapShare, offBall.HasPassRushSnapShare = "0902", 0.20, true
	rows := RankRookies(testAssembler(), []composition.PlayerSpec{rusher, offBall}, realRegistry())
	byID := map[string]RookieRow{}
	for _, r := range rows {
		byID[r.MFLID] = r
	}
	if got := byID["0901"].Position; got != "DE" {
		t.Fatalf("pass-rush-primary LB-tagged defender should route to DE rubric, got %q", got)
	}
	if got := byID["0902"].Position; got != "LB" {
		t.Fatalf("off-ball LB-tagged defender should stay LB, got %q", got)
	}
}

// TestRealCBRegistryFlips3I is the B5b-CB close gate: with the real CB rubric registered, case
// 3I (the NGS coverage anchor is present only at CB & S) flips PENDING → PASS — CB alone flips it
// because eval3I asserts the property per-registered-position (CB reports the anchor, the WR
// control does not), so it no longer waits on S. The suite has ZERO failures, and 3G stays
// PENDING (its PFFAlpha-assertion wiring is still deferred) — the three-state model holding.
func TestRealCBRegistryFlips3I(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	if r := find(t, results, "3I"); r.State != StatePass {
		t.Fatalf("3I should PASS with the real CB rubric (NGS anchor introspection), got %s (%s)", r.State, r.Detail)
	}
	if r := find(t, results, "3G"); r.State != StatePending {
		t.Fatalf("3G should stay PENDING (assertion-wiring deferred), got %s", r.State)
	}
	if s := Summarize(results); s.Fail != 0 {
		t.Fatalf("real registry must produce zero FAIL, got %d", s.Fail)
	}
}

// TestRealCBRankingDifferentiates proves the CB rubric separates two CBs through Module 1. Film
// is Data-Parity neutral (coverage weights unset), so the gap is driven by the active High-tier
// RAS component and the breakout composite — where SL-019 (at the reduced 0.30 strength) adds
// athletic credit. CB Alpha (high RAS + early breakout + dominant PD+INT share + Power Four)
// out-scores CB Bravo (low RAS + late breakout + thin share + Group of Five) on both the breakout
// component and the Layer-4 Combined.
func TestRealCBRankingDifferentiates(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), realRegistry())
	byID := map[string]RookieRow{}
	for _, r := range rows {
		byID[r.MFLID] = r
	}
	alpha, bravo := byID["0601"], byID["0602"]
	if alpha.Err != "" || bravo.Err != "" {
		t.Fatalf("CB rows errored: alpha=%q bravo=%q", alpha.Err, bravo.Err)
	}
	if !(alpha.Result.Layer4Output.BreakoutEffective > bravo.Result.Layer4Output.BreakoutEffective) {
		t.Fatalf("CB Alpha breakout %v should exceed Bravo %v (SL-019 + static profile)",
			alpha.Result.Layer4Output.BreakoutEffective, bravo.Result.Layer4Output.BreakoutEffective)
	}
	if !(alpha.Result.Layer4Output.Combined > bravo.Result.Layer4Output.Combined) {
		t.Fatalf("CB Alpha L4 %v should exceed Bravo %v",
			alpha.Result.Layer4Output.Combined, bravo.Result.Layer4Output.Combined)
	}
}

// TestRealSRegistryHolds3I is the B5b-S close gate: with the real S rubric registered ALONGSIDE
// CB, case 3I STAYS PASS — S co-asserts the NGS anchor (eval3I already loops {PosCB, PosS} and
// requires every registered NGS position to report it). There is no PENDING→PASS flip for S (3I
// is already green from CB); the point is S registering does not BREAK 3I. The suite has ZERO
// failures and 3G stays PENDING — the three-state model holding through a second NGS position.
func TestRealSRegistryHolds3I(t *testing.T) {
	results := RunValidationSuite(realRegistry())
	if r := find(t, results, "3I"); r.State != StatePass {
		t.Fatalf("3I should STAY PASS with the real S rubric co-asserting the NGS anchor, got %s (%s)", r.State, r.Detail)
	}
	if r := find(t, results, "3G"); r.State != StatePending {
		t.Fatalf("3G should stay PENDING (assertion-wiring deferred), got %s", r.State)
	}
	if s := Summarize(results); s.Fail != 0 {
		t.Fatalf("real registry with S must produce zero FAIL, got %d", s.Fail)
	}
}

// TestRealSRankingDifferentiates proves the S rubric separates two safeties through Module 1.
// Film is Data-Parity neutral (coverage weights unset), so the gap is driven by the active
// High-tier RAS component and the breakout composite — where SL-019 (at the 0.30 strength) adds
// athletic credit. S Alpha (high RAS + early breakout + dominant INT+Tackle share + Power Four)
// out-scores S Bravo (low RAS + late breakout + thin share + Group of Five) on both the breakout
// component and the Layer-4 Combined.
func TestRealSRankingDifferentiates(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), realRegistry())
	byID := map[string]RookieRow{}
	for _, r := range rows {
		byID[r.MFLID] = r
	}
	alpha, bravo := byID["0901"], byID["0902"]
	if alpha.Err != "" || bravo.Err != "" {
		t.Fatalf("S rows errored: alpha=%q bravo=%q", alpha.Err, bravo.Err)
	}
	if !(alpha.Result.Layer4Output.BreakoutEffective > bravo.Result.Layer4Output.BreakoutEffective) {
		t.Fatalf("S Alpha breakout %v should exceed Bravo %v (SL-019 + static profile)",
			alpha.Result.Layer4Output.BreakoutEffective, bravo.Result.Layer4Output.BreakoutEffective)
	}
	if !(alpha.Result.Layer4Output.Combined > bravo.Result.Layer4Output.Combined) {
		t.Fatalf("S Alpha L4 %v should exceed Bravo %v",
			alpha.Result.Layer4Output.Combined, bravo.Result.Layer4Output.Combined)
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

// TestRealKRankingDifferentiates proves the B5b-K rubric separates two kickers through Module 1
// purely by the active film component (DECISION-011): RAS + breakout are forced to exactly 1.000,
// so Combined == film. K Alpha (strong Madden 0.90 / NFLProduction 0.80) out-scores K Bravo (weak
// 0.30 / 0.25) on the Layer-4 Combined; both keep RAS + breakout at exactly 1.000 (SL-020 partial /
// no breakout framework). Base/age/salary are identical in the fixtures, so the gap is the film alone.
func TestRealKRankingDifferentiates(t *testing.T) {
	rows := RankRookies(testAssembler(), SampleRookies(), realRegistry())
	byID := map[string]RookieRow{}
	for _, r := range rows {
		byID[r.MFLID] = r
	}
	alpha, bravo := byID["1001"], byID["1002"]
	if alpha.Err != "" || bravo.Err != "" {
		t.Fatalf("K rows errored: alpha=%q bravo=%q", alpha.Err, bravo.Err)
	}
	if !(alpha.Result.Layer4Output.Combined > bravo.Result.Layer4Output.Combined) {
		t.Fatalf("K Alpha L4 %v should exceed Bravo %v (active Madden film)",
			alpha.Result.Layer4Output.Combined, bravo.Result.Layer4Output.Combined)
	}
	// Combined is film alone: RAS + breakout forced to exactly 1.000 for both kickers.
	for _, r := range []RookieRow{alpha, bravo} {
		o := r.Result.Layer4Output
		if o.RASEffective != 1.0 || o.BreakoutEffective != 1.0 {
			t.Fatalf("K %s RAS/breakout must be 1.000; got RAS=%v breakout=%v", r.MFLID, o.RASEffective, o.BreakoutEffective)
		}
		if o.Combined != o.FilmEffective {
			t.Fatalf("K %s Combined %v must equal FilmEffective %v", r.MFLID, o.Combined, o.FilmEffective)
		}
	}
}
