package market

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const maxWhatIfMoves = 20

// RawWhatIfStack is an ordered, unsaved owner scenario. Franchise is deliberately not an input.
type RawWhatIfStack struct {
	League     string          `json:"league"`
	ViewerZone string          `json:"viewerZone"`
	Moves      []RawWhatIfMove `json:"moves"`
}

// RawWhatIfMove carries only the inputs the contract pricer accepts for one move.
type RawWhatIfMove struct {
	Player      string       `json:"player"`
	Action      string       `json:"action"`
	Years       int          `json:"years"`
	Level       int          `json:"level"`
	MoveCents   domain.Money `json:"moveCents"`
	AmountCents domain.Money `json:"amountCents"`
	Reason      string       `json:"reason"`
}

func (r RawWhatIfStack) Validate() error {
	if err := (commissioner.RawNativeCapScope{League: r.League, ViewerZone: r.ViewerZone}).Validate(); err != nil {
		return fmt.Errorf("validate what-if scope: %w", err)
	}
	if len(r.Moves) > maxWhatIfMoves {
		return fmt.Errorf("stack at most %d what-if moves", maxWhatIfMoves)
	}
	for i, move := range r.Moves {
		if err := (RawContractQuote{League: r.League, Player: move.Player, Action: move.Action,
			ViewerZone: r.ViewerZone, Years: move.Years, Level: move.Level, MoveCents: move.MoveCents,
			AmountCents: move.AmountCents, Reason: move.Reason}).Validate(); err != nil {
			return fmt.Errorf("what-if move %d: %w", i+1, err)
		}
	}
	return nil
}

type WhatIfStack struct {
	League   string                       `json:"league"`
	Revision int64                        `json:"revision"`
	Season   int                          `json:"season"`
	Before   []commissioner.NativeCapYear `json:"before"`
	After    []commissioner.NativeCapYear `json:"after"`
	Moves    []ContractQuote              `json:"moves"`
}

// QuoteWhatIfStack applies each allowed move to a private in-memory view before quoting the next.
func QuoteWhatIfStack(raw RawWhatIfStack, records []commissioner.Record, now time.Time, zone *time.Location,
	facts nativecheck.Facts, franchise string) (WhatIfStack, error) {
	if err := raw.Validate(); err != nil {
		return WhatIfStack{}, fmt.Errorf("what-if stack input: %w", err)
	}
	base, err := commissioner.ReadNativeBase(raw.League, records, now)
	if err != nil {
		return WhatIfStack{}, fmt.Errorf("read what-if league: %w", err)
	}
	if !base.Assets.Seeded || base.Assets.Snapshot == nil {
		return WhatIfStack{}, fmt.Errorf("refuse quote without recorded rosters: %w",
			commissioner.Refused(fmt.Errorf("this app league has no recorded rosters to quote")))
	}
	if franchise == "" {
		return WhatIfStack{}, fmt.Errorf("refuse quote without owner seat: %w",
			commissioner.Refused(fmt.Errorf("a seated franchise owner is required")))
	}
	for _, move := range raw.Moves {
		if base.Assets.Snapshot.PlayerOf(franchise, move.Player) < 0 {
			return WhatIfStack{}, fmt.Errorf("refuse quote for unowned player: %w",
				commissioner.Refused(fmt.Errorf("choose a player on your franchise's roster")))
		}
	}
	before, err := commissioner.NativeCapRunway(base, franchise)
	if err != nil {
		return WhatIfStack{}, fmt.Errorf("read baseline cap runway: %w", err)
	}
	original := *base.Assets.Snapshot
	state := whatIfState{base: base, schedules: make(map[string][]nativecheck.SeasonAmount),
		removed: make(map[string]bool), statuses: make(map[string]domain.ContractStatus),
		trails: make(map[string]nativecheck.Trail), used: make(map[nativecheck.ContractAction]int),
		quotes: make([]ContractQuote, 0, len(raw.Moves))}
	out := WhatIfStack{League: raw.League, Revision: int64(len(records)), Season: base.Rules.Season, Before: before}
	if err := state.quoteMoves(raw, records, zone, facts, franchise, original); err != nil {
		return WhatIfStack{}, err
	}
	out.Moves = state.quotes
	out.After, err = commissioner.NativeCapRunwayProjected(base, franchise, commissioner.NativeCapProjection{
		Salaries: state.schedules, Removed: state.removed, Charges: state.charges,
	})
	if err != nil {
		return WhatIfStack{}, fmt.Errorf("read stacked cap runway: %w", err)
	}
	return out, nil
}

