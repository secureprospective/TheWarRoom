// Package scouting holds the scouting inputs Layer 4 (film, RAS, breakout) consumes: one
// Profile shape for all ten positions, populated by the assembly package. It is a leaf that
// imports only playerid.
//
// Zero-leak (hard constraint): no field may hold fantasy points, projected volume, MFL
// scoring config or format-dependent volume. Every field is a film, athletic or college
// signal.
package scouting

// SchoolTier is the college competition tier feeding Breakout. SchoolUnset means absent.
type SchoolTier string

const (
	SchoolUnset       SchoolTier = ""
	SchoolPowerFour   SchoolTier = "POWER_FOUR"
	SchoolGroupOfFive SchoolTier = "GROUP_OF_FIVE"
	SchoolFCS         SchoolTier = "FCS"
	SchoolNonFCS      SchoolTier = "NON_FCS"
)

// SafetyRole is reserved (AD-16) for the box/deep safety split (SL-OQ-035/036). It stays
// unset, and S is scored as one position until that decision is made.
type SafetyRole string

const (
	SafetyRoleUnset  SafetyRole = ""
	SafetyRoleBox    SafetyRole = "box"
	SafetyRoleDeep   SafetyRole = "deep"
	SafetyRoleHybrid SafetyRole = "hybrid"
)
