package planwatch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// RosterDraft prepares only local what-if envelopes from an already-held snapshot.
type RosterDraft struct {
	Ready        func() error
	Own          func(string) error
	Snapshot     func() (snapshot.Snapshot, bool)
	Lock, Unlock func()
	Target       func(string) (envelope.Target, []string)
	League       string
	Parent       func() context.Context
	Timeout      time.Duration
	Save         func(context.Context, envelope.Envelope) error
}

func (s RosterDraft) Draft(intent, franchiseID, playerID string) (envelope.Receipt, error) {
	if err := s.Ready(); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: startup: %w", err)
	}
	if err := s.Own(franchiseID); err != nil {
		return envelope.Receipt{}, err
	}
	pid, err := playerid.New(playerID)
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: player: %w", err)
	}
	snap, loaded := s.Snapshot()
	if !loaded {
		return envelope.Receipt{}, fmt.Errorf("target draft: no snapshot loaded")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: correlation ID: %w", err)
	}
	s.Lock()
	defer s.Unlock()
	option, prefix := "18", "ir-"
	if intent == "roster.taxi" {
		option, prefix = "98", "taxi-"
	}
	target, blocks := s.Target(option)
	req := envelope.IRRequest{
		LeagueID:    s.League,
		FranchiseID: franchiseID,
		Player:      pid,
		Target:      target,
		Blocks:      blocks,
	}
	id := prefix + hex.EncodeToString(random[:])
	var e envelope.Envelope
	if intent == "roster.taxi" {
		e, err = envelope.DraftTaxi(time.Now().UTC(), envelope.TaxiRequest(req), snap, func() string { return id })
	} else {
		e, err = envelope.DraftIR(time.Now().UTC(), req, snap, func() string { return id })
	}
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: %w", err)
	}
	ctx, cancel := context.WithTimeout(s.Parent(), s.Timeout)
	defer cancel()
	if err := s.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft: log: %w", err)
	}
	return e.Receipt(), nil
}

// RosterRail omits plans whose richer facts have a dedicated desk.
func RosterRail(receipts []envelope.Receipt) []envelope.Receipt {
	return slices.DeleteFunc(receipts, func(r envelope.Receipt) bool {
		return r.Spec.Intent == envelope.BidIntent || envelope.IsRelease(r.Spec.Intent) || envelope.IsContract(r.Spec.Intent)
	})
}
