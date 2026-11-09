package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// startingDefaults preserves recorded values and supplies only settings absent from older starts.
func startingDefaults(seed []SeedValue, source string) ([]SeedValue, error) {
	lock, assets := leaguerules.EachPlayerGame, leaguerules.EmptyAssets
	if source == SourceMFLImport {
		lock, assets = leaguerules.FirstKickoff, leaguerules.HeldAssets
	}
	defaults := []struct {
		path  string
		value leaguerules.Setting
	}{
		{"lineup.lock", leaguerules.Value[leaguerules.LineupLock]{State: leaguerules.IsKnown, Value: lock,
			Citation: "New leagues lock each player at their own game"}},
		{"league.startingAssets", leaguerules.Value[leaguerules.StartingAssets]{State: leaguerules.IsKnown, Value: assets,
			Citation: "New leagues start with empty rosters and their rules-defined auction lots"}},
	}
	out := slices.Clone(seed)
	for _, d := range defaults {
		if slices.ContainsFunc(out, func(s SeedValue) bool { return s.Field == d.path }) {
			continue
		}
		body, err := leaguerules.EncodeStored(d.value)
		if err != nil {
			return nil, fmt.Errorf("encode starting setting: %w", err)
		}
		out = append(out, SeedValue{Field: d.path, Value: body})
	}
	return out, nil
}

// initialStartingSettings keeps a native preset's values; an import supplies source values at creation.
func initialStartingSettings(seed []SeedValue, source string) ([]SeedValue, error) {
	if source == SourceMFLImport {
		seed = slices.DeleteFunc(slices.Clone(seed), func(s SeedValue) bool {
			return s.Field == "lineup.lock" || s.Field == "league.startingAssets"
		})
	}
	return startingDefaults(seed, source)
}

func onlyStartingDefaults(seed []SeedValue) bool {
	return !slices.ContainsFunc(seed, func(s SeedValue) bool {
		return s.Field != "lineup.lock" && s.Field != "league.startingAssets"
	})
}

// StartingAssetsAt reads the league setting, including normal recorded changes.
func StartingAssetsAt(records []Record, at time.Time) (leaguerules.StartingAssets, error) {
	rules, err := RulesNow(records, at)
	if err != nil {
		return "", err
	}
	if rules.League.StartingAssets.State != leaguerules.IsKnown {
		return "", fmt.Errorf("set the league's starting assets before recording them")
	}
	return rules.League.StartingAssets.Value, nil
}

// LeagueSource is the single compatibility read for records written before Source existed.
// A league imported from MFL reads as native once it has been moved over (Cutover).
func LeagueSource(records []Record) string {
	if source := importedSource(records); source != SourceMFLImport || !cutOver(records) {
		return source
	}
	return SourceNative
}
