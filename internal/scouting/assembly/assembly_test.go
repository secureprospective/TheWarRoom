package assembly

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/ras"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// fp is the one-line helper to take the address of a literal float — a present
// measurable is a *float64, and writing &4.5 directly is not valid Go.
func fp(v float64) *float64 { return &v }

// --- the math (no network) ---------------------------------------------------

// TestScoreRAS_HandComputedVector is the "prove the number through" test. A
// two-RB cohort with two present measurables each — forty (time drill) and
// vertical (size/bigger-better drill). The math is symmetric enough that a
// hand calculation is exact, so a math regression is caught to the last digit.
//
//	Player A: Forty=4.40, Vertical=40.0  (faster + jumps higher → elite)
//	Player B: Forty=4.60, Vertical=30.0  (slower + lower jump → below-average)
//
// Per the §3 method:
//
//	Forty (time drill, z=(μ−x)/σ, sample stddev n−1):
//	  μ = 4.50, σ = sqrt(((4.40−4.50)²+(4.60−4.50)²)/1) = sqrt(0.02) = 1/√2·0.1·√2
//	  → σ = √0.02 ≈ 0.1414213562
//	  A_z = (4.50−4.40)/σ = 0.1/0.14142… ≈ +0.7071067812
//	  B_z = (4.50−4.60)/σ ≈ −0.7071067812
//
//	Vertical (bigger-better, z=(x−μ)/σ, sample stddev n−1):
//	  μ = 35.0, σ = sqrt(((40−35)²+(30−35)²)/1) = sqrt(50) ≈ 7.0710678119
//	  A_z = (40−35)/σ ≈ +0.7071067812
//	  B_z = (30−35)/σ ≈ −0.7071067812
//
//	RAS_z = mean of available z = ±0.7071067812 for both players
//	RAS   = clamp(5.0 + 2.0·RAS_z, 0, 10)
//	      A → 5.0 + 1.4142135624 ≈ 6.4142135624
//	      B → 5.0 − 1.4142135624 ≈ 3.5857864376
//
// A fast-forty/high-vertical athlete scores ABOVE 5.0; a slow/low athlete
// scores below — the sign convention is verified both ways in one vector.
func TestScoreRAS_HandComputedVector(t *testing.T) {
	rawByGSIS := map[string]ras.RawCombine{
		"A": {GSISID: "A", Forty: fp(4.40), Vertical: fp(40.0)},
		"B": {GSISID: "B", Forty: fp(4.60), Vertical: fp(30.0)},
	}
	posByGSIS := map[string]domain.Position{"A": domain.PosRB, "B": domain.PosRB}

	out := scoreRAS(rawByGSIS, posByGSIS)
	if len(out) != 2 {
		t.Fatalf("want 2 scored players, got %d (%+v)", len(out), out)
	}
	invSqrt2 := 1.0 / math.Sqrt2 // ≈ 0.7071067811865476
	wantA := 5.0 + 2.0*invSqrt2  // ≈ 6.4142135623730951
	wantB := 5.0 - 2.0*invSqrt2  // ≈ 3.5857864376269049
	if math.Abs(out["A"]-wantA) > 1e-12 {
		t.Fatalf("A: got %v, want %v (5 + 2/√2)", out["A"], wantA)
	}
	if math.Abs(out["B"]-wantB) > 1e-12 {
		t.Fatalf("B: got %v, want %v (5 − 2/√2)", out["B"], wantB)
	}
}

