package commissioner

import (
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// AutoPickTurn is one open, unpaused draft whose current pick belongs to a vacant franchise.
type AutoPickTurn struct {
	Key       DraftKey
	Franchise string
	Round     int
	Slot      int
	Original  string
	Pool      []string
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
// clock has no seated owner, along with its available player pool IDs.
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
		if !ok || f.Seat != "" {
			continue
		}
		pool, err := availablePoolIDs(basis, records, at, facts)
		if err != nil {
			return AutoPickTurn{}, false, err
		}
		return AutoPickTurn{Key: key, Franchise: clock.Owner, Round: clock.Round, Slot: clock.Slot,
			Original: clock.Original, Pool: pool}, true, nil
	}
	return AutoPickTurn{}, false, nil
}

func availablePoolIDs(basis draftBasis, records []Record, at time.Time,
	facts nativecheck.Facts) ([]string, error) {
	through := RecordsThrough(records, at)
	assets, err := NativeAssetsReading(records[0].Command.League, through)
	if err != nil {
		return nil, err
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
	return ids, nil
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
// franchise must be vacant and on the clock, and PlanDraftPick must produce the same selection.
func CheckAutoDraftPickAppend(raw RawCommand, records []Record, at time.Time,
	facts nativecheck.Facts, standings DraftStandings) error {
	if raw.Command != DraftPick || raw.Args.Draft == nil || raw.Args.Draft.Pick == nil {
		return Refused(errors.New("automatic draft pick requires a draft selection"))
	}
	d := raw.Args.Draft
	pick := d.Pick
	auto, err := AutomaticFranchise(records, pick.Franchise)
	if err != nil {
		return err
	}
	if !auto {
		return Refused(errors.New("automatic picks apply only to vacant franchises"))
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
