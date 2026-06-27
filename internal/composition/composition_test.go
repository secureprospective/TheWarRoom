package composition

import (
	"errors"
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

type fakeParams struct {
	tiers    params.CapTiers
	decay    float64
	tiersErr error
	decayErr error
}

func (f fakeParams) GetCapTiers() (params.CapTiers, error) { return f.tiers, f.tiersErr }

func (f fakeParams) GetGlobal(string) (float64, error) {
	if f.decayErr != nil {
		return 0, f.decayErr
	}
	return f.decay, nil
}

type fakeCap struct{ cap string }

func (f fakeCap) GetSalaryCap() string { return f.cap }

func goodStores() (fakeParams, fakeCap) {
	return fakeParams{tiers: params.CapTiers{ColdCeiling: 1.2, HotFloor: 4.8}, decay: 0.03},
		fakeCap{cap: "1000"}
}

func goodSpec() PlayerSpec {
	return PlayerSpec{
		MFLID: "0001", Name: "Test Player", Position: domain.PosWR,
		BasePoints: 100, Age: 24, RAS: 9.0, HasRAS: true, Salary: 20, IsVeteran: false,
	}
}

func TestAssembleHappyPath(t *testing.T) {
	p, c := goodStores()
	a := New(p, c)
	in, cal, err := a.Assemble(goodSpec())
	if err != nil {
		t.Fatalf("Assemble: unexpected error: %v", err)
	}
	if in.Position != domain.PosWR || in.BasePoints != 100 || in.Age != 24 || !in.HasRAS {
		t.Fatalf("PlayerInput not mapped through: %+v", in)
	}
	if cal.DecayRate != 0.03 || cal.ColdCeiling != 1.2 || cal.HotFloor != 4.8 {
		t.Fatalf("globals not read from store: %+v", cal)
	}
	if cal.LeagueCap != 1000 {
		t.Fatalf("league cap not parsed: got %v want 1000", cal.LeagueCap)
	}
	if cal.PeakLimit != 29 { // WR peak (Engine_Specification:118)
		t.Fatalf("WR peak limit: got %v want 29", cal.PeakLimit)
	}
	if cal.RASFallback != DefaultRASFallback || cal.SalaryFloor != DefaultSalaryFloor {
		t.Fatalf("L1 defaults not supplied: %+v", cal)
	}
}

// TestAssembleRejectsBadSpec is the planted-failure gate (M3): the boundary must reject
// unscorable input so the pure engine never receives a poisoned value.
func TestAssembleRejectsBadSpec(t *testing.T) {
	p, c := goodStores()
	a := New(p, c)
	cases := map[string]func(s *PlayerSpec){
		"empty id":          func(s *PlayerSpec) { s.MFLID = "" },
		"empty name":        func(s *PlayerSpec) { s.Name = "" },
		"flag position":     func(s *PlayerSpec) { s.Position = domain.PosFlag },
		"unknown position":  func(s *PlayerSpec) { s.Position = domain.Position("ZZ") },
		"non-positive age":  func(s *PlayerSpec) { s.Age = 0 },
		"negative salary":   func(s *PlayerSpec) { s.Salary = -1 },
		"NaN base points":   func(s *PlayerSpec) { s.BasePoints = math.NaN() },
		"Inf age":           func(s *PlayerSpec) { s.Age = math.Inf(1) },
		"NaN ras with flag": func(s *PlayerSpec) { s.HasRAS = true; s.RAS = math.NaN() },
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			s := goodSpec()
			mut(&s)
			if _, _, err := a.Assemble(s); err == nil {
				t.Fatalf("expected error for %q, got nil", name)
			}
		})
	}
}

func TestCalibrationFailsLoudOnBadCap(t *testing.T) {
	p, _ := goodStores()
	for _, bad := range []string{"", "   ", "abc", "0", "-5", "NaN"} {
		a := New(p, fakeCap{cap: bad})
		if _, err := a.Calibration(domain.PosWR); err == nil {
			t.Fatalf("expected error for cap %q, got nil", bad)
		}
	}
}

