package l4

import (
	"math"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-3 }

func apply(pos domain.Position, in engine.Layer4Input) engine.Layer4Output {
	in.Player.Position = pos
	return New(Defaults()[pos]).Apply(in)
}

// A profile with nothing known, at the age where the trajectory is neutral, scores exactly 1.
func TestUnknownProfileIsNeutral(t *testing.T) {
	for pos, s := range Defaults() {
		peak := 0.0
		for _, b := range s.Curves.AgeTrajectory {
			if b.Y == NeutralNorm {
				peak = b.X
			}
		}
		if out := apply(pos, engine.Layer4Input{Player: engine.PlayerInput{Age: peak}}); out.Combined != 1 {
			t.Errorf("%s: unknown profile at age %v = %+v, want Combined exactly 1", pos, peak, out)
		}
	}
}

// Every component stays inside its cap, whatever the input.
func TestComponentsStayInsideTheirCaps(t *testing.T) {
	extremes := []float64{-5, 0, 1, 5, 99, math.NaN(), math.Inf(1)}
	for pos, s := range Defaults() {
		for _, v := range extremes {
			out := apply(pos, engine.Layer4Input{
				Player: engine.PlayerInput{Age: v, RAS: v * 10, HasRAS: true},
				Scouting: engine.ScoutingInput{FilmComposite: v, HasFilm: true, BreakoutAge: v * 20,
					HasBreakoutAge: true, SchoolTierNorm: v, HasSchoolTier: true, CollegeShare: v, HasCollegeShare: true},
			})
			for _, c := range []struct {
				name string
				got  float64
				cap  float64
			}{{"film", out.FilmEffective, s.Film.Cap}, {"RAS", out.RASEffective, s.RAS.Cap * s.RASWeight},
				{"breakout", out.BreakoutEffective, s.Breakout.Cap}} {
				if c.got < 1-c.cap-1e-12 || c.got > 1+c.cap+1e-12 {
					t.Errorf("%s at input %v: %s %v outside 1±%v", pos, v, c.name, c.got, c.cap)
				}
			}
		}
	}
}

// QB and K do not read RAS, and K has no breakout.
func TestOffMechanicsReturnExactlyOne(t *testing.T) {
	for _, pos := range []domain.Position{domain.PosQB, domain.PosK} {
		for _, ras := range []float64{0.1, 9.99} {
			if out := apply(pos, engine.Layer4Input{Player: engine.PlayerInput{RAS: ras, HasRAS: true}}); out.RASEffective != 1 {
				t.Errorf("%s RAS %v: RASEffective %v, want exactly 1", pos, ras, out.RASEffective)
			}
		}
	}
	k := apply(domain.PosK, engine.Layer4Input{
		Player:   engine.PlayerInput{Age: 40},
		Scouting: engine.ScoutingInput{FilmComposite: 0.9, HasFilm: true, BreakoutAge: 25, HasBreakoutAge: true},
	})
	if k.BreakoutEffective != 1 || k.Combined != k.FilmEffective || k.FilmEffective <= 1 {
		t.Errorf("K = %+v, want breakout exactly 1 and Combined = a lifted film", k)
	}
}

// The QB rubric's worked examples: an elite college profile at 25, and a good one at 35.
func TestQBWorkedExamples(t *testing.T) {
	for _, c := range []struct {
		name               string
		age, bo, tier, cs  float64
		breakout, combined float64
	}{
		{"elite profile, age 25", 25, 19, 1.00, 0.90, 1.050, 1.050},
		{"good profile, age 35", 35, 21, 0.70, 0.60, 1.039, 1.039},
	} {
		out := apply(domain.PosQB, engine.Layer4Input{
			Player: engine.PlayerInput{Age: c.age},
			Scouting: engine.ScoutingInput{BreakoutAge: c.bo, HasBreakoutAge: true, SchoolTierNorm: c.tier,
				HasSchoolTier: true, CollegeShare: c.cs, HasCollegeShare: true},
		})
		if !near(out.BreakoutEffective, c.breakout) || !near(out.Combined, c.combined) {
			t.Errorf("%s: breakout %v combined %v, want %v and %v", c.name, out.BreakoutEffective, out.Combined, c.breakout, c.combined)
		}
	}
}

