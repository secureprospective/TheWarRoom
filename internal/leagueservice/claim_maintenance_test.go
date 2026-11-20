package leagueservice

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func claimMaintenanceFixture(t *testing.T) (*dueFixture, commissioner.Receipt, commissioner.Receipt) {
	t.Helper()
	f := newDueFixtureWithPlayers(t, []commissioner.NativePlayer{{ID: "0100", Franchise: "franchise.03",
		Status: domain.RosterActive, Salary: 100000000, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA}})
	first := f.execute(t, "owner.two", commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "0099", Franchise: "franchise.02"}})
	second := f.execute(t, "owner.three", commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "0100", Franchise: "franchise.03"}})
	return f, first, second
}

func TestOwnerReordersOwnClaimsAndWaiverRunHonorsOrder(t *testing.T) {
	f, first, second := claimMaintenanceFixture(t)
	firstClaim := f.execute(t, "owner.one", commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Window: first.Entry, Franchise: "franchise.01"}})
	secondClaim := f.execute(t, "owner.one", commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Window: second.Entry, Franchise: "franchise.01"}})
	order := f.raw(t, commissioner.WaiverClaimOrderSet, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Franchise: "franchise.01", Order: []string{secondClaim.Entry, firstClaim.Entry}}})
	beforePreview := len(f.events(t))
	preview, err := f.store.PreviewCommissionerWithFacts(t.Context(), order, "owner.one", nil,
		commissioner.PreviewCatalogue{}, f.facts, time.UTC)
	if err != nil || len(preview.Blockers) != 0 {
		t.Fatalf("claim order preview = %+v, %v", preview, err)
	}
	if got := len(f.events(t)); got != beforePreview {
		t.Fatalf("preview wrote a record: before %d after %d", beforePreview, got)
	}
	if _, err := f.store.ExecuteCommissionerWithFacts(t.Context(), order, "owner.one", nil, f.facts, time.UTC); err != nil {
		t.Fatal(err)
	}

	reading, err := f.store.NativeWaivers(t.Context(), f.league, "franchise.01", "owner.one", f.clock.Now(), time.UTC, f.facts)
	if err != nil {
		t.Fatal(err)
	}
	if len(reading.Claims) != 2 || reading.Claims[0].ID != secondClaim.Entry || reading.Claims[0].Position != 1 ||
		reading.Claims[1].ID != firstClaim.Entry || reading.Claims[1].Position != 2 {
		t.Fatalf("claim reading did not return the saved order: %+v", reading.Claims)
	}

	f.clock.instant.Store(first.RecordedAt.Add(24 * time.Hour).UnixNano())
	if err := f.store.ProcessDueWaivers(t.Context(), f.league, f.facts); err != nil {
		t.Fatal(err)
	}
	var processed []string
	for _, event := range f.events(t) {
		if raw := event.Payload.Commissioner.Command; raw.Command == commissioner.WaiverProcess {
			processed = append(processed, raw.Args.Waiver.Window)
		}
	}
	if got, want := strings.Join(processed, ","), strings.Join([]string{second.Entry, first.Entry}, ","); got != want {
		t.Fatalf("waiver run order = %s, want %s", got, want)
	}
}

func TestOwnerCannotReorderAnotherFranchiseClaim(t *testing.T) {
	f, first, second := claimMaintenanceFixture(t)
	own := f.execute(t, "owner.one", commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Window: first.Entry, Franchise: "franchise.01"}})
	other := f.execute(t, "owner.two", commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Window: second.Entry, Franchise: "franchise.02"}})
	before := len(f.events(t))
	raw := f.raw(t, commissioner.WaiverClaimOrderSet, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Franchise: "franchise.01", Order: []string{own.Entry, other.Entry}}})
	if _, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "owner.one", nil, f.facts, time.UTC); err == nil {
		t.Fatal("reordering another franchise's claim succeeded")
	}
	if got := len(f.events(t)); got != before {
		t.Fatalf("refused reorder appended a record: before %d after %d", before, got)
	}
}

