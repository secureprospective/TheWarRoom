package governance

import "testing"

func TestCaretakerOnlyWhileVacantAndOwnerAssignmentClosesIt(t *testing.T) {
	s := change(t, State{}, LeagueCreated{Creator: "boss", Franchises: []FranchiseID{"f00", "f01"}})
	rejected(t, s, CaretakerSeated{Franchise: "f00", Principal: "other"})
	s = change(t, s, CaretakerSeated{Franchise: "f00", Principal: "boss"})
	if !s.HoldsSeat("f00", "boss") || s.HoldsSeat("f01", "boss") {
		t.Fatal("caretaker authority escaped its seat")
	}
	checkpoint := s.Checkpoint()
	resumed, err := Resume(checkpoint, nil, 2)
	if err != nil || !resumed.HoldsSeat("f00", "boss") {
		t.Fatalf("caretaker lost at checkpoint: %v", err)
	}
	s = change(t, s, SeatReplaced{Franchise: "f00", Principal: "owner"})
	if s.HoldsSeat("f00", "boss") || !s.HoldsSeat("f00", "owner") {
		t.Fatal("owner did not close caretaker")
	}
	rejected(t, s, CaretakerSeated{Franchise: "f00", Principal: "boss"})
	s = change(t, s, SeatRemoved{Franchise: "f00"})
	if s.HoldsSeat("f00", "boss") {
		t.Fatal("removing owner resurrected caretaker")
	}
}
