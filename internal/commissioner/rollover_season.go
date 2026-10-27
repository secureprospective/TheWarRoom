package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// recordedRollover is a season rollover entry as the ledger holds it. A command still being checked has no
// plan yet and is not one.
func recordedRollover(r Record) bool {
	return r.Command.Command == SeasonRollover && r.Receipt.Plan != nil && r.Before != nil
}

// explicitRolloverInstants lists when each in-force rollover act began its season. Once the ledger holds
// a season rollover act, rollover dates are reminders and no longer start seasons.
func explicitRolloverInstants(records []Record) ([]time.Time, bool) {
	active := Replay(records).Active
	starts := []time.Time{}
	explicit := false
	for _, r := range records {
		if !recordedRollover(r) {
			continue
		}
		explicit = true
		if active[r.Receipt.Entry] {
			starts = append(starts, r.Receipt.RecordedAt)
		}
	}
	return starts, explicit
}

// explicitSeasonRules reads a season's rules where rollover acts number the seasons. A season an act
// closed keeps the rules captured just before it; the season it opened starts from the rules the act
// recorded, then folds in the changes made since.
func explicitSeasonRules(records []Record, season int, at time.Time) (leaguerules.RuleSet, bool, error) {
	starts, explicit := explicitRolloverInstants(records)
	if !explicit {
		return leaguerules.RuleSet{}, false, nil
	}
	rules, err := rulesOfExplicitSeason(records, starts, season, at)
	return rules, true, err
}

func rulesOfExplicitSeason(records []Record, starts []time.Time, season int, at time.Time) (leaguerules.RuleSet, error) {
	active := Replay(records).Active
	var base *leaguerules.RuleSet
	since := 0
	for i, r := range records {
		if !recordedRollover(r) || !active[r.Receipt.Entry] {
			continue
		}
		if r.Receipt.Plan.From == season {
			return r.Before.Rules, nil
		}
		if r.Receipt.Plan.To == season {
			rules := r.Receipt.Plan.Rules
			base, since = &rules, i+1
		}
	}
	if base == nil {
		return firstExplicitRules(records, starts, season, at)
	}
	changes := leaguerules.SeedChanges(*base, base.LeagueID, season, records[since-1].Receipt.RecordedAt)
	fold := seasonFold{season: season, starts: starts, active: active, franchises: LeagueFranchises(records[:since])}
	for _, change := range changes {
		fold.timed = append(fold.timed, startedChange{change, change.RecordedAt})
	}
	for _, r := range records[since:] {
		if err := fold.add(r); err != nil {
			return leaguerules.RuleSet{}, err
		}
	}
	changes = changes[:0]
	for _, timed := range fold.timed {
		timed.change.Season = season
		changes = append(changes, timed.change)
	}
	return resolveSeason(changes, season, at)
}

// firstExplicitRules reads the league's first season, before any in-force act closed it.
func firstExplicitRules(records []Record, starts []time.Time, season int, at time.Time) (leaguerules.RuleSet, error) {
	first, err := firstSeason(records)
	if err != nil {
		return leaguerules.RuleSet{}, err
	}
	if season != first {
		return leaguerules.RuleSet{}, fmt.Errorf("read league rules: season %d has not started", season)
	}
	changes, err := seasonChanges(records, starts, season)
	if err != nil {
		return leaguerules.RuleSet{}, err
	}
	return resolveSeason(changes, season, at)
}

func resolveSeason(changes []leaguerules.Change, season int, at time.Time) (leaguerules.RuleSet, error) {
	rules, err := leaguerules.Resolve(changes, season, at)
	if err != nil {
		return leaguerules.RuleSet{}, fmt.Errorf("read season rules: %w", err)
	}
	return rules, nil
}

// applyNativeRollover replaces the native assets with the ones the rollover recorded.
func applyNativeRollover(assets *NativeAssets, record Record) error {
	plan := record.Receipt.Plan
	if plan == nil || record.Before == nil {
		return fmt.Errorf("the rollover entry has no recorded plan")
	}
	if !assets.Seeded {
		return nil
	}
	if assets.Snapshot.Season != plan.From {
		return fmt.Errorf("the rollover does not follow the league's recorded assets")
	}
	after := cloneRolloverAssets(plan.After)
	assets.Snapshot = &after
	return nil
}
