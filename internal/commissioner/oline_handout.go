package commissioner

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// OLineHandoutCandidate is one automatic franchise and unsold O-Line card ready for handout.
type OLineHandoutCandidate struct {
	Franchise string
	Side      string
	Player    string
}

// DueOLineHandouts returns the next automatic franchise without an O-Line card and the next
// unsold card on its side after the startup O-Line auction has closed. Franchises are ordered by
// ID ascending; unsold cards are ordered by card name then ID ascending (matching oline_close.go).
// When no more pairs can be formed, unfilled reports how many eligible automatic franchises have
// no O-Line card left to receive.
func DueOLineHandouts(league string, records []Record, now time.Time,
	skip map[string]bool) (OLineHandoutCandidate, int, bool, error) {
	f, ok, err := closedOLineFrame(league, records, now)
	if err != nil || !ok {
		return OLineHandoutCandidate{}, 0, false, err
	}
	state, err := GovernanceState(records)
	if err != nil {
		return OLineHandoutCandidate{}, 0, false, fmt.Errorf("read league seats: %w", err)
	}
	seed := f.Assets.Snapshot
	pendingTeam, pendingCard := pendingOLineReservations(seed, f.Book.OLine)
	franchises := sortedFranchises(seed.Franchises)
	lots := sortedOLineLots(seed.Pool)
	unfilled := 0
	for _, fr := range franchises {
		if skip[fr.ID] || pendingTeam[fr.ID] || !isVacantSeat(state, fr.ID) ||
			olineCount(seed, fr.ID) > 0 {
			continue
		}
		side := seed.SideOf(fr.ID)
		card, found := firstUnsoldOLineLot(seed, lots, side, pendingCard)
		if !found {
			unfilled++
			continue
		}
		return OLineHandoutCandidate{Franchise: fr.ID, Side: side, Player: card}, 0, true, nil
	}
	return OLineHandoutCandidate{}, unfilled, false, nil
}

// CheckOLineHandout verifies that an automatic O-Line handout is legal: the auction is closed,
// the franchise is automatic and holds no O-Line card, and the player is an unsold O-Line lot
// on the franchise's side.
func CheckOLineHandout(league string, records []Record, franchise, player string,
	now time.Time) error {
	if now.IsZero() && len(records) > 0 {
		now = records[len(records)-1].Receipt.RecordedAt
	}
	f, closed, err := closedOLineFrame(league, records, now)
	if err != nil {
		return err
	}
	if !closed {
		return Refused(errors.New("O-Line handouts run only after the O-Line auction closes"))
	}
	auto, err := AutomaticFranchise(records, franchise)
	if err != nil {
		return err
	}
	if !auto {
		return Refused(errors.New("O-Line handouts apply only to automatic franchises"))
	}
	if _, card := nativecheck.LineCardTeam(player); !card {
		return Refused(errors.New("O-Line handouts apply only to O-Line cards"))
	}
	seed := f.Assets.Snapshot
	if !slices.ContainsFunc(seed.Franchises, func(fr NativeFranchise) bool {
		return fr.ID == franchise
	}) {
		return Refused(errors.New("choose a franchise in this league"))
	}
	if olineCount(seed, franchise) > 0 {
		return Refused(errors.New("this franchise already holds an O-Line card"))
	}
	side := seed.SideOf(franchise)
	if seed.OnSide(side, player) {
		return Refused(errors.New("this O-Line card is already on a roster"))
	}
	if !slices.ContainsFunc(seed.Pool, func(l NativePoolEntry) bool {
		return l.Player == player && l.Side == side
	}) {
		return Refused(errors.New("this O-Line card is not an unsold lot for this side"))
	}
	return nil
}

func closedOLineFrame(league string, records []Record, now time.Time) (Frame, bool, error) {
	f, err := NewFrame(league, records, now, nil, time.UTC)
	if err != nil {
		return Frame{}, false, err
	}
	if !f.Assets.Seeded || f.Book.OLine == nil {
		return Frame{}, false, nil
	}
	if err := f.ReadRules(); err != nil {
		return Frame{}, false, err
	}
	if !leaguerules.OLineOn(f.Rules) {
		return Frame{}, false, nil
	}
	return f, true, nil
}

func pendingOLineReservations(seed *RawNativeSnapshot,
	outcome *OLineOutcome) (map[string]bool, map[string]bool) {
	teams := map[string]bool{}
	cards := map[string]bool{}
	if outcome == nil {
		return teams, cards
	}
	for _, p := range outcome.Pending {
		teams[p.Franchise] = true
		cards[seed.SideOf(p.Franchise)+":"+p.ID] = true
	}
	return teams, cards
}

func sortedFranchises(in []NativeFranchise) []NativeFranchise {
	out := slices.Clone(in)
	slices.SortFunc(out, func(a, b NativeFranchise) int { return strings.Compare(a.ID, b.ID) })
	return out
}

func sortedOLineLots(pool []NativePoolEntry) []NativePoolEntry {
	out := make([]NativePoolEntry, 0, len(pool))
	for _, l := range pool {
		if _, card := nativecheck.LineCardTeam(l.Player); card {
			out = append(out, l)
		}
	}
	slices.SortFunc(out, func(a, b NativePoolEntry) int {
		return cmp.Or(strings.Compare(a.Name, b.Name), strings.Compare(a.Player, b.Player))
	})
	return out
}

func isVacantSeat(state governance.State, franchise string) bool {
	fr, ok := state.Franchise(governance.FranchiseID(franchise))
	return ok && fr.Seat == ""
}

func firstUnsoldOLineLot(seed *RawNativeSnapshot, lots []NativePoolEntry, side string,
	pendingCard map[string]bool) (string, bool) {
	for _, l := range lots {
		if l.Side == side && !pendingCard[side+":"+l.Player] && !seed.OnSide(side, l.Player) {
			return l.Player, true
		}
	}
	return "", false
}
