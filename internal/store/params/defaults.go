package params

import "github.com/secureprospective/TheWarRoom/internal/engine/l4"

// defaultParams is the single source of the shipped defaults: the globals below and the
// Layer-4 settings, whose table ships with the engine (l4.Defaults). Ranges are deliberately wide: they catch admin typos
// without pre-judging the calibrated value, and IsCalibrated stays false until a calibration
// pass sets one.
func defaultParams() []ParamDef {
	return append([]ParamDef{
		{
			Key: KeyCapTierColdCeiling, Position: global, Type: TypeFloat,
			Default: 1.2, Min: 0, Max: 100,
			Description: "Layer 5: salary % of league cap below this is the Cold tier",
		},
		{
			Key: KeyCapTierHotFloor, Position: global, Type: TypeFloat,
			Default: 4.8, Min: 0, Max: 100,
			Description: "Layer 5: salary % of league cap above this is the Hot tier",
		},
		{
			Key: KeyLayer3DecayRate, Position: global, Type: TypeFloat,
			Default: 0.03, Min: 0, Max: 1,
			Description: "Layer 3: annual age-decay rate",
		},
		{
			Key: KeyCushionGuardRAS, Position: global, Type: TypeFloat,
			Default: 8.00, Min: 0, Max: 10,
			Description: "Cushion Guard: RAS threshold (DT) above which SL-019 is replaced",
		},
		{
			Key: KeyCushionGuardReduct, Position: global, Type: TypeFloat,
			Default: 0.10, Min: 0, Max: 1,
			Description: "Cushion Guard: reduction strength",
		},
	}, rubricParams()...)
}

// rubricParams is one row per adjustable Layer-4 number at each position where it is on.
func rubricParams() []ParamDef {
	var out []ParamDef
	for pos, s := range l4.Defaults() {
		for _, k := range l4.Knobs() {
			if k.AppliesTo(s) {
				out = append(out, ParamDef{Key: k.Key, Position: string(pos), Type: TypeFloat,
					Default: *k.Field(&s), Min: k.Min, Max: k.Max, Description: k.Description})
			}
		}
	}
	return out
}
