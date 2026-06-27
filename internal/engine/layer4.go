package engine

// identityLayer4 is the no-op Layer 4: every component multiplier is 1.0, so it leaves
// the scouting accumulation unchanged. It lets the whole pipeline run and be tested end
// to end before any position rubric exists; B5b-QB onward supply the first real Layer 4.
type identityLayer4 struct{}

func (identityLayer4) Apply(Layer4Input) Layer4Output {
	return Layer4Output{
		FilmEffective:     1.0,
		RASEffective:      1.0,
		BreakoutEffective: 1.0,
		Combined:          1.0,
	}
}

// IdentityLayer4 returns the no-op Layer 4 dispatch. Exposed as a function (not a
// package var) so the engine holds no mutable global state (gochecknoglobals).
func IdentityLayer4() Layer4 { return identityLayer4{} }