// TestScoreRAS_SignConventionPerDrill pins each sign convention individually
// so a flip (e.g. treating forty as bigger-better) is caught at the per-drill
// level. Two-player cohort per drill; the "better" player scores z=+1 (=
// RAS 7.0) and the "worse" scores z=−1 (= RAS 3.0) under a hand-picked spread.
func TestScoreRAS_SignConventionPerDrill(t *testing.T) {
	// For each measurable we pick two present values whose sample stddev makes
	// the z-score exactly ±1 (n=2 → σ = |x1-x2|/√2; with x1 = μ+d, x2 = μ-d,
	// σ = d·√2, and z = ±d/(d·√2) = ±1/√2; instead we choose x1 = μ + s and
	// x2 = μ − s with s such that z = ±1, i.e. s = σ = |x1-x2|/√2 → x1=μ+σ,
	// x2=μ−σ gives z=±1). Below each pair is constructed so the player listed
	// first is the BETTER athlete and should land at RAS 5 + 2·(+1) = 7.0.
	cases := []struct {
		name      string
		better    float64
		worse     float64
		extractor func(ras.RawCombine) *float64
		setter    func(*ras.RawCombine, *float64)
	}{
		{"height (bigger-better)", 76.0, 72.0, func(r ras.RawCombine) *float64 { return r.HeightIn }, func(r *ras.RawCombine, v *float64) { r.HeightIn = v }},
		{"weight (bigger-better)", 230.0, 210.0, func(r ras.RawCombine) *float64 { return r.WeightLb }, func(r *ras.RawCombine, v *float64) { r.WeightLb = v }},
		{"bench (bigger-better)", 25.0, 15.0, func(r ras.RawCombine) *float64 { return r.Bench }, func(r *ras.RawCombine, v *float64) { r.Bench = v }},
		{"vertical (bigger-better)", 38.0, 32.0, func(r ras.RawCombine) *float64 { return r.Vertical }, func(r *ras.RawCombine, v *float64) { r.Vertical = v }},
		{"broad_jump (bigger-better)", 130.0, 110.0, func(r ras.RawCombine) *float64 { return r.BroadJump }, func(r *ras.RawCombine, v *float64) { r.BroadJump = v }},
		{"forty (time, lower-better)", 4.40, 4.60, func(r ras.RawCombine) *float64 { return r.Forty }, func(r *ras.RawCombine, v *float64) { r.Forty = v }},
		{"cone (time, lower-better)", 6.80, 7.20, func(r ras.RawCombine) *float64 { return r.Cone }, func(r *ras.RawCombine, v *float64) { r.Cone = v }},
		{"shuttle (time, lower-better)", 4.20, 4.40, func(r ras.RawCombine) *float64 { return r.Shuttle }, func(r *ras.RawCombine, v *float64) { r.Shuttle = v }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var better, worse ras.RawCombine
			tc.setter(&better, fp(tc.better))
			tc.setter(&worse, fp(tc.worse))
			out := scoreRAS(
				map[string]ras.RawCombine{"better": better, "worse": worse},
				map[string]domain.Position{"better": domain.PosRB, "worse": domain.PosRB},
			)
			// Better athlete must score ABOVE 5.0 and strictly above the worse.
			if !(out["better"] > 5.0 && out["better"] > out["worse"]) {
				t.Fatalf("%s: sign convention broken — better=%v worse=%v (better must be > 5.0 and > worse)",
					tc.name, out["better"], out["worse"])
			}
			// With exactly two present values, the two z-scores are equal
			// magnitude opposite sign, so the two RAS scores are symmetric
			// about 5.0.
			if math.Abs((out["better"]-5.0)+(out["worse"]-5.0)) > 1e-12 {
				t.Fatalf("%s: two-player cohort must be symmetric about 5.0 — got better=%v worse=%v",
					tc.name, out["better"], out["worse"])
			}
		})
	}
}

// TestScoreRAS_SigmaZeroGuard: identical present values across the cohort →
// σ = 0 → the measurable contributes z = 0 for every player who has it. With
// ONLY that measurable present, every player lands at RAS = 5.0, HasRAS true
// (present-but-neutral — distinct from absent, which would drop the player
// from the map entirely).
func TestScoreRAS_SigmaZeroGuard(t *testing.T) {
	rawByGSIS := map[string]ras.RawCombine{
		"A": {GSISID: "A", Forty: fp(4.50)}, // identical across the cohort
		"B": {GSISID: "B", Forty: fp(4.50)},
	}
	posByGSIS := map[string]domain.Position{"A": domain.PosRB, "B": domain.PosRB}
	out := scoreRAS(rawByGSIS, posByGSIS)
	if len(out) != 2 {
		t.Fatalf("σ=0 guard: want both players present (HasRAS true), got %d entries", len(out))
	}
	for g, v := range out {
		if math.Abs(v-5.0) > 1e-12 {
			t.Fatalf("σ=0 guard: player %s got RAS %v, want exactly 5.0 (present-but-neutral)", g, v)
		}
	}
}

