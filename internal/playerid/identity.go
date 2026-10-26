package playerid

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// RawOutsideID preserves a provider's value exactly, including leading zeros.
type RawOutsideID struct {
	Provider string `json:"provider"`
	Value    string `json:"value"`
}

// RawIdentity is our permanent identity with its outside links. MFL PlayerID is
// deliberately still a separate type: existing imports speak that protocol.
type RawIdentity struct {
	Canonical string         `json:"canonical"`
	Links     []RawOutsideID `json:"links"`
}

func (r RawIdentity) Validate() error {
	suffix, ok := strings.CutPrefix(r.Canonical, "player.")
	decoded, err := hex.DecodeString(suffix)
	if !ok || err != nil || len(decoded) != 16 || strings.ToLower(suffix) != suffix || len(r.Links) == 0 {
		return fmt.Errorf("invalid canonical player identity")
	}
	seen := make(map[RawOutsideID]bool)
	mfl := 0
	for _, link := range r.Links {
		if strings.TrimSpace(link.Provider) == "" || link.Value == "" || seen[link] {
			return fmt.Errorf("invalid or repeated outside player link")
		}
		seen[link] = true
		if link.Provider == "mfl" {
			mfl++
			id, err := New(link.Value)
			if err != nil || id.String() != link.Value {
				return fmt.Errorf("invalid MFL compatibility link")
			}
		}
	}
	if mfl > 1 {
		return fmt.Errorf("a player cannot have two MFL compatibility links")
	}
	return nil
}
