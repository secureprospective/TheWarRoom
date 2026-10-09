package envelope

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type IRRequest struct {
	LeagueID    string
	FranchiseID string
	Player      playerid.PlayerID
	Target      Target
	Blocks      []string
}

func DraftIR(at time.Time, req IRRequest, snap snapshot.Snapshot, generate func() string) (Envelope, error) {
	// O=18 opens IR management; MFL does not honor player or franchise prefill.
	if req.Target.Kind == "" {
		req.Target.Kind = Unmapped
	}
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
		Target:  req.Target,
	}
	e, err := New(at, spec, generate)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft IR: %w", err)
	}
	checked, err := e.Check(at, snap, rosterDraftCheck{check: IRCheck{}, blocks: req.Blocks})
	if err != nil {
		return Envelope{}, fmt.Errorf("draft IR: check: %w", err)
	}
	return checked, nil
}

// Outside checks accompany the held roster check without changing its domain rules.
type rosterDraftCheck struct {
	check  Check
	blocks []string
}

func (c rosterDraftCheck) Evaluate(s Spec, snap snapshot.Snapshot) CheckResult {
	result := c.check.Evaluate(s, snap)
	if len(c.blocks) > 0 {
		result.Blocked = true
		result.Note = strings.TrimPrefix(result.Note+"; "+strings.Join(c.blocks, "; "), "; ")
	}
	return result
}
