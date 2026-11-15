package commissioner

import (
	"fmt"
	"time"
)

// RosterTimed identifies ordinary moves whose effective time may differ from receipt time.
func RosterTimed(command string) bool {
	switch command {
	case RosterCut, RosterReserve, CorrectRoster, CorrectPick, WaiverProcess, WaiverOverride, BidsProcess, BidsOverride,
		TradeAccept, TradeReview, TradeAppealRule, TradeOverride:
		return true
	}
	return false
}

func (r RawCommand) defaultRosterTime() RawCommand {
	if RosterTimed(r.Command) && r.Effective == "" {
		r.Effective = "now"
	}
	return r
}

func (r RawCommand) validateRosterPlacement() error {
	if r.Visibility != "league" {
		return fmt.Errorf("roster changes must be visible to every owner")
	}
	if r.Effective == "" || r.Effective == "now" {
		return nil
	}
	at, err := time.Parse(time.RFC3339Nano, r.Effective)
	if err != nil || at.IsZero() {
		return fmt.Errorf("choose when this roster change takes effect")
	}
	return nil
}

// EffectiveTime resolves a durable command's timing without changing its receipt.
func (r RawCommand) EffectiveTime(received time.Time) (time.Time, error) {
	if r.Effective == "" || r.Effective == "now" || r.Effective == "next_rollover" {
		return received, nil
	}
	at, err := time.Parse(time.RFC3339Nano, r.Effective)
	if err != nil {
		return time.Time{}, fmt.Errorf("read effective time: %w", err)
	}
	return at, nil
}

// CheckRosterTime runs under the append lock, using trusted commissioner authority.
func CheckRosterTime(raw RawCommand, now time.Time, role string) error {
	if !RosterTimed(raw.Command) {
		return nil
	}
	at, err := raw.EffectiveTime(now)
	if err != nil {
		return Refused(err)
	}
	if at.After(now) {
		return Refused(fmt.Errorf("a roster change cannot take effect in the future"))
	}
	if at.Before(now) && role != "commissioner" && role != "cocommissioner" {
		return Refused(fmt.Errorf("only a commissioner may backdate a roster change"))
	}
	return nil
}
