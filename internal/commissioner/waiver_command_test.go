package commissioner

import "testing"

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
