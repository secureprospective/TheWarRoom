package commissioner

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	DraftOrderAuto     = "draft.order.auto"
	DraftOrderSet      = "draft.order.set"
	DraftOrderPattern  = "draft.order.pattern"
	DraftOrderTiebreak = "draft.order.tiebreak"
	DraftStart         = "draft.start"
	DraftPick          = "draft.pick"

	// rookieDraftWindow is the calendar window the rookie draft opens in.
	rookieDraftWindow = "rookie_draft"
	draftPickAction   = leaguerules.Action("draft_pick")
)

// RawDraftArgs carry the details of one draft command. Running says the commissioner knows the draft
// clock is running, which makes an order change a recorded-reason change.
type RawDraftArgs struct {
	Season  int                      `json:"season,omitempty"`
	Round   int                      `json:"round,omitempty"`
	Order   []string                 `json:"order,omitempty"`
	Source  leaguerules.OrderSource  `json:"source,omitempty"`
	Pattern leaguerules.OrderPattern `json:"pattern,omitempty"`
	Tie     leaguerules.TieRule      `json:"tie,omitempty"`
	At      time.Time                `json:"at,omitempty"`
	Pick    *RawDraftSelection       `json:"pick,omitempty"`
	Running bool                     `json:"running,omitempty"`
}

// RawDraftSelection is one recorded owner pick. Go builds it from the owner's choice of player: the slot
// and the contract are fixed here, at the moment of the pick.
type RawDraftSelection struct {
	Franchise   string            `json:"franchise"`
	Player      string            `json:"player"`
	EntrySeason int               `json:"entrySeason"`
	Round       int               `json:"round"`
	Slot        int               `json:"slot"`
	Original    string            `json:"original"`
	Terms       rookiedraft.Terms `json:"terms"`
}

func draftCommand(command string) bool {
	switch command {
	case DraftOrderAuto, DraftOrderSet, DraftOrderPattern, DraftOrderTiebreak, DraftStart, DraftPick:
		return true
	}
	return false
}

// standingRule is a draft order command that is a league rule rather than a choice for one draft.
func standingRule(command string) bool {
	return command == DraftOrderPattern || command == DraftOrderTiebreak
}

func (a RawArgs) validateDraft(command string) error {
	other := a
	other.Draft = nil
	if a.Draft == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return fmt.Errorf("send only the details of this draft change")
	}
	d := *a.Draft
	if standingRule(command) == (d.Season != 0) || d.Season < 0 || d.Season > 9999 {
		return fmt.Errorf("choose the draft season for a start, an order or a pick, and none for a standing rule")
	}
	kept, err := d.kept(command)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(d, kept) {
		return fmt.Errorf("send only the details of this draft change")
	}
	return nil
}

// kept is the command's own details, validated; anything else sent with it is refused by the caller.
func (d RawDraftArgs) kept(command string) (RawDraftArgs, error) {
	keep := RawDraftArgs{Season: d.Season, Running: d.Running}
	switch command {
	case DraftOrderAuto:
		if d.Source != leaguerules.WorstStandingsFirst {
			return keep, fmt.Errorf("choose the final standings as the automatic draft order")
		}
		keep.Source = d.Source
	case DraftOrderSet:
		if d.Round < 0 || len(d.Order) < 2 || slices.ContainsFunc(d.Order, blank) {
			return keep, fmt.Errorf("list every franchise, for one round or for all rounds")
		}
		keep.Round, keep.Order = d.Round, d.Order
	case DraftOrderPattern:
		if d.Pattern != leaguerules.Linear && d.Pattern != leaguerules.Snake {
			return keep, fmt.Errorf("choose a linear or snake draft order for the later rounds")
		}
		keep.Pattern = d.Pattern
	case DraftOrderTiebreak:
		if blank(string(d.Tie)) {
			return keep, fmt.Errorf("write down how ties in the draft order are broken")
		}
		keep.Tie = d.Tie
	case DraftStart:
		if d.At.IsZero() || d.Running {
			return keep, fmt.Errorf("choose when the draft starts")
		}
		keep.At = d.At
	default:
		return d.keptPick()
	}
	return keep, nil
}

func (d RawDraftArgs) keptPick() (RawDraftArgs, error) {
	p := d.Pick
	if p == nil || d.Running || blank(p.Franchise) || blank(p.Original) || p.Round < 1 || p.Slot < 1 ||
		p.EntrySeason != d.Season || p.Terms.Years < 1 || p.Terms.Salary < 0 {
		return d, fmt.Errorf("record the current pick and its rookie contract")
	}
	if id, err := playerid.New(p.Player); err != nil || id.String() != p.Player {
		return d, fmt.Errorf("choose a valid rookie player")
	}
	return RawDraftArgs{Season: d.Season, Pick: p}, nil
}

func blank(s string) bool { return strings.TrimSpace(s) == "" || strings.TrimSpace(s) != s }

func draftLabel(r Record) string {
	d := r.Command.Args.Draft
	switch r.Command.Command {
	case DraftStart:
		return fmt.Sprintf("Set the %d rookie draft to start at %s", d.Season, d.At.Format(time.RFC3339))
	case DraftPick:
		return fmt.Sprintf("Picked player %s for round %d, pick %d of the %d rookie draft",
			d.Pick.Player, d.Pick.Round, d.Pick.Slot, d.Season)
	case DraftOrderAuto:
		return fmt.Sprintf("Set the %d rookie draft order from the final standings", d.Season)
	case DraftOrderSet:
		if d.Round == 0 {
			return fmt.Sprintf("Set the %d rookie draft order for every round", d.Season)
		}
		return fmt.Sprintf("Set the %d rookie draft order for round %d", d.Season, d.Round)
	case DraftOrderPattern:
		return "Set the later-round draft pattern to " + string(d.Pattern)
	default:
		return "Set the draft order tie rule to " + string(d.Tie)
	}
}

