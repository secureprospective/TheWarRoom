package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type RosterPlanReview struct {
	Store MoveStore
	Now   func() time.Time
	Log   func(string)
	Audit func(envelope.Envelope)
}

func (s RosterPlanReview) Decision(ctx context.Context, snap snapshot.Snapshot, id string) (eligibility.Decision, bool) {
	e, err := s.Store.Get(ctx, id)
	if err != nil || e.State() != envelope.Ready {
		return eligibility.Decision{}, false
	}
	spec := e.Receipt().Spec
	verdicts, ok := snap.Eligibility.Players[spec.Expected.Player.String()]
	if !ok {
		return eligibility.Decision{}, false
	}
	if spec.Intent == "roster.taxi" {
		return verdicts.Taxi, true
	}
	return verdicts.IR, true
}

func (s RosterPlanReview) ReturnToDraft(ctx context.Context, id string, decision eligibility.Decision, found bool) bool {
	if !found || decision.Verdict != eligibility.Ineligible {
		return false
	}
	e, err := s.Store.Get(ctx, id)
	if err != nil {
		s.Log(fmt.Sprintf("revalidate moves: load %s: %v", id, err))
		return false
	}
	if e.State() != envelope.Ready {
		return false
	}
	now := s.Now()
	reason := "player no longer eligible: " + decision.Reason
	invalidated, err := e.Invalidate(now, reason)
	if err != nil {
		s.Log(fmt.Sprintf("revalidate moves: invalidate %s: %v", id, err))
		return false
	}
	drafted, err := invalidated.Rebase(now, reason)
	if err != nil {
		s.Log(fmt.Sprintf("revalidate moves: rebase %s: %v", id, err))
		return false
	}
	if err := s.Store.Save(ctx, drafted); err != nil {
		s.Log(fmt.Sprintf("revalidate moves: save %s: %v", id, err))
		return false
	}
	s.Audit(drafted)
	return true
}

type RosterRevalidation struct {
	Release            func(context.Context)
	Contracts          func(context.Context) error
	ContractsAvailable bool
	Ready              func() error
	List               func(context.Context) ([]envelope.Receipt, error)
	Snapshot           func(context.Context) (snapshot.Snapshot, error)
	Lock, Unlock       func()
	Decision           func(context.Context, snapshot.Snapshot, string) (eligibility.Decision, bool)
	Return             func(context.Context, string, eligibility.Decision, bool) bool
	Changed            func(context.Context)
	Log                func(string)
}

func (s RosterRevalidation) Run(ctx context.Context) {
	s.Release(ctx)
	if s.ContractsAvailable {
		if err := s.Contracts(ctx); err != nil {
			s.Log(fmt.Sprintf("contract validation: %v", err))
		}
	}
	if err := s.Ready(); err != nil {
		return
	}
	receipts, err := s.List(ctx)
	if err != nil {
		s.Log(fmt.Sprintf("revalidate moves: list: %v", err))
		return
	}
	candidates := []string{}
	for _, r := range receipts {
		if r.State == envelope.Ready && (r.Spec.Intent == "roster.ir" || r.Spec.Intent == "roster.taxi") {
			candidates = append(candidates, r.CorrelationID)
		}
	}
	if len(candidates) == 0 {
		return
	}
	snap, err := s.Snapshot(ctx)
	if err != nil {
		s.Log(fmt.Sprintf("revalidate moves: build: %v", err))
		return
	}
	s.Lock()
	defer s.Unlock()
	changed := false
	for _, id := range candidates {
		decision, found := s.Decision(ctx, snap, id)
		if s.Return(ctx, id, decision, found) {
			changed = true
		}
	}
	if changed {
		s.Changed(ctx)
	}
}

func RevalidateRelease(ctx context.Context, available bool, validate func(context.Context) error, logLine func(string)) {
	if !available {
		return
	}
	if err := validate(ctx); err != nil {
		logLine(fmt.Sprintf("release validation: %v", err))
	}
}
