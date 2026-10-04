package powerrankings

import (
	"math"
	"testing"
)

func TestBlendEmpty(t *testing.T) {
	rows, err := Blend(nil, Weights{Roster: DefaultRosterWeight})
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
	// A dominates roster value, B dominates all-play — a symmetric field. With two
	// franchises each z-score is ±1, so at w=0.60 A's blend (0.6·1 + 0.4·−1 = 0.2)
	// beats B's (−0.2), and the display min-max maps A→1.0, B→0.0.
	in := []Input{
		{FranchiseID: "0002", RosterValue: 100, Performance: 0.0}, // A: roster high, perf low
		{FranchiseID: "0001", RosterValue: 0, Performance: 1.0},   // B: roster low, perf high
	}

	rows, err := Blend(in, Weights{Roster: DefaultRosterWeight})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0].FranchiseID != "0002" {
		t.Fatalf("roster-heavy A should rank first at w=0.60, got %s", rows[0].FranchiseID)
	}
	if rows[0].Rank != 1 || rows[1].Rank != 2 {
		t.Fatalf("ranks not dense 1..2: %d,%d", rows[0].Rank, rows[1].Rank)
	}
	// A is roster-high / perf-low: robust roster z > 0, all-play z = −1 (2-point
	// mean/std). B mirrors. (Robust z magnitude ≠ 1 — median+MAD, not mean/std.)
	if rows[0].RosterZ <= 0 || math.Abs(rows[0].MFLPerfZ+1) > 1e-9 {
		t.Fatalf("A components wrong: rosterZ %v (want >0) perfZ %v (want −1)", rows[0].RosterZ, rows[0].MFLPerfZ)
	}
	if rows[1].RosterZ >= 0 {
		t.Fatalf("B roster z should be < 0, got %v", rows[1].RosterZ)
	}
	// Display score min-max'd across the blend range → 1.0 / 0.0.
	if math.Abs(rows[0].PowerScore-1.0) > 1e-9 || math.Abs(rows[1].PowerScore-0.0) > 1e-9 {
		t.Fatalf("display score wrong: %v / %v", rows[0].PowerScore, rows[1].PowerScore)
	}
}

