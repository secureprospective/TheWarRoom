// Package l4 is Layer 4, the scouting multiplier: one routine for every position, driven by that
// position's Settings. Combined = film × RAS × breakout, three capped S-curves, so an absent or
// neutral profile scores exactly 1. The shipped settings are the table in settings.go; the
// adjustable numbers in it are stored as params, and composition reads them back for each run.
// Spec: docs/scoring-engine/Engine_Specification.md and the per-position rubric documents.
package l4

import (
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// Weights are the breakout composite's sub-signal weights. They sum to 1, so a profile of
// neutral sub-signals is neutral.
type Weights struct {
	BreakoutAge, SchoolTier, CollegeShare, AgeTrajectory float64
}

// Curves map the raw breakout sub-signals onto [0,1].
type Curves struct {
	BreakoutAge   []Breakpoint // age at the college breakout season
	CollegeShare  []Breakpoint // final-season share of the team's production
	AgeTrajectory []Breakpoint // current age, falling past the position's peak
}

// Settings are one position's rubric. A Component with Cap 0 is off.
type Settings struct {
	Film      Component
	RAS       Component
	RASWeight float64 // scales RAS's distance from 1; 0 means RAS is not read
	Breakout  Component
	Weights   Weights
	Curves    Curves
	Lift      float64 // athleticLift strength on breakout age and age trajectory
}

// Rubric applies one position's Settings. It implements engine.Layer4.
type Rubric struct {
	s Settings
}

// New returns the rubric for s.
func New(s Settings) Rubric { return Rubric{s: s} }

// Rubrics returns a rubric for each position in table.
func Rubrics(table map[domain.Position]Settings) map[domain.Position]engine.Layer4 {
	out := make(map[domain.Position]engine.Layer4, len(table))
	for pos, s := range table {
		out[pos] = New(s)
	}
	return out
}

// Apply scores one player. Film reads the upstream composite, RAS the raw score over 10, and
// breakout the weighted composite of four sub-signals.
func (r Rubric) Apply(in engine.Layer4Input) engine.Layer4Output {
	s, sc, p := r.s, in.Scouting, in.Player
	out := engine.Layer4Output{FilmEffective: 1, FilmRaw: NeutralNorm, RASEffective: 1}
	if sc.HasFilm {
		out.FilmRaw, out.FilmEffective = sc.FilmComposite, s.Film.Apply(sc.FilmComposite)
	}
	if p.HasRAS {
		out.RASEffective = 1 + s.RASWeight*(s.RAS.Apply(p.RAS/10.0)-1)
	}
	out.BreakoutEffective = s.Breakout.Apply(r.breakout(in))
	out.Combined = out.FilmEffective * out.RASEffective * out.BreakoutEffective
	return out
}

// breakout is the composite: breakout age and age trajectory get the athletic lift, and age
// trajectory the cushion, which composition hands only to the positions it protects.
func (r Rubric) breakout(in engine.Layer4Input) float64 {
	s, sc, p := r.s, in.Scouting, in.Player
	age := curved(sc.HasBreakoutAge, s.Curves.BreakoutAge, sc.BreakoutAge)
	if sc.HasBreakoutAge {
		age = athleticLift(age, p.RAS, s.Lift, p.HasRAS)
	}
	traj := athleticLift(interp(s.Curves.AgeTrajectory, p.Age), p.RAS, s.Lift, p.HasRAS)
	traj = in.Cushion.Slow(traj, NeutralNorm, p.RAS, p.HasRAS)
	w := s.Weights
	return w.BreakoutAge*age +
		w.SchoolTier*given(sc.HasSchoolTier, sc.SchoolTierNorm) +
		w.CollegeShare*curved(sc.HasCollegeShare, s.Curves.CollegeShare, sc.CollegeShare) +
		w.AgeTrajectory*traj
}
