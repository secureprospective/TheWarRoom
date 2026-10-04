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
	// KeyCapTierColdCeiling: salary below this % of the cap is Cold.
	KeyCapTierColdCeiling = "captier.cold_ceiling_pct"
	// KeyCapTierHotFloor: salary above this % of the cap is Hot.
	KeyCapTierHotFloor = "captier.hot_floor_pct"
	// KeyLayer3DecayRate: the annual age-decay rate.
	KeyLayer3DecayRate = "layer3.decay_rate"
	// KeyCushionGuardRAS: the DT RAS threshold above which Cushion Guard applies (SL-021).
	KeyCushionGuardRAS = "cushion_guard.ras_threshold"
	// KeyCushionGuardReduct: how much Cushion Guard slows the DT decline.
	KeyCushionGuardReduct = "cushion_guard.reduction"
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

// CapTiers are the cap-tier boundaries as percentages of the league cap.
type CapTiers struct {
	ColdCeiling float64 // Cold when salary% < ColdCeiling
	HotFloor    float64 // Hot when salary% > HotFloor; Neutral in between
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
