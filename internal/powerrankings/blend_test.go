package powerrankings

import (
	"math"
	"testing"
)

func TestBlendEmpty(t *testing.T) {
	rows, err := Blend(nil, DefaultScoutingWeight)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows == nil {
		t.Fatal("want non-nil empty slice (React nil-guard contract)")
	}
	if len(rows) != 0 {
		t.Fatalf("want 0 rows, got %d", len(rows))
	}
}

func TestBlendZScoreAndOrder(t *testing.T) {
	// A dominates scouting, B dominates all-play — a symmetric field. With two
	// franchises each z-score is ±1, so at w=0.60 A's blend (0.6·1 + 0.4·−1 = 0.2)
	// beats B's (−0.2), and the display min-max maps A→1.0, B→0.0.
	in := []Input{
		{FranchiseID: "0002", ScoutingScore: 100, AllPlayWinPct: 0.0}, // A: scout high, perf low
		{FranchiseID: "0001", ScoutingScore: 0, AllPlayWinPct: 1.0},   // B: scout low, perf high
	}

	rows, err := Blend(in, DefaultScoutingWeight)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0].FranchiseID != "0002" {
		t.Fatalf("scouting-heavy A should rank first at w=0.60, got %s", rows[0].FranchiseID)
	}
	if rows[0].Rank != 1 || rows[1].Rank != 2 {
		t.Fatalf("ranks not dense 1..2: %d,%d", rows[0].Rank, rows[1].Rank)
	}
	// A is scouting-high / perf-low: robust scouting z > 0, all-play z = −1 (2-point
	// mean/std). B mirrors. (Robust z magnitude ≠ 1 — median+MAD, not mean/std.)
	if rows[0].ScoutingZ <= 0 || math.Abs(rows[0].MFLPerfZ+1) > 1e-9 {
		t.Fatalf("A components wrong: scoutZ %v (want >0) perfZ %v (want −1)", rows[0].ScoutingZ, rows[0].MFLPerfZ)
	}
	if rows[1].ScoutingZ >= 0 {
		t.Fatalf("B scouting z should be < 0, got %v", rows[1].ScoutingZ)
	}
	// Display score min-max'd across the blend range → 1.0 / 0.0.
	if math.Abs(rows[0].PowerScore-1.0) > 1e-9 || math.Abs(rows[1].PowerScore-0.0) > 1e-9 {
		t.Fatalf("display score wrong: %v / %v", rows[0].PowerScore, rows[1].PowerScore)
	}
}

func TestBlendWeightClamp(t *testing.T) {
	in := []Input{
		{FranchiseID: "0001", ScoutingScore: 10, AllPlayWinPct: 0.2},
		{FranchiseID: "0002", ScoutingScore: 20, AllPlayWinPct: 0.8},
	}
	// w=1.5 clamps to 1.0 → pure scouting → 0002 (higher scout) leads at display 1.0.
	rows, err := Blend(in, 1.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].FranchiseID != "0002" || math.Abs(rows[0].PowerScore-1.0) > 1e-9 {
		t.Fatalf("w>1 should clamp to pure scouting; got %s @ %v", rows[0].FranchiseID, rows[0].PowerScore)
	}
	// w=-1 clamps to 0 → pure all-play (0002 also higher there).
	rows, err = Blend(in, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].FranchiseID != "0002" || math.Abs(rows[0].PowerScore-1.0) > 1e-9 {
		t.Fatalf("w<0 should clamp to pure all-play; got %s @ %v", rows[0].FranchiseID, rows[0].PowerScore)
	}
}

