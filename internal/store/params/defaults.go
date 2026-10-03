package params

// defaultParams is the single source of the shipped defaults. Per-position tables ship with
// the engine layer that reads them. Ranges are deliberately wide: they catch admin typos
// without pre-judging the calibrated value, and IsCalibrated stays false until a calibration
// pass sets one.
func defaultParams() []ParamDef {
	return []ParamDef{
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
	}
}