func TestOwnerCancelsOwnPendingClaim(t *testing.T) {
	f, first, _ := claimMaintenanceFixture(t)
	claim := f.execute(t, "owner.one", commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Window: first.Entry, Franchise: "franchise.01"}})
	f.execute(t, "owner.one", commissioner.WaiverClaimCancel, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Claim: claim.Entry, Franchise: "franchise.01"}})

	reading, err := f.store.NativeWaivers(t.Context(), f.league, "franchise.01", "owner.one", f.clock.Now(), time.UTC, f.facts)
	if err != nil {
		t.Fatal(err)
	}
	if len(reading.Claims) != 0 {
		t.Fatalf("cancelled claim remains in owner reading: %+v", reading.Claims)
	}
}

func TestOwnerCannotCancelAnotherFranchiseClaim(t *testing.T) {
	f, _, second := claimMaintenanceFixture(t)
	claim := f.execute(t, "owner.two", commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Window: second.Entry, Franchise: "franchise.02"}})
	before := len(f.events(t))
	raw := f.raw(t, commissioner.WaiverClaimCancel, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Claim: claim.Entry, Franchise: "franchise.01"}})
	if _, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "owner.one", nil, f.facts, time.UTC); err == nil ||
		!strings.Contains(err.Error(), "own waiver claims") {
		t.Fatalf("cancelling another franchise's claim: %v", err)
	}
	if got := len(f.events(t)); got != before {
		t.Fatalf("refused cancellation appended a record: before %d after %d", before, got)
	}
}

func TestOwnerCannotCancelClaimAfterWaiverRun(t *testing.T) {
	f, first, _ := claimMaintenanceFixture(t)
	claim := f.execute(t, "owner.one", commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Window: first.Entry, Franchise: "franchise.01"}})
	f.clock.instant.Store(first.RecordedAt.Add(24 * time.Hour).UnixNano())
	if err := f.store.ProcessDueWaivers(t.Context(), f.league, f.facts); err != nil {
		t.Fatal(err)
	}
	before := len(f.events(t))
	raw := f.raw(t, commissioner.WaiverClaimCancel, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Claim: claim.Entry, Franchise: "franchise.01"}})
	if _, err := f.store.ExecuteCommissionerWithFacts(t.Context(), raw, "owner.one", nil, f.facts, time.UTC); err == nil ||
		!strings.Contains(err.Error(), "already processed") {
		t.Fatalf("cancelling a processed claim: %v", err)
	}
	if got := len(f.events(t)); got != before {
		t.Fatalf("refused cancellation appended a record: before %d after %d", before, got)
	}
}

func TestClaimMaintenanceEventsStayPrivate(t *testing.T) {
	f, first, _ := claimMaintenanceFixture(t)
	claim := f.execute(t, "owner.one", commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Window: first.Entry, Franchise: "franchise.01"}})
	f.execute(t, "owner.one", commissioner.WaiverClaimOrderSet, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Franchise: "franchise.01", Order: []string{claim.Entry}}})
	f.execute(t, "owner.one", commissioner.WaiverClaimCancel, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Claim: claim.Entry, Franchise: "franchise.01"}})
	events := f.events(t)
	seen := map[string]bool{}
	for _, event := range events {
		raw := event.Payload.Commissioner.Command
		if raw.Command != commissioner.WaiverClaimOrderSet && raw.Command != commissioner.WaiverClaimCancel {
			continue
		}
		if raw.Visibility != "private" {
			t.Fatalf("%s event visibility = %q", raw.Command, raw.Visibility)
		}
		seen[raw.Command] = true
	}
	if !seen[commissioner.WaiverClaimOrderSet] || !seen[commissioner.WaiverClaimCancel] {
		t.Fatalf("maintenance events missing from ledger: %v", seen)
	}
}
