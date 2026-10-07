package envelope

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type CheckResult struct {
	Blocked bool
	Note    string
}
type Check interface {
	Evaluate(Spec, snapshot.Snapshot) CheckResult
}
type IRCheck struct{}

func (IRCheck) Evaluate(s Spec, snap snapshot.Snapshot) CheckResult {
	if s.Intent != "roster.ir" {
		return CheckResult{Blocked: true, Note: "IR check does not cover this intent"}
	}
	if snap.Rosters.Provenance.Freshness.State == domain.FreshFail {
		return CheckResult{Blocked: true, Note: "roster unavailable"}
	}
	status, found := rosterStatus(s, snap)
	if !found {
		return CheckResult{Blocked: true, Note: "player is not on this franchise's roster"}
	}
	if status == domain.RosterIR {
		return CheckResult{Blocked: true, Note: "player already on IR"}
	}
	return CheckResult{Note: "not verified (league rules not captured)"}
}
func (e Envelope) Check(at time.Time, snap snapshot.Snapshot, check Check) (Envelope, error) {
	if check == nil {
		return Envelope{}, fmt.Errorf("envelope: check required")
	}
	result := check.Evaluate(cloneSpec(e.spec), snap)
	if e.spec.Deadline != nil && at.After(*e.spec.Deadline) {
		result = CheckResult{Blocked: true, Note: "supplied deadline passed"}
	}
	if e.spec.Target.Kind != Mapped {
		result = CheckResult{Blocked: true, Note: strings.TrimPrefix(result.Note+"; MFL target not verified", "; ")}
	}
	event := ChecksPass
	if result.Blocked {
		event = ChecksBlock
	}
	return e.move(at, event, result.Note)
}

// Observation is a later snapshot of the league; its roster fetch time is when it was observed.
type Observation struct {
	LeagueID string
	Snapshot snapshot.Snapshot
}
type Verdict struct {
	Event Event
	Note  string
}
type Predicate interface {
	Evaluate(Spec, Observation) Verdict
}
type IRPredicate struct{}

func (IRPredicate) Evaluate(s Spec, obs Observation) Verdict {
	if s.Intent != "roster.ir" || obs.LeagueID != s.LeagueID ||
		obs.Snapshot.Rosters.Provenance.Freshness.State == domain.FreshFail {
		return Verdict{Event: Partial, Note: "Not verified: authoritative scoped roster unavailable"}
	}
	present := false
	for _, r := range obs.Snapshot.Rosters.Value {
		if r.FranchiseID == s.FranchiseID {
			present = true
		}
	}
	if !present {
		return Verdict{Event: Partial, Note: "Not verified: franchise roster missing"}
	}
	status, found := rosterStatus(s, obs.Snapshot)
	if !found {
		return Verdict{Event: Contradiction, Note: "player gone from franchise"}
	}
	if status == s.Expected.RosterStatus {
		return Verdict{Event: Match, Note: "observed player on IR on the same franchise"}
	}
	if status == "" {
		return Verdict{Event: Partial, Note: "Not verified: roster status missing"}
	}
	return Verdict{Event: NoChange, Note: "No matching MFL change observed"}
}
func rosterStatus(s Spec, snap snapshot.Snapshot) (domain.RosterStatus, bool) {
	for _, r := range snap.Rosters.Value {
		if r.FranchiseID != s.FranchiseID {
			continue
		}
		for _, p := range r.Players {
			if p.ID == s.Expected.Player {
				return p.RosterStatus, true
			}
		}
	}
	return "", false
}
func (e Envelope) Observe(at time.Time, obs Observation, predicate Predicate) (Envelope, error) {
	if predicate == nil {
		return Envelope{}, fmt.Errorf("envelope: predicate required")
	}
	observed, err := time.Parse(time.RFC3339, obs.Snapshot.Rosters.Provenance.Freshness.FetchedAt)
	if err != nil || observed.After(at) || !observed.After(e.handedAt()) {
		return Envelope{}, ErrStaleObservation
	}
	verdict := predicate.Evaluate(cloneSpec(e.spec), obs)
	if !slices.Contains([]Event{Match, Partial, NoChange, Contradiction}, verdict.Event) {
		return Envelope{}, fmt.Errorf("envelope: invalid predicate event %q", verdict.Event)
	}
	// Unchanged again after a supplied deadline invalidates the plan; it is not a rejection by MFL.
	if verdict.Event == NoChange && e.state == NotYetDone && e.spec.Deadline != nil &&
		observed.After(*e.spec.Deadline) {
		verdict = Verdict{Event: Invalidate, Note: "unchanged after deadline; revalidate plan"}
	}
	return e.move(at, verdict.Event, verdict.Note)
}
func (e Envelope) handedAt() time.Time {
	at := e.created
	for _, entry := range e.audit {
		if entry.Event == HandOff {
			at = entry.At
		}
	}
	return at
}
