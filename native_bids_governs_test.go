package main

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// ownerOf names the seat that holds each franchise in the bid fixture.
func ownerOf(franchise string) string {
	return map[string]string{"franchise.01": "owner.one", "franchise.02": "owner.two", "franchise.03": "owner.three"}[franchise]
}

// probe bids as a franchise and, when the bid is accepted, withdraws it, so the auction is left as found.
// It returns the points the app gave the bid, or the refusal.
func (f *bidFixture) probe(t *testing.T, franchise, player string, salary int64, years int) (string, error) {
	t.Helper()
	principal := ownerOf(franchise)
	if err := f.tryPlace(t, principal, franchise, player, salary, years); err != nil {
		return "", err
	}
	row := playerRow(f.bids(t, principal, franchise), player)
	if row == nil || row.YourBid == nil {
		t.Fatal("an accepted bid is missing from the owner's reading")
	}
	if err := f.withdraw(t, principal, franchise, commissioner.Receipt{Entry: row.YourBid.ID}); err != nil {
		t.Fatal(err)
	}
	return row.YourBid.Points, nil
}

func (f *bidFixture) accepted(t *testing.T, franchise, player string, salary int64, years int) string {
	t.Helper()
	points, err := f.probe(t, franchise, player, salary, years)
	if err != nil {
		t.Fatalf("%s bid on %s refused: %v", franchise, player, err)
	}
	return points
}

func (f *bidFixture) refused(t *testing.T, franchise, player string, salary int64, years int, words string) {
	t.Helper()
	if _, err := f.probe(t, franchise, player, salary, years); err == nil || !strings.Contains(err.Error(), words) {
		t.Fatalf("%s bid on %s: %v, want a refusal containing %q", franchise, player, err, words)
	}
}

// governs is the wiring proof of one bid rule: the value is set through the real command and its consequence
// read; a compensating revert is made and the old consequence read again; then the database is reopened and
// both are read and repeated from the replayed ledger.
func (f *bidFixture) governs(t *testing.T, field, value string, was, now func()) {
	t.Helper()
	was()
	for range 2 {
		change := f.set(t, field, value, "now")
		now()
		f.restart(t)
		now()
		f.revert(t, change)
		was()
		f.restart(t)
		was()
	}
}

func TestNativeBidOneYearLimitGovernsRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.governs(t, "bids.oneYearMaximum", `1000000000`,
		func() { f.accepted(t, "franchise.01", "5002", 11*million, 1) },
		func() { f.refused(t, "franchise.01", "5002", 11*million, 1, "no more than $10,000,000") })
}

func TestNativeBidTierYearsGovernRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	tiers := `[{"Salary":{"MaximumUnlimited":true,"Min":1200000000},"MinimumYears":4,"LowerInclusive":false}]`
	f.governs(t, "bids.tiers", tiers,
		func() { f.accepted(t, "franchise.01", "5002", 13*million, 2) },
		func() { f.refused(t, "franchise.01", "5002", 13*million, 2, "needs at least 4 years") })
}

func TestNativeBidStepGovernsRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.two", "franchise.02", "5001", 5*million, 1)
	f.governs(t, "bids.minimumIncrement", `5000`,
		func() { f.accepted(t, "franchise.01", "5001", 51*million/10, 1) },
		func() { f.refused(t, "franchise.01", "5001", 51*million/10, 1, "at least 5.500") })
}

func TestNativeBidSnipeMarkGovernsRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.two", "franchise.02", "5001", 5*million, 1)
	f.advance(10 * time.Hour)
	f.governs(t, "bids.snipeAfter", `28800000000000`,
		func() { f.accepted(t, "franchise.01", "5001", 51*million/10, 1) },
		func() { f.refused(t, "franchise.01", "5001", 51*million/10, 1, "late bid must beat") })
}

func TestNativeBidSnipeStepGovernsRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.two", "franchise.02", "5001", 5*million, 1)
	f.advance(21 * time.Hour)
	f.governs(t, "bids.snipeIncrement", `5000`,
		func() { f.refused(t, "franchise.01", "5001", 55*million/10, 1, "at least 6.000") },
		func() { f.accepted(t, "franchise.01", "5001", 55*million/10, 1) })
}

func TestNativeBidYearMultipliersGovernPoints(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.governs(t, "bids.yearMultipliers", `[10000,15000,14000,16000]`,
		func() {
			if got := f.accepted(t, "franchise.01", "5002", 5*million, 2); got != "6.000" {
				t.Fatalf("two years at $5M scored %s", got)
			}
		},
		func() {
			if got := f.accepted(t, "franchise.01", "5002", 5*million, 2); got != "7.500" {
				t.Fatalf("two years at $5M scored %s under the league's table", got)
			}
		})
}

func TestNativeBidPointDecimalsGovernTheReading(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.governs(t, "bids.pointDecimals", `1`,
		func() { f.accepted(t, "franchise.01", "5002", 5*million, 1) },
		func() {
			if got := f.accepted(t, "franchise.01", "5002", 5*million, 1); got != "5.0" {
				t.Fatalf("points read %s with one decimal place", got)
			}
		})
}

func TestNativeBidActiveLimitGovernsRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.one", "franchise.01", "5001", 5*million, 1)
	f.governs(t, "bids.maximumActive", `1`,
		func() { f.accepted(t, "franchise.01", "5002", 5*million, 1) },
		func() { f.refused(t, "franchise.01", "5002", 5*million, 1, "already have 1 active bid,") })
}

func TestNativeBidModeGovernsWhatOtherOwnersSee(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.two", "franchise.02", "5001", 5*million, 1)
	seen := func() bool {
		row := playerRow(f.bids(t, "owner.three", "franchise.03"), "5001")
		return row != nil && row.Leader != nil && row.Leader.Franchise == "franchise.02"
	}
	f.governs(t, "bids.ufa", `"blind"`,
		func() {
			if !seen() {
				t.Fatal("open bidding hid the leading bid")
			}
		},
		func() {
			if seen() || playerRow(f.bids(t, "owner.three", "franchise.03"), "5001") != nil {
				t.Fatal("blind bidding showed another owner's bid")
			}
		})
}

func TestNativeBidMultiYearCloseGovernsRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.season.week = 11
	f.governs(t, "bids.multiYearEndWeek", `10`,
		func() { f.accepted(t, "franchise.01", "5002", 5*million, 2) },
		func() { f.refused(t, "franchise.01", "5002", 5*million, 2, "closed at the end of week 10") })
}

func TestNativeBidPlayoffOnlyWeekGovernsRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.season.week = 13
	f.season.playoff = map[string]nativecheck.PlayoffStanding{"franchise.02": nativecheck.PlayoffOut}
	f.governs(t, "bids.playoffOnlyAfterWeek", `14`,
		func() { f.refused(t, "franchise.02", "5002", 5*million, 1, "only playoff teams may bid") },
		func() { f.accepted(t, "franchise.02", "5002", 5*million, 1) })
}

func TestNativeBidPlayoffYearsGovernRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.season.week = 13
	f.season.playoff = map[string]nativecheck.PlayoffStanding{"franchise.01": nativecheck.PlayoffIn}
	f.governs(t, "bids.playoffMaximumYears", `2`,
		func() { f.refused(t, "franchise.01", "5002", 5*million, 2, "1 year at most") },
		func() { f.accepted(t, "franchise.01", "5002", 5*million, 2) })
}

func TestNativeBidPlayoffCapLiftGovernsRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.season.week = 14
	f.season.playoff = map[string]nativecheck.PlayoffStanding{"franchise.01": nativecheck.PlayoffIn}
	f.governs(t, "bids.playoffCapLifted", `false`,
		func() { f.accepted(t, "franchise.01", "5002", 30*million, 1) },
		func() { f.refused(t, "franchise.01", "5002", 30*million, 1, "no more than $12,000,000") })
}
