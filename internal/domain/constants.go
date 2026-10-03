// Package domain holds the typed records and value sets the engine and stores consume. It is a
// leaf: it imports only playerid. Mapping from raw MFL lives in normalize.
package domain

// Position is the normalized engine position set; normalize maps MFL's raw codes onto it.
// PosFlag marks an unclassified record an admin must resolve before scoring.
type Position string

const (
	PosQB   Position = "QB"
	PosRB   Position = "RB"
	PosWR   Position = "WR"
	PosTE   Position = "TE"
	PosK    Position = "K"  // MFL "PK" normalizes here
	PosDE   Position = "DE" // MFL "EDGE" maps here (OQ-004)
	PosDT   Position = "DT"
	PosLB   Position = "LB"
	PosCB   Position = "CB"
	PosS    Position = "S"
	PosFlag Position = "FLAG" // MFL "XX" or unknown
)

// ContractStatus is the normalized contract state. MFL's field is dirty; CStatusFlag marks a
// value no known prefix matched.
type ContractStatus string

const (
	CStatusUFA  ContractStatus = "UFA"
	CStatusRFA  ContractStatus = "RFA"
	CStatusFT1  ContractStatus = "FT1"
	CStatusFT2  ContractStatus = "FT2"
	CStatusFlag ContractStatus = "FLAG"
)

// RosterStatus is where a player sits on a franchise.
type RosterStatus string

const (
	RosterActive RosterStatus = "ROSTER"
	RosterTaxi   RosterStatus = "TAXI_SQUAD"
	RosterIR     RosterStatus = "IR"
)
