package commissioner

import (
	"slices"
	"testing"
)

func TestAnOwnerSetsTheirOwnDirectionAndTheCommissionerSideMaySetAny(t *testing.T) {
	records := lineupGovernance()
	set := RawCommand{League: "native.test", Command: FranchiseDirection,
		Args: RawArgs{People: &RawPeople{Franchise: "franchise.01", Direction: "rebuild"}}}
	allowed := map[string]bool{"owner-1": true, "owner-2": false, "boss": true, "deputy": true, "stranger": false}
	for principal, want := range allowed {
		if _, err := AuthorizeRecords(set, principal, records); (err == nil) != want {
			t.Fatalf("%s setting franchise.01's direction: %v", principal, err)
		}
	}
	recorded := append(slices.Clone(records), Record{Receipt: Receipt{Entry: "direction"}, Principal: "owner-1",
		Command: set})
	got, err := Directions(recorded)
	if err != nil || got["franchise.01"] != "rebuild" || got["franchise.02"] != "" {
		t.Fatalf("directions %v: %v", got, err)
	}
}
