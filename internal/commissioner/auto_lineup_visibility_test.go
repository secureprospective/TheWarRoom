package commissioner

import (
	"testing"
	"time"
)

func TestAutomaticLineupIsVisibleToItsOwnerOnly(t *testing.T) {
	records := contractDecisionSeats(t)
	at := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	ownerBefore := RecipientRevision(records, "owner.one", at)
	otherBefore := RecipientRevision(records, "owner.two", at)
	auto := Record{Command: RawCommand{League: "native.test", Command: LineupSet,
		Args: RawArgs{Lineup: &RawLineupChange{Season: 2026, Week: 1, Franchise: "franchise.01",
			Starters: []string{"0001"}}}, Effective: "now", Visibility: "private"}, Principal: LineupClockPrincipal,
		Receipt: Receipt{Entry: "auto.lineup", League: "native.test", Revision: int64(len(records) + 1), RecordedAt: at}}
	records = append(records, auto)
	if RecipientRevision(records, "owner.one", at) == ownerBefore {
		t.Fatal("the owner's revision did not move for their automatic lineup")
	}
	if !ledgerEntryVisible(auto, auto, records, "owner.one", false) {
		t.Fatal("the owner cannot see their automatic lineup in the ledger")
	}
	if RecipientRevision(records, "owner.two", at) != otherBefore {
		t.Fatal("another owner's revision moved for a private lineup")
	}
	if ledgerEntryVisible(auto, auto, records, "owner.two", false) {
		t.Fatal("another owner can see a private lineup")
	}
}