func TestBlendDegenerateComponent(t *testing.T) {
	// All-play disabled → every AllPlayWinPct == 0 → zero variance → that component's
	// z-score is 0 for all (neutral). Scouting still differentiates.
	in := []Input{
		{FranchiseID: "0001", ScoutingScore: 0, AllPlayWinPct: 0},
		{FranchiseID: "0002", ScoutingScore: 100, AllPlayWinPct: 0},
	}
	rows, err := Blend(in, 0.60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range rows {
		if r.MFLPerfZ != 0 {
			t.Fatalf("zero-variance component should z-score to 0, got %v", r.MFLPerfZ)
		}
	}
	// Scouting still orders: 0002 (z +1) beats 0001 (z −1) → display 1.0 / 0.0.
	if rows[0].FranchiseID != "0002" || math.Abs(rows[0].PowerScore-1.0) > 1e-9 {
		t.Fatalf("degenerate-component ordering wrong: %s @ %v", rows[0].FranchiseID, rows[0].PowerScore)
	}
}

func TestBlendTieBreakDeterministic(t *testing.T) {
	// Identical inputs → zero variance both components → all z 0 → all blends equal →
	// display degenerate 0.5 → FranchiseID ascending decides.
	in := []Input{
		{FranchiseID: "0003", ScoutingScore: 5, AllPlayWinPct: 0.5},
		{FranchiseID: "0001", ScoutingScore: 5, AllPlayWinPct: 0.5},
		{FranchiseID: "0002", ScoutingScore: 5, AllPlayWinPct: 0.5},
	}
	rows, err := Blend(in, 0.60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"0001", "0002", "0003"}
	for i, w := range want {
		if rows[i].FranchiseID != w {
			t.Fatalf("tie-break order[%d] = %s, want %s", i, rows[i].FranchiseID, w)
		}
		if math.Abs(rows[i].PowerScore-0.5) > 1e-9 {
			t.Fatalf("degenerate display score should be 0.5, got %v", rows[i].PowerScore)
		}
	}
}

func TestBlendRejectsNonFinite(t *testing.T) {
	cases := []Input{
		{FranchiseID: "0001", ScoutingScore: math.NaN(), AllPlayWinPct: 0.5},
		{FranchiseID: "0001", ScoutingScore: math.Inf(1), AllPlayWinPct: 0.5},
		{FranchiseID: "0001", ScoutingScore: 1, AllPlayWinPct: 1.5},
		{FranchiseID: "0001", ScoutingScore: 1, AllPlayWinPct: -0.1},
		{FranchiseID: "0001", ScoutingScore: 1, AllPlayWinPct: math.NaN()},
	}
	for i, c := range cases {
		if _, err := Blend([]Input{c}, 0.60); err == nil {
			t.Fatalf("case %d: want error for non-finite/out-of-range input, got nil", i)
		}
	}
}

func TestBlendNonFiniteWeightFallsBack(t *testing.T) {
	in := []Input{
		{FranchiseID: "0001", ScoutingScore: 10, AllPlayWinPct: 0.2},
		{FranchiseID: "0002", ScoutingScore: 20, AllPlayWinPct: 0.8},
	}
	for _, w := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		rows, err := Blend(in, w)
		if err != nil {
			t.Fatalf("w=%v: unexpected error: %v", w, err)
		}
		for _, r := range rows {
			if math.IsNaN(r.PowerScore) || math.IsInf(r.PowerScore, 0) {
				t.Fatalf("w=%v produced non-finite PowerScore %v", w, r.PowerScore)
			}
		}
	}
}

func TestMeanStd(t *testing.T) {
	in := []Input{
		{ScoutingScore: 0}, {ScoutingScore: 100},
	}
	mean, std := meanStd(in, func(i Input) float64 { return i.ScoutingScore })
	if math.Abs(mean-50) > 1e-9 || math.Abs(std-50) > 1e-9 {
		t.Fatalf("meanStd = (%v,%v), want (50,50)", mean, std)
	}
}

func TestMedianMADRobustToOutlier(t *testing.T) {
	// A super-team outlier must NOT drag the center/scale the way mean/std would.
	// Cluster of 5 at 100 + one at 1000. Median stays 100; MAD stays 0-ish for the
	// cluster (they're identical), so the outlier's presence doesn't inflate scale.
	in := []Input{
		{ScoutingScore: 100}, {ScoutingScore: 100}, {ScoutingScore: 100},
		{ScoutingScore: 100}, {ScoutingScore: 100}, {ScoutingScore: 1000},
	}
	center, _ := medianMAD(in, func(i Input) float64 { return i.ScoutingScore })
	if math.Abs(center-100) > 1e-9 {
		t.Fatalf("median center should be 100 (outlier-robust), got %v", center)
	}
	// Mean, by contrast, would be dragged to 250 — proof the robust estimator matters.
	mean, _ := meanStd(in, func(i Input) float64 { return i.ScoutingScore })
	if math.Abs(mean-250) > 1e-9 {
		t.Fatalf("sanity: mean should be 250, got %v", mean)
	}
}

func TestMedianEvenOdd(t *testing.T) {
	if got := median([]float64{3, 1, 2}); got != 2 {
		t.Fatalf("odd median = %v, want 2", got)
	}
	if got := median([]float64{4, 1, 3, 2}); got != 2.5 {
		t.Fatalf("even median = %v, want 2.5", got)
	}
}
