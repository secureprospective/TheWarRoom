package rookiedraft

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// ActKind is one thing that can happen to a draft once it has started.
type ActKind string

const (
	// Select is a franchise choosing a rookie for the pick on its clock.
	Select ActKind = "select"
	// Skip is the commissioner skipping the pick on the clock.
	Skip ActKind = "skip"
	// Assign is the commissioner choosing a rookie for a pick: the one on the clock, or one the clock skipped.
	Assign ActKind = "assign"
	// Insert is the commissioner adding an extra pick, with its franchise and rookie, after another pick.
	Insert ActKind = "insert"
	// Change is the commissioner swapping the rookie a made pick holds.
	Change ActKind = "change"
	// Pause stops the clock where it stands; Resume lets it run on; SetClock sets the time it has left.
	Pause    ActKind = "pause"
	Resume   ActKind = "resume"
	SetClock ActKind = "set_clock"
	// Restart puts the draft back to how it stood at the checkpoint (Since): the pick then on the clock gets a
	// fresh clock, and the time that went by since is not counted.
	Restart ActKind = "restart"
	// Close seals the draft: every pick still unused is skipped for good.
	Close ActKind = "close"
)

// How a pick came to be used, in the words a screen can show.
const (
	HowOwner        = "owner"
	HowCommissioner = "commissioner"
	HowInserted     = "inserted"
	HowExpired      = "expired"
	HowSkipped      = "skipped"
	HowOverrode     = "overrode"
)

// PickRef names one pick of a draft: its round, the franchise it began with and, for an extra pick a person
// inserted, which insertion after that pick it is.
type PickRef struct {
	Round    int    `json:"round"`
	Original string `json:"original"`
	Extra    int    `json:"extra,omitempty"`
}

// Act is one recorded happening, supplied in ledger order. Terms is the contract fixed when it happened.
// Owner is the franchise making a selection or receiving an inserted pick. Replaces is the rookie the
// commissioner saw on the pick they changed: a change aimed at anything else is refused.
type Act struct {
	Kind     ActKind
	At       time.Time
	Pick     PickRef
	Owner    string
	Player   string
	Replaces string
	Terms    *Terms
	Duration time.Duration
	// Since is the instant of the checkpoint a Restart puts the draft back to.
	Since time.Time
}

// timeline is the clock of a draft being replayed. remaining is only read while the clock is paused.
type timeline struct {
	picks     []Result
	cursor    int
	deadline  time.Time
	opened    time.Time
	paused    bool
	remaining time.Duration
	clock     time.Duration
	last      time.Time
}

func newTimeline(picks []Result, start time.Time, clock time.Duration) *timeline {
	t := &timeline{picks: picks, clock: clock, opened: start, deadline: start.Add(clock), last: start}
	t.settle()
	return t
}

// settle moves the cursor past picks that never take a turn on the clock.
func (t *timeline) settle() {
	for t.cursor < len(t.picks) && (t.picks[t.cursor].Forfeited || t.picks[t.cursor].Extra > 0) {
		t.cursor++
	}
}

// open starts the clock for the next pick that has a turn.
func (t *timeline) open(at time.Time) {
	t.cursor++
	t.settle()
	t.opened, t.deadline, t.remaining = at, at.Add(t.clock), t.clock
}

// advance skips every pick whose clock ran out by the instant, each at its own deadline. A paused clock
// runs out of nothing.
func (t *timeline) advance(until time.Time) {
	t.settle()
	for t.cursor < len(t.picks) && !t.paused && !until.Before(t.deadline) {
		deadline := t.deadline
		pick := &t.picks[t.cursor]
		pick.Skipped, pick.MadeAt, pick.How = true, &deadline, HowExpired
		t.open(deadline)
	}
}

func (t *timeline) onClock() bool { return t.cursor < len(t.picks) }

func (t *timeline) find(ref PickRef) (int, error) {
	i := slices.IndexFunc(t.picks, func(p Result) bool {
		return p.Round == ref.Round && p.Original == ref.Original && p.Extra == ref.Extra
	})
	if i < 0 {
		return 0, fmt.Errorf("choose a pick in this draft")
	}
	return i, nil
}

// apply replays one act against the draft. Every act first lets the clock run out up to its instant, except
// a Restart, which lets it run out only up to its checkpoint and then starts the clock again.
func (t *timeline) apply(r leaguerules.DraftRules, a Act) error {
	if a.At.Before(t.last) {
		return fmt.Errorf("draft selections must follow their clock in time order")
	}
	t.last = a.At
	if a.Kind == Restart {
		t.advance(a.Since)
	} else {
		t.advance(a.At)
	}
	switch a.Kind {
	case Select, Assign:
		return t.choose(r, a)
	case Skip:
		return t.skip(a)
	case Insert:
		return t.insert(r, a)
	case Change:
		return t.change(a)
	case Pause, Resume, SetClock:
		return t.timer(a)
	case Close:
		t.seal(a.At)
		return nil
	case Restart:
		t.settle()
		t.opened, t.deadline, t.paused, t.remaining = a.At, a.At.Add(t.clock), false, t.clock
		return nil
	}
	return fmt.Errorf("this draft action is not available")
}

func (t *timeline) choose(r leaguerules.DraftRules, a Act) error {
	if !t.onClock() {
		return fmt.Errorf("there is no pick on the clock for this selection")
	}
	i, err := t.find(a.Pick)
	if err != nil {
		return err
	}
	pick := &t.picks[i]
	if i == t.cursor {
		if a.Kind == Select && (t.paused || pick.Owner != a.Owner) {
			return t.refuseSelect(*pick, a)
		}
		how := HowOwner
		if a.Kind == Assign {
			how = HowCommissioner
		}
		if err := t.fill(r, pick, a, how); err != nil {
			return err
		}
		t.open(a.At)
		return nil
	}
	if a.Kind == Assign && pick.Skipped && pick.Player == "" {
		return t.fill(r, pick, a, HowOverrode)
	}
	return fmt.Errorf("only the pick on the clock, or a pick that was skipped, can be given a rookie")
}

