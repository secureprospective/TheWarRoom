package commissioner

import (
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
)

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
