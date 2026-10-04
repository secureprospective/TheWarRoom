package l4

import "github.com/secureprospective/TheWarRoom/internal/domain"

// Defaults is the shipped rubric for each position. QB and K do not read RAS, and K has no
// breakout. Positions without the athletic lift have Lift 0; DT is protected by the cushion
// instead.
func Defaults() map[domain.Position]Settings {
	table := offense()
	for pos, s := range defense() {
		table[pos] = s
	}
	return table
}

// components are the film and breakout curves most positions share; DT, LB and K compress film.
func components() (standardFilm, compressedFilm, breakoutCurve Component) {
	return Component{Steepness: 12, Cap: 0.05}, Component{Steepness: 10, Cap: 0.03}, Component{Steepness: 11, Cap: 0.05}
}

func offense() map[domain.Position]Settings {
	standardFilm, compressedFilm, breakoutCurve := components()
	return map[domain.Position]Settings{
		domain.PosQB: {
			Film: standardFilm, Breakout: breakoutCurve,
			Weights: Weights{BreakoutAge: 0.30, SchoolTier: 0.25, CollegeShare: 0.30, AgeTrajectory: 0.15},
			Curves: Curves{
				BreakoutAge:  []Breakpoint{{20, 1.00}, {21, 0.80}, {22, 0.50}, {23, 0.10}},
				CollegeShare: []Breakpoint{{0.35, 0.15}, {0.50, 0.55}, {0.65, 1.00}},
				AgeTrajectory: []Breakpoint{{28, 1.00}, {29, 0.90}, {30, 0.80}, {31, 0.65}, {32, 0.50},
					{33, 0.35}, {34, 0.25}, {35, 0.15}, {36, 0.10}, {37, 0.00}},
			},
		},
		domain.PosRB: {
			Film: standardFilm, RAS: Component{Steepness: 8, Cap: 0.04}, RASWeight: 0.60, Breakout: breakoutCurve,
			Weights: Weights{BreakoutAge: 0.35, SchoolTier: 0.20, CollegeShare: 0.30, AgeTrajectory: 0.15},
			Curves: Curves{
				BreakoutAge:  []Breakpoint{{19.5, 1.00}, {20, 0.80}, {20.5, 0.50}, {21, 0.20}},
				CollegeShare: []Breakpoint{{0.20, 0.15}, {0.30, 0.60}, {0.40, 1.00}},
				AgeTrajectory: []Breakpoint{{21, 1.00}, {22, 0.85}, {23, 0.75}, {24, 0.60}, {25, 0.50},
					{26, 0.30}, {27, 0.15}, {28, 0.05}, {29, 0.00}},
			},
		},
		domain.PosWR: {
			Film: standardFilm, RAS: Component{Steepness: 10, Cap: 0.08}, RASWeight: 1, Breakout: breakoutCurve,
			Weights: Weights{BreakoutAge: 0.40, SchoolTier: 0.25, CollegeShare: 0.20, AgeTrajectory: 0.15},
			Curves: Curves{
				BreakoutAge:   []Breakpoint{{19, 1.00}, {20, 0.75}, {21, 0.40}, {22, 0.10}},
				CollegeShare:  []Breakpoint{{0.15, 0.10}, {0.25, 0.50}, {0.35, 1.00}},
				AgeTrajectory: peakArc(29),
			},
		},
		domain.PosTE: {
			Film: standardFilm, RAS: Component{Steepness: 11, Cap: 0.08}, RASWeight: 1, Breakout: breakoutCurve, Lift: 0.35,
			Weights: Weights{BreakoutAge: 0.35, SchoolTier: 0.20, CollegeShare: 0.30, AgeTrajectory: 0.15},
			Curves: Curves{
				BreakoutAge:   []Breakpoint{{20, 1.00}, {21, 0.80}, {22, 0.50}, {23, 0.15}},
				CollegeShare:  []Breakpoint{{0.08, 0.10}, {0.15, 0.50}, {0.22, 1.00}},
				AgeTrajectory: peakArc(29),
			},
		},
		domain.PosK: {Film: compressedFilm},
	}
}

