package contracts

import (
	"context"
	"fmt"
	"slices"

	mflwhatif "github.com/secureprospective/TheWarRoom/internal/mflref/whatif"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// WhatIf is the twin of the root RawWhatIfRequest: the same fields in the same order.
type WhatIf struct {
	Mine   string `json:"mine"`
	Action string `json:"action"`
	Player string `json:"player"`
	Move   string `json:"move"`
}

func (r WhatIf) Validate() error {
	if err := (snapshot.RawContractScope{Mine: r.Mine, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("what-if scope: %w", err)
	}
	if r.Mine == "" {
		return fmt.Errorf("what-if: choose your franchise")
	}
	if !slices.Contains([]string{"read", "cut", "buyout", "undo", "reset"}, r.Action) {
		return fmt.Errorf("what-if: unknown action %q", r.Action)
	}
	if r.Action == "cut" || r.Action == "buyout" {
		if _, err := playerid.New(r.Player); err != nil {
			return fmt.Errorf("what-if player: %w", err)
		}
	}
	if r.Action == "undo" && r.Move == "" {
		return fmt.Errorf("what-if: missing move")
	}
	return nil
}

// WhatIfMove is a modelled cut or buyout; it is main.WhatIfMoveView.
type WhatIfMove struct {
	ID          string                  `json:"id"`
	Player      string                  `json:"player"`
	Tag         string                  `json:"tag"`
	Status      string                  `json:"status"`
	Label       string                  `json:"label"`
	Charge      snapshot.ContractAmount `json:"charge"`
	Explanation string                  `json:"explanation"`
	Plan        string                  `json:"plan"`
}

// WhatIfFigure is one figure before and after; it is main.WhatIfFigure.
type WhatIfFigure struct {
	Label  string `json:"label"`
	Before string `json:"before"`
	After  string `json:"after"`
	Change string `json:"change"`
}

// WhatIfChoice is what the owner may model for one held player; it is main.WhatIfChoice.
type WhatIfChoice struct {
	Player        string `json:"player"`
	CutLabel      string `json:"cutLabel"`
	BuyoutLabel   string `json:"buyoutLabel"`
	BuyoutAllowed bool   `json:"buyoutAllowed"`
	Reason        string `json:"reason"`
}

// WhatIfView is the what-if as the owner reads it; it is main.WhatIfView.
type WhatIfView struct {
	Stale   bool           `json:"stale"`
	Banner  string         `json:"banner"`
	Empty   string         `json:"empty"`
	Moves   []WhatIfMove   `json:"moves"`
	Figures []WhatIfFigure `json:"figures"`
	Choices []WhatIfChoice `json:"choices"`
	Notice  string         `json:"notice"`
}

// Ledger is the what-if's replayed ledger, read for cap used and each player's future cells.
type Ledger interface {
	CapUsed(franchiseID string) (domain.Money, bool)
	LedgerCells(ctx context.Context, mflID string) (map[int]domain.Money, error)
}

// DescribeWhatIf reads the what-if's figures through horizon: cap used and room before and after the modelled
// moves, and each later season's commitments. label marks the moves that are open plans; it is the
// caller's, which reads the plan ledger.
func DescribeWhatIf(ctx context.Context, s mflwhatif.WhatIfSession, ledger Ledger, stale bool, horizon int, label func([]WhatIfMove) error) (WhatIfView, error) {
	out := WhatIfView{Stale: stale, Moves: []WhatIfMove{}, Figures: []WhatIfFigure{}, Choices: []WhatIfChoice{}, Empty: "Model a cut or buyout from your team's contract rows; nothing is sent anywhere."}
	if stale {
		out.Banner = "MFL changed since this what-if started"
	}
	if ledger == nil {
		return out, fmt.Errorf("what-if: current ledger unavailable")
	}
	moves, removed, charge := Moves(s)
	out.Moves = moves
	if err := label(out.Moves); err != nil {
		return out, err
	}

	beforeFuture, afterFuture := map[int]domain.Money{}, map[int]domain.Money{}
	last := max(s.Snapshot.Season, horizon)
	for _, p := range s.Snapshot.Players {
		if p.FranchiseID != s.Mine {
			continue
		}
		last = max(last, p.ContractYear)
		for year := s.Snapshot.Season + 1; year <= p.ContractYear; year++ {
			beforeFuture[year] += p.Salary
		}
		if removed[p.MFLID] {
			continue
		}
		if err := Committed(ctx, ledger, p, s.Snapshot.Season, afterFuture); err != nil {
			return out, err
		}
		out.Choices = append(out.Choices, Choice(p, s.Snapshot.Season))
	}
	used, _ := ledger.CapUsed(s.Mine)
	used = s.Used + used - s.SeedUsed
	capText, roomBefore, roomAfter := "Unavailable", "Unavailable", "Unavailable"
	if s.Cap != nil {
		capText = snapshot.ContractAmountOf(*s.Cap).Text
		roomBefore = snapshot.ContractAmountOf(*s.Cap - s.Used).Text
		roomAfter = snapshot.ContractAmountOf(*s.Cap - used).Text
	}
	out.Figures = append(out.Figures, WhatIfFigure{Label: fmt.Sprintf("%d cap", s.Snapshot.Season), Before: capText, After: capText}, Figure("Used", s.Used, used), WhatIfFigure{Label: "Room", Before: roomBefore, After: roomAfter}, Figure("Modelled dead cap this season (MFL adjustments excluded)", 0, charge))
	if s.Cap != nil {
		out.Figures[2].Change = Change(s.Used - used)
	}
	for year := s.Snapshot.Season + 1; year <= last; year++ {
		out.Figures = append(out.Figures, Figure(fmt.Sprintf("%d commitments at today's salaries", year), beforeFuture[year], afterFuture[year]))
	}
	return out, nil
}

// Committed adds a player's ledger cells after this season to into, keeping the held salary's sub-$10,000 cents.
func Committed(ctx context.Context, ledger Ledger, p state.MirrorPlayer, season int, into map[int]domain.Money) error {
	cells, err := ledger.LedgerCells(ctx, p.MFLID)
	if err != nil {
		return fmt.Errorf("what-if future cells: %w", err)
	}
	for year, cents := range cells {
		if year > season {
			into[year] += cents + p.Salary - domain.RoundToNearest10k(p.Salary)
		}
	}
	return nil
}

// Rule is the rule a modelled move's charge follows.
func Rule(kind string, p state.PlayerState, season int) string {
	if kind == string(transactions.KindBuyout) {
		return "Offseason only: can't be done until the offseason. Counts against two buyouts this season."
	}
	if p.ExpirationYear <= season {
		return "Final paid year: no paid cells after this season, so $0 dead cap."
	}
	if p.IsRestructured {
		return "50% of remaining cells: restructured."
	}
	return "35% assumed: MFL exports do not establish whether this contract was restructured."
}

// Choice is what the owner may model for a held player: a cut always, a buyout with 2–4 paid years left.
func Choice(p state.MirrorPlayer, season int) WhatIfChoice {
	remaining := p.ContractYear - season
	allowed := remaining >= 2 && remaining <= 4
	reason := "Offseason only: can't be done until the offseason."
	if !allowed {
		reason = "Buyout not offered: needs 2–4 paid years after this season; a 1-year-left exit uses a cut."
	}
	return WhatIfChoice{Player: p.MFLID, CutLabel: "What if I cut him", BuyoutLabel: "What if I buy him out", BuyoutAllowed: allowed, Reason: reason}
}

// Moves words the modelled moves, with the players they remove and their summed charge.
func Moves(s mflwhatif.WhatIfSession) ([]WhatIfMove, map[string]bool, domain.Money) {
	moves := []WhatIfMove{}
	removed := map[string]bool{}
	var charge domain.Money
	for _, m := range s.Moves {
		verb, tag, status := "Cut ", "Cut in what-if", "cut "
		if m.Kind == string(transactions.KindBuyout) {
			verb, tag, status = "Buy out ", "Bought out in what-if", "buyout "
		}
		moves = append(moves, WhatIfMove{ID: m.ID, Player: m.Player, Tag: tag, Status: "Modelled: " + status + m.Name + ", " + snapshot.ContractAmountOf(m.Charge).Text + "; see What-if at the top", Label: verb + m.Name, Charge: snapshot.ContractAmountOf(m.Charge), Explanation: m.Explanation})
		removed[m.Player] = true
		charge += m.Charge
	}
	return moves, removed, charge
}

// Figure is a money figure before and after, with its signed change.
func Figure(label string, before, after domain.Money) WhatIfFigure {
	return WhatIfFigure{Label: label, Before: snapshot.ContractAmountOf(before).Text, After: snapshot.ContractAmountOf(after).Text, Change: Change(after - before)}
}

// Change is a signed money change, empty when nothing changed.
func Change(delta domain.Money) string {
	if delta == 0 {
		return ""
	}
	if delta < 0 {
		return "−" + snapshot.ContractAmountOf(-delta).Text
	}
	return "+" + snapshot.ContractAmountOf(delta).Text
}
