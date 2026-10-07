package envelope

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type IRRequest struct {
	LeagueID    string
	FranchiseID string
	Player      playerid.PlayerID
}

func DraftIR(at time.Time, req IRRequest, snap snapshot.Snapshot, generate func() string) (Envelope, error) {
	// M-028's O=18 page is spec-only; ring 1 must verify it before mapping the target.
	spec := Spec{
		Intent:      "roster.ir",
		LeagueID:    req.LeagueID,
		FranchiseID: req.FranchiseID,
		Subject: Subject{
			Players: []playerid.PlayerID{req.Player},
		},
		Expected: ExpectedChange{
			Player:       req.Player,
			RosterStatus: domain.RosterIR,
		},
		Gravity: G2,
		Undo:    Reversible,
		Target: Target{
			Kind: Unmapped,
		},
	}
	e, err := New(at, spec, generate)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft IR: %w", err)
	}
	checked, err := e.Check(at, snap, IRCheck{})
	if err != nil {
		return Envelope{}, fmt.Errorf("draft IR: check: %w", err)
	}
	return checked, nil
}