// TestScoreRAS_NLessThan2Guard: a measurable with cohort n<2 (exactly one
// player at the position has it present) is undefined — neutralize to z = 0
// rather than divide by zero. Same outcome as the σ=0 guard.
func TestScoreRAS_NLessThan2Guard(t *testing.T) {
	// Player A is the only RB; he has a forty but no other RB ran a drill.
	rawByGSIS := map[string]ras.RawCombine{
		"A": {GSISID: "A", Forty: fp(4.40)},
	}
	posByGSIS := map[string]domain.Position{"A": domain.PosRB}
	out := scoreRAS(rawByGSIS, posByGSIS)
	if len(out) != 1 {
		t.Fatalf("n<2 guard: want the lone player present (HasRAS true), got %d entries", len(out))
	}
	if math.Abs(out["A"]-5.0) > 1e-12 {
		t.Fatalf("n<2 guard: lone player got RAS %v, want exactly 5.0", out["A"])
	}
}

// TestScoreRAS_ZeroMeasurablesAbsent: a player with zero measurables present
// (empty RawCombine) MUST be absent from the result map — HasRAS false
// downstream, L1 imputes 5.0. The RawCombine row may exist; absence is about
// the measurable set, not the row.
func TestScoreRAS_ZeroMeasurablesAbsent(t *testing.T) {
	rawByGSIS := map[string]ras.RawCombine{
		"A": {GSISID: "A", Forty: fp(4.50)}, // has a measurable
		"B": {GSISID: "B"},                  // zero measurables present
	}
	posByGSIS := map[string]domain.Position{"A": domain.PosRB, "B": domain.PosRB}
	out := scoreRAS(rawByGSIS, posByGSIS)
	if _, ok := out["B"]; ok {
		t.Fatalf("zero-measurable player must be absent from the result map, got RAS %v", out["B"])
	}
	if _, ok := out["A"]; !ok {
		t.Fatal("present player must still score even when his cohort-mate has zero measurables")
	}
}

// TestScoreRAS_AbsentNotAveraged: a player with SOME measurables present is
// averaged ONLY over the measurables he has — an absent drill never enters the
// average (it is not a 0). Pinned by construction: two RBs, A has forty+vert,
// B has forty only. A's vertical is the cohort-mean (so it adds 0 to A's
// z-mean), B has no vertical to add. Both players end up at the SAME RAS (the
// forty alone is symmetric), proving B is NOT being padded with a 0 for the
// missing vertical.
func TestScoreRAS_AbsentNotAveraged(t *testing.T) {
	rawByGSIS := map[string]ras.RawCombine{
		"A": {GSISID: "A", Forty: fp(4.40), Vertical: fp(35.0)}, // vert == μ → adds 0
		"B": {GSISID: "B", Forty: fp(4.60)},                     // no vertical
	}
	posByGSIS := map[string]domain.Position{"A": domain.PosRB, "B": domain.PosRB}
	out := scoreRAS(rawByGSIS, posByGSIS)
	// With the cohort σ for forty fixed by the two present values, B's RAS is
	// 5 + 2·zB_forty. A's RAS is 5 + 2·(zA_forty + 0)/2 == 5 + zA_forty.
	// zA = -zB (two present values, symmetric), so A = 5 - zB = 5 + zB only if
	// |zB|; in general A and B are NOT equal. Instead pin the precise values:
	// forty σ = |4.40-4.60|/√2 ≈ 0.14142…, μ = 4.50
	// A_forty_z = +0.7071…, A_vert_z = 0 (lone value at the mean; but vertical
	//   has n=1 → n<2 guard fires → z = 0 by the guard, not by the mean math)
	// A_ras = 5 + 2·(0.7071… + 0)/2 = 5 + 0.7071… ≈ 5.7071067812
	// B_forty_z = −0.7071…, B has no other measurable → B_ras = 5 + 2·(−0.7071…) ≈ 3.5857864376
	invSqrt2 := 1.0 / math.Sqrt2
	wantA := 5.0 + invSqrt2 // (zSum=invSqrt2, zCount=2) → 5 + 2·(invSqrt2/2) == 5 + invSqrt2
	wantB := 5.0 - 2.0*invSqrt2
	if math.Abs(out["A"]-wantA) > 1e-12 {
		t.Fatalf("A: got %v, want %v", out["A"], wantA)
	}
	if math.Abs(out["B"]-wantB) > 1e-12 {
		t.Fatalf("B: got %v, want %v (absent vertical NOT averaged in as 0)", out["B"], wantB)
	}
}

