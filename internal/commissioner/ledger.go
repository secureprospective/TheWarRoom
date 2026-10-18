package commissioner

import (
	"fmt"
	"strings"
	"time"
)

type Receipt struct {
	Entry      string        `json:"entry"`
	League     string        `json:"league"`
	Revision   int64         `json:"revision"`
	RecordedAt time.Time     `json:"recordedAt"`
	Message    string        `json:"message"`
	Plan       *RolloverPlan `json:"plan,omitempty"`
}

type Record struct {
	GeneratedSchedule []ScheduleWeek  `json:"generatedSchedule,omitempty"`
	Before            *RolloverBefore `json:"before,omitempty"`
	Command           RawCommand      `json:"command"`
	Principal         string          `json:"principal"`
	Role              string          `json:"role"`
	Gravity           string          `json:"gravity"`
	Canonical         string          `json:"canonical"`
	Receipt           Receipt         `json:"receipt"`
}

func (r Record) Validate() error {
	if err := r.validateRolloverRecord(); err != nil {
		return err
	}
	if err := r.Command.Validate(); err != nil {
		return err
	}
	if err := r.validateGeneratedSchedule(); err != nil {
		return err
	}
	definition, err := r.Command.EffectiveDefinition()
	if err != nil {
		return err
	}
	if strings.TrimSpace(r.Principal) == "" || strings.TrimSpace(r.Role) == "" ||
		r.Gravity != definition.Gravity || r.Command.Gravity != definition.Gravity ||
		r.Receipt.League != r.Command.League || r.Receipt.Entry == "" ||
		r.Receipt.Revision < 1 || r.Receipt.RecordedAt.IsZero() {
		return fmt.Errorf("league ledger has invalid identity or receipt metadata")
	}
	canonical, err := r.Command.Canonical(r.Principal)
	if err != nil {
		return err
	}
	if canonical != r.Canonical {
		return fmt.Errorf("league ledger command does not match its recorded request")
	}
	return nil
}

type Projection struct {
	Creator string
	Name    string
	Active  map[string]bool
}

// Replay walks compensation links backwards, then sets aside what a draft restore compensated: a reversed reversal no longer
// compensates its target. Originals and every correction remain in the ledger.
func Replay(records []Record) Projection {
	p := Projection{Active: make(map[string]bool)}
	for _, r := range records {
		p.Active[r.Receipt.Entry] = true
		if r.Command.Command == Create {
			p.Creator, p.Name = r.Principal, r.Command.Args.Name
		}
	}
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Command.Command == Revert && p.Active[r.Receipt.Entry] {
			p.Active[r.Command.Args.Entry] = !p.Active[r.Command.Args.Entry]
		}
	}
	setAsideByDraftRestores(records, p.Active)
	return p
}

type CreatorAuthorizer struct{ Creator string }

func (a CreatorAuthorizer) Allow(_ string, principal, _ string) (string, error) {
	if a.Creator != "" && a.Creator == principal {
		return "commissioner", nil
	}
	return "", fmt.Errorf("only this league's commissioner may make this change")
}

type Entry struct {
	ID      string    `json:"id"`
	Who     string    `json:"who"`
	What    string    `json:"what"`
	Why     string    `json:"why"`
	When    time.Time `json:"when"`
	Gravity string    `json:"gravity"`
	// Undo is "league" or "private" where this principal may reverse the entry, and says how to send it.
	Undo       string `json:"undo"`
	ReplacedAt string `json:"replacedAt,omitempty"`
}

type Ledger struct {
	League   string  `json:"league"`
	Name     string  `json:"name"`
	Revision int64   `json:"revision"`
	Entries  []Entry `json:"entries"`
}

// Reading deliberately excludes payloads, request bodies and private metadata.
func Reading(league, principal string, records []Record, auth Authorizer) (Ledger, error) {
	p := Replay(records)
	replaced, err := replacedWaiting(records)
	if err != nil {
		return Ledger{}, err
	}
	out := Ledger{League: league, Name: p.Name, Entries: []Entry{}}
	role, err := auth.Allow(league, principal, "commish.ledger.read")
	private := err == nil && (role == "commissioner" || role == "cocommissioner")
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Receipt.Revision > out.Revision {
			out.Revision = r.Receipt.Revision
		}
		if !ledgerEntryVisible(r, records, principal, private) {
			continue
		}
		what, err := entryLabel(records, i)
		if err != nil {
			return Ledger{}, err
		}
		who := r.Principal
		if strings.HasPrefix(who, "local.") {
			who = "Local operator"
		}
		gravity := "Confirmation required"
		if r.Gravity == "G3" {
			gravity = "Recorded reason required"
		}
		if replaced[r.Receipt.Entry] != "" {
			what = strings.TrimSuffix(what, ", from the next rollover")
		}
		out.Entries = append(out.Entries, Entry{ID: r.Receipt.Entry, Who: who, What: what,
			Why: r.Command.Reason, When: r.Receipt.RecordedAt, Gravity: gravity,
			Undo: undoPlacement(records, i, p, principal, auth), ReplacedAt: replaced[r.Receipt.Entry]})
	}
	return out, nil
}

