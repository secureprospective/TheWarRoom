package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// IsDefault32 reports whether a league was created from the 32-Team Default.
func IsDefault32(records []Record) bool {
	return len(records) != 0 && records[0].Command.Command == Create && records[0].Command.Args.Preset == PresetDefault32
}

// Default32SeedCommand is the starting snapshot command of a 32-Team Default league. Such a league holds
// no MFL data: it starts with its own franchises and unassigned O-Line auction lots. A retried request reuses
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

// Default32Start opens the league with empty rosters and all 32 cards as auction lots.
func Default32Start(league string, records []Record, now time.Time) (RawNativeSnapshot, error) {
	if !IsDefault32(records) {
		return RawNativeSnapshot{}, fmt.Errorf("only a 32-Team Default league starts with O-Line cards")
	}
	season := records[0].Command.Args.Season
	directory, err := NativeDirectoryReading(league, records, now)
	if err != nil {
		return RawNativeSnapshot{}, err
	}
	teams := nativecheck.LineCardTeams()
	if len(directory.Franchises) != len(teams) {
		return RawNativeSnapshot{}, fmt.Errorf("a 32-Team Default league needs its %d franchises", len(teams))
	}
	at := now.UTC()
	seed := RawNativeSnapshot{Season: season, RosterAt: at, DraftAt: at, FutureAt: at,
		Players: []NativePlayer{}, Picks: []NativePick{}, Adjustments: []NativeCharge{}}
	for _, f := range directory.Franchises {
		seed.Franchises = append(seed.Franchises, NativeFranchise{ID: f.Franchise, HeldID: f.Franchise,
			Name: f.Name, Abbreviation: f.Abbreviation, Division: f.Division})
	}
	for _, team := range teams {
		card, _ := nativecheck.LineCardID(team)
		seed.Pool = append(seed.Pool, NativePoolEntry{Player: card, Name: team + " O-Line", Season: season})
	}

	if err := seed.Validate(); err != nil {
		return RawNativeSnapshot{}, fmt.Errorf("32-Team Default starting snapshot: %w", err)
	}
	return seed, nil
}

// checkDefault32Start admits only the empty start and the complete, unassigned auction pool.
func checkDefault32Start(records []Record, seed *RawNativeSnapshot) error {
	if seed.Season != records[0].Command.Args.Season || len(seed.Picks) != 0 || len(seed.Adjustments) != 0 || len(seed.Players) != 0 || len(seed.Pool) != 32 {
		return fmt.Errorf("a 32-Team Default league starts with empty rosters and 32 O-Line auction lots")
	}
	cards := map[string]bool{}
	for _, lot := range seed.Pool {
		facts, ok := nativecheck.LineCardFacts(lot.Player)
		if !ok || cards[lot.Player] || lot.Season != seed.Season || lot.Name != facts.Name {
			return fmt.Errorf("the starting auction needs each O-Line card exactly once")
		}
		cards[lot.Player] = true
	}
	return nil
}

// startingCardLot distinguishes the open start auction from later free-agent bidding and older
// leagues that already assigned cards. The immutable seed, not today's roster size, defines it.
func startingCardLot(records []Record, player string) bool {
	if !IsDefault32(records) {
		return false
	}
	if _, ok := nativecheck.LineCardTeam(player); !ok {
		return false
	}
	for _, r := range records {
		if r.Command.Command == NativeSeed && r.Command.Args.Native != nil {
			return len(r.Command.Args.Native.Players) == 0 && len(r.Command.Args.Native.Pool) == 32
		}
	}
	return false
}
