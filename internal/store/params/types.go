package params

// ValueType tags how a parameter's stored string is parsed and bounded. v1.0 holds
// only floating-point calibration params; the tag exists so a future int/string
// parameter slots into the same generic store without reshaping it.
type ValueType string

// TypeFloat is a float64 parameter validated against an inclusive [Min,Max] range.
const TypeFloat ValueType = "float"

// global is the sentinel Position for a league-wide (non-per-position) parameter.
// A future per-position calibration param (shipped WITH the engine layer that reads
// it) carries a real position code here instead; because (Key, Position) is the
// store's primary key, a per-position table is N rows sharing one Key — no schema
// change when those layers land. v1.0 holds only globals.
const global = ""

// Parameter keys. Canonical, dotted, stable — the engine reads by these constants.
const (
	// KeyCapTierColdCeiling: salary as a % of the league cap below this is Cold (L5).
	KeyCapTierColdCeiling = "captier.cold_ceiling_pct"
	// KeyCapTierHotFloor: salary as a % of the league cap above this is Hot (L5).
	KeyCapTierHotFloor = "captier.hot_floor_pct"
	// KeyLayer3DecayRate: the annual age-decay rate (Layer 3).
	KeyLayer3DecayRate = "layer3.decay_rate"
	// KeyCushionGuardRAS: the RAS threshold above which Cushion Guard replaces SL-019.
	KeyCushionGuardRAS = "cushion_guard.ras_threshold"
	// KeyCushionGuardReduct: the Cushion Guard reduction strength.
	KeyCushionGuardReduct = "cushion_guard.reduction"
)

// ParamDef is one calibration parameter's definition: its identity (Key+Position),
// type, shipped default, and the inclusive bounds the override gate enforces. The
// seeded defaults are the engine's STARTING calibration — OQ-006 tuning happens later
// via M9a — so IsCalibrated is false until a real calibration pass sets a value,
// letting the engine report how much of its input is still on placeholder defaults.
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

// CapTiers is the Layer-5 cap-scaling boundary set. AD-21: these are PERCENTAGES of
// the league cap, not the cap AMOUNT (the amount is the rulebook store's concern). A
// player's salary as a percent of the cap below ColdCeiling is Cold, above HotFloor is
// Hot, and Neutral in between.
type CapTiers struct {
	ColdCeiling float64 // Cold when salary% < ColdCeiling
	HotFloor    float64 // Hot when salary% > HotFloor; Neutral in between
}

// Override is an admin delta layered over a shipped default at read time. It is a
// SEPARATE record (the default stays immutable), so re-seeding never clobbers it;
// clearing an override back to the default is a future capability (a delete path),
// not yet implemented. Admin-only write path (AD-05) — never routed through B7.
// UpdatedAt is last-modified: an upsert refreshes it on every write.
type Override struct {
	Key       string
	Position  string
	Value     float64
	Note      string
	UpdatedAt string
}

// defKey is the in-memory map key for a (parameter, position) pair.
func defKey(key, position string) string { return key + "\x00" + position }
