package commissioner

import (
	"slices"
	"testing"
)

func TestRosterAddUsesTypedOwnerCommandPolicy(t *testing.T) {
	if !WaiverCommand(RosterAdd) || !RosterTimed(RosterAdd) || OwnerWaiverAction(RosterAdd) != "ufa_bid" {
		t.Fatal("roster.add is missing the shared market-command path")
	}
	definition, err := DefinitionFor(RosterAdd)
	if err != nil || definition.Gravity != "G3" {
		t.Fatalf("roster.add definition = %+v, %v; want G3", definition, err)
	}
	valid := RawArgs{Waiver: &RawWaiverArgs{Player: "6001", Franchise: "franchise.01", Drop: "0102"}}
	if err := valid.Validate(RosterAdd); err != nil {
		t.Fatalf("valid typed add refused: %v", err)
	}
	for name, args := range map[string]RawWaiverArgs{
		"missing player":      {Franchise: "franchise.01"},
		"missing franchise":   {Player: "6001"},
		"same add and drop":   {Player: "6001", Franchise: "franchise.01", Drop: "6001"},
		"invalid drop":        {Player: "6001", Franchise: "franchise.01", Drop: "not-a-player"},
		"untrimmed franchise": {Player: "6001", Franchise: " franchise.01 "},
	} {
		t.Run(name, func(t *testing.T) {
			if err := (RawArgs{Waiver: &args}).Validate(RosterAdd); err == nil {
				t.Fatal("invalid roster.add input accepted")
			}
		})
	}
	if err := (RawArgs{Waiver: &RawWaiverArgs{Player: "6001", Franchise: "franchise.01", Drop: "0102"}}).Validate(RosterCut); err == nil {
		t.Fatal("roster.cut accepted an add-only drop field")
	}
}

func TestRosterAddIsOwnerOnlyEvenForCommissioners(t *testing.T) {
	records := contractDecisionSeats(t)
	command := RawCommand{League: "native.test", Command: RosterAdd,
		Args: RawArgs{Waiver: &RawWaiverArgs{Player: "6001", Franchise: "franchise.01"}}}
	role, err := AuthorizeRecords(command, "owner.one", records)
	if err != nil || role != "owner" {
		t.Fatalf("owner cannot add to own franchise: %q %v", role, err)
	}
	for _, principal := range []string{"owner.two", "commissioner"} {
		if _, err := AuthorizeRecords(command, principal, records); err == nil {
			t.Fatalf("%s added a free agent to another owner's franchise", principal)
		}
	}
}

func TestRosterAddIsNotADeputyDelegation(t *testing.T) {
	records := permissionTestLedger(t)
	command := RawCommand{League: "native.test", Command: RosterAdd,
		Args: RawArgs{Waiver: &RawWaiverArgs{Player: "6001", Franchise: "franchise.02"}}}
	if _, err := AuthorizeRecords(command, "full-deputy", records); err == nil {
		t.Fatal("a deputy with every permission added a player to another franchise")
	}
}

func TestClaimMaintenanceUsesPrivateOwnerWaiverCommands(t *testing.T) {
	for command, args := range map[string]RawWaiverArgs{
		WaiverClaimOrderSet: {Franchise: "franchise.01", Order: []string{"claim.one"}},
		WaiverClaimCancel:   {Franchise: "franchise.01", Claim: "claim.one"},
	} {
		t.Run(command, func(t *testing.T) {
			if !WaiverCommand(command) || OwnerWaiverAction(command) != "waiver_claim" {
				t.Fatalf("%s is not on the owner waiver path", command)
			}
			if err := (RawArgs{Waiver: &args}).Validate(command); err != nil {
				t.Fatalf("valid %s args refused: %v", command, err)
			}
			raw := RawCommand{League: "native.test", Command: command, Args: RawArgs{Waiver: &args},
				Expected: 1, Request: "claim-maintenance", Effective: "now", Visibility: "private"}
			if err := raw.Validate(); err != nil {
				t.Fatalf("valid private %s command refused: %v", command, err)
			}
			raw.Visibility = "league"
			if err := raw.Validate(); err == nil {
				t.Fatalf("%s accepted public placement", command)
			}
		})
	}
	for command, args := range map[string]RawWaiverArgs{
		WaiverClaimOrderSet: {Franchise: "franchise.01"},
		WaiverClaimCancel:   {Franchise: "franchise.01"},
	} {
		if err := (RawArgs{Waiver: &args}).Validate(command); err == nil {
			t.Fatalf("%s accepted incomplete args", command)
		}
		if _, err := DefinitionFor(command); err != nil {
			t.Fatalf("%s has no command definition: %v", command, err)
		}
	}
}

