package market

import "time"

// NativeMove uses the HQ receipt shape without MFL targets or synthetic MFL transitions.
type NativeMove struct {
	CorrelationID string            `json:"correlationId"`
	Spec          NativeMoveSpec    `json:"spec"`
	State         string            `json:"state"`
	Audit         []NativeMoveAudit `json:"audit"`
}

type NativeMoveSpec struct {
	Intent      string            `json:"intent"`
	LeagueID    string            `json:"leagueId"`
	FranchiseID string            `json:"franchiseId"`
	Subject     NativeMoveSubject `json:"subject"`
	Expected    struct{}          `json:"expected"`
	Gravity     string            `json:"gravity"`
	Undo        string            `json:"undo"`
	Target      NativeMoveTarget  `json:"target"`
	Deadline    *time.Time        `json:"deadline,omitempty"`
}

type NativeMoveSubject struct {
	Players []string `json:"players"`
	Picks   []string `json:"picks"`
}

type NativeMoveTarget struct {
	Kind string `json:"kind"`
}

type NativeMoveAudit struct {
	At    time.Time `json:"at"`
	From  string    `json:"from"`
	Event string    `json:"event"`
	To    string    `json:"to"`
	Note  string    `json:"note"`
}

type NativeMoves struct {
	League    string       `json:"league"`
	Franchise string       `json:"franchise"`
	Planned   []NativeMove `json:"planned"`
	Reason    string       `json:"reason"`
	Moves     []NativeMove `json:"moves"`
}

func nativeMove(league, franchise, id, intent, state, note string, at time.Time) NativeMove {
	return NativeMove{CorrelationID: id, State: state,
		Spec: NativeMoveSpec{Intent: intent, LeagueID: league, FranchiseID: franchise,
			Subject: NativeMoveSubject{Players: []string{}, Picks: []string{}},
			Gravity: "G0", Undo: "irreversible", Target: NativeMoveTarget{Kind: "native"}},
		Audit: []NativeMoveAudit{{At: at, From: "draft", Event: "recorded", To: state, Note: note}}}
}
