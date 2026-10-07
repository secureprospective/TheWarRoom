// Package envelope keeps plans separate from MFL truth: an envelope is a planned move whose state
// changes only through the transition table, and "Ready" means the app checked it, never that MFL
// accepted it. The core performs no I/O. internal/store/moves owns the append-only SQLite
// envelope and audit tables.
package envelope

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

type State string

const (
	Draft         State = "draft"
	Blocked       State = "blocked"
	Ready         State = "ready"
	HandedOff     State = "handed_off"
	NotYetDone    State = "not_yet_done"
	Landed        State = "landed"
	NotVerified   State = "not_verified"
	Failed        State = "failed"
	Stale         State = "stale"
	DOTReview     State = "dot_review"
	BidPending    State = "bid_pending"
	WaiverPending State = "waiver_pending"
)

type Event string

const (
	ChecksPass    Event = "checks_pass"
	ChecksBlock   Event = "checks_block"
	HandOff       Event = "hand_off"
	NoChange      Event = "no_change"
	Match         Event = "match"
	Partial       Event = "partial"
	Contradiction Event = "contradiction"
	Invalidate    Event = "invalidate"
	Rebase        Event = "rebase"
	AwaitDOT      Event = "await_dot"
	AwaitBid      Event = "await_bid"
	AwaitWaiver   Event = "await_waiver"
)

type Gravity string

const (
	G0 Gravity = "G0"
	G1 Gravity = "G1"
	G2 Gravity = "G2"
	G3 Gravity = "G3"
)

type UndoClass string

const (
	Instant      UndoClass = "instant"
	Reversible   UndoClass = "reversible"
	Irreversible UndoClass = "irreversible"
)

// TargetKind says whether the MFL page for an intent is known. Unmapped targets never leave
// draft or blocked (spec §6: "MFL target not verified").
type TargetKind string

const (
	Mapped   TargetKind = "mapped"
	Unmapped TargetKind = "unmapped"
)

type Target struct {
	Kind TargetKind `json:"kind"`
	URL  string     `json:"url,omitempty"`
}

// ts_type: PlayerID marshals as its string form; without it Wails emits an undefined TS type.
type Subject struct {
	Players []playerid.PlayerID `json:"players" ts_type:"string[]"`
	Picks   []string            `json:"picks"`
}

// ExpectedChange is what the landing predicate looks for. Ring 0 knows one shape, a roster
// status change; trade and bid shapes arrive with their intents.
type ExpectedChange struct {
	Player       playerid.PlayerID   `json:"player" ts_type:"string"`
	RosterStatus domain.RosterStatus `json:"rosterStatus"`
}

type Spec struct {
	Intent      string         `json:"intent"`
	LeagueID    string         `json:"leagueId"`
	FranchiseID string         `json:"franchiseId"`
	Subject     Subject        `json:"subject"`
	Expected    ExpectedChange `json:"expected"`
	Gravity     Gravity        `json:"gravity"`
	Undo        UndoClass      `json:"undo"`
	Target      Target         `json:"target"`
	Deadline    *time.Time     `json:"deadline,omitempty"`
}

type AuditEntry struct {
	At    time.Time `json:"at"`
	From  State     `json:"from"`
	Event Event     `json:"event"`
	To    State     `json:"to"`
	Note  string    `json:"note"`
}

type Receipt struct {
	CorrelationID string       `json:"correlationId"`
	Spec          Spec         `json:"spec"`
	State         State        `json:"state"`
	Audit         []AuditEntry `json:"audit"`
}

// Envelope has no exported fields; construction and restoration validate every transition.
type Envelope struct {
	id      string
	spec    Spec
	state   State
	created time.Time
	audit   []AuditEntry
}

type ErrIllegalTransition struct {
	From  State
	Event Event
}

func (e ErrIllegalTransition) Error() string {
	return fmt.Sprintf("envelope: illegal %s from %s", e.Event, e.From)
}

// ErrStaleObservation rejects evidence that is not newer than the hand-off; it changes nothing.
var ErrStaleObservation = errors.New("envelope: observation is not newer than the hand-off")

func New(at time.Time, spec Spec, generate func() string) (Envelope, error) {
	if err := validateSpec(spec); err != nil {
		return Envelope{}, err
	}
	if at.IsZero() || generate == nil {
		return Envelope{}, fmt.Errorf("envelope: timestamp and ID generator required")
	}
	id := generate()
	if strings.TrimSpace(id) == "" {
		return Envelope{}, fmt.Errorf("envelope: empty correlation ID")
	}
	return Envelope{id: id, spec: cloneSpec(spec), state: Draft, created: at.UTC(), audit: []AuditEntry{}}, nil
}

func validateSpec(s Spec) error {
	if s.Intent == "" || s.LeagueID == "" || s.FranchiseID == "" {
		return fmt.Errorf("envelope: intent and scope required")
	}
	if s.Expected.Player.String() == "" || s.Expected.RosterStatus == "" ||
		!slices.Contains(s.Subject.Players, s.Expected.Player) {
		return fmt.Errorf("envelope: expected change must name a subject player and status")
	}
	if !slices.Contains([]Gravity{G0, G1, G2, G3}, s.Gravity) ||
		!slices.Contains([]UndoClass{Instant, Reversible, Irreversible}, s.Undo) {
		return fmt.Errorf("envelope: invalid gravity or undo class")
	}
	for _, p := range s.Subject.Players {
		if p.String() == "" {
			return fmt.Errorf("envelope: empty subject player")
		}
	}
	return validateTarget(s.Target)
}

