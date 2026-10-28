package commissioner

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func CheckDraftAppend(raw RawCommand, records []Record, at time.Time, standings DraftStandings) error {
	if done, err := checkBeyondDraftCommands(raw, records, at, standings); done {
		return err
	}
	if !draftCommand(raw.Command) {
		return nil
	}
	if len(records) == 0 {
		return Refused(fmt.Errorf("create this app league before setting its draft"))
	}
	if raw.Command == DraftPick {
		return Refused(fmt.Errorf("a rookie pick is made by the franchise on the clock"))
	}
	if raw.Command == DraftStartupOpen {
		return Refused(checkStartupOpen(raw, records, at))
	}
	if liveDraftEntry(raw.Command) {
		return Refused(checkLiveDraft(raw, records, at, standings))
	}
	if _, closed := closeRecord(records, raw.Args.Draft.key()); closed && !standingRule(raw.Command) {
		return Refused(fmt.Errorf("the %s is closed", raw.Args.Draft.noun()))
	}
	if raw.Command == DraftOrderSet {
		if err := checkOrderList(raw.Args.Draft, records, at); err != nil {
			return Refused(err)
		}
	}
	if err := checkDraftTiming(raw, records, at, standings); err != nil {
		return Refused(err)
	}
	return Refused(draftStaysConsistent(raw, records, at, standings))
}

// checkBeyondDraftCommands holds the changes that are not draft commands but bear on a draft: a reversal,
// a draft rule and the draft's window. done says one of them was checked, with err its answer.
func checkBeyondDraftCommands(raw RawCommand, records []Record, at time.Time,
	standings DraftStandings) (done bool, err error) {
	switch {
	case correctionCommand(raw.Command):
		return true, Refused(draftReadable(withProposed(raw, records, at), draftsWithPicks(records), at, standings))
	case raw.Command == Revert:
		return true, Refused(checkDraftReversal(raw, records, at, standings))
	case raw.Command == RulesSet && draftRuleField(raw.Args.Field):
		trial := withProposed(raw, records, at)
		if err := checkStartupList(trial, at); err != nil {
			return true, Refused(err)
		}
		return true, Refused(draftReadable(trial, draftsWithPicks(records), at, standings))
	case windowCommand(raw.Command) && raw.Args.Window.ID == rookieDraftWindow && draftStartSet(records):
		return true, Refused(fmt.Errorf("the rookie draft start was set with the draft start; change it there " +
			"so the calendar and the draft agree"))
	}
	return false, nil
}

// checkDraftTiming holds a draft change to whether the clock runs.
func checkDraftTiming(raw RawCommand, records []Record, at time.Time, standings DraftStandings) error {
	running := false
	for _, key := range draftSeasons(raw, records) {
		running = running || draftClockRuns(records, key, at, standings)
	}
	switch {
	case raw.Command == DraftStart && running:
		return fmt.Errorf("the draft has started; its start can no longer move")
	case raw.Command == DraftStart && raw.Args.Draft.At.Before(at):
		return fmt.Errorf("choose a start time that has not passed")
	case raw.Command != DraftStart && running != raw.Args.Draft.Running:
		return runningMismatch(running)
	}
	return nil
}

func runningMismatch(running bool) error {
	if running {
		return fmt.Errorf("the draft clock is running; say so and give a reason to change the draft now")
	}
	return fmt.Errorf("the draft clock is not running; this is a regular draft change")
}

// draftSeasons are the drafts a command bears on: its own, or every draft with a recorded start for a
// standing rule.
func draftSeasons(raw RawCommand, records []Record) []DraftKey {
	if !standingRule(raw.Command) {
		return sideDraftKeys(records, raw.Args.Draft.key())
	}
	keys := []DraftKey{}
	for _, r := range records {
		if r.Command.Command == DraftStart && !slices.Contains(keys, r.Command.Args.Draft.key()) {
			keys = append(keys, sideDraftKeys(records, r.Command.Args.Draft.key())...)
		}
	}
	return keys
}

// draftStaysConsistent reads the draft as it would stand with the change and refuses a change that leaves
// a pick already made without its place on the clock.
func draftStaysConsistent(raw RawCommand, records []Record, at time.Time, standings DraftStandings) error {
	return draftReadable(withProposed(raw, records, at), draftSeasons(raw, records), at, standings)
}

