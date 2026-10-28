package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// DraftTrayCommand says whether a command is previewed by DraftPreview.
func DraftTrayCommand(command string) bool {
	return trayCommand(command) || command == DraftRevert || command == DraftArchive || command == DraftStartupOpen
}

// DraftPreview describes a live draft change for a GM: what the pick, the clock or the draft is now, what it
// becomes, and what to know before confirming. The checks that may refuse the change run elsewhere.
func DraftPreview(raw RawCommand, records []Record, catalogue PreviewCatalogue, now time.Time,
	standings DraftStandings, facts nativecheck.Facts) (Preview, error) {
	basis, err := readDraft(records, raw.Args.Draft.key(), now, standings)
	if err != nil {
		return Preview{}, err
	}
	safeguard, err := safeguardOf(raw)
	if err != nil {
		return Preview{}, err
	}
	control := controlOfCommand(raw.Command)
	room := draftRoomView{basis: basis, facts: facts, records: records, now: now}
	field, warning := room.describe(raw)
	return Preview{Question: catalogue.questions[control], Safeguard: safeguard, Blockers: []string{},
		Target:  PreviewTarget{League: raw.League, Name: Replay(records).Name, Version: raw.Expected},
		Fields:  []PreviewField{field},
		Affects: warned(catalogue.affects(control), warning)}, nil
}

// warned puts what to know before confirming where the tray reads what a change affects. The tray shows
// those words only for a count it knows, so the count is the number of readings the change reaches.
func warned(affects PreviewAffects, warning string) PreviewAffects {
	count := len(affects.Readings)
	affects.Count, affects.Words = &count, warning
	return affects
}

func controlOfCommand(command string) string { return commandControls()[command] }

// draftRoomView names things the way the draft room does.
type draftRoomView struct {
	basis   draftBasis
	facts   nativecheck.Facts
	records []Record
	now     time.Time
}

func (v draftRoomView) player(id string) string { return playerNamed(id, v.facts).Name }

func (v draftRoomView) franchise(id string) string {
	for _, o := range v.basis.owners {
		if o.ID == id {
			return o.Name
		}
	}
	return id
}

func (v draftRoomView) pickWords(ref rookiedraft.PickRef) string {
	i := slices.IndexFunc(v.basis.draft.Picks, func(p rookiedraft.Result) bool {
		return p.Round == ref.Round && p.Original == ref.Original && p.Extra == ref.Extra
	})
	if i < 0 {
		return fmt.Sprintf("Round %d, %s's pick", ref.Round, v.franchise(ref.Original))
	}
	p := v.basis.draft.Picks[i]
	return fmt.Sprintf("Round %d, pick %d (%s's pick)", p.Round, p.Slot, v.franchise(p.Original))
}

// stateOf says what a pick is now.
func (v draftRoomView) stateOf(ref rookiedraft.PickRef) string {
	for _, p := range v.basis.draft.Picks {
		if p.Round != ref.Round || p.Original != ref.Original || p.Extra != ref.Extra {
			continue
		}
		switch {
		case p.Player != "":
			return "Taken: " + v.player(p.Player)
		case p.Skipped:
			return "Skipped"
		case v.basis.draft.OnClock != nil && v.basis.draft.OnClock.Round == p.Round &&
			v.basis.draft.OnClock.Original == p.Original && p.Extra == 0:
			return "On the clock"
		}
		return "Waiting for its turn"
	}
	return "Not in this draft"
}

func (v draftRoomView) field(label, before, after string) PreviewField {
	return PreviewField{Label: label, Before: before, BeforeSource: "The draft as it stands now",
		After: after, AfterSource: proposed, When: "now"}
}