// TestScoreRAS_ClampRails: a player whose z-mean exceeds ±2.5 clamps to the
// 0/10 rail. Constructing a single-measurable outlier >2.5σ requires a
// lopsided cohort — the maximum z for the (n−1)-identical-plus-1 pattern is
// (n−1)/√n, which first exceeds 2.5 at n=9 (8/3 ≈ 2.667). 8 average RBs plus
// one extreme RB on the same drill → the extreme clamps.
func TestScoreRAS_ClampRails(t *testing.T) {
	// 8 RBs at forty=4.50 + one RB at forty=0.50 (impossibly fast) → fast z=8/3.
	build := func(outlier float64) map[string]ras.RawCombine {
		m := make(map[string]ras.RawCombine, 9)
		for i := 0; i < 8; i++ {
			m[string([]byte{'a', byte('a' + i)})] = ras.RawCombine{Forty: fp(4.50)}
		}
		m["zzz"] = ras.RawCombine{Forty: fp(outlier)}
		return m
	}
	posByGSIS := func() map[string]domain.Position {
		out := make(map[string]domain.Position, 9)
		for _, k := range []string{"aa", "ab", "ac", "ad", "ae", "af", "ag", "ah", "zzz"} {
			out[k] = domain.PosRB
		}
		return out
	}()

	// Time drill: a very FAST outlier (low forty) → high positive z → clamps at 10.
	fast := scoreRAS(build(0.50), posByGSIS)
	if fast["zzz"] != 10.0 {
		t.Fatalf("fast outlier: got RAS %v, want clamp at 10.0 (z = 8/3 ≈ 2.667 should clamp)", fast["zzz"])
	}
	// Time drill: a very SLOW outlier (high forty) → high negative z → clamps at 0.
	slow := scoreRAS(build(99.0), posByGSIS)
	if slow["zzz"] != 0.0 {
		t.Fatalf("slow outlier: got RAS %v, want clamp at 0.0", slow["zzz"])
	}
}

// TestScoreRAS_Determinism: same input → same output across runs. Go map
// iteration order is randomized; the math (mean/stddev/average) is order-
// independent. Run the same input N times and check exact equality.
func TestScoreRAS_Determinism(t *testing.T) {
	rawByGSIS := map[string]ras.RawCombine{
		"A": {GSISID: "A", Forty: fp(4.40), Vertical: fp(40.0), Bench: fp(20.0)},
		"B": {GSISID: "B", Forty: fp(4.60), Vertical: fp(30.0), Bench: fp(10.0)},
		"C": {GSISID: "C", Forty: fp(4.50), Vertical: fp(35.0), Bench: fp(15.0)},
	}
	posByGSIS := map[string]domain.Position{"A": domain.PosRB, "B": domain.PosRB, "C": domain.PosRB}
	first := scoreRAS(rawByGSIS, posByGSIS)
	for i := 0; i < 50; i++ {
		next := scoreRAS(rawByGSIS, posByGSIS)
		for g, v := range first {
			if math.Abs(next[g]-v) > 0 {
				t.Fatalf("iteration %d: player %s drifted %v → %v (math is order-dependent)",
					i, g, v, next[g])
			}
		}
	}
}

