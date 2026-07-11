package transactions_test

import (
	"context"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// draftDir is a fake transactions.Directory mapping seeded ids to a MFL draft year — the experience
// source for the §6 min-salary floor. An id absent from the map resolves to ok=false (no players-DB
// record), and a draft year of 0 models MFL's undrafted sentinel (a real draft year is positive).
type draftDir map[string]int

func (d draftDir) Facts(mflID string) (normalize.PlayerFacts, bool) {
	y, ok := d[mflID]
	if !ok {
		return normalize.PlayerFacts{}, false
	}
	if y <= 0 { // undrafted sentinel typed away at normalize → present record, no draft year
		return normalize.PlayerFacts{}, true
	}
	return normalize.PlayerFacts{DraftYear: y, HasDraftYear: true}, true
}

// signUSD converts whole dollars to exact cents (domain.Money), all on the $10k grid here.
func signUSD(d int64) domain.Money { return domain.Money(d * 100) }

// waiveIntoPool cuts a rostered seed player so he is a signable free agent.
func waiveIntoPool(t *testing.T, c *transactions.Coordinator, mflID string) {
	t.Helper()
	if _, err := c.Execute(context.Background(), transactions.Waiver{MFLID: mflID}); err != nil {
		t.Fatalf("waive %s: %v", mflID, err)
	}
}

// TestIntegration_SignEnforcesVeteranFloor proves the §6 floor is enforced from the resolved draft
// year: a 2020 draftee in the 2026 season has 6 years' experience → the $530k floor, so a $500k
// signing is rejected and a $530k signing clears.
func TestIntegration_SignEnforcesVeteranFloor(t *testing.T) {
	s, c := signStore(t)
	ctx := context.Background()
	waiveIntoPool(t, c, "0011")
	dir := draftDir{"0011": 2020} // season 2026 − 2020 = 6 years → $530k floor

	if _, err := c.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(500_000), Years: 2}, dir); err == nil {
		t.Fatal("signed a 6-year veteran at $500k (below the $530k §6 floor), want rejection")
	}
	if _, err := c.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(530_000), Years: 2}, dir); err != nil {
		t.Fatalf("sign a 6-year veteran at the $530k floor: %v", err)
	}
	if _, ok := s.Reader().Player("0011"); !ok {
		t.Fatal("after a floor-clearing signing, 0011 is not rostered")
	}
}

// TestIntegration_SignSentinelDraftYearGetsRookieFloor gives the plausibility window real teeth:
// MFL's "1970" epoch placeholder must resolve to UNKNOWN experience → the rookie $330k floor (the
// lenient direction), NOT season−1970 = 56 years → the highest $630k floor. A $400k signing that
// would be REJECTED under the spurious $630k floor must be ACCEPTED under the correct $330k rookie
// floor — so a regression that trusted the sentinel is caught.
func TestIntegration_SignSentinelDraftYearGetsRookieFloor(t *testing.T) {
	_, c := signStore(t)
	ctx := context.Background()
	waiveIntoPool(t, c, "0011")
	dir := draftDir{"0011": 1970} // MFL undrafted/unknown epoch sentinel

	// Below the rookie floor → still rejected (the floor IS enforced, just at the rookie level).
	if _, err := c.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(300_000), Years: 1}, dir); err == nil {
		t.Fatal("signed at $300k (below the $330k rookie floor), want rejection")
	}
	// Above the rookie floor but BELOW the spurious 56-year $630k floor → must be accepted.
	if _, err := c.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(400_000), Years: 1}, dir); err != nil {
		t.Fatalf("sign at $400k with a 1970 sentinel draft year: %v — the sentinel must yield the rookie floor, not $630k", err)
	}
}