// describe is the one field a draft change shows, and the warning to read before confirming it.
func (v draftRoomView) describe(raw RawCommand) (PreviewField, string) {
	d := raw.Args.Draft
	switch raw.Command {
	case DraftSkip:
		ref := d.Act.ref()
		return v.field(v.pickWords(ref), v.stateOf(ref), "Skipped; the next pick opens with a full clock"),
			"The pick is passed over at once. The franchise does not get it back unless you choose a player for it later."
	case DraftAssign:
		ref := d.Act.ref()
		return v.field(v.pickWords(ref), v.stateOf(ref), "Taken by "+v.player(d.Act.Player)+" at the slot's contract"),
			"You choose for the franchise. A pick the clock skipped is given back; the clock does not move."
	case DraftInsert:
		ref := d.Act.ref()
		return v.field("Extra pick after "+v.pickWords(ref), "No extra pick", v.franchise(d.Act.Franchise)+
				" takes "+v.player(d.Act.Player)+" at the contract of that slot"),
			"An extra pick takes no turn on the clock and does not move anyone else's pick."
	case DraftChange:
		ref := d.Act.ref()
		return v.field(v.pickWords(ref), v.stateOf(ref), "Taken by "+v.player(d.Act.Player)), changeWarning(v, d.Act)
	case DraftTimer:
		return v.timer(d.Act)
	case DraftRevert:
		return v.restore(d)
	case DraftStartupOpen:
		return v.opening(d)
	}
	if d.Startup {
		return v.field("The "+d.noun(), "Open", "Closed: every pick still unused is skipped"),
			"Nobody goes to the free-agent pool and no picks are issued. A closed draft can only be reopened by restoring it."
	}
	return v.closing(d.Season)
}

// changeWarning warns before a pick already made is swapped, rather than doing it quietly.
func changeWarning(v draftRoomView, a *RawDraftAct) string {
	warning := fmt.Sprintf("This swaps the rookie a pick already made holds: %s goes back to the undrafted rookies "+
		"and %s takes the same contract.", v.player(a.Replaces), v.player(a.Player))
	if v.basis.draft.Started && v.basis.closed == nil {
		warning += " The draft is running; picks made after this one are not changed."
	}
	return warning
}

func (v draftRoomView) timer(a *RawDraftAct) (PreviewField, string) {
	now := "Not running"
	if c := v.basis.draft.OnClock; c != nil {
		left := leaguerules.DurationWords(c.Remaining.Round(time.Second))
		now = fmt.Sprintf("Running, %s left", left)
		if c.Paused {
			now = fmt.Sprintf("Paused with %s left", left)
		}
	}
	after := map[string]string{TimerPause: "Paused where it stands", TimerResume: "Running again with the time it had left",
		TimerSet: fmt.Sprintf("%s left for the pick on the clock",
			leaguerules.DurationWords(time.Duration(a.Seconds)*time.Second))}[a.Timer]
	return v.field("The pick clock", now, after),
		"Only the pick on the clock is affected. A paused clock lets no one pick and skips no one."
}

func (v draftRoomView) restore(d *RawDraftArgs) (PreviewField, string) {
	at, err := checkpointOf(v.records, d.key(), d.Act.To)
	label := "an earlier point"
	setAside := 0
	if err == nil {
		label = namedLabel(v.records[at], v.facts)
		active := Replay(v.records).Active
		for _, r := range v.records[at+1:] {
			if liveDraftEntry(r.Command.Command) && ofDraft(r, d.key()) && active[r.Receipt.Entry] {
				setAside++
			}
		}
	}
	made := 0
	for _, p := range v.basis.draft.Picks {
		if p.Player != "" {
			made++
		}
	}
	return v.field("The "+d.noun(), countWords(made, "pick made", "picks made"),
			"Back to how it stood after: "+label),
		fmt.Sprintf("%s set aside, not erased: the ledger keeps both states. The pick that comes back on the "+
			"clock gets a fresh clock.", countWords(setAside, "later change is", "later changes are"))
}

func (v draftRoomView) closing(season int) (PreviewField, string) {
	return v.field("The "+fmt.Sprint(season)+" rookie draft", "Open",
			"Closed: undrafted rookies join the free-agent pool and picks for the next two seasons are issued"),
		"Every rookie nobody took goes to the free-agent pool. Picks for two seasons out are issued to every " +
			"franchise and can be traded at once. A closed draft can only be reopened by restoring it."
}

// opening is the startup draft before and after it opens.
func (v draftRoomView) opening(d *RawDraftArgs) (PreviewField, string) {
	rounds := "its rounds"
	if s, err := rookiedraft.ResolveStartup(v.basis.rules); err == nil {
		rounds = countWords(s.Rounds, "round", "rounds")
	}
	return v.field("The "+d.noun(), "Not open", "Open now, "+rounds+"; the first pick is on the clock"),
		"The first round's order is fixed as the draft opens: the commissioner's list, or drawn at random and " +
			"recorded. Every player taken signs the startup draft's contract."
}

func countWords(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