func TestBlendWeightClamp(t *testing.T) {
	in := []Input{
		{FranchiseID: "0001", RosterValue: 10, Performance: 0.2},
		{FranchiseID: "0002", RosterValue: 20, Performance: 0.8},
	}
	// w=1.5 clamps to 1.0 → pure roster value → 0002 (higher roster) leads at display 1.0.
	rows, err := Blend(in, Weights{Roster: 1.5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].FranchiseID != "0002" || math.Abs(rows[0].PowerScore-1.0) > 1e-9 {
		t.Fatalf("w>1 should clamp to pure roster value; got %s @ %v", rows[0].FranchiseID, rows[0].PowerScore)
	}
	// w=-1 clamps to 0 → pure all-play (0002 also higher there).
	rows, err = Blend(in, Weights{Roster: -1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].FranchiseID != "0002" || math.Abs(rows[0].PowerScore-1.0) > 1e-9 {
		t.Fatalf("w<0 should clamp to pure all-play; got %s @ %v", rows[0].FranchiseID, rows[0].PowerScore)
	}
}

func TestBlendDegenerateComponent(t *testing.T) {
	// No results yet → every Performance == 0 → zero variance → that component's
	// z-score is 0 for all (neutral). Roster value still differentiates.
	in := []Input{
		{FranchiseID: "0001", RosterValue: 0, Performance: 0},
		{FranchiseID: "0002", RosterValue: 100, Performance: 0},
	}
	rows, err := Blend(in, Weights{Roster: 0.60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range rows {
		if r.MFLPerfZ != 0 {
			t.Fatalf("zero-variance component should z-score to 0, got %v", r.MFLPerfZ)
		}
	}
	// Roster value still orders: 0002 (z +1) beats 0001 (z −1) → display 1.0 / 0.0.
	if rows[0].FranchiseID != "0002" || math.Abs(rows[0].PowerScore-1.0) > 1e-9 {
		t.Fatalf("degenerate-component ordering wrong: %s @ %v", rows[0].FranchiseID, rows[0].PowerScore)
	}
}

func TestBlendTieBreakDeterministic(t *testing.T) {
	// Identical inputs → zero variance both components → all z 0 → all blends equal →
	// display degenerate 0.5 → FranchiseID ascending decides.
	in := []Input{
		{FranchiseID: "0003", RosterValue: 5, Performance: 0.5},
		{FranchiseID: "0001", RosterValue: 5, Performance: 0.5},
		{FranchiseID: "0002", RosterValue: 5, Performance: 0.5},
	}
	rows, err := Blend(in, Weights{Roster: 0.60})
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
		{FranchiseID: "0001", RosterValue: math.NaN(), Performance: 0.5},
		{FranchiseID: "0001", RosterValue: math.Inf(1), Performance: 0.5},
		{FranchiseID: "0001", RosterValue: 1, Performance: 1.5},
		{FranchiseID: "0001", RosterValue: 1, Performance: -0.1},
		{FranchiseID: "0001", RosterValue: 1, Performance: math.NaN()},
	}
	for i, c := range cases {
		if _, err := Blend([]Input{c}, Weights{Roster: 0.60}); err == nil {
			t.Fatalf("case %d: want error for non-finite/out-of-range input, got nil", i)
		}
	}
}

func TestBlendNonFiniteWeightFallsBack(t *testing.T) {
	in := []Input{
		{FranchiseID: "0001", RosterValue: 10, Performance: 0.2},
		{FranchiseID: "0002", RosterValue: 20, Performance: 0.8},
	}
	for _, w := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		rows, err := Blend(in, Weights{Roster: w})
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
		{RosterValue: 0}, {RosterValue: 100},
	}
	mean, std := meanStd(in, func(i Input) float64 { return i.RosterValue })
	if math.Abs(mean-50) > 1e-9 || math.Abs(std-50) > 1e-9 {
		t.Fatalf("meanStd = (%v,%v), want (50,50)", mean, std)
	}
}

func TestMedianMADRobustToOutlier(t *testing.T) {
	// A super-team outlier must NOT drag the center/scale the way mean/std would.
	// Cluster of 5 at 100 + one at 1000. Median stays 100; MAD stays 0-ish for the
	// cluster (they're identical), so the outlier's presence doesn't inflate scale.
	in := []Input{
		{RosterValue: 100}, {RosterValue: 100}, {RosterValue: 100},
		{RosterValue: 100}, {RosterValue: 100}, {RosterValue: 1000},
	}
	center, _ := medianMAD(in, func(i Input) float64 { return i.RosterValue })
	if math.Abs(center-100) > 1e-9 {
		t.Fatalf("median center should be 100 (outlier-robust), got %v", center)
	}
	// Mean, by contrast, would be dragged to 250 — proof the robust estimator matters.
	mean, _ := meanStd(in, func(i Input) float64 { return i.RosterValue })
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

func TestBlendAgeCountsAgainstAnOlderRoster(t *testing.T) {
	in := []Input{
		{FranchiseID: "0001", RosterValue: 50, Age: 29},
		{FranchiseID: "0002", RosterValue: 50, Age: 25},
		{FranchiseID: "0003", RosterValue: 50, Age: math.NaN()},
	}
	rows, err := Blend(in, Weights{Roster: 1, Age: FranchiseAgeWeight})
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].FranchiseID != "0002" || rows[2].FranchiseID != "0001" {
		t.Fatalf("want the young roster first and the old one last, got %s %s %s",
			rows[0].FranchiseID, rows[1].FranchiseID, rows[2].FranchiseID)
	}
	if rows[1].AgeZ != 0 {
		t.Fatalf("an unknown age must count as the league's average, got z %v", rows[1].AgeZ)
	}
	flat, err := Blend(in, Weights{Roster: 1})
	if err != nil {
		t.Fatal(err)
	}
	if flat[0].PowerScore != flat[2].PowerScore {
		t.Fatal("with no age weight, equal rosters must score the same")
	}
}

func TestClampWeight(t *testing.T) {
	for _, c := range []struct{ in, want float64 }{
		{0.5, 0.5}, {-1, 0}, {2, 1}, {math.NaN(), DefaultRosterWeight}, {math.Inf(1), DefaultRosterWeight},
	} {
		if got := ClampWeight(c.in); got != c.want {
			t.Errorf("ClampWeight(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
