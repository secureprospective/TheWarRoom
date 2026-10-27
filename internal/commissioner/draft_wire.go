package commissioner

import (
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
	Act     *RawDraftAct             `json:"act,omitempty"`
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
	return trayCommand(command) || command == DraftRevert || command == DraftArchive
}

// standingRule is a draft order command that is a league rule rather than a choice for one draft.
func standingRule(command string) bool {
	return command == DraftOrderPattern || command == DraftOrderTiebreak
}

func (a RawArgs) validateDraft(command string) error {
	other := a
	other.Draft, other.DraftOutcome = nil, nil
	if a.Draft == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return fmt.Errorf("send only the details of this draft change")
	}
	if err := a.DraftOutcome.validate(command); err != nil {
		return err
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
	switch command {
	case DraftPick:
		return d.keptPick()
	case DraftOrderAuto, DraftOrderSet, DraftOrderPattern, DraftOrderTiebreak, DraftStart:
		return d.keptSetup(command)
	}
	return d.keptTray(command)
}

// keptSetup is the details of the commands that order the draft and start it.
func (d RawDraftArgs) keptSetup(command string) (RawDraftArgs, error) {
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
	if liveDraftEntry(r.Command.Command) && r.Command.Command != DraftPick {
		return trayLabel(r)
	}
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

func draftRuleField(field string) bool { return strings.HasPrefix(field, "draft.") }

// checkDraftReversal and checkDraftRule keep a drafted class intact: undoing the start or the order, or
// changing a draft rule, must leave every pick already made with its place on the clock.

// The live draft tray (census CM-043), the restore (CM-044) and the close (CM-045).
const (
	DraftSkip    = "draft.skip"
	DraftAssign  = "draft.assign"
	DraftInsert  = "draft.insert"
	DraftChange  = "draft.change"
	DraftTimer   = "draft.timer"
	DraftRevert  = "draft.revert"
	DraftArchive = "draft.archive"

	// maxDraftClock keeps a set clock to something a person could mean.
	maxDraftClock = 14 * 24 * 3600
)

// The three ways to work the clock.
const (
	TimerPause  = "pause"
	TimerResume = "resume"
	TimerSet    = "set"
)

// RawDraftAct is what a person sends to work the live draft. A pick is named by its round and the franchise
// it began with (Extra counts the picks inserted after it). Replaces is the rookie the commissioner saw on a
// pick they change. Go fills the contract itself, in DraftOutcome.
type RawDraftAct struct {
	Round     int    `json:"round,omitempty"`
	Original  string `json:"original,omitempty"`
	Extra     int    `json:"extra,omitempty"`
	Player    string `json:"player,omitempty"`
	Replaces  string `json:"replaces,omitempty"`
	Franchise string `json:"franchise,omitempty"`
	Timer     string `json:"timer,omitempty"`
	Seconds   int    `json:"seconds,omitempty"`
	To        string `json:"to,omitempty"`
}

// DraftOutcome is what Go fixed when the act was recorded: the contract a rookie signed, or the rookies a
// closed draft released to the free-agent pool and the picks it issued. The sender never supplies it.
type DraftOutcome struct {
	Terms *rookiedraft.Terms `json:"terms,omitempty"`
	// Franchise and Slot are who receives the rookie and where in the round his pick falls.
	Franchise string            `json:"franchise,omitempty"`
	Slot      int               `json:"slot,omitempty"`
	Pool      []NativePoolEntry `json:"pool,omitempty"`
	Issued    []NativePick      `json:"issued,omitempty"`
}

func trayCommand(command string) bool {
	switch command {
	case DraftSkip, DraftAssign, DraftInsert, DraftChange, DraftTimer:
		return true
	}
	return false
}

// liveDraftEntry is a record that changes a draft once it runs: the picks, the tray, the restore and the close.
func liveDraftEntry(command string) bool {
	return trayCommand(command) || command == DraftPick || command == DraftRevert || command == DraftArchive
}

func rookieID(player string) error {
	if id, err := playerid.New(player); err != nil || id.String() != player {
		return fmt.Errorf("choose a valid rookie player")
	}
	return nil
}

// keptTray is the details of a tray, restore or close command, validated. Anything else sent with it is
// refused by the caller, which compares the result with what was sent.
func (d RawDraftArgs) keptTray(command string) (RawDraftArgs, error) {
	keep := RawDraftArgs{Season: d.Season}
	if command == DraftArchive {
		return keep, nil
	}
	if d.Act == nil {
		return keep, fmt.Errorf("send the details of this draft change")
	}
	act, err := d.Act.kept(command)
	keep.Act = &act
	return keep, err
}

func (a RawDraftAct) kept(command string) (RawDraftAct, error) {
	switch command {
	case DraftRevert:
		if blank(a.To) {
			return a, fmt.Errorf("choose the point to restore the draft to")
		}
		return RawDraftAct{To: a.To}, nil
	case DraftTimer:
		return a.keptTimer()
	}
	if a.Round < 1 || a.Extra < 0 || blank(a.Original) {
		return a, fmt.Errorf("choose a pick in this draft")
	}
	pick := RawDraftAct{Round: a.Round, Original: a.Original, Extra: a.Extra}
	if command == DraftSkip {
		return pick, nil
	}
	if err := rookieID(a.Player); err != nil {
		return a, err
	}
	pick.Player = a.Player
	switch command {
	case DraftInsert:
		if blank(a.Franchise) {
			return a, fmt.Errorf("choose the franchise that receives the extra pick")
		}
		pick.Franchise = a.Franchise
	case DraftChange:
		if err := rookieID(a.Replaces); err != nil || a.Replaces == a.Player {
			return a, fmt.Errorf("name the rookie the pick holds now and a different rookie to give it")
		}
		pick.Replaces = a.Replaces
	}
	return pick, nil
}

func (a RawDraftAct) keptTimer() (RawDraftAct, error) {
	switch {
	case a.Timer == TimerSet && a.Seconds >= 1 && a.Seconds <= maxDraftClock:
		return RawDraftAct{Timer: a.Timer, Seconds: a.Seconds}, nil
	case a.Timer == TimerSet:
		return a, fmt.Errorf("set the draft clock to between one second and 14 days")
	case (a.Timer == TimerPause || a.Timer == TimerResume) && a.Seconds == 0:
		return RawDraftAct{Timer: a.Timer}, nil
	}
	return a, fmt.Errorf("pause the draft clock, resume it, or set the time it has left")
}

// ref is the pick a tray command acts on.
func (a RawDraftAct) ref() rookiedraft.PickRef {
	return rookiedraft.PickRef{Round: a.Round, Original: a.Original, Extra: a.Extra}
}

// trayAct is the engine's act for one recorded tray command.

func trayLabel(r Record) string {
	d := r.Command.Args.Draft
	a := d.Act
	if a == nil {
		return fmt.Sprintf("Closed the %d rookie draft: undrafted rookies went to the free-agent pool and new "+
			"picks were issued", d.Season)
	}
	pick := fmt.Sprintf("round %d, %s's pick", a.Round, a.Original)
	switch r.Command.Command {
	case DraftSkip:
		return fmt.Sprintf("Skipped %s in the %d rookie draft", pick, d.Season)
	case DraftAssign:
		return fmt.Sprintf("Chose player %s for %s in the %d rookie draft", a.Player, pick, d.Season)
	case DraftInsert:
		return fmt.Sprintf("Added an extra pick after %s: player %s for %s (%d rookie draft)", pick, a.Player,
			a.Franchise, d.Season)
	case DraftChange:
		return fmt.Sprintf("Changed %s from player %s to player %s in the %d rookie draft", pick, a.Replaces,
			a.Player, d.Season)
	case DraftTimer:
		return timerLabel(d)
	}
	return fmt.Sprintf("Restored the %d rookie draft to an earlier point", d.Season)
}

func timerLabel(d *RawDraftArgs) string {
	switch d.Act.Timer {
	case TimerPause:
		return fmt.Sprintf("Paused the %d rookie draft clock", d.Season)
	case TimerResume:
		return fmt.Sprintf("Resumed the %d rookie draft clock", d.Season)
	}
	return fmt.Sprintf("Set the %d rookie draft clock to %s", d.Season, (time.Duration(d.Act.Seconds) * time.Second))
}

// validate holds what Go recorded to the command that may carry it: a contract with a rookie given to a pick,
// the releases and issued picks with a close. Nothing else carries an outcome.

func (o *DraftOutcome) validate(command string) error {
	if o == nil {
		return nil
	}
	switch {
	case (command == DraftAssign || command == DraftInsert) && o.Terms != nil && o.Pool == nil && o.Issued == nil:
		if o.Terms.Years < 1 || o.Terms.Salary < 0 || blank(o.Franchise) || o.Slot < 1 {
			return fmt.Errorf("a rookie contract needs a franchise, a slot, at least a year and no negative salary")
		}
		return nil
	case command == DraftArchive && o.Terms == nil && o.Franchise == "" && o.Slot == 0:
		return nil
	}
	return fmt.Errorf("send only the details of this draft change")
}
