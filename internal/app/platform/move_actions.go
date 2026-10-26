package platform

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

// MoveStore is the local envelope ledger, not an MFL writer.
type MoveStore interface {
	Get(context.Context, string) (envelope.Envelope, error)
	Save(context.Context, envelope.Envelope) error
	Awaiting(context.Context) ([]envelope.Envelope, error)
	BidFacts(context.Context, string) (envelope.BidFacts, error)
	ContractFacts(context.Context, string) (envelope.ContractFacts, error)
}
type MoveActions struct {
	Store         MoveStore
	Own           func(string) error
	Now           func() time.Time
	Open          func(string)
	Audit         func(envelope.Envelope)
	Changed, Wake func()
}

func (s MoveActions) HandOff(ctx context.Context, correlationID string) (envelope.Receipt, error) {
	e, err := s.Store.Get(ctx, correlationID)
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target hand-off: load: %w", err)
	}
	if err := s.Own(e.Receipt().Spec.FranchiseID); err != nil {
		return envelope.Receipt{}, err
	}
	if envelope.IsTrade(e.Receipt().Spec.Intent) && e.Receipt().Spec.Intent != "trade.accept" {
		return envelope.Receipt{}, fmt.Errorf("trade Act needs its trade-desk pre-flight and confirmation")
	}
	if envelope.IsRelease(e.Receipt().Spec.Intent) {
		return envelope.Receipt{}, fmt.Errorf("target hand-off: release needs its confirmation and checks")
	}
	next, err := e.HandOff(s.Now())
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target hand-off: %w", err)
	}
	target := next.Receipt().Spec.Target.URL
	if err := ValidateMoveURL(target); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target hand-off from %s: %w", e.State(), err)
	}
	if err := s.Store.Save(ctx, next); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target hand-off: save: %w", err)
	}
	s.Open(target)
	s.Audit(next)
	s.Changed()
	s.Wake()
	return next.Receipt(), nil
}

func (s MoveActions) Withdraw(ctx context.Context, correlationID string) (envelope.Receipt, error) {
	e, err := s.Store.Get(ctx, correlationID)
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("withdraw roster: load: %w", err)
	}
	if err := s.Own(e.Receipt().Spec.FranchiseID); err != nil {
		return envelope.Receipt{}, err
	}
	next, err := e.WithdrawRoster(s.Now())
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("withdraw roster: %w", err)
	}
	if err := s.Store.Save(ctx, next); err != nil {
		return envelope.Receipt{}, fmt.Errorf("withdraw roster: save: %w", err)
	}
	s.Audit(next)
	s.Changed()
	return next.Receipt(), nil
}

func RequestMovesPass(ready func() error, wake func()) error {
	if err := ready(); err != nil {
		return fmt.Errorf("target check moves: startup: %w", err)
	}
	wake()
	return nil
}
func MoveLog(e envelope.Envelope) string {
	r := e.Receipt()
	entry := r.Audit[len(r.Audit)-1]
	return fmt.Sprintf("move %s %s → %s: %s", e.ID(), entry.From, entry.To, entry.Note)
}

// MoveWatch refreshes without the ledger lock, then rereads under it before observing.
type MoveWatch struct {
	Store        MoveStore
	Unknown      map[string]bool
	Log          func(string)
	Audit        func(envelope.Envelope)
	Now          func() time.Time
	Observation  func(context.Context, map[envelope.Source]bool) (envelope.Observation, error)
	Lock, Unlock func()
	Changed      func(context.Context)
}

func (s MoveWatch) WithBidFacts(ctx context.Context, id string, p envelope.Predicate) (envelope.Predicate, error) {
	if _, ok := p.(envelope.ContractPredicate); ok {
		facts, err := s.Store.ContractFacts(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("watch moves: contract facts: %w", err)
		}
		return envelope.ContractPredicate{Facts: facts}, nil
	}
	if _, ok := p.(envelope.BidPredicate); !ok {
		return p, nil
	}
	facts, err := s.Store.BidFacts(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("watch moves: bid facts %s: %w", id, err)
	}
	return envelope.BidPredicate{Facts: facts}, nil
}

func (s MoveWatch) Sources(entries []envelope.Envelope) map[envelope.Source]bool {
	sources := make(map[envelope.Source]bool)
	for _, e := range entries {
		intent := e.Receipt().Spec.Intent
		predicate := MovePredicate(intent)
		if predicate == nil {
			if !s.Unknown[intent] {
				s.Unknown[intent] = true
				s.Log(fmt.Sprintf("move intent %s: no landing predicate", intent))
			}
			continue
		}
		for _, source := range predicate.Sources() {
			sources[source] = true
		}
	}
	return sources
}

func (s MoveWatch) Check(ctx context.Context) error {
	entries, err := s.Store.Awaiting(ctx)
	if err != nil {
		return fmt.Errorf("watch moves: awaiting: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}
	sources := s.Sources(entries)
	if len(sources) == 0 {
		return nil
	}
	obs, err := s.Observation(ctx, sources)
	if err != nil {
		return err
	}
	// Bindings take movesMu too, so it covers the local observe-and-save only, never MFL.
	s.Lock()
	defer s.Unlock()
	changed := false
	defer func() {
		if changed {
			s.Changed(ctx)
		}
	}()
	for _, listed := range entries {
		predicate := MovePredicate(listed.Receipt().Spec.Intent)
		if predicate == nil {
			continue
		}
		// Re-read: a binding may have moved it while MFL was being fetched.
		e, err := s.Store.Get(ctx, listed.ID())
		if err != nil {
			return fmt.Errorf("watch moves: reload %s: %w", listed.ID(), err)
		}
		if predicate, err = s.WithBidFacts(ctx, e.ID(), predicate); err != nil {
			return err
		}
		if !WatchableMove(e, listed.State()) {
			continue
		}
		next, err := e.Observe(s.Now(), obs, predicate)
		if errors.Is(err, envelope.ErrStaleObservation) {
			continue
		}
		if err != nil {
			return fmt.Errorf("watch moves: observe %s: %w", e.ID(), err)
		}
		if !SaveMoveEvidence(e, next) {
			continue
		}
		if err := s.Store.Save(ctx, next); err != nil {
			return fmt.Errorf("watch moves: save %s: %w", e.ID(), err)
		}
		changed = true
		s.Audit(next)
	}
	return nil
}