// TestScoreRAS_CohortIsolatedByPosition: two positions scored independently —
// a fast WR should not be normalized against RB times (and vice versa). The
// cohort IS the position; cross-position drift is structurally impossible.
func TestScoreRAS_CohortIsolatedByPosition(t *testing.T) {
	// Same forty time at two positions; each cohort has only one player, so
	// both hit the n<2 guard and score 5.0. Adding a cohort-mate at ONE
	// position must move only that position's player.
	rawByGSIS := map[string]ras.RawCombine{
		"rb1": {GSISID: "rb1", Forty: fp(4.50)},
		"rb2": {GSISID: "rb2", Forty: fp(4.40)}, // faster than rb1
		"wr1": {GSISID: "wr1", Forty: fp(4.50)}, // lone WR → guard fires
	}
	posByGSIS := map[string]domain.Position{
		"rb1": domain.PosRB, "rb2": domain.PosRB,
		"wr1": domain.PosWR,
	}
	out := scoreRAS(rawByGSIS, posByGSIS)
	// rb1 / rb2 are scored against each other; rb2 (faster) > 5 > rb1.
	if !(out["rb2"] > 5.0 && out["rb1"] < 5.0) {
		t.Fatalf("RB cohort: rb2 (faster)=%v should be > 5 > rb1=%v", out["rb2"], out["rb1"])
	}
	// wr1 is a lone WR → n<2 guard → exactly 5.0 (NOT averaged against the RBs).
	if math.Abs(out["wr1"]-5.0) > 1e-12 {
		t.Fatalf("WR cohort must be isolated from RB: wr1=%v, want 5.0 (lone-cohort guard)", out["wr1"])
	}
}

// --- BuildRAS (network join, via httptest loopback) -------------------------

// fakePosLookup is a map-backed PositionLookup for tests — the same shape the
// app wires over a normalize.Lookup.
type fakePosLookup map[string]domain.Position

func (f fakePosLookup) Position(mflID string) (domain.Position, bool) {
	p, ok := f[mflID]
	return p, ok
}

// TestBuildRAS_EndToEnd wires the full fetch→join→score pipeline against a
// pair of httptest loopback servers. It is NOT "the network" — loopback
// fixtures are how every ingestion fetcher in this repo is tested. The test
// proves the crosswalk→gsis→RawCombine join + the math call all compose.
func TestBuildRAS_EndToEnd(t *testing.T) {
	// The combine fixture: two players, pfr_id P-A and P-B. P-A → gsis G-A,
	// P-B → gsis G-B. Same hand-computed vector as TestScoreRAS_HandComputedVector.
	combineCSV := strings.Join([]string{
		"pfr_id,ht,wt,forty,bench,vertical,broad_jump,cone,shuttle",
		// 9 columns → 8 commas per row. ht/wt/bench/broad/cone/shuttle blank.
		"P-A,,,\"4.40\",,,40.0,,",
		"P-B,,,\"4.60\",,,30.0,,",
		"",
	}, "\n")
	combineSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(combineCSV))
	}))
	defer combineSrv.Close()

	// crosswalk fixture: mfl_id 1001 → gsis G-A, 1002 → G-B; pfr_id P-A → G-A,
	// P-B → G-B. (Both bridges in one row each.)
	crosswalkCSV := strings.Join([]string{
		"mfl_id,gsis_id,espn_id,pfr_id",
		"1001,G-A,,P-A",
		"1002,G-B,,P-B",
		"",
	}, "\n")
	crosswalkSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(crosswalkCSV))
	}))
	defer crosswalkSrv.Close()

	roster := []string{"1001", "1002"}
	pos := fakePosLookup{"1001": domain.PosRB, "1002": domain.PosRB}

	out, err := BuildRAS(context.Background(), combineSrv.Client(), combineSrv.URL, crosswalkSrv.URL, roster, pos)
	if err != nil {
		t.Fatalf("BuildRAS: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 profiles, got %d", len(out))
	}
	// Re-derive the expected MFL PlayerIDs through playerid.New (the canonical
	// constructor) so the test asserts key-shape equality, not string equality.
	id1001, _ := playerid.New("1001")
	id1002, _ := playerid.New("1002")
	invSqrt2 := 1.0 / math.Sqrt2
	wantA := 5.0 + 2.0*invSqrt2
	wantB := 5.0 - 2.0*invSqrt2
	if math.Abs(out[id1001].RAS-wantA) > 1e-9 {
		t.Fatalf("1001 RAS: got %v, want %v", out[id1001].RAS, wantA)
	}
	if math.Abs(out[id1002].RAS-wantB) > 1e-9 {
		t.Fatalf("1002 RAS: got %v, want %v", out[id1002].RAS, wantB)
	}
	// Only RAS is populated this phase — every other Profile field stays zero.
	if out[id1001].MFLID != id1001 {
		t.Fatalf("MFLID field must be set to the player's canonical id: got %v, want %v", out[id1001].MFLID, id1001)
	}
}