func defense() map[domain.Position]Settings {
	standardFilm, compressedFilm, breakoutCurve := components()
	return map[domain.Position]Settings{
		domain.PosDT: {
			Film: compressedFilm, RAS: Component{Steepness: 10, Cap: 0.08}, RASWeight: 1, Breakout: breakoutCurve,
			Weights: Weights{BreakoutAge: 0.20, SchoolTier: 0.20, CollegeShare: 0.45, AgeTrajectory: 0.15},
			Curves: Curves{
				BreakoutAge:   []Breakpoint{{20, 1.00}, {21, 0.75}, {22, 0.45}, {23, 0.15}},
				CollegeShare:  []Breakpoint{{0.08, 0.15}, {0.15, 0.55}, {0.22, 1.00}},
				AgeTrajectory: peakArc(30),
			},
			PassRushAlpha: [2]float64{0.50, 0.10},
		},
		domain.PosDE: {
			Film: standardFilm, RAS: Component{Steepness: 10, Cap: 0.08}, RASWeight: 1, Breakout: breakoutCurve, Lift: 0.35,
			Weights: Weights{BreakoutAge: 0.30, SchoolTier: 0.20, CollegeShare: 0.35, AgeTrajectory: 0.15},
			Curves: Curves{
				BreakoutAge:   []Breakpoint{{19.5, 1.00}, {20, 0.80}, {20.5, 0.50}, {21, 0.15}},
				CollegeShare:  []Breakpoint{{0.12, 0.15}, {0.20, 0.55}, {0.28, 1.00}},
				AgeTrajectory: peakArc(30),
			},
			PassRushAlpha: [2]float64{0.15, 0.15},
		},
		domain.PosLB: {
			Film: compressedFilm, RAS: Component{Steepness: 11, Cap: 0.04}, RASWeight: 0.60, Breakout: breakoutCurve,
			Weights: Weights{BreakoutAge: 0.25, SchoolTier: 0.20, CollegeShare: 0.40, AgeTrajectory: 0.15},
			Curves: Curves{
				BreakoutAge:   []Breakpoint{{20, 1.00}, {21, 0.75}, {22, 0.45}, {23, 0.15}},
				CollegeShare:  []Breakpoint{{0.10, 0.15}, {0.18, 0.55}, {0.25, 1.00}},
				AgeTrajectory: peakArc(29),
			},
		},
		domain.PosCB: {
			Film: standardFilm, RAS: Component{Steepness: 11, Cap: 0.08}, RASWeight: 1,
			Breakout: Component{Steepness: 10, Cap: 0.05}, Lift: 0.30, CoverageAnchor: true,
			Weights: Weights{BreakoutAge: 0.20, SchoolTier: 0.25, CollegeShare: 0.40, AgeTrajectory: 0.15},
			Curves: Curves{
				BreakoutAge:   defensiveBackBreakout(),
				CollegeShare:  []Breakpoint{{0.08, 0.15}, {0.16, 0.55}, {0.24, 1.00}},
				AgeTrajectory: peakArc(28),
			},
		},
		domain.PosS: {
			Film: standardFilm, RAS: Component{Steepness: 10, Cap: 0.08}, RASWeight: 1, Breakout: breakoutCurve,
			Lift: 0.30, CoverageAnchor: true,
			Weights: Weights{BreakoutAge: 0.20, SchoolTier: 0.25, CollegeShare: 0.40, AgeTrajectory: 0.15},
			Curves: Curves{
				BreakoutAge:   defensiveBackBreakout(),
				CollegeShare:  []Breakpoint{{0.08, 0.15}, {0.14, 0.55}, {0.20, 1.00}},
				AgeTrajectory: peakArc(28),
			},
		},
	}
}

// peakArc is the age trajectory most positions share: 1 four years before the peak, neutral at
// it, 0 four years after.
func peakArc(peak float64) []Breakpoint {
	ys := []float64{1.00, 0.85, 0.70, 0.55, 0.50, 0.35, 0.20, 0.10, 0.00}
	out := make([]Breakpoint, len(ys))
	for i, y := range ys {
		out[i] = Breakpoint{X: peak - 4 + float64(i), Y: y}
	}
	return out
}

func defensiveBackBreakout() []Breakpoint {
	return []Breakpoint{{19.5, 1.00}, {20.5, 0.75}, {21.5, 0.45}, {22.5, 0.15}}
}

// Knob is one adjustable number in Settings, stored as a param under Key for each position whose
// shipped value is not zero. A zero there means the mechanic is off at that position; turning it
// on is a rubric change, not a setting.
type Knob struct {
	Key         string
	Min, Max    float64
	Description string
	field       func(*Settings) *float64
}

// Field points at the knob's value in s.
func (k Knob) Field(s *Settings) *float64 { return k.field(s) }

// AppliesTo reports whether the knob is a setting at a position with shipped settings s.
func (k Knob) AppliesTo(s Settings) bool { return *k.field(&s) != 0 }

// Knobs lists every adjustable number. The ranges catch typos; they do not judge the value.
func Knobs() []Knob {
	return []Knob{
		{"l4.film.steepness", 0, 50, "Film: S-curve steepness", func(s *Settings) *float64 { return &s.Film.Steepness }},
		{"l4.film.cap", 0, 0.5, "Film: largest lift or cut, as a fraction", func(s *Settings) *float64 { return &s.Film.Cap }},
		{"l4.ras.steepness", 0, 50, "RAS: S-curve steepness", func(s *Settings) *float64 { return &s.RAS.Steepness }},
		{"l4.ras.cap", 0, 0.5, "RAS: largest lift or cut, as a fraction", func(s *Settings) *float64 { return &s.RAS.Cap }},
		{"l4.ras.weight", 0, 1, "RAS: share of the S-curve's effect applied", func(s *Settings) *float64 { return &s.RASWeight }},
		{"l4.breakout.steepness", 0, 50, "Breakout: S-curve steepness", func(s *Settings) *float64 { return &s.Breakout.Steepness }},
		{"l4.breakout.cap", 0, 0.5, "Breakout: largest lift or cut, as a fraction", func(s *Settings) *float64 { return &s.Breakout.Cap }},
		{"l4.breakout.weight.breakout_age", 0, 1, "Breakout: weight of breakout age", func(s *Settings) *float64 { return &s.Weights.BreakoutAge }},
		{"l4.breakout.weight.school_tier", 0, 1, "Breakout: weight of school tier", func(s *Settings) *float64 { return &s.Weights.SchoolTier }},
		{"l4.breakout.weight.college_share", 0, 1, "Breakout: weight of college production share", func(s *Settings) *float64 { return &s.Weights.CollegeShare }},
		{"l4.breakout.weight.age_trajectory", 0, 1, "Breakout: weight of age trajectory", func(s *Settings) *float64 { return &s.Weights.AgeTrajectory }},
		{"l4.breakout.athletic_lift", 0, 1, "Breakout: how far RAS lifts breakout age and age trajectory", func(s *Settings) *float64 { return &s.Lift }},
	}
}
