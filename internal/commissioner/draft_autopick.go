package commissioner

import (
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// AutoPickTurn is one open, unpaused draft with a pick eligible for computer selection.
type AutoPickTurn struct {
	Key       DraftKey
	Franchise string
	Round     int
	Slot      int
	Original  string
	Pool      []string
	Lineup    leaguerules.LineupRules
	Held      map[leaguerules.Position]int
	Ranking   leaguerules.AutoPickRanking
}

// AutomaticFranchise reports whether the franchise has no seated owner (vacant, caretaker or not).
func AutomaticFranchise(records []Record, franchise string) (bool, error) {
	state, err := GovernanceState(records)
	if err != nil {
		return false, fmt.Errorf("read league seats: %w", err)
	}
	f, ok := state.Franchise(governance.FranchiseID(franchise))
	return ok && f.Seat == "", nil
}

// DueAutoPick returns the first open, unpaused draft (excluding skip keys) whose franchise on the
// clock is vacant or due for human timeout autopick, with its available player pool IDs.
func DueAutoPick(records []Record, at time.Time, facts nativecheck.Facts,
	standings DraftStandings, skip map[DraftKey]bool) (AutoPickTurn, bool, error) {
	state, err := GovernanceState(records)
	if err != nil {
		return AutoPickTurn{}, false, fmt.Errorf("read league seats: %w", err)
	}
	for _, key := range draftSeasonsKnown(records) {
		if skip[key] {
			continue
		}
		basis, err := readDraft(records, key, at, standings)
		if err != nil || basis.words != "" || basis.closed != nil || !basis.started ||
			basis.draft.Exhausted || basis.draft.OnClock == nil || basis.draft.OnClock.Paused {
			continue
		}
		clock := basis.draft.OnClock
		f, ok := state.Franchise(governance.FranchiseID(clock.Owner))
		if !ok || (f.Seat != "" && !humanAutoPickDue(basis, at)) {
			continue
		}
		pool, held, err := turnPoolAndHeld(basis, records, at, clock.Owner, facts)
		if err != nil {
			return AutoPickTurn{}, false, err
		}
		return AutoPickTurn{Key: key, Franchise: clock.Owner, Round: clock.Round, Slot: clock.Slot,
			Original: clock.Original, Pool: pool, Lineup: basis.rules.Lineup, Held: held,
			Ranking: basis.rules.Draft.AutoPickRanking.Value}, true, nil
	}
	return AutoPickTurn{}, false, nil
}

func turnPoolAndHeld(basis draftBasis, records []Record, at time.Time, franchise string,
	facts nativecheck.Facts) ([]string, map[leaguerules.Position]int, error) {
	through := RecordsThrough(records, at)
	assets, err := NativeAssetsReading(records[0].Command.League, through)
	if err != nil {
		return nil, nil, err
	}
	taken := draftTakenSet(basis, assets)
	var players []DraftPlayer
	if basis.key.Startup {
		players = startupPool(facts, taken)
	} else {
		players = classRookies(facts, basis.key.Season, taken)
	}
	ids := make([]string, len(players))
	for i, p := range players {
		ids[i] = p.ID
	}
	return ids, franchiseHeldPositions(basis, assets, franchise, facts), nil
}

func franchiseHeldPositions(basis draftBasis, assets NativeAssets, franchise string,
	facts nativecheck.Facts) map[leaguerules.Position]int {
	seen := map[string]bool{}
	held := map[leaguerules.Position]int{}
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		if pf, ok := facts.Of(id); ok && pf.Position != "" {
			held[pf.Position]++
		}
	}
	if assets.Seeded {
		for _, p := range assets.Snapshot.Players {
			if p.Franchise == franchise {
				add(p.ID)
			}
		}
	}
	for _, pick := range basis.draft.Picks {
		if pick.Owner == franchise && pick.Player != "" {
			add(pick.Player)
		}
	}
	return held
}

func draftTakenSet(basis draftBasis, assets NativeAssets) map[string]bool {
	taken := map[string]bool{}
	for _, pick := range basis.draft.Picks {
		if pick.Player != "" {
			taken[pick.Player] = true
		}
	}
	if !assets.Seeded {
		return taken
	}
	for _, p := range assets.Snapshot.Players {
		if assets.Snapshot.SideOf(p.Franchise) == basis.key.Side {
			taken[p.ID] = true
		}
	}
	for _, p := range assets.Snapshot.Pool {
		if p.Side == basis.key.Side {
			taken[p.Player] = true
		}
	}
	return taken
}

// CheckAutoDraftPickAppend verifies an automatic draft.pick inside the write transaction: the
// franchise must qualify for computer selection, and planning must produce the same selection.
func CheckAutoDraftPickAppend(raw RawCommand, records []Record, at time.Time,
	facts nativecheck.Facts, standings DraftStandings) error {
	if raw.Command != DraftPick || raw.Args.Draft == nil || raw.Args.Draft.Pick == nil {
		return Refused(errors.New("automatic draft pick requires a draft selection"))
	}
	d := raw.Args.Draft
	pick := d.Pick
	auto, err := AutomaticDraftTurn(records, DraftKey{Season: d.Season, Startup: d.Startup, Side: d.Side}, pick.Franchise, at, standings)
	if err != nil {
		return err
	}
	if !auto {
		return Refused(errors.New("this turn is not eligible for an automatic pick"))
	}
	planned, err := PlanDraftPick(DraftPickInput{League: raw.League, Season: d.Season,
		Franchise: pick.Franchise, Player: pick.Player, Request: raw.Request,
		Expected: raw.Expected, Startup: d.Startup}, records, facts, at, time.UTC, standings)
	if err != nil {
		return err
	}
	got := planned.Args.Draft
	if got == nil || got.Side != d.Side || got.Pick == nil || *got.Pick != *pick {
		return Refused(errors.New("the automatic pick no longer matches the slot on the clock"))
	}
	return nil
}

// holdVacantDraftTurn preserves a computer turn using the seats at its deadline.
// A later seating change must not rewrite an earlier human timeout.
func holdVacantDraftTurn(records []Record) func(string, time.Time) (bool, error) {
	return func(owner string, deadline time.Time) (bool, error) {
		return AutomaticFranchise(RecordsThrough(records, deadline), owner)
	}
}

// AutomaticDraftTurn restricts computer selections to vacant seats or a timed-out human autopick turn.
func AutomaticDraftTurn(records []Record, key DraftKey, owner string, at time.Time, standings DraftStandings) (bool, error) {
	vacant, err := AutomaticFranchise(records, owner)
	if err != nil || vacant {
		return vacant, err
	}
	basis, err := readDraft(records, key, at, standings)
	if err != nil {
		return false, err
	}
	clock := basis.draft.OnClock
	return basis.closed == nil && basis.started && clock != nil && !clock.Paused && clock.Owner == owner && basis.rules.Draft.OnExpire.Value == leaguerules.AutoPick && !at.Before(clock.Deadline), nil
}

func humanAutoPickDue(basis draftBasis, at time.Time) bool {
	return basis.rules.Draft.OnExpire.Value == leaguerules.AutoPick && !at.Before(basis.draft.OnClock.Deadline)
}
