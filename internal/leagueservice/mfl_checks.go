package leagueservice

import (
	"context"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/mflmirror"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	MflMovesBidCheck    = "mflmoves.bid.check"
	MflMovesLineupCheck = "mflmoves.lineup.check"

	mflPickTeamFirst = "Pick your MFL team first."
	mflRefreshFirst  = "Refresh first: the held copy has no MFL rosters or rules yet."
)

// The checks and plans themselves run in the MFL engine the host's fetcher carries (mflmirror.Engine); this
// file and mfl_plans.go keep the seat, the inputs and the plain refusals.
type (
	MflBidCheck    = mflmirror.BidCheck
	MflLineupCheck = mflmirror.LineupCheck
	MflPlan        = mflmirror.Plan
	MflPlans       = mflmirror.Plans
	MflPlanLink    = mflmirror.PlanLink
)

// RawMflBidCheck is one bid to judge: the player to add, the one annual salary in cents, the years, and the
// players to drop to make room.
type RawMflBidCheck struct {
	League      string       `json:"league"`
	Franchise   string       `json:"franchise,omitempty"`
	Add         string       `json:"add"`
	AmountCents domain.Money `json:"amountCents"`
	Years       int          `json:"years,omitempty"`
	Drop        []string     `json:"drop,omitempty"`
}

func (r RawMflBidCheck) Validate() error {
	if err := (RawMflMovesScope{League: r.League, Franchise: r.Franchise}).Validate(); err != nil {
		return err
	}
	if _, err := playerid.New(r.Add); err != nil {
		return fmt.Errorf("choose the player to add")
	}
	if r.AmountCents <= 0 || r.AmountCents > 1e11 {
		return fmt.Errorf("enter one annual salary above $0")
	}
	if r.Years < 0 || r.Years > bid.MaxYears {
		return fmt.Errorf("a bid runs 1 to %d years", bid.MaxYears)
	}
	if len(r.Drop) > 8 {
		return fmt.Errorf("drop at most 8 players")
	}
	for _, d := range r.Drop {
		if _, err := playerid.New(d); err != nil {
			return fmt.Errorf("choose players to drop from your roster")
		}
	}
	return nil
}

// RawMflLineupCheck is one week's starters, as MFL player ids.
type RawMflLineupCheck struct {
	League    string   `json:"league"`
	Franchise string   `json:"franchise,omitempty"`
	Week      int      `json:"week"`
	Starters  []string `json:"starters"`
}

func (r RawMflLineupCheck) Validate() error {
	if err := (RawMflMovesScope{League: r.League, Franchise: r.Franchise}).Validate(); err != nil {
		return err
	}
	if r.Week < 1 || r.Week > 22 {
		return fmt.Errorf("choose the NFL week, 1 to 22")
	}
	if len(r.Starters) == 0 || len(r.Starters) > 40 {
		return fmt.Errorf("choose your starters")
	}
	seen := map[playerid.PlayerID]bool{}
	for _, s := range r.Starters {
		id, err := playerid.New(s)
		if err != nil {
			return fmt.Errorf("choose starters from your roster")
		}
		if seen[id] {
			return fmt.Errorf("a starter is listed twice")
		}
		seen[id] = true
	}
	return nil
}

func (a *Service) mflEngine() (mflmirror.Engine, error) {
	if engine, ok := a.mflSource().(mflmirror.Engine); ok && engine != nil {
		return engine, nil
	}
	return nil, fmt.Errorf("MFL checks are not available on this host")
}