// withProposed is the ledger as it would stand once the change is recorded.
func withProposed(raw RawCommand, records []Record, at time.Time) []Record {
	return append(slices.Clone(records), Record{Command: raw, Principal: "proposed",
		Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: at}})
}

// draftReadable holds each draft to being readable: a draft with picks made must keep an order and a clock
// that still give every one of them its place.
func draftReadable(trial []Record, keys []DraftKey, at time.Time, standings DraftStandings) error {
	for _, key := range keys {
		basis, err := readDraft(trial, key, at, standings)
		if err == nil && basis.words != "" && len(draftActs(trial, key)) > 0 {
			err = errors.New(strings.TrimSuffix(basis.words, "."))
		}
		if err != nil {
			return fmt.Errorf("this change contradicts a pick already made: %v", err)
		}
	}
	return nil
}

// draftsWithPicks are the drafts that have a pick, or any other live change, in force.
func draftsWithPicks(records []Record) []DraftKey {
	active, keys := Replay(records).Active, []DraftKey{}
	for _, r := range records {
		if liveDraftEntry(r.Command.Command) && active[r.Receipt.Entry] &&
			!slices.Contains(keys, r.Command.Args.Draft.key()) {
			keys = append(keys, r.Command.Args.Draft.key())
		}
	}
	return keys
}

func checkDraftReversal(raw RawCommand, records []Record, at time.Time, standings DraftStandings) error {
	for _, r := range records {
		if r.Receipt.Entry == raw.Args.Entry && liveDraftEntry(r.Command.Command) {
			return fmt.Errorf("a change to the live draft cannot be undone on its own; restore the draft to " +
				"an earlier point instead")
		}
	}
	if err := draftReadable(withProposed(raw, records, at), draftsWithPicks(records), at, standings); err != nil {
		return err
	}
	for _, r := range records {
		if r.Receipt.Entry == raw.Args.Entry && draftCommand(r.Command.Command) {
			return draftReadable(withProposed(raw, records, at), draftSeasons(r.Command, records), at, standings)
		}
	}
	return nil
}

// checkOrderList holds a recorded order to the league's franchises and the draft's rounds.
func checkOrderList(d *RawDraftArgs, records []Record, at time.Time) error {
	if err := listsEveryFranchise(d.Order, LeagueFranchises(records)); err != nil {
		return err
	}
	rules, err := RulesNow(records, at)
	if err != nil {
		return err
	}
	if rounds := rules.Draft.Rounds; d.Round != 0 && (rounds.State != leaguerules.IsKnown || d.Round > rounds.Value) {
		return fmt.Errorf("choose a round in this draft")
	}
	return nil
}

// checkLiveDraft holds a change to the live draft to the state of the draft. The draft must be readable and
// running; only a restore may touch a closed one; and the change is tried against the draft as it would
// stand, so the same replay that will read it later is the one that refuses it now.
func checkLiveDraft(raw RawCommand, records []Record, at time.Time, standings DraftStandings) error {
	key := raw.Args.Draft.key()
	if err := checkDraftSide(records, key); err != nil {
		return err
	}
	basis, err := readDraft(records, key, at, standings)
	switch {
	case err != nil:
		return err
	case basis.words != "":
		return fmt.Errorf("the draft is not ready: %s", strings.TrimSuffix(basis.words, "."))
	case !basis.started:
		return fmt.Errorf("the %s has not started", key.noun())
	case basis.closed != nil && raw.Command != DraftRevert:
		return fmt.Errorf("the %s is closed; restore it to an earlier point to reopen it", key.noun())
	case raw.Command == DraftRevert:
		if _, err := checkpointOf(records, key, raw.Args.Draft.Act.To); err != nil {
			return err
		}
	}
	trial, err := readDraft(withProposed(raw, records, at), key, at, standings)
	if err != nil {
		return err
	}
	if raw.Command == DraftRevert && trial.draft.Exhausted && trial.closed == nil {
		return fmt.Errorf("every pick would still be used, so the draft would close itself again; restore it " +
			"to a point before the last pick")
	}
	return nil
}

// listsEveryFranchise holds an order to every franchise of the league exactly once.
func listsEveryFranchise(order, franchises []string) error {
	seen := map[string]bool{}
	for _, f := range order {
		if !slices.Contains(franchises, f) || seen[f] {
			return errors.New("list every franchise exactly once")
		}
		seen[f] = true
	}
	if len(seen) != len(franchises) {
		return errors.New("list every franchise exactly once")
	}
	return nil
}
