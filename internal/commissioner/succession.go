package commissioner

import (
	"fmt"
	"math"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// CM-022 is a composite census row with no single minimum role, so each succession command names
// the role it needs here. The governance fold stays the final authority: every proposed change is
// trial-folded (CheckGovernanceAppend), which enforces who may step down, accept, take over, open
// a vote, cast a ballot or break a runoff tie.
type successionGate uint8

const (
	gateCommissioner successionGate = iota // a current commissioner
	gateOpener                             // a co-commissioner or DOT member (who may open votes)
	gateCloser                             // an opener who is not the vote's target
	gateFold                               // any principal; the governance fold decides
)

type successionDefinition struct {
	gate           successionGate
	gravity, label string
	private, self  bool
}

func successionDefinitions() map[string]successionDefinition {
	return map[string]successionDefinition{
		"commish.succession.stepdown":       {gateCommissioner, "G3", "Stepped down as commissioner", false, true},
		"commish.succession.accept":         {gateFold, "G3", "Accepted the commissioner's seat", false, true},
		"commish.succession.decline":        {gateFold, "G3", "Declined the commissioner's seat", false, true},
		"commish.succession.inactivity.set": {gateCommissioner, "G3", "Changed the commissioner inactivity periods", false, false},
		"commish.succession.activity":       {gateCommissioner, "G1", "Recorded that the commissioner is active", false, true},
		"commish.succession.takeover":       {gateFold, "G3", "Took over after commissioner inactivity", false, true},
		"commish.succession.voteout.open":   {gateOpener, "G3", "Opened a vote to remove the commissioner", false, true},
		"commish.succession.election.open":  {gateOpener, "G3", "Opened a commissioner election", false, true},
		"commish.succession.periods.set":    {gateCommissioner, "G3", "Changed the election voting periods", false, false},
		"commish.vote.cast":                 {gateFold, "G2", "Cast a franchise ballot", true, true},
		"commish.vote.close":                {gateCloser, "G3", "Closed a commissioner vote", false, false},
		"commish.vote.tiebreak":             {gateFold, "G3", "Ruled on a tied runoff", false, true},
	}
}

func successionDefinitionFor(command string) (successionDefinition, bool) {
	d, ok := successionDefinitions()[command]
	return d, ok
}

func (r RawPeople) selectedSuccession(command string) (RawPeople, error) {
	p := RawPeople{Principal: r.Principal}
	switch command {
	case "commish.succession.stepdown":
		p.Successor = r.Successor
	case "commish.succession.accept", "commish.succession.decline", "commish.succession.activity":
	case "commish.succession.takeover":
		p.Commissioner = r.Commissioner
	case "commish.succession.voteout.open":
		p.ID, p.Commissioner = r.ID, r.Commissioner
	case "commish.succession.election.open":
		p.ID = r.ID
	case "commish.vote.cast", "commish.vote.tiebreak":
		p.ID, p.Choice = r.ID, r.Choice
	case "commish.vote.close":
		p = RawPeople{ID: r.ID}
	case "commish.succession.inactivity.set":
		p = RawPeople{InSeasonDays: r.InSeasonDays, OffseasonDays: r.OffseasonDays}
	case "commish.succession.periods.set":
		p = RawPeople{ElectionDays: r.ElectionDays, RunoffDays: r.RunoffDays}
	default:
		return RawPeople{}, fmt.Errorf("choose an available people change")
	}
	return p, nil
}

func (r RawPeople) validateSuccession(command string) error {
	d, ok := successionDefinitionFor(command)
	if !ok {
		return nil
	}
	if d.self && r.Principal == "" {
		return fmt.Errorf("identify who is acting")
	}
	switch command {
	case "commish.succession.inactivity.set":
		return validateDays(r.InSeasonDays, r.OffseasonDays)
	case "commish.succession.periods.set":
		return validateDays(r.ElectionDays, r.RunoffDays)
	}
	return nil
}

func validateDays(first, second int64) error {
	maximum := int64(math.MaxInt64 / int64(24*time.Hour))
	if first <= 0 || second <= 0 || first > maximum || second > maximum {
		return fmt.Errorf("choose positive voting or inactivity periods in whole days")
	}
	return nil
}

func days(n int64) time.Duration { return time.Duration(n) * 24 * time.Hour }

func (r RawPeople) successionEvent(command string, at time.Time) (governance.Event, error) {
	if err := r.Validate(command); err != nil {
		return nil, err
	}
	p := governance.Principal(r.Principal)
	switch command {
	case "commish.succession.stepdown":
		return governance.StepDown{Principal: p, Successor: governance.Principal(r.Successor)}, nil
	case "commish.succession.accept":
		return governance.SuccessorAccepted{Principal: p}, nil
	case "commish.succession.decline":
		return governance.SuccessorDeclined{Principal: p}, nil
	case "commish.succession.inactivity.set":
		return governance.InactivitySet{InSeason: days(r.InSeasonDays), Offseason: days(r.OffseasonDays)}, nil
	case "commish.succession.activity":
		return governance.ActivityObserved{Principal: p, At: at}, nil
	case "commish.succession.takeover":
		// The fold checks the offseason period; takeoverWaitedLongest also requires the in-season one,
		// because the league's season dates are not ruled yet and the caller must not pick the shorter.
		return governance.InactivityTakeover{Principal: p, Commissioner: governance.Principal(r.Commissioner),
			Now: at}, nil
	case "commish.succession.voteout.open":
		return governance.VoteOutOpened{ID: r.ID, Principal: p, Commissioner: governance.Principal(r.Commissioner)}, nil
	case "commish.succession.election.open":
		return governance.ElectionOpened{ID: r.ID, Principal: p, Now: at}, nil
	case "commish.succession.periods.set":
		return governance.ElectionPeriodsSet{Election: days(r.ElectionDays), Runoff: days(r.RunoffDays)}, nil
	case "commish.vote.cast":
		return governance.BallotCast{ID: r.ID, Principal: p, Choice: governance.Principal(r.Choice), Now: at}, nil
	case "commish.vote.close":
		return governance.VoteClosed{ID: r.ID, Now: at}, nil
	case "commish.vote.tiebreak":
		return governance.ElectionTieBreakCast{ID: r.ID, Principal: p,
			Choice: governance.Principal(r.Choice), Now: at}, nil
	default:
		return nil, fmt.Errorf("choose an available succession change")
	}
}

func authorizeSuccession(state governance.State, raw RawCommand, principal string) (string, error) {
	d, _ := successionDefinitionFor(raw.Command)
	if d.self && raw.Args.People.Principal != principal {
		return "", fmt.Errorf("%s", "Other owners act from their own app; this desktop can only act as you")
	}
	if d.gate == gateCloser {
		if v, ok := state.Vote(raw.Args.People.ID); ok && v.Target == governance.Principal(principal) {
			return "", fmt.Errorf("the person a vote is about cannot close it")
		}
	}
	return successionPermission(state, principal, d)
}

func successionPermission(state governance.State, principal string, d successionDefinition) (string, error) {
	p := governance.Principal(principal)
	held := highestRole(state, p)
	switch d.gate {
	case gateCommissioner:
		if !state.HasRole(p, governance.Commissioner) {
			return "", fmt.Errorf("succession permission: only the commissioner may do this")
		}
	case gateOpener, gateCloser:
		if !state.HasRole(p, governance.CoCommissioner) && !state.HasRole(p, governance.DOT) {
			return "", fmt.Errorf("succession permission: only a co-commissioner or DOT member may do this")
		}
	case gateFold:
	}
	return held, nil
}

func highestRole(state governance.State, p governance.Principal) string {
	switch {
	case state.HasRole(p, governance.Commissioner):
		return "commissioner"
	case state.HasRole(p, governance.CoCommissioner):
		return "cocommissioner"
	case state.HasRole(p, governance.DOT):
		return string(governance.DOT)
	default:
		return string(governance.Owner)
	}
}

// takeoverWaitedLongest refuses an inactivity takeover until the commissioner has been inactive for
// both periods, so the longer one always applies while the league's season dates need a ruling.
func takeoverWaitedLongest(state governance.State, raw RawCommand, at time.Time) error {
	if raw.Command != "commish.succession.takeover" {
		return nil
	}
	commissioner := governance.Principal(raw.Args.People.Commissioner)
	for _, inSeason := range []bool{true, false} {
		ok, err := governance.InactivityEligible(state, commissioner, at, inSeason)
		if err != nil {
			return fmt.Errorf("check commissioner inactivity: %w", err)
		}
		if !ok {
			return fmt.Errorf("the commissioner has not reached the longer inactivity period " +
				"(the league's season dates need a ruling, so the longer period applies)")
		}
	}
	return nil
}

// withActivity follows each recorded change with the acting person's activity at its recorded time,
// so a commissioner's own league changes count as activity for the inactivity rule.
func withActivity(events []governance.Event, r Record) []governance.Event {
	if r.Principal == "" || r.Receipt.RecordedAt.IsZero() {
		return events
	}
	return append(events, governance.ActivityObserved{Principal: governance.Principal(r.Principal),
		At: r.Receipt.RecordedAt})
}
