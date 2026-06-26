package params

// defaultParams returns the v1.0 shipped calibration defaults (the Option-C scope):
// the Layer-5 cap-tier percentages (AD-21, the present engine consumer) PLUS the few
// global scalars that already have stated defaults in the engine spec (Layer-3 decay,
// Cushion Guard). Per-position calibration TABLES ship WITH the engine layer that
// consumes them, never before — so this set is deliberately small, and seeding two
// distinct param groups proves the store shape is general, not cap-tier-specific.
//
// Returned from a function (not a package var) to satisfy gochecknoglobals; it is the
// single source of seed truth (M17). Ranges are wide on purpose (GLM-5.2 review):
// they bound out-of-range admin error without pre-judging the post-live calibrated
// value, which is revisited at M9a (OQ-006). IsCalibrated stays false until then.
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
