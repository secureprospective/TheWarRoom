// Package playerid is the single source of MFL player ids. Ids are strings, and ids under 1000
// carry leading zeros ("0099", never "99"); an id built any other way silently fails to match.
// New is the only constructor. There is deliberately no sql.Scanner/driver.Valuer, which would
// pull database/sql into a domain type: stores write id.String() into TEXT columns and read
// back through New.
package playerid

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// minWidth is MFL's id width: shorter ids are zero-padded to 4 digits.
const minWidth = 4

// PlayerID is a validated, normalized MFL player id. The zero value is not valid.
type PlayerID struct {
	id string
}

// New validates a raw id and normalizes every leading-zero variant to one form: "99", "099"
// and "0099" all become "0099".
func New(raw string) (PlayerID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return PlayerID{}, fmt.Errorf("playerid: empty id")
	}
	// Digits only: strconv.Atoi accepts a sign, which would survive as "0+99".
	for _, r := range trimmed {
		if r < '0' || r > '9' {
			return PlayerID{}, fmt.Errorf("playerid: %q is not numeric", raw)
		}
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

// UnmarshalJSON decodes through New, so wire input is validated too.
func (p *PlayerID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("playerid: unmarshal: %w", err)
	}
	id, err := New(s)
	if err != nil {
		return err
	}
	*p = id
	return nil
}
