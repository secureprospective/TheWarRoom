package commissioner

import (
	"testing"
	"time"
)

func contractDecisionSeats(t *testing.T) []Record {
	t.Helper()
	create, err := (RawCommand{League: "native.test", Command: Create,
		Args:    RawArgs{Name: "Decision league", Preset: PresetLegacy, FranchiseCount: 32, Season: 2026},
		Request: "create", Reason: "Create test league", Effective: "now", Visibility: "league"}).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	records := []Record{{Command: create, Principal: "commissioner", Role: "commissioner",
		Receipt: Receipt{Entry: "create", League: create.League, Revision: 1, RecordedAt: at}}}
	for _, seat := range []struct{ entry, franchise, owner string }{
		{"seat.one", "franchise.01", "owner.one"}, {"seat.two", "franchise.02", "owner.two"},
	} {
		command, err := (RawCommand{League: create.League, Command: "commish.seat.replace",
			Args:    RawArgs{People: &RawPeople{Franchise: seat.franchise, Principal: seat.owner}},
			Request: seat.entry, Reason: "Assign test owner", Effective: "now", Visibility: "league"}).Prepared()
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, Record{Command: command, Principal: "commissioner", Role: "commissioner",
			Receipt: Receipt{Entry: seat.entry, League: create.League, Revision: int64(len(records) + 1),
				RecordedAt: at.Add(time.Duration(len(records)) * time.Hour)}})
	}
	return records
}

func TestPrivateContractEffectsChangeEveryRecipientRevisionWithoutExposingTheLedger(t *testing.T) {
	records := contractDecisionSeats(t)
	at := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	before := RecipientRevision(records, "owner.two", at)
	decision := Record{Command: RawCommand{League: "native.test", Command: ContractDecisionSet,
		Args: RawArgs{ContractDecision: &RawContractDecision{Franchise: "franchise.01", Player: "0001",
			Action: "tag"}}, Effective: "now", Visibility: "private"}, Principal: "owner.one",
		Receipt: Receipt{Entry: "private.choice", League: "native.test", Revision: int64(len(records) + 1),
			RecordedAt: at}}
	records = append(records, decision)
	changed := RecipientRevision(records, "owner.two", at)
	if changed == before || ledgerEntryVisible(decision, decision, records, "owner.two", false) {
		t.Fatal("public contract projection changed without a revision, or leaked the private ledger body")
	}
	reversal := Record{Command: RawCommand{League: "native.test", Command: Revert,
		Args: RawArgs{Entry: decision.Receipt.Entry}, Effective: "now", Visibility: "private"}, Principal: "owner.one",
		Receipt: Receipt{Entry: "private.choice.reversal", League: "native.test", Revision: int64(len(records) + 1),
			RecordedAt: at.Add(time.Hour)}}
	records = append(records, reversal)
	if reverted := RecipientRevision(records, "owner.two", at); reverted == changed ||
		ledgerEntryVisible(reversal, decision, records, "owner.two", false) {
		t.Fatal("reversing a public contract projection failed to change the private recipient token")
	}
}

func TestContractDecisionIsOwnerOnlyEvenForCommissioners(t *testing.T) {
	records := contractDecisionSeats(t)
	command := RawCommand{League: "native.test", Command: ContractDecisionSet,
		Args: RawArgs{ContractDecision: &RawContractDecision{Franchise: "franchise.01", Player: "0001", Action: "tag"}}}
	role, err := AuthorizeRecords(command, "owner.one", records)
	if err != nil || role != "owner" {
		t.Fatalf("owner cannot decide for own franchise: %q %v", role, err)
	}
	for _, principal := range []string{"owner.two", "commissioner"} {
		if _, err := AuthorizeRecords(command, principal, records); err == nil {
			t.Fatalf("%s decided for another owner's franchise", principal)
		}
	}
	definition, err := DefinitionFor(ContractDecisionSet)
	if err != nil || definition.Gravity != "G3" {
		t.Fatalf("contract decision safeguard: %+v %v", definition, err)
	}
}
