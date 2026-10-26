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