// TestIntegration_SignNoDraftDataGetsRookieFloor proves the missing-data policy: a free agent with
// no players-DB draft year (commissioner-created / undrafted "0" sentinel / absent) is floored at
// the rookie minimum — $300k rejected, $330k accepted.
func TestIntegration_SignNoDraftDataGetsRookieFloor(t *testing.T) {
	_, c := signStore(t)
	ctx := context.Background()
	waiveIntoPool(t, c, "0011")
	dir := draftDir{"0011": 0} // present record, undrafted sentinel → HasDraftYear=false

	if _, err := c.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(300_000), Years: 1}, dir); err == nil {
		t.Fatal("signed at $300k with no draft data (below the $330k rookie floor), want rejection")
	}
	if _, err := c.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(330_000), Years: 1}, dir); err != nil {
		t.Fatalf("sign at the $330k rookie floor with no draft data: %v", err)
	}
}

// TestIntegration_SignRookieDraftClassFloor proves a current-season draftee is 0 experience → the
// rookie floor (the draft class = season boundary), independent of any missing-data path.
func TestIntegration_SignRookieDraftClassFloor(t *testing.T) {
	_, c := signStore(t)
	ctx := context.Background()
	waiveIntoPool(t, c, "0011")
	dir := draftDir{"0011": 2026} // drafted this season → 0 years experience → $330k floor

	if _, err := c.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(320_000), Years: 1}, dir); err == nil {
		t.Fatal("signed a rookie at $320k (below the $330k floor), want rejection")
	}
	if _, err := c.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(330_000), Years: 1}, dir); err != nil {
		t.Fatalf("sign a rookie at the $330k floor: %v", err)
	}
}

// TestIntegration_SignPlausibilityWindowBoundary pins the EXACT lower edge of the draft-year
// plausibility window (season − maxPlausibleCareer = 2026 − 30 = 1996). A draft year AT the bound
// (1996 → 30 yrs → the $630k 10+ floor) is honored; ONE year past it (1995 → 31 yrs, implausible)
// collapses to the rookie floor. Without this a narrowed window (a smaller maxPlausibleCareer, or
// `>=` → `>`) would silently mis-floor real veterans and the 1970-sentinel test would not notice.
func TestIntegration_SignPlausibilityWindowBoundary(t *testing.T) {
	ctx := context.Background()

	// AT the bound: 1996 → 30 years → $630k floor. $600k is rejected, $630k clears.
	_, c := signStore(t)
	waiveIntoPool(t, c, "0011")
	atBound := draftDir{"0011": 1996}
	if _, err := c.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(600_000), Years: 1}, atBound); err == nil {
		t.Fatal("signed a 1996 draftee (30 yrs → $630k floor) at $600k, want rejection — the window's lower bound must still honor experience")
	}
	if _, err := c.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(630_000), Years: 1}, atBound); err != nil {
		t.Fatalf("sign a 1996 draftee at the $630k floor: %v", err)
	}

	// ONE past the bound: 1995 → 31 yrs → implausible → rookie floor. $300k rejected, $330k clears.
	_, c2 := signStore(t)
	waiveIntoPool(t, c2, "0011")
	pastBound := draftDir{"0011": 1995}
	if _, err := c2.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(300_000), Years: 1}, pastBound); err == nil {
		t.Fatal("signed at $300k with a 1995 (implausible, 31-yr) draft year, want rejection at the rookie floor")
	}
	if _, err := c2.ExecuteSign(ctx, transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(330_000), Years: 1}, pastBound); err != nil {
		t.Fatalf("sign at the $330k rookie floor with an implausible draft year: %v", err)
	}
}

// TestIntegration_ExecuteSignNilDirectoryFails proves ExecuteSign fails loud without the players-DB
// join it needs to resolve the §6 experience input.
func TestIntegration_ExecuteSignNilDirectoryFails(t *testing.T) {
	_, c := signStore(t)
	if _, err := c.ExecuteSign(context.Background(), transactions.Sign{MFLID: "0011", FranchiseID: "0002", Salary: signUSD(500_000), Years: 1}, nil); err == nil {
		t.Fatal("ExecuteSign with a nil directory succeeded, want a wiring error")
	}
}
