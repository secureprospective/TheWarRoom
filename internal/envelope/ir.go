package envelope

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
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
	status, found := rosterStatus(s, snap.Rosters.Value)
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

// Source names an MFL feed a predicate reads; Observe checks only the declared ones' freshness.
type Source string

const (
	Rosters       Source = "rosters"
	Transactions  Source = "transactions"
	Lineups       Source = "lineups"
	PendingTrades Source = "pendingTrades"
)

// Observation is the league as MFL reported it after a hand-off, one feed per field, each with
// its own fetch time.
type Observation struct {
	LeagueID      string
	Rosters       snapshot.Sourced[[]snapshot.Roster]
	Transactions  snapshot.Sourced[[]leaguefeed.Transaction]
	Lineups       snapshot.Sourced[leaguefeed.Lineups]
	PendingTrades snapshot.Sourced[[]leaguefeed.PendingTrade]
}

func (o Observation) freshness(source Source) (domain.Freshness, bool) {
	switch source {
	case Rosters:
		return o.Rosters.Provenance.Freshness, true
	case Transactions:
		return o.Transactions.Provenance.Freshness, true
	case Lineups:
		return o.Lineups.Provenance.Freshness, true
	case PendingTrades:
		return o.PendingTrades.Provenance.Freshness, true
	default:
		return domain.Freshness{}, false
	}
}

type Verdict struct {
	Event Event
	Note  string
}
type Predicate interface {
	Evaluate(Spec, Observation) Verdict
	Sources() []Source
}
type IRPredicate struct{}

func (IRPredicate) Sources() []Source { return []Source{Rosters} }

func (IRPredicate) Evaluate(s Spec, obs Observation) Verdict {
	if s.Intent != "roster.ir" || obs.LeagueID != s.LeagueID ||
		obs.Rosters.Provenance.Freshness.State == domain.FreshFail {
		return Verdict{Event: Partial, Note: "Not verified: authoritative scoped roster unavailable"}
	}
	present := false
	for _, r := range obs.Rosters.Value {
		if r.FranchiseID == s.FranchiseID {
			present = true
		}
	}
	if !present {
		return Verdict{Event: Partial, Note: "Not verified: franchise roster missing"}
	}
	status, found := rosterStatus(s, obs.Rosters.Value)
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
func rosterStatus(s Spec, rosters []snapshot.Roster) (domain.RosterStatus, bool) {
	for _, r := range rosters {
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
	observed, err := e.observedAt(at, obs, predicate.Sources())
	if err != nil {
		return Envelope{}, err
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

func (e Envelope) observedAt(at time.Time, obs Observation, sources []Source) (time.Time, error) {
	if len(sources) == 0 {
		return time.Time{}, ErrStaleObservation
	}
	var oldest time.Time
	for _, source := range sources {
		fresh, known := obs.freshness(source)
		fetched, err := time.Parse(time.RFC3339, fresh.FetchedAt)
		if !known || fresh.State == domain.FreshFail || err != nil ||
			fetched.After(at) || !fetched.After(e.handedAt()) {
			return time.Time{}, ErrStaleObservation
		}
		if oldest.IsZero() || fetched.Before(oldest) {
			oldest = fetched
		}
	}
	return oldest, nil
}
