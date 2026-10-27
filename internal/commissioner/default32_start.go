package commissioner

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// lineCardContract is the contract note on a card handed out at league start (DEFAULT-32 spec §3).
const lineCardContract = "O-Line card: 1-year league-minimum contract"

// IsDefault32 reports whether a league was created from the 32-Team Default.
func IsDefault32(records []Record) bool {
	return len(records) != 0 && records[0].Command.Command == Create && records[0].Command.Args.Preset == PresetDefault32
}

// Default32SeedCommand is the starting snapshot command of a 32-Team Default league. Such a league holds
// no MFL data: it starts with its own franchises, each holding one O-Line card. A retried request reuses
// its recorded body.
func Default32SeedCommand(records []Record, league, request, reason string, expected int64,
	now time.Time) (RawCommand, error) {
	if raw, found := NativeSeedRetry(records, request); found {
		if raw.Command != NativeSeed || raw.Expected != expected || raw.Reason != reason {
			return raw, fmt.Errorf("this request was already used for a different change")
		}
		return raw, nil
	}
	seed, err := Default32Start(league, records, now)
	if err != nil {
		return RawCommand{}, err
	}
	return RawCommand{League: league, Command: NativeSeed, Args: RawArgs{Native: &seed}, Expected: expected,
		Request: request, Reason: reason, Effective: "now", Visibility: "league"}, nil
}

// Default32Start auto-assigns each franchise its O-Line card at league start: franchise N (creation order)
// gets the card of the Nth team in the fixed card list, on a 1-year contract at the league minimum salary.
func Default32Start(league string, records []Record, now time.Time) (RawNativeSnapshot, error) {
	if !IsDefault32(records) {
		return RawNativeSnapshot{}, fmt.Errorf("only a 32-Team Default league starts with O-Line cards")
	}
	season := records[0].Command.Args.Season
	directory, err := NativeDirectoryReading(league, records, now)
	if err != nil {
		return RawNativeSnapshot{}, err
	}
	rules, err := RulesNow(records, now)
	if err != nil {
		return RawNativeSnapshot{}, err
	}
	salary, err := leagueMinimum(rules.Contracts)
	if err != nil {
		return RawNativeSnapshot{}, err
	}
	teams := nativecheck.LineCardTeams()
	if len(directory.Franchises) != len(teams) {
		return RawNativeSnapshot{}, fmt.Errorf("a 32-Team Default league needs its %d franchises", len(teams))
	}
	at := now.UTC()
	seed := RawNativeSnapshot{Season: season, RosterAt: at, DraftAt: at, FutureAt: at,
		Picks: []NativePick{}, Adjustments: []NativeCharge{}}
	for i, f := range directory.Franchises {
		seed.Franchises = append(seed.Franchises, NativeFranchise{ID: f.Franchise, HeldID: f.Franchise,
			Name: f.Name, Abbreviation: f.Abbreviation, Division: f.Division})
		card, _ := nativecheck.LineCardID(teams[i])
		seed.Players = append(seed.Players, NativePlayer{ID: card, Franchise: f.Franchise, Name: teams[i] + " O-Line",
			Status: domain.RosterActive, Salary: salary, ExpirationYear: season, ContractStatus: domain.CStatusUFA,
			ContractInfo: lineCardContract, AcquiredVia: "League start"})
	}
	if err := seed.Validate(); err != nil {
		return RawNativeSnapshot{}, fmt.Errorf("32-Team Default starting snapshot: %w", err)
	}
	return seed, nil
}

// leagueMinimum is the lowest recorded minimum salary, the price of an auto-assigned card.
func leagueMinimum(rules leaguerules.ContractsRules) (domain.Money, error) {
	if rules.MinimumSalaries.State != leaguerules.IsKnown {
		return 0, fmt.Errorf("record the league's minimum salaries before starting the league")
	}
	low := domain.Money(-1)
	for _, tier := range rules.MinimumSalaries.Value {
		if tier.Minimum.State == leaguerules.IsKnown && (low < 0 || domain.Money(tier.Minimum.Value) < low) {
			low = domain.Money(tier.Minimum.Value)
		}
	}
	if low < 0 {
		return 0, fmt.Errorf("record the league's minimum salaries before starting the league")
	}
	return low, nil
}

// checkDefault32Start refuses a 32-Team Default snapshot that is anything but one card per franchise.
func checkDefault32Start(records []Record, seed *RawNativeSnapshot) error {
	if seed.Season != records[0].Command.Args.Season || len(seed.Picks) != 0 || len(seed.Adjustments) != 0 ||
		len(seed.Players) != len(seed.Franchises) {
		return fmt.Errorf("a 32-Team Default league starts with one O-Line card per franchise and nothing else")
	}
	held, cards := map[string]bool{}, map[string]bool{}
	for _, p := range seed.Players {
		if _, ok := nativecheck.LineCardTeam(p.ID); !ok || held[p.Franchise] || cards[p.ID] ||
			p.Status != domain.RosterActive || p.ExpirationYear != seed.Season {
			return fmt.Errorf("a 32-Team Default league starts with one O-Line card per franchise and nothing else")
		}
		held[p.Franchise], cards[p.ID] = true, true
	}
	if !slices.Equal(slices.Sorted(maps.Keys(held)), slices.Sorted(slices.Values(records[0].Command.Args.Franchises))) {
		return fmt.Errorf("a 32-Team Default league starts with one O-Line card per franchise and nothing else")
	}
	return nil
}
