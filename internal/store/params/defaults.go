package params

// defaultParams returns the shipped settings and fitted values.
func defaultParams() []ParamDef {
	return append([]ParamDef{
		{
			Key: KeyLayer3DecayRate, Position: global, Type: TypeFloat,
			Default: 0.03, Min: 0, Max: 1,
			Description: "Layer 3: annual age-decay rate",
		},
		{
			Key: KeyDynastySeasons, Position: global, Type: TypeFloat,
			Default: 3, Min: 1, Max: 10,
			Description: "Dynasty: seasons counted, this one first (whole seasons)",
		},
		{
			Key: KeyDynastyDiscount, Position: global, Type: TypeFloat,
			Default: 0.75, Min: 0, Max: 1,
			Description: "Dynasty: weight of each later season against the one before",
		},
		{
			Key: KeyDynastyContractTilt, Position: global, Type: TypeFloat,
			Default: 0.06, Min: 0, Max: 0.25,
			Description: "Dynasty: most a contract's price and length move a value, as a share",
		},
	}, fittedParams()...)
}
