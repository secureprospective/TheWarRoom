package leagueservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/mflmirror"
)

const (
	MflMovesPlanSave     = "mflmoves.plan.save"
	MflMovesPlansRead    = "mflmoves.plans.read"
	MflMovesPlanLink     = "mflmoves.plan.link"
	MflMovesPlanDone     = "mflmoves.plan.done"
	MflMovesPlanWithdraw = "mflmoves.plan.withdraw"

	mflPlanNotFound = mflmirror.PlanNotFound
)

// RawMflPlanSave saves one checked plan. Kind is "bid" or "lineup"; Body holds that kind's fields:
// a bid takes add, amount, years, drop (at most one) and snipe; a lineup takes week and starters.
type RawMflPlanSave struct {
	League    string          `json:"league"`
	Franchise string          `json:"franchise,omitempty"`
	Kind      string          `json:"kind"`
	Body      json.RawMessage `json:"body"`
}

func (r RawMflPlanSave) Validate() error {
	if err := (RawMflMovesScope{League: r.League, Franchise: r.Franchise}).Validate(); err != nil {
		return err
	}
	if r.Kind != "bid" && r.Kind != "lineup" {
		return fmt.Errorf("a plan is a bid or a lineup")
	}
	if len(r.Body) == 0 || len(r.Body) > 8192 {
		return fmt.Errorf("give the plan's details")
	}
	return nil
}

// RawMflPlanScope names one plan of the seat's own.
type RawMflPlanScope struct {
	League    string `json:"league"`
	Franchise string `json:"franchise,omitempty"`
	Plan      string `json:"plan"`
}

func (r RawMflPlanScope) Validate() error {
	if err := (RawMflMovesScope{League: r.League, Franchise: r.Franchise}).Validate(); err != nil {
		return err
	}
	if r.Plan == "" || len(r.Plan) > 64 || strings.TrimSpace(r.Plan) != r.Plan {
		return fmt.Errorf("choose one of your plans")
	}
	return nil
}

func (a *Service) MflMovesPlanSave(raw RawMflPlanSave) (MflPlan, error) {
	return a.MflMovesPlanSaveContext(a.host.CommissionerContext(), raw)
}

// MflMovesPlanSaveContext re-runs the check on the server and saves the plan for the seated franchise only.
func (a *Service) MflMovesPlanSaveContext(ctx context.Context, raw RawMflPlanSave) (MflPlan, error) {
	if err := raw.Validate(); err != nil {
		return MflPlan{}, fmt.Errorf("MFL plan input: %w", err)
	}
	ctx, seat, done, err := a.mflOwnerSeat(ctx, raw.League, MflMovesPlanSave, "execute", true)
	if err != nil {
		return MflPlan{}, err
	}
	defer done()
	engine, err := a.mflEngine()
	if err != nil {
		return MflPlan{}, err
	}
	out, err := engine.PlanSave(ctx, seat, raw.Kind, raw.Body)
	return out, mflRefusal(err)
}

func (a *Service) MflMovesPlansRead(raw RawMflMovesScope) (MflPlans, error) {
	return a.MflMovesPlansReadContext(a.host.CommissionerContext(), raw)
}

// MflMovesPlansReadContext lists the seated franchise's own plans, newest first. Results are never cached.
func (a *Service) MflMovesPlansReadContext(ctx context.Context, raw RawMflMovesScope) (MflPlans, error) {
	if err := raw.Validate(); err != nil {
		return MflPlans{}, fmt.Errorf("MFL plans input: %w", err)
	}
	ctx, seat, done, err := a.mflOwnerSeat(ctx, raw.League, MflMovesPlansRead, "read", false)
	if err != nil {
		return MflPlans{}, err
	}
	defer done()
	engine, err := a.mflEngine()
	if err != nil {
		return MflPlans{}, err
	}
	out, err := engine.PlansRead(ctx, seat)
	return out, mflRefusal(err)
}

// planStep is one plan change or reading on the seat's own plan.
func (a *Service) planStep(ctx context.Context, raw RawMflPlanScope, operation, mode string,
	run func(mflmirror.Engine, context.Context, mflmirror.Seat, string) error) error {
	if err := raw.Validate(); err != nil {
		return fmt.Errorf("MFL plan input: %w", err)
	}
	ctx, seat, done, err := a.mflOwnerSeat(ctx, raw.League, operation, mode, false)
	if err != nil {
		return err
	}
	defer done()
	engine, err := a.mflEngine()
	if err != nil {
		return err
	}
	return mflRefusal(run(engine, ctx, seat, raw.Plan))
}

func (a *Service) MflMovesPlanLink(raw RawMflPlanScope) (MflPlanLink, error) {
	return a.MflMovesPlanLinkContext(a.host.CommissionerContext(), raw)
}

// MflMovesPlanLinkContext returns the MFL page for a plan. It only names the page; the server never opens it.
func (a *Service) MflMovesPlanLinkContext(ctx context.Context, raw RawMflPlanScope) (MflPlanLink, error) {
	var out MflPlanLink
	err := a.planStep(ctx, raw, MflMovesPlanLink, "read", func(e mflmirror.Engine, ctx context.Context, s mflmirror.Seat, plan string) (err error) {
		out, err = e.PlanLink(ctx, s, plan)
		return err //nolint:wrapcheck // the engine speaks plain words
	})
	return out, err
}

func (a *Service) MflMovesPlanDone(raw RawMflPlanScope) (MflPlan, error) {
	return a.MflMovesPlanDoneContext(a.host.CommissionerContext(), raw)
}

// MflMovesPlanDoneContext records the owner's word that they did the step. It is a local change only; nothing is sent to MFL.
func (a *Service) MflMovesPlanDoneContext(ctx context.Context, raw RawMflPlanScope) (MflPlan, error) {
	var out MflPlan
	err := a.planStep(ctx, raw, MflMovesPlanDone, "execute", func(e mflmirror.Engine, ctx context.Context, s mflmirror.Seat, plan string) (err error) {
		out, err = e.PlanDone(ctx, s, plan)
		return err //nolint:wrapcheck // the engine speaks plain words
	})
	return out, err
}

func (a *Service) MflMovesPlanWithdraw(raw RawMflPlanScope) (MflPlan, error) {
	return a.MflMovesPlanWithdrawContext(a.host.CommissionerContext(), raw)
}

// MflMovesPlanWithdrawContext ends a plan that is still only planned. A step already marked done stays on record.
func (a *Service) MflMovesPlanWithdrawContext(ctx context.Context, raw RawMflPlanScope) (MflPlan, error) {
	var out MflPlan
	err := a.planStep(ctx, raw, MflMovesPlanWithdraw, "execute", func(e mflmirror.Engine, ctx context.Context, s mflmirror.Seat, plan string) (err error) {
		out, err = e.PlanWithdraw(ctx, s, plan)
		return err //nolint:wrapcheck // the engine speaks plain words
	})
	return out, err
}