func (s *whatIfState) quoteMoves(raw RawWhatIfStack, records []commissioner.Record, zone *time.Location,
	facts nativecheck.Facts, franchise string, original commissioner.RawNativeSnapshot) error {
	blocked := false
	for _, move := range raw.Moves {
		if blocked {
			s.quotes = append(s.quotes, unquotedMove(raw.League, s.base.Rules.Season, move,
				"Not quoted because an earlier move was refused."))
			continue
		}
		if s.removed[move.Player] {
			s.quotes = append(s.quotes, unquotedMove(raw.League, s.base.Rules.Season, move,
				"A previous move removes this player from the roster."))
			blocked = true
			continue
		}
		input := RawContractQuote{League: raw.League, Player: move.Player, Franchise: franchise,
			Action: move.Action, ViewerZone: raw.ViewerZone, Years: move.Years, Level: move.Level,
			MoveCents: move.MoveCents, AmountCents: move.AmountCents, Reason: move.Reason}
		trail := s.trails[move.Player]
		trail.Used = s.used
		quote, err := quoteContractOnBase(input, records, s.base.Now, zone, facts, s.base,
			trail, s.schedules[move.Player])
		if err != nil {
			return fmt.Errorf("quote what-if move for player %s: %w", move.Player, err)
		}
		s.quotes = append(s.quotes, quote)
		if !quote.Allowed {
			blocked = true
			continue
		}
		s.apply(move, quote, franchise)
		s.base = s.readingBase(original, s.base.Rules.Season, franchise)
	}
	return nil
}

type whatIfState struct {
	base      commissioner.NativeBase
	quotes    []ContractQuote
	schedules map[string][]nativecheck.SeasonAmount
	removed   map[string]bool
	statuses  map[string]domain.ContractStatus
	trails    map[string]nativecheck.Trail
	used      map[nativecheck.ContractAction]int
	charges   []commissioner.NativeCharge
}

func (s *whatIfState) apply(move RawWhatIfMove, quote ContractQuote, franchise string) {
	action := nativecheck.ContractAction(move.Action)
	trail := s.trails[move.Player]
	s.used[action]++
	trail.Used = s.used
	if action == nativecheck.ActionRestructure {
		trail.Restructures++
	}
	if action == nativecheck.ActionExtension {
		trail.Extensions++
	}
	if action == nativecheck.ActionTag {
		trail.ConsecutiveTags++
		if quote.SalaryCent != nil {
			trail.FirstTag = *quote.SalaryCent
		}
	}
	s.trails[move.Player] = trail
	if len(quote.Salaries) > 0 {
		s.schedules[move.Player] = slices.Clone(quote.Salaries)
	}
	if actionRemovesContract(action) {
		s.removed[move.Player] = true
	}
	if action == nativecheck.ActionTag {
		s.statuses[move.Player] = domain.CStatusFT1
	}
	for i, dead := range quote.DeadMoney {
		s.charges = append(s.charges, commissioner.NativeCharge{ID: fmt.Sprintf("whatif:%d:%d", len(s.charges), i),
			Kind: "dead_money", Season: dead.Season, Franchise: franchise, Amount: dead.Cents,
			Description: "Stacked what-if dead money"})
	}
	if quote.Relief > 0 {
		s.charges = append(s.charges, commissioner.NativeCharge{ID: fmt.Sprintf("whatif:relief:%d", len(s.charges)),
			Season: quote.Season, Franchise: franchise, Amount: -quote.Relief,
			Description: "Stacked what-if cap relief"})
	}
}

func (s whatIfState) readingBase(original commissioner.RawNativeSnapshot, season int, franchise string) commissioner.NativeBase {
	seed := original
	seed.Players = make([]commissioner.NativePlayer, 0, len(original.Players))
	for _, player := range original.Players {
		if player.Franchise == franchise && s.removed[player.ID] {
			continue
		}
		if schedule, ok := s.schedules[player.ID]; ok {
			player.Salary = scheduleAmount(schedule, season, player.Salary)
			for _, cell := range schedule {
				player.ExpirationYear = max(player.ExpirationYear, cell.Season)
			}
		}
		if status, ok := s.statuses[player.ID]; ok {
			player.ContractStatus = status
		}
		seed.Players = append(seed.Players, player)
	}
	seed.Adjustments = append(slices.Clone(original.Adjustments), s.charges...)
	assets := s.base.Assets
	assets.Snapshot = &seed
	base := s.base
	base.Assets = assets
	return base
}

func scheduleAmount(schedule []nativecheck.SeasonAmount, season int, fallback domain.Money) domain.Money {
	for _, cell := range schedule {
		if cell.Season == season {
			return cell.Cents
		}
	}
	return fallback
}

func actionRemovesContract(action nativecheck.ContractAction) bool {
	return action == nativecheck.ActionCut || action == nativecheck.ActionBuyout ||
		action == nativecheck.ActionRetire || action == nativecheck.ActionDeath
}

func unquotedMove(league string, season int, move RawWhatIfMove, reason string) ContractQuote {
	action := nativecheck.ContractAction(move.Action)
	return ContractQuote{League: league, Season: season, Player: move.Player, Action: move.Action,
		ActionName: action.Name(), Reason: reason, Allowed: false}
}
