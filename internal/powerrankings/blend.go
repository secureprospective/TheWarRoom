// Package powerrankings holds the M2 Power-Ranking blend math and nothing else: it
// is a PURE aggregation package (no I/O, no MFL, no store) so the 60/40 formula is
// unit-testable in isolation and depguard keeps it leaf-level. The app layer fetches
// the two inputs (M1 scouting Adjusted Score per franchise + MFL all-play record),
// aggregates them into []Input, and calls Blend; display passthrough (names, pwr,
// records) is joined by FranchiseID at that layer, never here.
//
// Normalization is Z-SCORE, not min-max (decision 2026-07-18/19, GLM-5.2 math panel):
// each component is standardized BEFORE weighting, so the weight w controls each
// component's true amplitude in the blended spread — min-max equalizes only the
// range, which lets a single dynasty super-team or tanked roster compress the field
// and silently distort the intended 60/40. Scouting uses a ROBUST center/scale
// (median + MAD) so real dynasty outliers can't shift the scale; all-play win% is
// bounded [0,1] and keeps classic mean/std. The weighted z-blend is then min-max'd
// to [0,1] for a clean display score.
package powerrankings

import (
	"fmt"
	"math"
	"sort"
)

// DefaultScoutingWeight is the opinionated base: scouting roster strength is the
// forward-looking 60, MFL in-season all-play result the 40. The UI exposes w as a
// free 0–100% slider (Christopher, 2026-07-18); this is only the default.
const DefaultScoutingWeight = 0.60

// Input is one franchise's already-aggregated blend inputs. ScoutingScore is the
// franchise's aggregated M1 AdjustedScore (sum or top-N, chosen upstream);
// AllPlayWinPct is its luck-adjusted all-play win rate in [0,1]. Both are RAW
// magnitudes — Blend does the standardization, so callers pass real values.
type Input struct {
	FranchiseID   string
	ScoutingScore float64 // aggregated AdjustedScore (raw)
	AllPlayWinPct float64 // all-play win rate, in [0,1]
}

// Row is one ranked franchise: the display PowerScore plus the two STANDARDIZED
// (z-score) components that produced it, so the UI can show each team's talent and
// results standing relative to the field (0 = league average, +1 = one std above).
// PowerScore is min-max'd to [0,1] for display; the z components are unbounded.
type Row struct {
	Rank          int
	FranchiseID   string
	PowerScore    float64 // weighted z-blend, min-max'd to [0,1] for display
	ScoutingZ     float64 // robust-standardized scouting (0 = league median)
	MFLPerfZ      float64 // standardized all-play component (0 = league mean)
	ScoutingScore float64 // raw passthrough (display + audit)
	AllPlayWinPct float64 // raw passthrough
}

