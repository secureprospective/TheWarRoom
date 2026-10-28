package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// LeagueSource is the single compatibility read for records written before Source existed.
// A league imported from MFL reads as native once it has been moved over (Cutover).
func LeagueSource(records []Record) string {
	if source := importedSource(records); source != SourceMFLImport || !cutOver(records) {
		return source
	}
	return SourceNative
}

// NativeSeedCommand prepares a native starting snapshot, preserving retries.
func NativeSeedCommand(records []Record, league, request, reason string, expected int64,
	now time.Time) (RawCommand, error) {
	if raw, found := NativeSeedRetry(records, request); found {
		if raw.Command != NativeSeed || raw.Expected != expected || raw.Reason != reason {
			return raw, fmt.Errorf("this request was already used for a different change")
		}
		return raw, nil
	}
	seed, err := NativeStart(league, records, now)
	if err != nil {
		return RawCommand{}, err
	}
	return RawCommand{League: league, Command: NativeSeed, Args: RawArgs{Native: &seed}, Expected: expected,
		Request: request, Reason: reason, Effective: "now", Visibility: "league"}, nil
}

// NativeStart captures empty rosters and, when OL is bounded, the unassigned O-Line lots.
func NativeStart(league string, records []Record, now time.Time) (RawNativeSnapshot, error) {
	if len(records) == 0 || LeagueSource(records) != SourceNative {
		return RawNativeSnapshot{}, fmt.Errorf("only a native league starts without held data")
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
	cards := leaguerules.OLineOn(rules)
	teams := nativecheck.LineCardTeams()
	at := now.UTC()
	seed := RawNativeSnapshot{Season: season, RosterAt: at, DraftAt: at, FutureAt: at,
		Players: []NativePlayer{}, Picks: []NativePick{}, Adjustments: []NativeCharge{}}
	// Only a settled division is copied; the directory's words for an unset one are never a name.
	alignment := rules.League.Alignment
	for _, f := range directory.Franchises {
		division := ""
		if alignment.State == leaguerules.IsKnown {
			division = alignment.Value[f.Franchise]
		}
		seed.Franchises = append(seed.Franchises, NativeFranchise{ID: f.Franchise, HeldID: f.Franchise,
			Name: f.Name, Abbreviation: f.Abbreviation, Division: division})
	}
	if cards {
		for _, team := range teams {
			card, _ := nativecheck.LineCardID(team)
			seed.Pool = append(seed.Pool, NativePoolEntry{Player: card, Name: team + " O-Line", Season: season})
		}
	}

	if err := seed.Validate(); err != nil {
		return RawNativeSnapshot{}, fmt.Errorf("native starting snapshot: %w", err)
	}
	return seed, nil
}

func nativeStartCards(records []Record, now time.Time) (bool, error) {
	rules, err := RulesNow(records, now)
	if err != nil {
		return false, err
	}
	return leaguerules.OLineOn(rules), nil
}

func checkNativeStart(records []Record, seed *RawNativeSnapshot) error {
	cards, err := nativeStartCards(records, seed.RosterAt)
	if err != nil {
		return err
	}
	if cards {
		return checkCardStart(records, seed)
	}
	if len(seed.Players) != 0 || len(seed.Picks) != 0 || len(seed.Pool) != 0 || len(seed.Adjustments) != 0 {
		return fmt.Errorf("a native league starts with empty rosters, picks and pool")
	}
	return nil
}

// checkCardStart admits only the empty start and the complete, unassigned auction pool.
func checkCardStart(records []Record, seed *RawNativeSnapshot) error {
	if seed.Season != records[0].Command.Args.Season || len(seed.Picks) != 0 || len(seed.Adjustments) != 0 || len(seed.Players) != 0 || len(seed.Pool) != len(nativecheck.LineCardTeams()) {
		return fmt.Errorf("an O-Line league starts with empty rosters and all O-Line auction lots")
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
func StartingCardLot(records []Record, player string) bool {
	if LeagueSource(records) != SourceNative {
		return false
	}
	if _, ok := nativecheck.LineCardTeam(player); !ok {
		return false
	}
	for _, r := range records {
		if r.Command.Command == NativeSeed && r.Command.Args.Native != nil {
			return len(r.Command.Args.Native.Players) == 0 && len(r.Command.Args.Native.Pool) == len(nativecheck.LineCardTeams())
		}
	}
	return false
}
