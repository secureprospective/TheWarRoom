package commissioner

import (
	"testing"
	"time"
)

func TestCaretakerLineupSeatEndsWhenOwnerArrives(t *testing.T) {
	records := lineupGovernance()[:1]
	records[0].Command.Args.Season = 2026
	caretaker := Record{Receipt: Receipt{Entry: "caretaker"}, Principal: "boss", Command: RawCommand{League: "native.test", Command: CaretakerSeat, Args: RawArgs{People: &RawPeople{Franchise: "franchise.01", Principal: "boss"}}}}
	if _, err := AuthorizeRecords(caretaker.Command, "boss", records); err != nil {
		t.Fatal(err)
	}
	if err := CheckGovernanceAppend(caretaker.Command, "boss", records, time.Now()); err != nil {
		t.Fatal(err)
	}
	records = append(records, caretaker)
	raw := lineupRecord("lineup", LineupSet, 1, "1001").Command
	raw.League = "native.test"
	if _, err := AuthorizeRecords(raw, "boss", records); err != nil {
		t.Fatal(err)
	}
	people, err := PeopleReading("native.test", "visitor", records)
	if err != nil || people.Seats[0].Caretaker != "boss" || people.Seats[0].Owner != "Vacant" {
		t.Fatalf("caretaker not public: %+v %v", people, err)
	}
	records = append(records, lineupGovernance()[1])
	if _, err := AuthorizeRecords(raw, "boss", records); err == nil {
		t.Fatal("former caretaker retained lineup authority")
	}
	if err := CheckGovernanceAppend(caretaker.Command, "boss", records, time.Now()); err == nil {
		t.Fatal("caretaker seated over owner")
	}
}
