// Package domain holds the leaf domain types the engine and stores consume:
// the typed, validated records that normalize (B3) produces from raw MFL data.
// It is a LEAF — it imports only internal/playerid (the shared id value type) and
// nothing else internal. Raw→typed mapping lives in normalize, not here; this
// package only defines the value types and their legal sets.
package domain

// Position is an engine position code. It is the NORMALIZED set the engine scores
// against — MFL's raw codes (PK, EDGE, XX, team aggregates) are mapped onto it by
// normalize, never stored raw. PosFlag marks a record whose position could not be
// classified (MFL "XX" or an unknown code) and that an admin must resolve before
// the engine trusts it.
type Position string

const (
	PosQB   Position = "QB"
	PosRB   Position = "RB"
	PosWR   Position = "WR"
	PosTE   Position = "TE"
	PosK    Position = "K"  // MFL "PK" normalizes here
	PosDE   Position = "DE" // MFL "EDGE" normalizes here (OQ-004: MFL labels edge rushers DE)
	PosDT   Position = "DT"
	PosLB   Position = "LB"
	PosCB   Position = "CB"
	PosS    Position = "S"
	PosFlag Position = "FLAG" // unclassified (MFL "XX"/unknown) — admin must resolve
)

// ContractStatus is the normalized dynasty contract state. MFL's contractStatus
// field is dirty (trailing spaces, typos, parenthetical notes, combined values);
// normalize cleans every known variant onto this set. CStatusFlag marks a value
// that did not match any known prefix and needs admin review.
type ContractStatus string

const (
	CStatusUFA  ContractStatus = "UFA"
	CStatusRFA  ContractStatus = "RFA"
	CStatusFT1  ContractStatus = "FT1"
	CStatusFT2  ContractStatus = "FT2"
	CStatusFlag ContractStatus = "FLAG" // unknown/dirty value — admin must resolve
)

// RosterStatus is where a player sits on a franchise. MFL sends "ROSTER",
// "TAXI_SQUAD", or "IR"; an unexpected value fails loud in normalize rather than
// mapping to a default.
type RosterStatus string

const (
	RosterActive RosterStatus = "ROSTER"
	RosterTaxi   RosterStatus = "TAXI_SQUAD"
	RosterIR     RosterStatus = "IR"
)