// The athletic lift's worked examples at the TE strength, 0.35.
func TestAthleticLift(t *testing.T) {
	for _, c := range []struct{ base, ras, want float64 }{
		{0.50, 9.5, 0.666}, {0.50, 7.0, 0.623}, {0.50, 4.0, 0.570}, {0.10, 9.5, 0.399}, {0.10, 5.4, 0.270}, {1.00, 9.5, 1.000},
	} {
		if got := athleticLift(c.base, c.ras, 0.35, true); !near(got, c.want) {
			t.Errorf("lift(%v, RAS %v) = %v, want %v", c.base, c.ras, got, c.want)
		}
	}
	if got := athleticLift(0.5, 9.5, 0.35, false); got != 0.5 {
		t.Errorf("an absent RAS must not lift: %v", got)
	}
	if got := athleticLift(0.5, 9.5, 0, true); got != 0.5 {
		t.Errorf("strength 0 must not lift: %v", got)
	}
	// An absent breakout age stays neutral at a lift position: unknown data is never lifted.
	te := func(ras float64) float64 {
		return apply(domain.PosTE, engine.Layer4Input{Player: engine.PlayerInput{Age: 29, RAS: ras, HasRAS: true}}).BreakoutEffective
	}
	if hi, lo := te(9.5), te(4.0); !(hi > lo) {
		t.Errorf("TE age trajectory should rise with RAS: %v vs %v", hi, lo)
	}
}

// The cushion slows a high-RAS DT's decline past peak, and only there.
func TestCushionSlowsDecline(t *testing.T) {
	sc := engine.ScoutingInput{BreakoutAge: 22, HasBreakoutAge: true, SchoolTierNorm: 0.70, HasSchoolTier: true,
		CollegeShare: 0.15, HasCollegeShare: true}
	guard := engine.CushionGuard{RASThreshold: 8, DeclineFactor: 0.9}
	breakout := func(age, ras float64, g engine.CushionGuard) float64 {
		return apply(domain.PosDT, engine.Layer4Input{
			Player: engine.PlayerInput{Age: age, RAS: ras, HasRAS: true}, Scouting: sc, Cushion: g}).BreakoutEffective
	}
	if hi, lo := breakout(32, 9, guard), breakout(32, 7, guard); !(hi > lo) {
		t.Errorf("past peak a cushioned DT %v should beat an uncushioned one %v", hi, lo)
	}
	if hi, lo := breakout(26, 9, guard), breakout(26, 7, guard); hi != lo {
		t.Errorf("before peak the cushion must be inert: %v vs %v", hi, lo)
	}
	if hi, lo := breakout(32, 9, engine.CushionGuard{}), breakout(32, 7, engine.CushionGuard{}); hi != lo {
		t.Errorf("with the guard off RAS must not reach the trajectory: %v vs %v", hi, lo)
	}
}

// A knob is a setting only where the shipped value is on, and every shipped value is in range.
func TestKnobsCoverTheOnMechanics(t *testing.T) {
	offAtQB := []string{"l4.ras.steepness", "l4.ras.cap", "l4.ras.weight", "l4.breakout.athletic_lift"}
	onAtK := []string{"l4.film.steepness", "l4.film.cap"}
	qb, k := Defaults()[domain.PosQB], Defaults()[domain.PosK]
	for _, knob := range Knobs() {
		if knob.AppliesTo(qb) == slices.Contains(offAtQB, knob.Key) {
			t.Errorf("QB: %s applies = %v", knob.Key, knob.AppliesTo(qb))
		}
		if knob.AppliesTo(k) != slices.Contains(onAtK, knob.Key) {
			t.Errorf("K: %s applies = %v", knob.Key, knob.AppliesTo(k))
		}
		for pos, s := range Defaults() {
			if v := *knob.Field(&s); knob.AppliesTo(s) && (v < knob.Min || v > knob.Max) {
				t.Errorf("%s at %s: %v outside [%v, %v]", knob.Key, pos, v, knob.Min, knob.Max)
			}
		}
	}
}
