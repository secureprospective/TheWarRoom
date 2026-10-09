package envelope

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type TaxiRequest IRRequest

type TaxiCheck struct{}

func (TaxiCheck) Evaluate(s Spec, snap snapshot.Snapshot) CheckResult {
	if s.Intent != "roster.taxi" {
		return CheckResult{Blocked: true, Note: "taxi check does not cover this intent"}
	}
	if snap.Rosters.Provenance.Freshness.State == domain.FreshFail {
		return CheckResult{Blocked: true, Note: "roster unavailable"}
	}
	status, found := rosterStatus(s, snap.Rosters.Value)
	if !found {
		return CheckResult{Blocked: true, Note: "player is not on this franchise's roster"}
	}
	if status == domain.RosterIR {
		return CheckResult{Blocked: true, Note: "player on IR cannot move to or from taxi"}
	}
	if taxiDestination(status) != s.Expected.RosterStatus {
		return CheckResult{Blocked: true, Note: "active or taxi roster status required"}
	}
	return CheckResult{Note: "taxi capacity, eligibility and lock rules not held; MFL checks them"}
}

func taxiDestination(status domain.RosterStatus) domain.RosterStatus {
	switch status {
	case domain.RosterActive:
		return domain.RosterTaxi
	case domain.RosterTaxi:
		return domain.RosterActive
	case domain.RosterIR:
		return ""
	default:
		return ""
	}
}

func DraftTaxi(
	at time.Time, req TaxiRequest, snap snapshot.Snapshot, generate func() string,
) (Envelope, error) {
	spec := Spec{
		Intent: "roster.taxi", LeagueID: req.LeagueID, FranchiseID: req.FranchiseID,
		Subject:  Subject{Players: []playerid.PlayerID{req.Player}},
		Expected: ExpectedChange{Player: req.Player, RosterStatus: domain.RosterTaxi},
		Gravity:  G2, Undo: Reversible, Target: req.Target,
	}
	if spec.Target.Kind == "" {
		spec.Target.Kind = Unmapped
	}
	status, found := rosterStatus(spec, snap.Rosters.Value)
	if found && status == domain.RosterTaxi {
		spec.Expected.RosterStatus = domain.RosterActive
	}
	e, err := New(at, spec, generate)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft taxi: %w", err)
	}
	checked, err := e.Check(at, snap, rosterDraftCheck{check: TaxiCheck{}, blocks: req.Blocks})
	if err != nil {
		return Envelope{}, fmt.Errorf("draft taxi: check: %w", err)
	}
	return checked, nil
}

type TaxiPredicate struct{}

func (TaxiPredicate) Sources() []Source { return []Source{Rosters} }

func (TaxiPredicate) Evaluate(s Spec, obs Observation) Verdict {
	if s.Intent != "roster.taxi" || obs.LeagueID != s.LeagueID ||
		obs.Rosters.Provenance.Freshness.State == domain.FreshFail {
		return Verdict{Event: Partial, Note: "Not verified: authoritative scoped roster unavailable"}
	}
	if !slices.ContainsFunc(obs.Rosters.Value, func(roster snapshot.Roster) bool {
		return roster.FranchiseID == s.FranchiseID
	}) {
		return Verdict{Event: Partial, Note: "Not verified: franchise roster missing"}
	}
	status, found := rosterStatus(s, obs.Rosters.Value)
	if !found {
		return Verdict{Event: Contradiction, Note: "player gone from franchise"}
	}
	if status == s.Expected.RosterStatus {
		return Verdict{Event: Match, Note: "observed requested taxi roster status on the same franchise"}
	}
	if taxiDestination(status) == s.Expected.RosterStatus {
		return Verdict{Event: NoChange, Note: "No matching MFL change observed"}
	}
	if status == domain.RosterIR {
		return Verdict{Event: Contradiction, Note: "player moved to IR instead of requested taxi status"}
	}
	return Verdict{Event: Partial, Note: "Not verified: roster status missing or unknown"}
}
