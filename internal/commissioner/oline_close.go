package commissioner

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const OLineAuctionClose = "oline.auction.close"
const OLineClockPrincipal = "native.oline.clock"

// OLineOutcome reserves a fixed unit and contract even while its franchise must make room.
// Only the append transaction supplies it, never a caller.
type OLineOutcome struct {
	Assigned []NativePlayer `json:"assigned"`
	Pending  []NativePlayer `json:"pending"`
	Words    []string       `json:"words"`
}

// PrepareOLineClose closes the starting auction, or retries its reserved assignments.
func PrepareOLineClose(raw *RawCommand, records []Record, now time.Time) error {
	if raw.Command != OLineAuctionClose {
		return nil
	}
	f, err := NewFrame(raw.League, records, now, nil, time.UTC)
	if err != nil {
		return err
	}
	if err := f.Seeded("record starting assets before closing the O-Line auction"); err != nil {
		return err
	}
	if err := f.ReadRules(); err != nil {
		return err
	}
	if !leaguerules.OLineOn(f.Rules) {
		return fmt.Errorf("this league does not use O-Line units")
	}
	var pending []NativePlayer
	if f.Book.OLine != nil {
		pending = slices.Clone(f.Book.OLine.Pending)
		if len(pending) == 0 {
			return fmt.Errorf("the O-Line auction is already closed and every unit is assigned")
		}
	} else {
		pending, err = initialOLineAssignments(f)
		if err != nil {
			return err
		}
	}
	outcome := &OLineOutcome{Assigned: []NativePlayer{}, Pending: []NativePlayer{}, Words: []string{}}
	for _, p := range pending {
		if err := olineRoom(f.Assets.Snapshot, f.Rules, p); err != nil {
			outcome.Pending = append(outcome.Pending, p)
			outcome.Words = append(outcome.Words, fmt.Sprintf("%s: cut a player to make room for the %s at the minimum contract. %s", olineFranchiseName(f.Assets.Snapshot, p.Franchise), p.Name, err))
			continue
		}
		outcome.Assigned = append(outcome.Assigned, p)
		outcome.Words = append(outcome.Words, fmt.Sprintf("Assigned the %s to %s at the minimum contract", p.Name, olineFranchiseName(f.Assets.Snapshot, p.Franchise)))
	}
	raw.Args.OLineOutcome = outcome
	return nil
}

func initialOLineAssignments(f Frame) ([]NativePlayer, error) {
	seed := f.Assets.Snapshot
	for _, a := range f.Book.Bids.Auctions {
		if _, card := nativecheck.LineCardTeam(a.Player); card && !a.Decided && len(a.LiveBids()) > 0 {
			return nil, fmt.Errorf("decide the standing O-Line bids before closing the auction; settle any ties first")
		}
	}
	salary, err := olineMinimum(f.Rules)
	if err != nil {
		return nil, err
	}
	franchises := slices.Clone(seed.Franchises)
	slices.SortFunc(franchises, func(a, b NativeFranchise) int { return strings.Compare(a.ID, b.ID) })
	lots := slices.Clone(seed.Pool)
	slices.SortFunc(lots, func(a, b NativePoolEntry) int {
		return cmp.Or(strings.Compare(a.Name, b.Name), strings.Compare(a.Player, b.Player))
	})
	reserved := map[string]bool{}
	pending := []NativePlayer{}
	for _, franchise := range franchises {
		held := olineCount(seed, franchise.ID)
		if held > 1 {
			return nil, fmt.Errorf("%s holds more than one O-Line unit; correct its roster first", franchise.Name)
		}
		if held == 1 {
			continue
		}
		side := seed.SideOf(franchise.ID)
		lot := slices.IndexFunc(lots, func(l NativePoolEntry) bool {
			_, card := nativecheck.LineCardTeam(l.Player)
			return card && l.Side == side && !reserved[side+":"+l.Player] && !seed.OnSide(side, l.Player)
		})
		if lot < 0 {
			return nil, fmt.Errorf("not enough leftover O-Line units for %s", franchise.Name)
		}
		l := lots[lot]
		reserved[side+":"+l.Player] = true
		pending = append(pending, NativePlayer{ID: l.Player, Name: l.Name, Franchise: franchise.ID, Status: domain.RosterActive,
			Salary: salary, ExpirationYear: f.Rules.Season, ContractStatus: domain.CStatusUFA, ContractInfo: "Assigned at the minimum contract after the O-Line auction"})
	}
	return pending, nil
}

func olineMinimum(rules leaguerules.RuleSet) (domain.Money, error) {
	if rules.Contracts.MinimumSalaries.State != leaguerules.IsKnown {
		return 0, fmt.Errorf("the minimum contract salary needs a ruling")
	}
	var minimum domain.Money
	for _, row := range rules.Contracts.MinimumSalaries.Value {
		if row.Minimum.State != leaguerules.IsKnown || row.Minimum.Value <= 0 {
			return 0, fmt.Errorf("the minimum contract salary needs a ruling")
		}
		value := domain.Money(row.Minimum.Value)
		if minimum == 0 || value < minimum {
			minimum = value
		}
	}
	if minimum == 0 {
		return 0, fmt.Errorf("the minimum contract salary needs a ruling")
	}
	return minimum, nil
}

