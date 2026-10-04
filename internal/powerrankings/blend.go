// Package powerrankings is the M2 blend math, with no I/O.
//
// Each component is z-scored before weighting, so a weight sets each component's real share of
// the spread; min-max would let one super-team or tanked roster compress the field and distort
// the split. Roster value uses median and MAD so one stacked roster cannot move the scale; the
// performance share and the roster's age are bounded and use mean and std. The weighted blend is
// min-max'd to [0,1] for display.
package powerrankings

import (
	"fmt"
	"math"
	"sort"
)

// DefaultRosterWeight is the weight a non-finite one falls back to: 60 roster value, 40 results.
const DefaultRosterWeight = 0.60

// The numbers below were fitted on Legacy NFL's own seasons, 2022–2025, with the model's Now and
// Dyn re-run as of each past week (docs/modules/M2_Power_Ranking_Factors.md).
const (
	// RosterCredibilityWeeks is how many weeks of results weigh as much as the roster. The roster
	// weight after k weeks is m ÷ (m + k): 80% after week 1, 50% after week 4, 29% after week 10.
	// It fitted best between 3 and 6 weeks; today's fixed 60% was right only around week 3.
	RosterCredibilityWeeks = 4.0
	// FranchiseAgeWeight is how much a roster's age counts against its dynasty value, z for z.
	// An older roster scored worse the next season and the one after than its dynasty value
	// alone said (fitted 0.42 and 0.55).
	FranchiseAgeWeight = 0.5
	// GameSpread is the standard deviation, in league points, of a game's margin around the gap
	// between the two teams' expected scores.
	GameSpread = 50.0
)

// AutoRosterWeight is the roster's share of the blend after weeks of results.
func AutoRosterWeight(weeks int) float64 {
	return RosterCredibilityWeeks / (RosterCredibilityWeeks + float64(max(weeks, 0)))
}

// Input is one franchise's raw, already-aggregated inputs; Blend standardizes them.
type Input struct {
	FranchiseID string
	RosterValue float64
	Performance float64 // the franchise's results this season, in [0,1]
	Age         float64 // the counted players' value-weighted age; NaN when none is known
}

// Weights are the blend's: Roster against results (1 − Roster), and how much an older roster
// counts against it, on the same z scale.
type Weights struct {
	Roster float64
	Age    float64
}

// Row is one ranked franchise: the display PowerScore plus the z components (0 = league
// center, +1 = one std above).
type Row struct {
	Rank        int
	FranchiseID string
	PowerScore  float64 // [0,1]
	RosterZ     float64
	MFLPerfZ    float64
	AgeZ        float64
	RosterValue float64
	Performance float64
	Age         float64
}

// Blend returns rows sorted by w·rosterZ + (1−w)·perfZ − age·ageZ, descending, FranchiseID
// breaking ties. w is clamped to [0,1] and the age weight to ≥ 0, never rejected. A
// zero-variance component contributes 0 to everyone, and so does an unknown age. Empty input
// returns an empty, non-nil slice.
func Blend(inputs []Input, wt Weights) ([]Row, error) {
	w := ClampWeight(wt.Roster)
	ageW := wt.Age
	if !(ageW > 0) || math.IsInf(ageW, 0) { // NaN fails the comparison
		ageW = 0
	}

	rows := make([]Row, 0, len(inputs))
	if len(inputs) == 0 {
		return rows, nil
	}

	if err := validate(inputs); err != nil {
		return nil, err
	}

	// Median and MAD·1.4826 both estimate σ for normal data, so the w:(1−w) ratio holds across
	// the robust and classic components.
	rosterCenter, rosterScale := medianMAD(inputs, func(in Input) float64 { return in.RosterValue })
	perfMean, perfStd := meanStd(inputs, func(in Input) float64 { return in.Performance })
	ageMean, ageStd := knownMeanStd(inputs)

	blends := make([]float64, len(inputs))
	blendLo, blendHi := math.Inf(1), math.Inf(-1)
	for i, in := range inputs {
		sz := zscore(in.RosterValue, rosterCenter, rosterScale)
		pz := zscore(in.Performance, perfMean, perfStd)
		az := 0.0
		if !math.IsNaN(in.Age) {
			az = zscore(in.Age, ageMean, ageStd)
		}
		b := w*sz + (1-w)*pz - ageW*az
		blends[i] = b
		blendLo = math.Min(blendLo, b)
		blendHi = math.Max(blendHi, b)
		rows = append(rows, Row{
			FranchiseID: in.FranchiseID,
			RosterZ:     sz,
			MFLPerfZ:    pz,
			AgeZ:        az,
			RosterValue: in.RosterValue,
			Performance: in.Performance,
			Age:         in.Age,
		})
	}

	// A degenerate range maps everyone to 0.5.
	for i := range rows {
		rows[i].PowerScore = minmax(blends[i], blendLo, blendHi)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].PowerScore != rows[j].PowerScore {
			return rows[i].PowerScore > rows[j].PowerScore
		}
		return rows[i].FranchiseID < rows[j].FranchiseID
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows, nil
}

