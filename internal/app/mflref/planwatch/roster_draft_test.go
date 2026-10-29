package planwatch

import (
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestRosterDraftRefusesBeforeLocksAndWrites(t *testing.T) {
	ownErr := errors.New("not yours")
	s := RosterDraft{Ready: func() error { return nil }, Own: func(string) error { return ownErr }, Snapshot: func() (snapshot.Snapshot, bool) {
		t.Fatal("captured unauthorized snapshot")
		return snapshot.Snapshot{}, false
	}}
	if _, err := s.Draft("roster.ir", "other", "0042"); !errors.Is(err, ownErr) {
		t.Fatal(err)
	}
	s.Own = func(string) error { return nil }
	if _, err := s.Draft("roster.ir", "mine", "bad"); err == nil {
		t.Fatal("invalid player accepted")
	}
	s.Snapshot = func() (snapshot.Snapshot, bool) { return snapshot.Snapshot{}, false }
	s.Lock = func() { t.Fatal("locked with no snapshot") }
	if _, err := s.Draft("roster.ir", "mine", "0042"); err == nil {
		t.Fatal("missing snapshot accepted")
	}
	receipts := []envelope.Receipt{{Spec: envelope.Spec{Intent: envelope.BidIntent}}, {Spec: envelope.Spec{Intent: "roster.ir"}}, {Spec: envelope.Spec{Intent: envelope.CutIntent}}, {Spec: envelope.Spec{Intent: "contract.extend"}}, {Spec: envelope.Spec{Intent: "mfl.trade.accept"}}}
	rail := RosterRail(receipts)
	if len(rail) != 2 || rail[0].Spec.Intent != "roster.ir" || rail[1].Spec.Intent != "mfl.trade.accept" {
		t.Fatal(rail)
	}
}