// Blend standardizes each component to a z-score across the franchises, combines
// them as w·scoutingZ + (1−w)·perfZ, and min-max's the weighted blend to [0,1] for
// display. Rows come back sorted by the blend descending (FranchiseID ascending as a
// deterministic tiebreak). w is the scouting weight, clamped to [0,1] rather than
// rejected so a stray slider value can never error the view. When a component has
// zero variance (every franchise equal — e.g. all-play disabled → all zero) its
// z-score is 0 for all, a neutral contribution. An empty input yields an empty,
// non-nil slice (the Wails nil→null guard still applies at the React edge).
func Blend(inputs []Input, w float64) ([]Row, error) {
	// A non-finite weight (a malformed IPC call — the React slider can't emit one)
	// would propagate through math.Min/Max as NaN, making every PowerScore NaN and
	// the sort comparator undefined. Fall back to the default rather than error.
	if math.IsNaN(w) || math.IsInf(w, 0) {
		w = DefaultScoutingWeight
	}
	w = math.Max(0, math.Min(1, w))

	rows := make([]Row, 0, len(inputs))
	if len(inputs) == 0 {
		return rows, nil
	}

	// Reject non-finite inputs LOUD — a NaN would silently poison mean/std and every
	// downstream score. AllPlayWinPct is a rate; assert its domain too.
	for _, in := range inputs {
		if math.IsNaN(in.ScoutingScore) || math.IsInf(in.ScoutingScore, 0) {
			return nil, fmt.Errorf("powerrankings: franchise %s has non-finite scouting score", in.FranchiseID)
		}
		if math.IsNaN(in.AllPlayWinPct) || math.IsInf(in.AllPlayWinPct, 0) || in.AllPlayWinPct < 0 || in.AllPlayWinPct > 1 {
			return nil, fmt.Errorf("powerrankings: franchise %s all-play win%% %v out of [0,1]", in.FranchiseID, in.AllPlayWinPct)
		}
	}

	// Scouting uses a ROBUST center/scale (median + MAD·1.4826): dynasty fields have
	// real super-teams and tanked/empty rosters, and mean/std would let one outlier
	// shift the whole scale (50% breakdown point for median+MAD vs 0% for min/max).
	// All-play win% is bounded [0,1] and can't throw an extreme tail, so it keeps the
	// classic mean/std. Both are consistent estimators of the std for normal data, so
	// the w:(1−w) amplitude ratio (the Lead-1 fix) is preserved either way.
	scoutCenter, scoutScale := medianMAD(inputs, func(in Input) float64 { return in.ScoutingScore })
	perfMean, perfStd := meanStd(inputs, func(in Input) float64 { return in.AllPlayWinPct })

	// First pass: standardize + weighted blend (unbounded), tracking the blend range
	// for the display normalization.
	blends := make([]float64, len(inputs))
	blendLo, blendHi := math.Inf(1), math.Inf(-1)
	for i, in := range inputs {
		sz := zscore(in.ScoutingScore, scoutCenter, scoutScale)
		pz := zscore(in.AllPlayWinPct, perfMean, perfStd)
		b := w*sz + (1-w)*pz
		blends[i] = b
		blendLo = math.Min(blendLo, b)
		blendHi = math.Max(blendHi, b)
		rows = append(rows, Row{
			FranchiseID:   in.FranchiseID,
			ScoutingZ:     sz,
			MFLPerfZ:      pz,
			ScoutingScore: in.ScoutingScore,
			AllPlayWinPct: in.AllPlayWinPct,
		})
	}

	// Second pass: min-max the blend to [0,1] for a stable display score. A
	// degenerate range (every franchise identical) maps all to 0.5.
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

// meanStd returns the mean and POPULATION standard deviation of f over inputs
// (len ≥ 1 guaranteed by caller). Population (÷n) not sample (÷n−1): the 32
// franchises are the whole field, not a sample of a larger one.
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

// madScale converts a MAD to a std-consistent scale: for normal data
// median(|x−median|)·1.4826 ≈ σ, so a robust z-score is comparable in magnitude to
// a classic one and the w:(1−w) amplitude ratio holds across both components.
const madScale = 1.4826

// medianMAD returns a ROBUST center (median) and scale (MAD·1.4826) of f over
// inputs (len ≥ 1 guaranteed by caller). Unlike mean/std, a handful of extreme
// franchises (super-teams, empty rosters) cannot shift these — the breakdown point
// is 50%. A zero MAD (over half the field identical) yields scale 0, which zscore
// maps to a neutral 0 for all, exactly as a zero std does.
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

// median returns the median of vs (len ≥ 1). It sorts a COPY so the caller's slice
// is untouched; the even-length median is the mean of the two central values.
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

// zscore standardizes v against the field's center/scale. Zero scale (every
// franchise equal, or over half identical for MAD) yields 0 — a neutral, so a
// constant component neither helps nor hurts any franchise.
func zscore(v, center, scale float64) float64 {
	if scale == 0 {
		return 0
	}
	return (v - center) / scale
}

// minmax maps v into [0,1]. A degenerate range (hi==lo) yields 0.5 — a neutral
// display value when every franchise's blend is identical.
func minmax(v, lo, hi float64) float64 {
	if hi == lo {
		return 0.5
	}
	return (v - lo) / (hi - lo)
}
