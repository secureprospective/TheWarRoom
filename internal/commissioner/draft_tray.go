package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

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

// NativePoolEntry is a rookie nobody drafted, released to the free-agent pool when his draft closed.
type NativePoolEntry struct {
	Player string `json:"player"`
	Name   string `json:"name,omitempty"`
	Season int    `json:"season"`
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
func trayAct(r Record) (rookiedraft.Act, bool) {
	d := r.Command.Args.Draft
	act := rookiedraft.Act{At: r.Receipt.RecordedAt}
	if o := r.Command.Args.DraftOutcome; o != nil {
		act.Terms = o.Terms
	}
	if d.Act != nil {
		a := d.Act
		act.Pick, act.Player, act.Replaces, act.Owner = a.ref(), a.Player, a.Replaces, a.Franchise
		act.Duration = time.Duration(a.Seconds) * time.Second
	}
	switch r.Command.Command {
	case DraftSkip:
		act.Kind = rookiedraft.Skip
	case DraftAssign:
		act.Kind = rookiedraft.Assign
	case DraftInsert:
		act.Kind = rookiedraft.Insert
	case DraftChange:
		act.Kind = rookiedraft.Change
	case DraftRevert:
		act.Kind = rookiedraft.Restart
	case DraftArchive:
		act.Kind = rookiedraft.Close
	case DraftTimer:
		act.Kind = timerKinds()[d.Act.Timer]
	default:
		return act, false
	}
	return act, true
}

func timerKinds() map[string]rookiedraft.ActKind {
	return map[string]rookiedraft.ActKind{TimerPause: rookiedraft.Pause, TimerResume: rookiedraft.Resume,
		TimerSet: rookiedraft.SetClock}
}

// trayLabel describes a tray, restore or close entry for the ledger.
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