func validateTarget(target Target) error {
	if target.Kind == Unmapped && target.URL == "" {
		return nil
	}
	u, err := url.Parse(target.URL)
	if err != nil {
		return fmt.Errorf("envelope: target URL: %w", err)
	}
	if target.Kind != Mapped || u.Scheme != "https" || u.Host == "" || u.User != nil {
		return fmt.Errorf("envelope: mapped HTTPS target or unmapped target required")
	}
	return nil
}

func cloneSpec(s Spec) Spec {
	s.Subject.Players = append([]playerid.PlayerID{}, s.Subject.Players...)
	s.Subject.Picks = append([]string{}, s.Subject.Picks...)
	if s.Deadline != nil {
		d := s.Deadline.UTC()
		s.Deadline = &d
	}
	return s
}

func (e Envelope) Receipt() Receipt {
	return Receipt{CorrelationID: e.id, Spec: cloneSpec(e.spec), State: e.state,
		Audit: append([]AuditEntry{}, e.audit...)}
}

func (e Envelope) State() State { return e.state }

type step struct {
	From  State
	Event Event
}

// transitions returns the whole state machine. After hand-off, observation drives the envelope: a
// matching change lands it, a partial one is Not verified, a contradiction fails it. A trade,
// bid or waiver claim that MFL has accepted but not yet resolved waits in its named stage.
func transitions() map[step]State {
	t := map[step]State{
		{Draft, ChecksPass}: Ready, {Draft, ChecksBlock}: Blocked,
		{Blocked, ChecksPass}: Ready, {Blocked, ChecksBlock}: Blocked,
		{Ready, HandOff}: HandedOff, {Ready, Invalidate}: Stale,
		{Stale, Rebase}: Draft, {Failed, Rebase}: Draft,
	}
	pending := map[Event]State{AwaitDOT: DOTReview, AwaitBid: BidPending, AwaitWaiver: WaiverPending}
	for _, s := range []State{HandedOff, NotYetDone, NotVerified} {
		t[step{s, NoChange}] = NotYetDone
		for event, stage := range pending {
			t[step{s, event}] = stage
		}
	}
	for _, s := range []State{HandedOff, NotYetDone, NotVerified, DOTReview, BidPending, WaiverPending} {
		t[step{s, Match}], t[step{s, Partial}] = Landed, NotVerified
		t[step{s, Contradiction}], t[step{s, Invalidate}] = Failed, Stale
	}
	for _, stage := range pending {
		t[step{stage, NoChange}] = stage
	}
	return t
}

func (e Envelope) move(at time.Time, event Event, note string) (Envelope, error) {
	illegal := ErrIllegalTransition{From: e.state, Event: event}
	if e.id == "" || at.IsZero() || at.Before(e.created) ||
		(len(e.audit) > 0 && at.Before(e.audit[len(e.audit)-1].At)) {
		return Envelope{}, illegal
	}
	to, ok := transitions()[step{e.state, event}]
	if !ok || (e.spec.Target.Kind != Mapped && to != Draft && to != Blocked) {
		return Envelope{}, illegal
	}
	next := e
	next.audit = append(append([]AuditEntry{}, e.audit...),
		AuditEntry{At: at.UTC(), From: e.state, Event: event, To: to, Note: note})
	next.state = to
	return next, nil
}

func (e Envelope) HandOff(at time.Time) (Envelope, error) {
	if e.spec.Deadline != nil && at.After(*e.spec.Deadline) {
		return Envelope{}, ErrIllegalTransition{From: e.state, Event: HandOff}
	}
	return e.move(at, HandOff, "MFL opened; acceptance not verified")
}

func (e Envelope) Invalidate(at time.Time, note string) (Envelope, error) {
	return e.move(at, Invalidate, note)
}

func (e Envelope) Rebase(at time.Time) (Envelope, error) {
	return e.move(at, Rebase, "explicit revalidation required")
}

// Await records that MFL accepted the submission and it now waits on a resolution only the
// intent's own process can give (DOT review, bid award, waiver run).
func (e Envelope) Await(at time.Time, event Event) (Envelope, error) {
	intents := map[Event]string{AwaitDOT: "trade.propose", AwaitBid: "bid.submit", AwaitWaiver: "waiver.claim"}
	if intent, ok := intents[event]; !ok || intent != e.spec.Intent {
		return Envelope{}, ErrIllegalTransition{From: e.state, Event: event}
	}
	return e.move(at, event, "submission observed; awaiting resolution")
}

func (e Envelope) ID() string { return e.id }

func (e Envelope) Created() time.Time { return e.created }

// Restore rebuilds a stored envelope by replaying its audit through the transition table; a
// history the table cannot produce is an error naming the entry, never repaired.
func Restore(id string, created time.Time, spec Spec, audit []AuditEntry) (Envelope, error) {
	e, err := New(created, spec, func() string { return id })
	if err != nil {
		return Envelope{}, fmt.Errorf("envelope: restore construction: %w", err)
	}
	for i, entry := range audit {
		if entry.From != e.state {
			return Envelope{}, fmt.Errorf("envelope: restore entry %d: from %s, want %s", i, entry.From, e.state)
		}
		next, err := e.move(entry.At, entry.Event, entry.Note)
		if err != nil {
			return Envelope{}, fmt.Errorf("envelope: restore entry %d: %w", i, err)
		}
		if next.state != entry.To {
			return Envelope{}, fmt.Errorf("envelope: restore entry %d: to %s, want %s", i, entry.To, next.state)
		}
		e = next
	}
	return e, nil
}