func TestClaimMaintenanceIsOwnerOnlyEvenForCommissioners(t *testing.T) {
	records := contractDecisionSeats(t)
	for command, args := range map[string]RawWaiverArgs{
		WaiverClaimOrderSet: {Franchise: "franchise.01", Order: []string{"claim.one"}},
		WaiverClaimCancel:   {Franchise: "franchise.01", Claim: "claim.one"},
	} {
		raw := RawCommand{League: "native.test", Command: command, Args: RawArgs{Waiver: &args}}
		if role, err := AuthorizeRecords(raw, "owner.one", records); err != nil || role != "owner" {
			t.Errorf("owner's %s authorization = %q, %v", command, role, err)
		}
		for _, principal := range []string{"owner.two", "commissioner"} {
			if _, err := AuthorizeRecords(raw, principal, records); err == nil {
				t.Errorf("%s ran %s for another franchise", principal, command)
			}
		}
	}
}

func TestClaimMaintenanceIsNotADeputyDelegation(t *testing.T) {
	records := permissionTestLedger(t)
	for command, args := range map[string]RawWaiverArgs{
		WaiverClaimOrderSet: {Franchise: "franchise.02", Order: []string{"claim.one"}},
		WaiverClaimCancel:   {Franchise: "franchise.02", Claim: "claim.one"},
	} {
		if _, err := AuthorizeRecords(RawCommand{League: "native.test", Command: command, Args: RawArgs{Waiver: &args}},
			"full-deputy", records); err == nil {
			t.Errorf("deputy with every permission ran %s for another franchise", command)
		}
	}
}

func TestClaimCancelCannotCrossPrincipalsOnSameFranchise(t *testing.T) {
	book := MarketBook{Windows: []WaiverWindow{{ID: "window.one", Claims: []WaiverBookClaim{{
		ID: "former.claim", Franchise: "franchise.01", Principal: "former.owner",
	}}}}}
	record := Record{Principal: "owner.one", Command: RawCommand{Command: WaiverClaimCancel,
		Args: RawArgs{Waiver: &RawWaiverArgs{Franchise: "franchise.01", Claim: "former.claim"}}}}
	if err := projectOtherWaiver(nil, &book, record, nil); err == nil {
		t.Fatal("current franchise owner cancelled a private claim from another principal")
	}
}

func TestClaimOrderPreservesOtherPrincipalsQueueSlots(t *testing.T) {
	book := MarketBook{Windows: []WaiverWindow{{Claims: []WaiverBookClaim{
		{ID: "old.one", Franchise: "franchise.01", Principal: "former.owner"},
		{ID: "new.one", Franchise: "franchise.01", Principal: "owner.one"},
		{ID: "old.two", Franchise: "franchise.01", Principal: "former.owner"},
		{ID: "new.two", Franchise: "franchise.01", Principal: "owner.one"},
	}}}}
	if err := book.SetClaimOrder("franchise.01", "owner.one", nil, []string{"new.two", "new.one"}); err != nil {
		t.Fatal(err)
	}
	if got, want := book.PendingClaimIDs("franchise.01"), []string{"old.one", "new.two", "old.two", "new.one"}; !slices.Equal(got, want) {
		t.Fatalf("reordered queue = %v, want %v", got, want)
	}
}

func TestClaimMaintenanceNeverLeaksToOtherLedgerReaders(t *testing.T) {
	entries := []Record{
		{Principal: "owner.one", Receipt: Receipt{Entry: "claim.one"}, Command: RawCommand{Command: WaiverClaim,
			Visibility: "private", Args: RawArgs{Waiver: &RawWaiverArgs{Window: "window.one", Franchise: "franchise.01"}}}},
		{Principal: "owner.one", Receipt: Receipt{Entry: "order.one"}, Command: RawCommand{Command: WaiverClaimOrderSet,
			Visibility: "private", Args: RawArgs{Waiver: &RawWaiverArgs{Franchise: "franchise.01", Order: []string{"claim.one"}}}}},
		{Principal: "owner.one", Receipt: Receipt{Entry: "cancel.one"}, Command: RawCommand{Command: WaiverClaimCancel,
			Visibility: "private", Args: RawArgs{Waiver: &RawWaiverArgs{Franchise: "franchise.01", Claim: "claim.one"}}}},
	}
	for _, entry := range entries {
		if !privateWaiverRecord(entry, entries) {
			t.Fatalf("%s is not classified as private", entry.Command.Command)
		}
		if !ledgerEntryVisible(entry, entry, entries, "owner.one", false) {
			t.Fatalf("owner cannot read own %s", entry.Command.Command)
		}
		if ledgerEntryVisible(entry, entry, entries, "other.owner", true) {
			t.Fatalf("privileged reader can read private %s", entry.Command.Command)
		}
	}
}