// TestCalibrationRejectsPoisonedStoreValues is the B1 defense-in-depth gate: the boundary
// must not pass a non-finite or out-of-range STORE value into the engine even though B4
// range-gates on write.
func TestCalibrationRejectsPoisonedStoreValues(t *testing.T) {
	_, c := goodStores()
	good := params.CapTiers{ColdCeiling: 1.2, HotFloor: 4.8}
	cases := map[string]fakeParams{
		"NaN decay":        {tiers: good, decay: math.NaN()},
		"Inf decay":        {tiers: good, decay: math.Inf(1)},
		"decay > 1":        {tiers: good, decay: 1.5},
		"decay < 0":        {tiers: good, decay: -0.1},
		"inverted tiers":   {tiers: params.CapTiers{ColdCeiling: 5.0, HotFloor: 1.0}, decay: 0.03},
		"negative tier":    {tiers: params.CapTiers{ColdCeiling: -1, HotFloor: 4.8}, decay: 0.03},
		"NaN cold ceiling": {tiers: params.CapTiers{ColdCeiling: math.NaN(), HotFloor: 4.8}, decay: 0.03},
	}
	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := New(p, c).Calibration(domain.PosWR); err == nil {
				t.Fatalf("expected error for %q, got nil", name)
			}
		})
	}
}

// TestAssembleZerosAbsentRAS is the l1 guard: when HasRAS is false the raw RAS is zeroed at
// the boundary so a stray/non-finite value cannot ride into the engine.
func TestAssembleZerosAbsentRAS(t *testing.T) {
	p, c := goodStores()
	s := goodSpec()
	s.HasRAS = false
	s.RAS = math.NaN()
	in, _, err := New(p, c).Assemble(s)
	if err != nil {
		t.Fatalf("Assemble should accept absent RAS: %v", err)
	}
	if in.RAS != 0 || in.HasRAS {
		t.Fatalf("absent RAS must be zeroed, got RAS=%v HasRAS=%v", in.RAS, in.HasRAS)
	}
}

func TestCalibrationPropagatesStoreErrors(t *testing.T) {
	_, c := goodStores()
	sentinel := errors.New("store down")
	if _, err := New(fakeParams{tiersErr: sentinel}, c).Calibration(domain.PosWR); err == nil {
		t.Fatal("expected cap-tier error to propagate")
	}
	if _, err := New(fakeParams{decayErr: sentinel}, c).Calibration(domain.PosWR); err == nil {
		t.Fatal("expected decay error to propagate")
	}
}

func TestPeakLimitByPosition(t *testing.T) {
	want := map[domain.Position]float64{
		domain.PosQB: 32, domain.PosRB: 25, domain.PosWR: 29, domain.PosTE: 29,
		domain.PosDE: 30, domain.PosDT: 30, domain.PosLB: 29, domain.PosCB: 28,
		domain.PosS: 28, domain.PosK: 30,
	}
	for pos, w := range want {
		if got := peakLimit(pos); got != w {
			t.Errorf("peakLimit(%s) = %v, want %v", pos, got, w)
		}
	}
}

// TestAssembleFeedsEngineEndToEnd proves the boundary output flows straight into the
// pure engine: a player assembled here scores through the identity-L4 pipeline. With
// L4 identity (Combined 1.0) and a neutral salary tier, AdjustedScore = BasePoints.
func TestAssembleFeedsEngineEndToEnd(t *testing.T) {
	p, c := goodStores()
	a := New(p, c)
	s := goodSpec()
	s.Age = 22    // below WR peak 29 → AgePull 1.0
	s.Salary = 30 // 30/1000 = 3% → between 1.2 and 4.8 → Neutral ×1.0
	in, cal, err := a.Assemble(s)
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	res, err := engine.NewPipeline(nil).Score(in, cal)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if res.CapTier != engine.CapTierNeutral {
		t.Fatalf("expected Neutral tier, got %s", res.CapTier)
	}
	if math.Abs(res.AdjustedScore-100) > 1e-9 {
		t.Fatalf("identity path: AdjustedScore = %v, want 100", res.AdjustedScore)
	}
}
