package engine

// Pipeline runs the fixed layer order over a player, accumulating at the two points
// defined by Backend_Architecture. It holds only the injected Layer 4 dispatch and is
// otherwise stateless, so a single Pipeline is safe for concurrent Score calls.
type Pipeline struct {
	layer4 Layer4
}

// NewPipeline builds a pipeline with the given Layer 4 dispatch. A nil dispatch falls
// back to the identity Layer 4, so callers without a position rubric (B5a, tests) get a
// working pipeline; B5b injects a real per-position Layer 4.
func NewPipeline(layer4 Layer4) *Pipeline {
	if layer4 == nil {
		layer4 = identityLayer4{}
	}
	return &Pipeline{layer4: layer4}
}

// Score runs L1 → L3 → L4 → L5 → L6 and returns the full Result. The accumulation is
// exactly Backend_Architecture's:
//
//	ScoutingAdjusted = BasePoints × AgePull × Layer4Output.Combined
//	AdjustedScore    = ScoutingAdjusted × CapMultiplier
//
// L1's cleaned salary feeds L5 and its cleaned RAS feeds L6, so hygiene is applied once
// and reused. The scouting sub-signals (sc) feed L4 only. An invalid league cap surfaces
// as an error rather than a poisoned score.
func (pl *Pipeline) Score(p PlayerInput, sc ScoutingInput, c Calibration) (Result, error) {
	cleaned := ApplyHygiene(p, c)
	agePull, err := ApplyDecay(p.Age, c.PeakLimit, c.DecayRate)
	if err != nil {
		return Result{}, err
	}
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
