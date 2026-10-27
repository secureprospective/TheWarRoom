package market

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// DueBids uses the manual decision policy, including current roster and cap checks. A tie remains
// undecided for the commissioner; waivers must clear before any automatic award.
func DueBids(league string, records []commissioner.Record, at time.Time, facts nativecheck.Facts) ([]string, error) {
	f, err := commissioner.NewFrame(league, records, at, facts, time.UTC)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return nil, err
	}
	return dueBidsFrame(f)
}

func dueBidsFrame(f commissioner.Frame) ([]string, error) {
	records, at, facts := f.Records, f.At, f.Facts
	assets, state := f.Assets, f.Book
	// A league with no starting assets has no bids; its season may not even be named yet.
	due := []string{}
	if !assets.Seeded {
		return due, nil
	}
	if err := f.ReadRules(); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return nil, err
	}
	rules := f.Rules
	stand := bidStandFor(rules)
	if stand.State != leaguerules.IsKnown || stand.Value <= 0 {
		return due, nil
	}
	x := bidExecution{assets: assets, state: state, rules: rules, records: records, now: at, facts: facts, zone: time.UTC}
	for i := range state.Bids.Auctions {
		auction := &state.Bids.Auctions[i]
		if auction.Decided || at.Before(auction.StandingAt().Add(stand.Value)) {
			continue
		}
		raw := commissioner.RawCommand{Command: commissioner.BidsProcess, Args: commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: auction.Player}}}
		if err := x.decide(&raw); err != nil {
			continue
		}
		if len(raw.Args.BidOutcome.Tied) == 0 {
			due = append(due, auction.Player)
		}
	}
	slices.Sort(due)
	return due, nil
}

// CheckDueBid repeats the worker's eligibility check under the append lock, so a stale clock scan
// cannot award a player after a new bid, waiver restoration or commissioner rule change.
func CheckDueBid(raw commissioner.RawCommand, records []commissioner.Record, at time.Time, facts nativecheck.Facts) error {
	due, err := DueBids(raw.League, records, at, facts)
	if err != nil {
		return err
	}
	if raw.Command != commissioner.BidsProcess || raw.Args.Bid == nil || !slices.Contains(due, raw.Args.Bid.Player) {
		return fmt.Errorf("only an untied bid that has stood its full time and cleared waivers can be awarded automatically")
	}
	return nil
}

// BidOffseasonAt recognizes an otherwise undated offseason only after a held rollover and before
// September of the named season. An unrecorded rollover never becomes an invented season boundary.
func BidOffseasonAt(records []commissioner.Record, at time.Time, season int) (bool, error) {
	starts, err := commissioner.RolloverInstants(commissioner.RecordsThrough(records, at))
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return false, err
	}
	rules, err := commissioner.RulesNow(records, at)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return false, err
	}
	if rules.Calendar.Timezone.State != leaguerules.IsKnown {
		return false, nil
	}
	zone, err := time.LoadLocation(rules.Calendar.Timezone.Value)
	if err != nil {
		return false, fmt.Errorf("read bidding calendar timezone: %w", err)
	}
	september := time.Date(season, time.September, 1, 0, 0, 0, 0, zone)
	for _, start := range starts {
		if !at.Before(start) && at.Before(september) && start.Before(september) {
			return true, nil
		}
	}
	return false, nil
}
