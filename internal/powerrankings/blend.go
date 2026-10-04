// Package powerrankings is the M2 blend math, with no I/O.
//
// Each component is z-scored before weighting, so w sets each component's real share of the
// spread; min-max would let one super-team or tanked roster compress the field and distort
// the split. Roster value uses median and MAD so one stacked roster cannot move the scale; the
// performance share is bounded and uses mean and std. The weighted blend is min-max'd to [0,1] for display.
package powerrankings

import (
	"fmt"
	"math"
	"sort"
)

// DefaultRosterWeight is the slider's starting point: 60 roster value, 40 results.
const DefaultRosterWeight = 0.60

// Input is one franchise's raw, already-aggregated inputs; Blend standardizes them.
type Input struct {
	FranchiseID string
	RosterValue float64
	Performance float64 // the franchise's results this season, in [0,1]
}

// Row is one ranked franchise: the display PowerScore plus the two z components (0 = league
// center, +1 = one std above).
type Row struct {
	Rank        int
	FranchiseID string
	PowerScore  float64 // [0,1]
	RosterZ     float64
	MFLPerfZ    float64
	RosterValue float64
	Performance float64
}

// Blend returns rows sorted by w·rosterZ + (1−w)·perfZ, descending, FranchiseID breaking
// ties. w is clamped to [0,1], never rejected. A zero-variance component contributes 0 to
// everyone. Empty input returns an empty, non-nil slice.
func Blend(inputs []Input, w float64) ([]Row, error) {
	// A non-finite weight would make every score NaN; use the default.
	if math.IsNaN(w) || math.IsInf(w, 0) {
		w = DefaultRosterWeight
	}
	w = math.Max(0, math.Min(1, w))

	rows := make([]Row, 0, len(inputs))
	if len(inputs) == 0 {
		return rows, nil
	}

	for _, in := range inputs {
		if math.IsNaN(in.RosterValue) || math.IsInf(in.RosterValue, 0) {
			return nil, fmt.Errorf("powerrankings: franchise %s has a non-finite roster value", in.FranchiseID)
		}
		if math.IsNaN(in.Performance) || math.IsInf(in.Performance, 0) || in.Performance < 0 || in.Performance > 1 {
			return nil, fmt.Errorf("powerrankings: franchise %s performance %v out of [0,1]", in.FranchiseID, in.Performance)
		}
	}

	// Median and MAD·1.4826 both estimate σ for normal data, so the w:(1−w) ratio holds across
	// the robust and classic components.
	rosterCenter, rosterScale := medianMAD(inputs, func(in Input) float64 { return in.RosterValue })
	perfMean, perfStd := meanStd(inputs, func(in Input) float64 { return in.Performance })

	blends := make([]float64, len(inputs))
	blendLo, blendHi := math.Inf(1), math.Inf(-1)
	for i, in := range inputs {
		sz := zscore(in.RosterValue, rosterCenter, rosterScale)
		pz := zscore(in.Performance, perfMean, perfStd)
		b := w*sz + (1-w)*pz
		blends[i] = b
		blendLo = math.Min(blendLo, b)
		blendHi = math.Max(blendHi, b)
		rows = append(rows, Row{
			FranchiseID: in.FranchiseID,
			RosterZ:     sz,
			MFLPerfZ:    pz,
			RosterValue: in.RosterValue,
			Performance: in.Performance,
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
