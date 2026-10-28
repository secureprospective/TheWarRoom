package commissioner

import (
	"fmt"
	"reflect"
)

// Cutover moves a league imported from MFL over to the app. Until it is recorded the league follows MFL
// (its source is the MFL import); from it on, the league is an app league and MFL no longer changes it.
// The MFL league itself is never touched, so there is nothing to undo: a move given up just starts the
// import again.
const Cutover = "commish.league.cutover"

// cutoverLabel is the ledger's words for the act.
const cutoverLabel = "Moved the league over from MFL: the app runs it now, and MFL no longer changes it"

func (a RawArgs) validateCutover() error {
	if !reflect.DeepEqual(a, RawArgs{}) {
		return fmt.Errorf("moving a league over from MFL takes only a reason")
	}
	return nil
}

// cutOver says the league's ledger holds the cutover act.
func cutOver(records []Record) bool {
	for _, r := range records {
		if r.Command.Command == Cutover {
			return true
		}
	}
	return false
}

// importedSource is the league's source as its creation recorded it, before any cutover.
func importedSource(records []Record) string {
	if len(records) != 0 && records[0].Command.Command == Create {
		args := records[0].Command.Args
		if args.Source != "" {
			return args.Source
		}
		if p, ok := presetByID(args.Preset); ok {
			return p.source
		}
	}
	return SourceNative
}

// CheckCutoverAppend admits the cutover once, on a seeded league that still follows MFL, and never its
// reversal: the app keeps no way back because the MFL league was never changed.
func CheckCutoverAppend(raw RawCommand, records []Record) error {
	if raw.Command == Revert {
		for _, r := range records {
			if r.Receipt.Entry == raw.Args.Entry && r.Command.Command == Cutover {
				return Refused(fmt.Errorf("moving the league over from MFL is permanent; " +
					"to start again, import the MFL league again"))
			}
		}
	}
	if raw.Command != Cutover {
		return nil
	}
	switch {
	case importedSource(records) != SourceMFLImport:
		return Refused(fmt.Errorf("only a league imported from MFL can be moved over"))
	case cutOver(records):
		return Refused(fmt.Errorf("this league has already been moved over from MFL"))
	case !seeded(records):
		return Refused(fmt.Errorf("record the league's rosters, contracts and picks from MFL before moving it over"))
	}
	return nil
}

// CutOver says whether the league imported from MFL has been moved over to the app.
func CutOver(records []Record) bool { return cutOver(records) }
