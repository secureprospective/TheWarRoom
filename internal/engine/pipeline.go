package engine

// Pipeline runs the layers in order. It holds only the Layer 4 dispatch, so one Pipeline is
// safe for concurrent Score calls.
type Pipeline struct {
	layer4 Layer4
}

// NewPipeline builds a pipeline with the given Layer 4 dispatch; nil means identity.
func NewPipeline(layer4 Layer4) *Pipeline {
	if layer4 == nil {
		layer4 = identityLayer4{}
	}
	return &Pipeline{layer4: layer4}
}

// Score runs L1 → L3 → L4 → L5 → L6:
//
//	ScoutingAdjusted = BasePoints × AgePull × Layer4Output.Combined
//	AdjustedScore    = ScoutingAdjusted × CapMultiplier
//
// L1's cleaned salary feeds L5 and its cleaned RAS feeds L6. The scouting sub-signals feed
// L4 only.
func (pl *Pipeline) Score(p PlayerInput, sc ScoutingInput, c Calibration) (Result, error) {
	cleaned := ApplyHygiene(p, c)
	agePull, err := ApplyDecay(p.Age, c.PeakLimit, c.DecayRate)
	if err != nil {
		return Result{}, err
	}
	agePull = ApplyCushionGuard(agePull, p.RAS, p.HasRAS, c.CushionRASThreshold, c.CushionDeclineFactor)
	l4 := pl.layer4.Apply(Layer4Input{Player: p, Scouting: sc})

	scoutingAdjusted := p.BasePoints * agePull * l4.Combined

	cap5, err := ApplyCapScaling(scoutingAdjusted, cleaned.Salary, c.LeagueCap, c.ColdCeiling, c.HotFloor)
	if err != nil {
		return Result{}, err
	}

	return Result{
		BasePoints:       p.BasePoints,
		AgePull:          agePull,
		Layer4Output:     l4,
		ScoutingAdjusted: scoutingAdjusted,
		CapMultiplier:    cap5.Multiplier,
		CapTier:          cap5.Tier,
		AdjustedScore:    cap5.AdjustedScore,
		Tiebreaker:       BuildTiebreaker(p, cleaned.RAS, c.ScarcityRank),
	}, nil
}

// Cleaned is L1's output. RASImputed marks a fallback RAS; it is internal, never shown.
type Cleaned struct {
	RAS        float64
	Salary     float64
	RASImputed bool
}

// ApplyHygiene is Layer 1: impute the fallback RAS when absent and raise salary to the
// floor (Backend_Architecture:235).
func ApplyHygiene(p PlayerInput, c Calibration) Cleaned {
	ras := p.RAS
	imputed := false
	if !p.HasRAS {
		ras = c.RASFallback
		imputed = true
	}
	salary := p.Salary
	if salary < c.SalaryFloor {
		salary = c.SalaryFloor
	}
	return Cleaned{RAS: ras, Salary: salary, RASImputed: imputed}
}

// identityLayer4 is the no-op Layer 4: every multiplier is 1.0.
type identityLayer4 struct{}

func (identityLayer4) Apply(Layer4Input) Layer4Output {
	return Layer4Output{
		FilmEffective:     1.0,
		RASEffective:      1.0,
		BreakoutEffective: 1.0,
		Combined:          1.0,
	}
}

// IdentityLayer4 returns the no-op Layer 4.
func IdentityLayer4() Layer4 { return identityLayer4{} }

// BuildTiebreaker is Layer 6: the key that orders exact score ties
// (Backend_Architecture:268). It uses the cleaned RAS, so an imputed one sorts consistently.
func BuildTiebreaker(p PlayerInput, cleanedRAS float64, scarcityRank int) TiebreakerKey {
	return TiebreakerKey{
		IsVeteran:    p.IsVeteran,
		RAS:          cleanedRAS,
		ScarcityRank: scarcityRank,
	}
}
