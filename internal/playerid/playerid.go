// Package playerid is the single source of truth for MFL player identifiers.
//
// AD-06 / RISK-003 enforcement (decided 2026-06-13, Confidence-80 session,
// Gate 1 → struct-wrap). MFL player IDs are strings, and IDs under 1000 require
// leading zeros (project hard constraint: "0099", never "99"). If those two
// rules are not applied uniformly, an id built one way silently fails to match
// the same player stored another way — a data-integrity bug that can be planted
// in any of the 38 build sessions and is very hard to trace back to its source.
//
// The mechanism. PlayerID is a struct with an UNEXPORTED field. Outside this
// package, `playerid.PlayerID("99")` does not compile — you cannot convert a
// string to a struct type — so the validating constructor New is the only way
// to obtain a PlayerID. This converts a lint-time social contract (which
// forbidigo could not actually express — Fable Friction #6) into a compile-time
// guarantee, which is the standard this repo holds itself to. JSON and SQL both
// route through New, so an id arriving over the wire or read back from the DB is
// validated and normalized identically to one built in code.
package playerid

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// minWidth is MFL's canonical id width: IDs are left-zero-padded to 4 digits
// ("0099"). Larger IDs (5+ digits) keep their natural width.
const minWidth = 4

// PlayerID is a validated, normalized MFL player identifier. The zero value is
// not a valid id; New never returns it on success (see IsZero).
type PlayerID struct {
	id string
}

// New validates and normalizes a raw MFL id. It rejects empty/non-numeric
// input and collapses every leading-zero variant of the same number to one
// canonical form, so "99", "099", and "0099" all become "0099".
func New(raw string) (PlayerID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return PlayerID{}, fmt.Errorf("playerid: empty id")
	}
	if _, err := strconv.Atoi(trimmed); err != nil {
		return PlayerID{}, fmt.Errorf("playerid: %q is not numeric: %w", raw, err)
	}

	digits := strings.TrimLeft(trimmed, "0")
	if digits == "" { // input was all zeros, e.g. "0000"
		digits = "0"
	}
	if len(digits) < minWidth {
		digits = strings.Repeat("0", minWidth-len(digits)) + digits
	}
	return PlayerID{id: digits}, nil
}

// String returns the canonical zero-padded id.
func (p PlayerID) String() string { return p.id }

// IsZero reports whether p is the zero value (never produced by a successful New).
func (p PlayerID) IsZero() bool { return p.id == "" }

// MarshalJSON encodes the id as its canonical string form.
func (p PlayerID) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(p.id)
	if err != nil {
		return nil, fmt.Errorf("playerid: marshal: %w", err)
	}
	return data, nil
}

// UnmarshalJSON decodes a JSON string THROUGH New, so an id arriving over the
// wire cannot smuggle in an unvalidated or unnormalized value.
func (p *PlayerID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("playerid: unmarshal: %w", err)
	}
	id, err := New(s)
	if err != nil {
		return err // already prefixed with "playerid:"
	}
	*p = id
	return nil
}

// Note on database serialization. PlayerID deliberately implements NEITHER
// driver.Valuer NOR sql.Scanner. Doing so would force this package to import
// database/sql/driver, which the G0 overlay's depguard `sql-confined-to-data-
// layer` rule correctly forbids outside internal/db and internal/store (a
// domain identifier is not part of the data layer). Persistence happens at the
// store boundary instead: a store writes id.String() (all SQLite id columns are
// TEXT — project hard constraint) and reads back through playerid.New(text), so
// a value loaded from the DB is re-validated exactly like one built in code.
// This keeps PlayerID a pure domain value with zero I/O-layer coupling, in
// keeping with the three-layer law.
//
// (sql.Scanner's Scan(any) signature, were it implemented in the store layer,
// is the canonical case for the ifaceguard analyzer's //ifaceguard:allow
// directive — see templates/go/tools/ifaceguard/README.md.)