type NativeLeague struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func entryLabel(records []Record, i int) (string, error) {
	r := records[i]
	if label, ok, err := changeLabel(r); ok || err != nil {
		return label, err
	}
	switch r.Command.Command {
	case ContractEdit, CapAdjust, CapRelief, ReserveException:
		return overrideLabel(r), nil
	case RulesSet, ContractActionSet:
		return ruleLabel(r, records[:i])
	case DraftRevert:
		return draftRestoreLabel(records, i), nil
	case Revert:
		label, ok, err := revertRuleLabel(records, i)
		if err != nil || ok {
			return label, err
		}
		if label, ok := revertNoteLabel(records, i); ok {
			return label, nil
		}
	}
	return recordLabel(r), nil
}

// changeLabel words the entries whose arguments decode into a typed change; ok is false for any other.
func changeLabel(r Record) (label string, ok bool, err error) {
	command := r.Command.Command
	switch command {
	case PlayerPosition, PlayerPositionClear, ScorePlayer, ScoreFranchise:
		change, err := DecodeScoreChange(r.Command.Args, command)
		if err != nil {
			return "", true, err
		}
		return scoreChangeLabel(command, change), true, nil
	case StandingsRecordAdjust, StandingsOverride, StandingsOverrideClear:
		change, err := DecodeStandingsChange(r.Command.Args, command)
		if err != nil {
			return "", true, err
		}
		return standingsLabel(command, change), true, nil
	case StatApprove, StatReject, StatCorrect:
		change, err := DecodeStatChange(r.Command.Args, command)
		if err != nil {
			return "", true, err
		}
		return statChangeLabel(command, change), true, nil
	case LineupSet, LineupCorrect:
		change, err := DecodeLineupChange(r.Command.Args, command)
		if err != nil {
			return "", true, err
		}
		return lineupLabel(command, change), true, nil
	}
	return "", false, nil
}

func recordLabel(r Record) string {
	if label, ok := familyLabel(r); ok {
		return label
	}
	switch r.Command.Command {
	case ContractEdit, CapAdjust, CapRelief, ReserveException:
		return overrideLabel(r)
	case SeasonRollover:
		return fmt.Sprintf("Moved the league to season %d", r.Command.Args.Rollover.To)
	case NativeSeed:
		return "Recorded the starting rosters, contracts, reserves, franchises and picks from held data"
	case WindowSet, WindowMove, WindowRemove:
		return windowLabel(r)
	}

	if r.Command.Command == CalendarException {
		e := r.Command.Args.Exception
		return fmt.Sprintf("Allowed %s for %s outside its window until %s: %s",
			e.Action, e.Franchise, e.Until.Format(time.RFC3339), r.Command.Reason)
	}
	if r.Command.Command == Create {
		preset := "every rule waiting for a ruling"
		if r.Command.Args.Preset == PresetLegacy {
			preset = "the Legacy NFL rules"
		}
		return fmt.Sprintf("Created league “%s” with %d franchises and %s", r.Command.Args.Name,
			len(r.Command.Args.Franchises), preset)
	}
	if _, ok := peopleDefinitionFor(r.Command.Command); ok && r.Command.Args.People != nil {
		return r.Command.Args.People.label(r.Command.Command)
	}
	if _, ok := noteDefinitionFor(r.Command.Command); ok && r.Command.Args.Note != nil {
		return noteLabel(r)
	}
	return "Reversed a recorded league change"
}

// familyLabel describes an entry of a command family that writes its own labels: draft, waiver, schedule.
func familyLabel(r Record) (string, bool) {
	switch c := r.Command.Command; {
	case draftCommand(c):
		return draftLabel(r), true
	case WaiverCommand(c):
		return waiverLabel(c), true
	case scheduleCommand(c):
		return scheduleLabel(r), true
	case playoffCommand(c):
		return playoffLabel(r), true
	}
	return "", false
}

// scheduleLabel describes a schedule entry for the ledger.
func scheduleLabel(r Record) string {
	switch r.Command.Command {
	case ScheduleGenerate:
		return "Generated a draft league schedule"
	case SchedulePublish:
		return "Published the league schedule"
	default:
		return fmt.Sprintf("Changed the league schedule for week %d", r.Command.Args.Schedule.Week)
	}
}

// undoPlacement says whether this principal may reverse the entry and who may read the reversal.
// It asks the same checks a submitted reversal meets; Go still decides when it is sent.
func undoPlacement(records []Record, i int, p Projection, principal string, auth Authorizer) string {
	r := records[i]
	if r.Command.Command == Create || r.Command.Command == NativeSeed || liveDraftEntry(r.Command.Command) ||
		!p.Active[r.Receipt.Entry] {
		return ""
	}
	if _, err := auth.Allow(r.Command.League, principal, Revert); err != nil {
		return ""
	}
	if rolloverReversalRefusal(records, r.Receipt.Entry) != "" {
		return ""
	}
	if governed, ok := auth.(GovernanceAuthorizer); ok &&
		authorizeReversal(governed, r.Receipt.Entry, principal, records[:i+1]) != nil {
		return ""
	}
	return r.Command.Visibility
}

func ledgerEntryVisible(r Record, records []Record, principal string, privileged bool) bool {
	if privateWaiverRecord(r, records) {
		return (r.Principal == principal && r.Command.Command == WaiverClaim) ||
			(privileged && processedWaiverClaim(r, records))
	}
	return r.Command.Visibility != "private" || privileged
}