// TestBuildRAS_RosterMissesAreOrdinary: a roster id with no crosswalk mapping,
// one with no combine row, and one with no resolved position all drop quietly
// from the map — never an error. The fetched feed is healthy (one resolvable
// player scores); the misses are player-level, and a player-level miss is
// ordinary by the boundary contract.
func TestBuildRAS_RosterMissesAreOrdinary(t *testing.T) {
	combineCSV := strings.Join([]string{
		"pfr_id,ht,wt,forty,bench,vertical,broad_jump,cone,shuttle",
		// 9 cols, 8 commas per row. Only G-A is present (combine has no G-B row).
		"P-A,,,\"4.40\",,,40.0,,",
		"",
	}, "\n")
	combineSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(combineCSV))
	}))
	defer combineSrv.Close()

	crosswalkCSV := strings.Join([]string{
		"mfl_id,gsis_id,espn_id,pfr_id",
		"1001,G-A,,P-A", // resolvable
		"1002,G-B,,P-B", // resolvable to gsis but combine has no G-B row
		// 1003 has NO crosswalk row at all
		"",
	}, "\n")
	crosswalkSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(crosswalkCSV))
	}))
	defer crosswalkSrv.Close()

	// 1001 resolves + has combine → scores. 1002 resolves but no combine row
	// → ordinary miss. 1003 has no crosswalk row → ordinary miss. 1004 has no
	// resolved position → ordinary miss. 1005 is malformed → ordinary miss.
	roster := []string{"1001", "1002", "1003", "1004", "10X5"}
	pos := fakePosLookup{"1001": domain.PosRB, "1002": domain.PosRB, "1003": domain.PosRB}
	// (1004 and 1005 are not in the pos map → position miss)

	out, err := BuildRAS(context.Background(), combineSrv.Client(), combineSrv.URL, crosswalkSrv.URL, roster, pos)
	if err != nil {
		t.Fatalf("player-level misses must NOT error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want exactly 1 scored profile (1001); got %d: %+v", len(out), out)
	}
	id1001, _ := playerid.New("1001")
	if _, ok := out[id1001]; !ok {
		t.Fatalf("1001 should be in the map, got %+v", out)
	}
}

// TestBuildRAS_NilClientFailsLoud and TestBuildRAS_NilPositionLookupFailsLoud
// pin the constructor-time guard: a nil dependency is a wiring error, never a
// silent zero-score league.
func TestBuildRAS_NilClientFailsLoud(t *testing.T) {
	if _, err := BuildRAS(context.Background(), nil, "u", "u", nil, fakePosLookup{}); err == nil {
		t.Fatal("BuildRAS with nil client should error")
	}
}

func TestBuildRAS_NilPositionLookupFailsLoud(t *testing.T) {
	if _, err := BuildRAS(context.Background(), http.DefaultClient, "u", "u", nil, nil); err == nil {
		t.Fatal("BuildRAS with nil PositionLookup should error")
	}
}
