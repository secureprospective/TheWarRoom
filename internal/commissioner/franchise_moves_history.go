package commissioner

import (
	"fmt"
	"slices"
	"strings"
)

// RawFranchiseMovesHistory selects one season of the append-only native ledger.
type RawFranchiseMovesHistory struct {
	League    string `json:"league"`
	Franchise string `json:"franchise"`
	Season    int    `json:"season"`
}

func (r RawFranchiseMovesHistory) Validate() error {
	if (!strings.HasPrefix(r.League, "native.") || r.League == "native." || len(r.League) > 128 || strings.TrimSpace(r.League) != r.League) || strings.TrimSpace(r.Franchise) == "" || strings.TrimSpace(r.Franchise) != r.Franchise || len(r.Franchise) > 128 || r.Season < 1920 || r.Season > 9999 {
		return fmt.Errorf("choose a league, franchise and season")
	}
	return nil
}

// FranchiseMovesHistoryReading filters the safe audience reading, never raw command payloads.
// Corrections and their originals remain separate rows. Seasons follow the league clock,
// not calendar years or the year of a traded future pick.
func FranchiseMovesHistoryReading(scope RawFranchiseMovesHistory, principal string, records []Record, auth Authorizer) (Ledger, error) {
	if err := scope.Validate(); err != nil {
		return Ledger{}, err
	}
	if !leagueFranchise(records, scope.Franchise) {
		return Ledger{}, fmt.Errorf("choose a franchise in this league")
	}
	if auth == nil {
		state, err := GovernanceState(records)
		if err != nil {
			return Ledger{}, fmt.Errorf("read franchise history permissions: %w", err)
		}
		auth = GovernanceAuthorizer{State: state}
	}
	reading, err := Reading(scope.League, principal, records, auth)
	if err != nil {
		return Ledger{}, err
	}
	matches, err := franchiseHistoryMatches(scope, records)
	if err != nil {
		return Ledger{}, err
	}
	entries := make([]Entry, 0, len(reading.Entries))
	for _, entry := range reading.Entries {
		if matches[entry.ID] {
			entries = append(entries, entry)
		}
	}
	reading.Entries = entries
	return reading, nil
}

func franchiseHistoryMatches(scope RawFranchiseMovesHistory, records []Record) (map[string]bool, error) {
	season, err := firstSeason(records)
	if err != nil {
		return nil, err
	}
	starts, err := RolloverInstants(records)
	if err != nil {
		return nil, err
	}
	matches := make(map[string]bool)
	targetsByEntry := make(map[string][]string)
	for i, r := range records {
		targets, err := historyTargets(r, records[:i], targetsByEntry)
		if err != nil {
			return nil, err
		}
		targetsByEntry[r.Receipt.Entry] = slices.Clone(targets)
		for len(starts) > 0 && !r.Receipt.RecordedAt.Before(starts[0]) {
			season++
			starts = starts[1:]
		}
		// Unscoped league changes (rules, rollover, setup) affect every franchise.
		matches[r.Receipt.Entry] = season == scope.Season && (len(targets) == 0 || slices.Contains(targets, scope.Franchise))
	}
	return matches, nil
}

func historyTargets(r Record, before []Record, byEntry map[string][]string) ([]string, error) {
	targets, err := nativePreviewTargets(r.Command, before, r.Receipt.RecordedAt, previewTargets(r.Command))
	if err != nil {
		return nil, err
	}
	a := r.Command.Args
	targets = append(targets, byEntry[a.Entry]...)
	if a.Correction != nil {
		targets = append(targets, byEntry[a.Correction.Entry]...)
		if a.Correction.Scope != nil {
			targets = append(targets, byEntry[a.Correction.Scope.Transaction]...)
		}
	}
	if a.Bid != nil {
		targets = append(targets, a.Bid.Franchise)
		targets = append(targets, byEntry[a.Bid.Bid]...)
	}
	if a.Waiver != nil {
		targets = append(targets, a.Waiver.Franchise)
		targets = append(targets, byEntry[a.Waiver.Claim]...)
	}
	if a.Trade != nil {
		targets = append(targets, a.Trade.Franchise)
	}
	owner, err := historyPickOwner(r, before)
	if err != nil {
		return nil, err
	}
	targets = append(targets, owner)
	return historyOutcomeTargets(a, byEntry, targets), nil
}

func historyPickOwner(r Record, before []Record) (string, error) {
	c := r.Command.Args.Correction
	if c == nil || c.Pick == nil || c.History != nil {
		return "", nil
	}
	assets, err := NativeAssetsReading(r.Command.League, before)
	if err != nil {
		return "", err
	}
	if assets.Snapshot != nil {
		for _, pick := range assets.Snapshot.Picks {
			p := c.Pick
			if pick.Season == p.Season && pick.Round == p.Round && pick.Original == p.Original && pick.Slot == p.Slot {
				return pick.Owner, nil
			}
		}
	}
	return "", nil
}

func historyOutcomeTargets(a RawArgs, byEntry map[string][]string, targets []string) []string {
	if a.BidOutcome != nil {
		targets = append(targets, a.BidOutcome.Winner)
		for _, result := range a.BidOutcome.Results {
			targets = append(targets, byEntry[result.ID]...)
		}
	}
	if a.WaiverOutcome != nil {
		targets = append(targets, a.WaiverOutcome.Winner)
		for _, claim := range a.WaiverOutcome.Claims {
			targets = append(targets, byEntry[claim.ID]...)
		}
	}
	if a.TradeOutcome != nil {
		targets = append(targets, a.TradeOutcome.Parties...)
	}
	if a.DraftOutcome != nil {
		targets = append(targets, a.DraftOutcome.Franchise)
	}
	if a.Draft != nil {
		if a.Draft.Pick != nil {
			targets = append(targets, a.Draft.Pick.Franchise)
		}
		if a.Draft.Act != nil {
			targets = append(targets, a.Draft.Act.Franchise)
		}
	}
	if a.OLineOutcome != nil {
		for _, p := range a.OLineOutcome.Assigned {
			targets = append(targets, p.Franchise)
		}
		for _, p := range a.OLineOutcome.Pending {
			targets = append(targets, p.Franchise)
		}
	}
	targets = slices.DeleteFunc(targets, func(s string) bool { return s == "" })
	return targets
}