// mflOwnerSeat is the seated owner's request with the held copy, the picked team and the rules loaded. Checks
// and saves need the picked team and the held rosters; listing and changing plans need neither.
func (a *Service) mflOwnerSeat(ctx context.Context, league, operation, mode string, needTeam bool) (context.Context, mflmirror.Seat, func(), error) {
	ctx, seat, err := a.mflAccess(ctx, league, operation, mode)
	if err != nil {
		return nil, mflmirror.Seat{}, nil, err
	}
	out := mflmirror.Seat{League: league, Franchise: seat.franchise, MFLLeague: seat.mflLeague,
		Pools: seat.store.Pools(), Now: a.host.CommissionerNow()}
	fail := func(err error) (context.Context, mflmirror.Seat, func(), error) {
		seat.cancel()
		return nil, mflmirror.Seat{}, nil, err
	}
	team, hasTeam, err := seat.store.Team(ctx, seat.franchise, seat.mflLeague)
	if err != nil {
		return fail(fmt.Errorf("read your MFL team: %w", err))
	}
	if needTeam && !hasTeam {
		return fail(commissioner.Refused(fmt.Errorf("%s", mflPickTeamFirst)))
	}
	out.Team = team
	held, found, err := seat.store.Read(ctx, seat.mflLeague)
	if err != nil {
		return fail(fmt.Errorf("read held MFL copy: %w", err))
	}
	if needTeam && (!found || !held.HasLeague) {
		return fail(commissioner.Refused(fmt.Errorf("%s", mflRefreshFirst)))
	}
	if found {
		out.Held = held
		if source := a.mflSource(); source != nil {
			if lookup, lerr := source.Directory(held.PlayersJSON); lerr == nil {
				out.Lookup = lookup
			} else if needTeam {
				return fail(fmt.Errorf("held MFL player directory: %w", lerr))
			}
		}
	}
	if rules, rerr := commissioner.RulesNow(seat.records, out.Now); rerr == nil {
		out.Rules, out.RulesOK = rules, true
	}
	return ctx, out, seat.cancel, nil
}

func (a *Service) MflMovesBidCheck(raw RawMflBidCheck) (MflBidCheck, error) {
	return a.MflMovesBidCheckContext(a.host.CommissionerContext(), raw)
}

// MflMovesBidCheckContext judges one bid on the owner's MFL team from the held copy; it never calls MFL
// and saves nothing.
func (a *Service) MflMovesBidCheckContext(ctx context.Context, raw RawMflBidCheck) (MflBidCheck, error) {
	if err := raw.Validate(); err != nil {
		return MflBidCheck{}, fmt.Errorf("MFL bid check input: %w", err)
	}
	ctx, seat, done, err := a.mflOwnerSeat(ctx, raw.League, MflMovesBidCheck, "read", true)
	if err != nil {
		return MflBidCheck{}, err
	}
	defer done()
	// The one-drop rule is a refusal in plain words, not an input error: Validate runs while the hosted
	// request is decoded, where any error becomes "invalid native request".
	if len(raw.Drop) > 1 {
		return MflBidCheck{}, mflRefused("a bid can hold one drop")
	}
	engine, err := a.mflEngine()
	if err != nil {
		return MflBidCheck{}, err
	}
	return engine.BidCheck(ctx, seat, mflmirror.BidInput{Add: raw.Add, AmountCents: raw.AmountCents, Years: raw.Years, Drop: raw.Drop}), nil
}

func (a *Service) MflMovesLineupCheck(raw RawMflLineupCheck) (MflLineupCheck, error) {
	return a.MflMovesLineupCheckContext(a.host.CommissionerContext(), raw)
}

// MflMovesLineupCheckContext judges one week's lineup on the owner's MFL team from the held copy.
func (a *Service) MflMovesLineupCheckContext(ctx context.Context, raw RawMflLineupCheck) (MflLineupCheck, error) {
	if err := raw.Validate(); err != nil {
		return MflLineupCheck{}, fmt.Errorf("MFL lineup check input: %w", err)
	}
	ctx, seat, done, err := a.mflOwnerSeat(ctx, raw.League, MflMovesLineupCheck, "read", true)
	if err != nil {
		return MflLineupCheck{}, err
	}
	defer done()
	engine, err := a.mflEngine()
	if err != nil {
		return MflLineupCheck{}, err
	}
	out, err := engine.LineupCheck(ctx, seat, mflmirror.LineupInput{Week: raw.Week, Starters: raw.Starters})
	return out, mflRefusal(err)
}