func olineRoom(seed *RawNativeSnapshot, rules leaguerules.RuleSet, incoming NativePlayer) error {
	players := []nativecheck.CapPlayer{{Salary: incoming.Salary, Status: domain.RosterActive}}
	roster := []nativecheck.RosterPlayer{{ID: incoming.ID, Status: domain.RosterActive}}
	for _, p := range seed.Players {
		if p.Franchise == incoming.Franchise {
			players = append(players, nativecheck.CapPlayer{Salary: p.Salary, Status: p.Status})
			roster = append(roster, nativecheck.RosterPlayer{ID: p.ID, Status: p.Status})
		}
	}
	if seed.OnSide(seed.SideOf(incoming.Franchise), incoming.ID) {
		return fmt.Errorf("the reserved unit is already on a roster; correct it first")
	}
	if err := nativecheck.LineCardsHeld(rules.Roster, roster); err != nil {
		return fmt.Errorf("check assigned O-Line unit: %w", err)
	}
	charges := []domain.Money{}
	for _, c := range seed.Adjustments {
		if c.Franchise == incoming.Franchise && (c.Season == 0 || c.Season == rules.Season) {
			charges = append(charges, c.Amount)
		}
	}
	capState, err := nativecheck.CheckCap(rules.Cap, players, charges, time.UTC)
	if err != nil {
		return fmt.Errorf("check assigned O-Line salary: %w", err)
	}
	if capState.RoomCents == nil {
		return fmt.Errorf("the salary cap needs a ruling")
	}
	if *capState.RoomCents < 0 {
		return fmt.Errorf("the minimum contract would exceed the salary cap")
	}
	if rules.Roster.Limits.State != leaguerules.IsKnown || rules.Roster.Limits.Value.Max.State != leaguerules.IsKnown {
		return fmt.Errorf("the roster maximum needs a ruling")
	}
	candidate := *seed
	candidate.Players = append(slices.Clone(seed.Players), incoming)
	if rosterOverfull(&candidate, rules.Roster, incoming.Franchise) {
		return fmt.Errorf("the roster is full")
	}
	return nil
}

func olineFranchiseName(seed *RawNativeSnapshot, id string) string {
	for _, f := range seed.Franchises {
		if f.ID == id {
			return f.Name
		}
	}
	return id
}

func projectOLineClose(seed *RawNativeSnapshot, book *MarketBook, outcome *OLineOutcome) error {
	if seed == nil {
		return fmt.Errorf("record starting assets before closing the O-Line auction")
	}
	if outcome == nil {
		return fmt.Errorf("the O-Line close has no recorded assignments")
	}
	for _, p := range outcome.Assigned {
		if seed.OnSide(seed.SideOf(p.Franchise), p.ID) {
			return fmt.Errorf("an assigned O-Line unit already belongs to a roster")
		}
		seed.Players = append(seed.Players, p)
	}
	book.OLine = outcome
	return nil
}

func olineJobs(book MarketBook, seed *RawNativeSnapshot) []Job {
	jobs := []Job{}
	if book.OLine == nil {
		return jobs
	}
	for _, p := range book.OLine.Pending {
		jobs = append(jobs, Job{ID: "oline.must_cut." + p.Franchise, Name: "O-Line assignment: must cut", State: JobLive, Controls: []string{"CM-003", "CM-036"},
			Words: fmt.Sprintf("%s: cut a player to make room for the %s at the minimum contract. Assignment completes automatically once there is room.", olineFranchiseName(seed, p.Franchise), p.Name)})
	}
	return jobs
}

// OLineAssignmentsDue never opens an auction; it only retries a recorded reservation.
func OLineAssignmentsDue(league string, records []Record, now time.Time) (bool, error) {
	f, err := NewFrame(league, records, now, nil, time.UTC)
	if err != nil {
		return false, err
	}
	if f.Book.OLine == nil || len(f.Book.OLine.Pending) == 0 {
		return false, nil
	}
	if err := f.ReadRules(); err != nil {
		return false, err
	}
	if !leaguerules.OLineOn(f.Rules) {
		return false, nil
	}
	raw := RawCommand{League: league, Command: OLineAuctionClose}
	if err := PrepareOLineClose(&raw, records, now); err != nil {
		return false, err
	}
	return len(raw.Args.OLineOutcome.Assigned) > 0, nil
}

// PendingOLineCutRules allows a unit-less franchise to make the cut it was asked for.
// It does not permit cutting a held unit, acquiring a second one, or ignoring the cap.
func PendingOLineCutRules(book MarketBook, before, after *RawNativeSnapshot, rules leaguerules.RuleSet, franchise string) leaguerules.RuleSet {
	if book.OLine == nil || !slices.ContainsFunc(book.OLine.Pending, func(p NativePlayer) bool { return p.Franchise == franchise }) || len(after.Players) >= len(before.Players) {
		return rules
	}
	for _, p := range before.Players {
		if p.Franchise == franchise {
			if _, card := nativecheck.LineCardTeam(p.ID); card {
				return rules
			}
		}
	}
	rules.Roster.PositionBounds.Value = maps.Clone(rules.Roster.PositionBounds.Value)
	delete(rules.Roster.PositionBounds.Value, leaguerules.OL)
	rules.Roster.Limits.Value.Min = leaguerules.Known(0)
	return rules
}

func olineCount(seed *RawNativeSnapshot, franchise string) int {
	count := 0
	for _, p := range seed.Players {
		if _, card := nativecheck.LineCardTeam(p.ID); p.Franchise == franchise && card {
			count++
		}
	}
	return count
}

func olineLabel(outcome *OLineOutcome) string {
	if outcome == nil {
		return "Close O-Line auction"
	}
	return "Closed the O-Line auction. " + strings.Join(outcome.Words, ". ")
}
