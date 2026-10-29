package params

// ValueType says how a stored value is parsed and bounded. Only floats exist today.
type ValueType string

// TypeFloat is a float64 parameter validated against an inclusive [Min,Max] range.
const TypeFloat ValueType = "float"

// global is the Position of a league-wide parameter. A per-position parameter is one row per
// position under the same Key.
const global = ""

// Parameter keys. The engine reads by these constants.
const (
	// KeyLayer3DecayRate: the annual age-decay rate.
	KeyLayer3DecayRate = "layer3.decay_rate"
	// KeyDynastySeasons: how many seasons the dynasty measurable counts, this one first.
	KeyDynastySeasons = "dynasty.seasons"
	// KeyDynastyDiscount: each later season's weight in the dynasty measurable.
	KeyDynastyDiscount   = "dynasty.discount"
	KeyPickValueDiscount = "pickvalue.discount"
	KeyPowerPickWeight   = "power.pick_weight"
	// KeyDynastyContractTilt: the most a contract's price and length move a dynasty value.
	KeyDynastyContractTilt = "dynasty.contract_tilt"
)

// ParamDef is one parameter: its identity (Key, Position), type, shipped default and the
// inclusive range an override must fall in. IsCalibrated is false while the value is still a
// placeholder.
type ParamDef struct {
	Key          string
	Position     string
	Type         ValueType
	Default      float64
	Min          float64
	Max          float64
	IsCalibrated bool
	Description  string
}

// Override is an admin value layered over a shipped default; a release may replace the default,
// never the override.
// Clearing an override is not implemented yet.
type Override struct {
	Key       string
	Position  string
	Value     float64
	Note      string
	UpdatedAt string
}

// defKey is the in-memory map key for a (parameter, position) pair.
func defKey(key, position string) string { return key + "\x00" + position }
