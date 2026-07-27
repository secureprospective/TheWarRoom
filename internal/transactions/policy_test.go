package transactions

import (
	"context"
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// fakeRosterPolicy is a minimal RosterPolicy for unit-testing the pure enforcement helpers
// (checkRosterSize / checkPositionLimit / isRosterLimitReject) without a real rulebook or
// players-DB — the composition-root adapter (roster_policy_app.go) is exercised separately.
type fakeRosterPolicy struct {
	rosterSize   int
	taxiSquad    int
	ir           int
	positionCaps map[domain.Position]int
}

func (f *fakeRosterPolicy) RosterSize() int     { return f.rosterSize }
func (f *fakeRosterPolicy) TaxiSquad() int      { return f.taxiSquad }
func (f *fakeRosterPolicy) InjuredReserve() int { return f.ir }
func (f *fakeRosterPolicy) PositionLimit(pos domain.Position) int {
	return f.positionCaps[pos]
}
func (f *fakeRosterPolicy) Position(context.Context, string) (domain.Position, bool) {
	return domain.PosFlag, false
}

func TestCheckRosterSize(t *testing.T) {
	p := &fakeRosterPolicy{rosterSize: 20}

	if err := checkRosterSize(p, "F1", 19, 1); err != nil {
		t.Fatalf("adding to exactly the cap should be legal, got: %v", err)
	}
	err := checkRosterSize(p, "F1", 20, 1)
	if err == nil {
		t.Fatal("adding past the cap should be rejected")
	}
	if !isRosterLimitReject(err) {
		t.Fatalf("rejection should be an errRosterLimit, got %T: %v", err, err)
	}

	unlimited := &fakeRosterPolicy{rosterSize: 0}
	if err := checkRosterSize(unlimited, "F1", 999, 50); err != nil {
		t.Fatalf("a 0 roster-size cap means unlimited, got: %v", err)
	}
}

func TestCheckPositionLimit(t *testing.T) {
	p := &fakeRosterPolicy{positionCaps: map[domain.Position]int{domain.PosQB: 3}}

	if err := checkPositionLimit(p, "F1", domain.PosQB, 2, 1); err != nil {
		t.Fatalf("adding to exactly the per-position cap should be legal, got: %v", err)
	}
	err := checkPositionLimit(p, "F1", domain.PosQB, 3, 1)
	if err == nil {
		t.Fatal("exceeding the per-position cap should be rejected")
	}
	if !isRosterLimitReject(err) {
		t.Fatalf("rejection should be an errRosterLimit, got %T: %v", err, err)
	}

	if err := checkPositionLimit(p, "F1", domain.PosFlag, 100, 100); err != nil {
		t.Fatalf("PosFlag (unresolved position) must never be gated, got: %v", err)
	}

	unconfigured := &fakeRosterPolicy{positionCaps: map[domain.Position]int{}}
	if err := checkPositionLimit(unconfigured, "F1", domain.PosRB, 50, 10); err != nil {
		t.Fatalf("a position with no configured cap (0) means unlimited, got: %v", err)
	}
}

func TestIsRosterLimitReject_DistinguishesOtherErrors(t *testing.T) {
	if isRosterLimitReject(errors.New("some other transaction rejection")) {
		t.Fatal("a plain error must not be misread as a roster-limit rejection")
	}
	if isRosterLimitReject(nil) {
		t.Fatal("a nil error must not be misread as a roster-limit rejection")
	}
}
