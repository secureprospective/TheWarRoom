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
	return checkRosterMove(s, snap, rosterMoveRule{
		intent: "roster.taxi", label: "taxi", squad: domain.RosterTaxi,
		exclusionNote: "player on IR cannot move to or from taxi",
	})
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
	if rosterDestination(status, domain.RosterTaxi) == s.Expected.RosterStatus {
		return Verdict{Event: NoChange, Note: "No matching MFL change observed"}
	}
	if status == domain.RosterIR {
		return Verdict{Event: Contradiction, Note: "player moved to IR instead of requested taxi status"}
	}
	return Verdict{Event: Partial, Note: "Not verified: roster status missing or unknown"}
}