// meanStd uses the population std: the franchises are the whole field, not a sample.
func meanStd(inputs []Input, f func(Input) float64) (mean, std float64) {
	n := float64(len(inputs))
	var sum float64
	for _, in := range inputs {
		sum += f(in)
	}
	mean = sum / n
	var ss float64
	for _, in := range inputs {
		d := f(in) - mean
		ss += d * d
	}
	return mean, math.Sqrt(ss / n)
}

// validate rejects a non-finite roster value and a result outside [0,1]; an unknown age is NaN
// by contract and allowed.
func validate(inputs []Input) error {
	for _, in := range inputs {
		if math.IsNaN(in.RosterValue) || math.IsInf(in.RosterValue, 0) {
			return fmt.Errorf("powerrankings: franchise %s has a non-finite roster value", in.FranchiseID)
		}
		if math.IsNaN(in.Performance) || math.IsInf(in.Performance, 0) || in.Performance < 0 || in.Performance > 1 {
			return fmt.Errorf("powerrankings: franchise %s performance %v out of [0,1]", in.FranchiseID, in.Performance)
		}
	}
	return nil
}

// knownMeanStd is meanStd over the franchises whose age is known.
func knownMeanStd(inputs []Input) (mean, std float64) {
	known := make([]Input, 0, len(inputs))
	for _, in := range inputs {
		if !math.IsNaN(in.Age) {
			known = append(known, in)
		}
	}
	if len(known) == 0 {
		return 0, 0
	}
	return meanStd(known, func(in Input) float64 { return in.Age })
}

// ClampWeight is the roster weight Blend applies: a non-finite one falls back to the default,
// and any other is clamped to [0,1].
func ClampWeight(w float64) float64 {
	if math.IsNaN(w) || math.IsInf(w, 0) {
		w = DefaultRosterWeight
	}
	return math.Max(0, math.Min(1, w))
}

// madScale makes MAD comparable to σ for normal data.
const madScale = 1.4826

// medianMAD returns the median and MAD·madScale (breakdown point 50%). A zero MAD gives scale 0,
// which zscore treats as neutral.
func medianMAD(inputs []Input, f func(Input) float64) (center, scale float64) {
	vals := make([]float64, len(inputs))
	for i, in := range inputs {
		vals[i] = f(in)
	}
	center = median(vals)
	dev := make([]float64, len(vals))
	for i, v := range vals {
		dev[i] = math.Abs(v - center)
	}
	return center, median(dev) * madScale
}

// median sorts a copy; an even-length median averages the middle two.
func median(vs []float64) float64 {
	cp := make([]float64, len(vs))
	copy(cp, vs)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

// zscore returns 0 for a zero scale, so a constant component favors no one.
func zscore(v, center, scale float64) float64 {
	if scale == 0 {
		return 0
	}
	return (v - center) / scale
}

// minmax maps v into [0,1]; a degenerate range gives 0.5.
func minmax(v, lo, hi float64) float64 {
	if hi == lo {
		return 0.5
	}
	return (v - lo) / (hi - lo)
}