// standingChange is the league rule a pattern or tie rule command sets, from the moment it is recorded.
func standingChange(r Record) (leaguerules.Change, bool) {
	d := r.Command.Args.Draft
	change := leaguerules.Change{LeagueID: r.Command.League, Effective: leaguerules.Effective{Kind: leaguerules.Now},
		RecordedAt: r.Receipt.RecordedAt}
	switch r.Command.Command {
	case DraftOrderPattern:
		change.Field, change.New = "draft.laterPattern", leaguerules.Known(d.Pattern)
	case DraftOrderTiebreak:
		change.Field, change.New = "draft.tieRule", leaguerules.Known(d.Tie)
	default:
		return change, false
	}
	return change, true
}

// CheckDraftAppend is run under the ledger append lock. It holds a draft change to the state of the draft:
// a start cannot move once the clock runs, an order change cannot contradict a pick already made, and a
// change made while the clock runs must say so, and give its reason.
func CheckDraftAppend(raw RawCommand, records []Record, at time.Time) error {
	if raw.Command == Revert {
		return Refused(checkDraftReversal(raw, records, at))
	}
	if raw.Command == RulesSet && draftRuleField(raw.Args.Field) {
		return Refused(draftReadable(withProposed(raw, records, at), draftsWithPicks(records), at))
	}
	if windowCommand(raw.Command) && raw.Args.Window.ID == rookieDraftWindow && draftStartSet(records) {
		return Refused(fmt.Errorf("the rookie draft start was set with the draft start; change it there so the " +
			"calendar and the draft agree"))
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
	if raw.Command == DraftOrderSet {
		if err := checkOrderList(raw.Args.Draft, records, at); err != nil {
			return Refused(err)
		}
	}
	if err := checkDraftTiming(raw, records, at); err != nil {
		return Refused(err)
	}
	return Refused(draftStaysConsistent(raw, records, at))
}

// checkDraftTiming holds a draft change to whether the clock runs.
func checkDraftTiming(raw RawCommand, records []Record, at time.Time) error {
	running := false
	for _, season := range draftSeasons(raw, records) {
		running = running || draftClockRuns(records, season, at)
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
func draftSeasons(raw RawCommand, records []Record) []int {
	if !standingRule(raw.Command) {
		return []int{raw.Args.Draft.Season}
	}
	seasons := []int{}
	for _, r := range records {
		if r.Command.Command == DraftStart && !slices.Contains(seasons, r.Command.Args.Draft.Season) {
			seasons = append(seasons, r.Command.Args.Draft.Season)
		}
	}
	return seasons
}

// draftStaysConsistent reads the draft as it would stand with the change and refuses a change that leaves
// a pick already made without its place on the clock.
func draftStaysConsistent(raw RawCommand, records []Record, at time.Time) error {
	return draftReadable(withProposed(raw, records, at), draftSeasons(raw, records), at)
}

// withProposed is the ledger as it would stand once the change is recorded.
func withProposed(raw RawCommand, records []Record, at time.Time) []Record {
	return append(slices.Clone(records), Record{Command: raw, Principal: "proposed",
		Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: at}})
}

// draftReadable holds each draft to being readable: a draft with picks made must keep an order and a clock
// that still give every one of them its place.
func draftReadable(trial []Record, seasons []int, at time.Time) error {
	for _, season := range seasons {
		basis, err := readDraft(trial, season, at)
		if err == nil && basis.words != "" && len(draftSelections(trial, season)) > 0 {
			err = errors.New(strings.TrimSuffix(basis.words, "."))
		}
		if err != nil {
			return fmt.Errorf("this change contradicts a pick already made: %v", err)
		}
	}
	return nil
}

// draftsWithPicks are the drafts that have a pick in force.
func draftsWithPicks(records []Record) []int {
	active, seasons := Replay(records).Active, []int{}
	for _, r := range records {
		if season := r.Command.Args.Draft; r.Command.Command == DraftPick && active[r.Receipt.Entry] &&
			!slices.Contains(seasons, season.Season) {
			seasons = append(seasons, season.Season)
		}
	}
	return seasons
}

func draftRuleField(field string) bool { return strings.HasPrefix(field, "draft.") }

// checkDraftReversal and checkDraftRule keep a drafted class intact: undoing the start or the order, or
// changing a draft rule, must leave every pick already made with its place on the clock.
func checkDraftReversal(raw RawCommand, records []Record, at time.Time) error {
	for _, r := range records {
		if r.Receipt.Entry != raw.Args.Entry {
			continue
		}
		c := r.Command
		if TradeCommand(c.Command) {
			return draftReadable(withProposed(raw, records, at), draftsWithPicks(records), at)
		}
		if draftCommand(c.Command) && c.Command != DraftPick {
			return draftReadable(withProposed(raw, records, at), draftSeasons(c, records), at)
		}
		if c.Command == RulesSet && draftRuleField(c.Args.Field) {
			return draftReadable(withProposed(raw, records, at), draftsWithPicks(records), at)
		}
	}
	return nil
}

// checkOrderList holds a recorded order to the league's franchises and the draft's rounds.
func checkOrderList(d *RawDraftArgs, records []Record, at time.Time) error {
	franchises := records[0].Command.Args.Franchises
	seen := map[string]bool{}
	for _, f := range d.Order {
		if !slices.Contains(franchises, f) || seen[f] {
			return fmt.Errorf("list every franchise exactly once")
		}
		seen[f] = true
	}
	if len(seen) != len(franchises) {
		return fmt.Errorf("list every franchise exactly once")
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
