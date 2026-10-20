package commissioner

import (
	"errors"
	"slices"
)

// CheckRolloverAppend refuses a reversal that would leave a season rollover and the changes around it
// disagreeing about which season they belong to. Everything else about a rollover is checked as it is
// prepared.
func CheckRolloverAppend(raw RawCommand, records []Record) error {
	if raw.Command != Revert {
		return nil
	}
	if reason := rolloverReversalRefusal(records, raw.Args.Entry); reason != "" {
		return Refused(errors.New(reason))
	}
	return nil
}

// rolloverReversalRefusal says why the entry cannot be reversed (or restored by reversing its reversal)
// without breaking a rollover: a rollover cannot be reversed while later changes depend on its season,
// and a change a rollover captured cannot be reversed once the league has moved past it. It is empty when
// the reversal is fine.
func rolloverReversalRefusal(records []Record, entry string) string {
	target, found := reversalRoots(records)[entry]
	if !found {
		return ""
	}
	active := Replay(records).Active
	later := records[target+1:]
	if records[target].Command.Command == SeasonRollover {
		if slices.ContainsFunc(later, func(r Record) bool {
			return active[r.Receipt.Entry] && rolloverKind(r.Command.Command)
		}) {
			return "Reverse the later changes first: rules, contracts, windows or another rollover were " +
				"recorded after this season rollover."
		}
		return ""
	}
	if rolloverKind(records[target].Command.Command) && slices.ContainsFunc(later, func(r Record) bool {
		return active[r.Receipt.Entry] && r.Command.Command == SeasonRollover
	}) {
		return "This change is already part of a season the league has moved past; reverse the season " +
			"rollover first."
	}
	return ""
}