func (t *timeline) refuseSelect(pick Result, a Act) error {
	if t.paused {
		return fmt.Errorf("the draft clock is paused")
	}
	return fmt.Errorf("only the current owner may select for the pick on the clock (%s, not %s)", pick.Owner, a.Owner)
}

// fill gives a pick its rookie and contract. A Select must also name the pick on the clock exactly.
func (t *timeline) fill(r leaguerules.DraftRules, pick *Result, a Act, how string) error {
	if err := t.rookie(a.Player); err != nil {
		return err
	}
	terms := a.Terms
	if terms == nil {
		slot, err := SlotTerms(r, pick.Round, pick.Slot)
		if err != nil {
			return err
		}
		terms = &slot
	}
	at := a.At
	pick.Player, pick.Skipped, pick.MadeAt, pick.Contract, pick.How = a.Player, false, &at, terms, how
	return nil
}

// rookie holds a chosen player to being a valid, undrafted player.
func (t *timeline) rookie(player string) error {
	id, err := playerid.New(player)
	if err != nil || id.String() != player {
		return fmt.Errorf("choose a valid rookie player")
	}
	if slices.ContainsFunc(t.picks, func(p Result) bool { return p.Player == player }) {
		return fmt.Errorf("this player has already been drafted")
	}
	return nil
}

// seal skips what is left of the draft at once. Picks already used stay as they are.
func (t *timeline) seal(at time.Time) {
	for ; t.cursor < len(t.picks); t.cursor++ {
		if pick := &t.picks[t.cursor]; !pick.Forfeited && pick.Extra == 0 {
			closedAt := at
			pick.Skipped, pick.MadeAt, pick.How = true, &closedAt, HowSkipped
		}
	}
}

func (t *timeline) skip(a Act) error {
	i, err := t.find(a.Pick)
	if err != nil {
		return err
	}
	if !t.onClock() || i != t.cursor {
		return fmt.Errorf("only the pick on the clock can be skipped")
	}
	at := a.At
	t.picks[i].Skipped, t.picks[i].MadeAt, t.picks[i].How = true, &at, HowSkipped
	t.open(a.At)
	return nil
}

// insert adds an extra pick right after the pick it follows (and after any extras already there). It takes
// no turn on the clock and costs the franchise's next pick nothing.
func (t *timeline) insert(r leaguerules.DraftRules, a Act) error {
	after, err := t.find(a.Pick)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(t.picks, func(p Result) bool { return p.Owner == a.Owner }) {
		return fmt.Errorf("choose a franchise in this draft")
	}
	base := t.picks[after]
	extra := 1
	at := after + 1
	for ; at < len(t.picks) && t.picks[at].Extra > 0 && t.picks[at].Round == base.Round &&
		t.picks[at].Original == base.Original; at++ {
		extra++
	}
	added := Result{Pick: Pick{OwnedPick: OwnedPick{Round: base.Round, Original: base.Original, Owner: a.Owner},
		Slot: base.Slot}, Extra: extra}
	if err := t.fill(r, &added, a, HowInserted); err != nil {
		return err
	}
	t.picks = slices.Insert(t.picks, at, added)
	if at <= t.cursor {
		t.cursor++
	}
	return nil
}

func (t *timeline) change(a Act) error {
	i, err := t.find(a.Pick)
	if err != nil {
		return err
	}
	pick := &t.picks[i]
	if pick.Player == "" {
		return fmt.Errorf("only a pick that has been made can be changed")
	}
	if pick.Player != a.Replaces {
		return fmt.Errorf("this pick now holds a different player than the one you were shown; read the draft again")
	}
	if err := t.rookie(a.Player); err != nil {
		return err
	}
	pick.Player, pick.Changed = a.Player, true
	return nil
}

func (t *timeline) timer(a Act) error {
	if !t.onClock() {
		return fmt.Errorf("the draft has no pick on the clock")
	}
	switch {
	case a.Kind == Pause && t.paused:
		return fmt.Errorf("the draft clock is already paused")
	case a.Kind == Pause:
		t.paused, t.remaining = true, max(0, t.deadline.Sub(a.At))
	case a.Kind == Resume && !t.paused:
		return fmt.Errorf("the draft clock is not paused")
	case a.Kind == Resume:
		t.paused, t.deadline = false, a.At.Add(t.remaining)
	case a.Duration <= 0:
		return fmt.Errorf("set a positive time for the draft clock")
	case t.paused:
		t.remaining = a.Duration
	default:
		t.deadline = a.At.Add(a.Duration)
	}
	return nil
}

// clockAt is the pick on the clock at an instant, or nil when every pick is used.
func (t *timeline) clockAt(r leaguerules.DraftRules, at time.Time) (*Clock, error) {
	if !t.onClock() {
		return nil, nil
	}
	terms, err := SlotTerms(r, t.picks[t.cursor].Round, t.picks[t.cursor].Slot)
	if err != nil {
		return nil, err
	}
	remaining := t.remaining
	if !t.paused {
		remaining = max(0, t.deadline.Sub(at))
	}
	return &Clock{Pick: t.picks[t.cursor].Pick, OpenedAt: t.opened, Deadline: t.deadline, Contract: terms,
		Paused: t.paused, Remaining: remaining}, nil
}
