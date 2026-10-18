package commissioner

import (
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func recordEvents(r Record) ([]governance.Event, error) {
	if r.Command.Command == Create {
		return creationEvents(r), nil
	}
	if r.Command.Args.People != nil {
		people := r.Command.Args.People
		var event governance.Event
		var err error
		if _, ok := successionDefinitionFor(r.Command.Command); ok {
			event, err = people.successionEvent(r.Command.Command, r.Receipt.RecordedAt)
		} else {
			event, err = people.event(r.Command.Command)
		}
		if err != nil {
			return nil, err
		}
		return []governance.Event{event}, nil
	}
	return nil, nil
}

func GovernanceState(records []Record) (governance.State, error) {
	projection := Replay(records)
	events := []governance.Event{}
	for _, r := range records {
		// Creation remains bootstrap metadata, just as in the ledger's identity projection.
		if !projection.Active[r.Receipt.Entry] && r.Command.Command != Create {
			// A reversed change was still the person's activity when it was recorded.
			events = withActivity(events, r)
			continue
		}
		recorded, err := recordEvents(r)
		if err != nil {
			return governance.State{}, err
		}
		events = append(events, recorded...)
		events = withActivity(events, r)
	}
	state, err := governance.Replay(events)
	if err != nil {
		return governance.State{}, fmt.Errorf("read league people: %w", err)
	}
	return state, nil
}

type GovernanceAuthorizer struct{ State governance.State }

func (a GovernanceAuthorizer) Allow(_ string, principal, command string) (string, error) {
	if command == ContractActionSet && !a.State.HasRole(governance.Principal(principal), governance.Commissioner) {
		return "", fmt.Errorf("only this league's commissioner may set contract actions")
	}
	if d, ok := successionDefinitionFor(command); ok {
		return successionPermission(a.State, principal, d)
	}
	if command == "commish.ledger.read" {
		return ledgerReader(a.State, principal), nil
	}
	control, ok := commandControls()[command]
	if !ok {
		if d, isPeople := peopleDefinitionFor(command); isPeople {
			control = d.control
		}
		if d, ok := noteDefinitionFor(command); ok {
			control = d.control
		}
	}
	row, ok := governance.CensusControl(control)
	if !ok {
		return "", fmt.Errorf("this league action is not available")
	}
	role, err := governance.Allow(a.State, governance.Principal(principal), governance.CommandRef{
		ID: row.ID, MinimumRole: row.MinimumRole, COnly: row.COnly})
	if err != nil {
		return "", fmt.Errorf("league permission: %w", err)
	}
	if role == governance.CoCommissioner {
		return "cocommissioner", nil
	}
	return string(role), nil
}

// commandControls names the census control that governs each commissioner command.
func commandControls() map[string]string {
	return map[string]string{
		DraftStart: "CM-042", DraftOrderAuto: "CM-041", DraftOrderSet: "CM-041",
		DraftOrderPattern: "CM-041", DraftOrderTiebreak: "CM-041",
		DraftSkip: "CM-043", DraftAssign: "CM-043", DraftInsert: "CM-043", DraftChange: "CM-043",
		DraftTimer: "CM-043", DraftRevert: "CM-044", DraftArchive: "CM-045",
		WaiverProcess: "CM-035", WaiverOrderSet: "CM-035", WaiverOrderAuto: "CM-035", WaiverOverride: "CM-035",
		Create: "CM-001", NativeSeed: "CM-001", Revert: "CM-039", ContractActionSet: "CM-009",
		ContractEdit: "CM-034", CapAdjust: "CM-034", CapRelief: "CM-034", ReserveException: "CM-038",
		PlayerPosition: "CM-030", PlayerPositionClear: "CM-030",
		ScorePlayer: "CM-032", ScoreFranchise: "CM-032",
		StatApprove: "CM-031", StatReject: "CM-031", StatCorrect: "CM-031",
		WindowSet: "CM-023", WindowMove: "CM-023", WindowRemove: "CM-023",
		CalendarException: "CM-026", SeasonRollover: "CM-027",
		StandingsRecordAdjust: "CM-033", StandingsOverride: "CM-033", StandingsOverrideClear: "CM-033",
		ScheduleGenerate: "CM-015", ScheduleEdit: "CM-015", SchedulePublish: "CM-015",
		LineupCorrect: "CM-029",
		BracketBuild:  "CM-046", BracketEdit: "CM-046", BracketSeed: "CM-046", BracketPresetSave: "CM-046",
		BracketAdvance: "CM-047", TitleSet: "CM-048", HistoryLink: "CM-049",
	}
}

// ledgerReader says how much of the ledger a principal may read.
func ledgerReader(state governance.State, principal string) string {
	p := governance.Principal(principal)
	switch {
	case state.HasRole(p, governance.Commissioner):
		return "commissioner"
	case state.HasRole(p, governance.CoCommissioner):
		return "cocommissioner"
	}
	return "owner"
}

func AuthorizeRecords(raw RawCommand, principal string, records []Record) (string, error) {
	if len(records) == 0 && raw.Command == Create {
		return "commissioner", nil
	}
	if OwnerWaiverAction(raw.Command) != "" {
		if err := AuthorizeWaiverOwner(records, principal, raw.Args.Waiver.Franchise); err != nil {
			return "", err
		}
		return "owner", nil
	}
	state, err := GovernanceState(records)
	if err != nil {
		return "", err
	}
	if raw.Command == "commish.seat.invite.redeem" {
		return authorizeRedemption(state, raw, principal)
	}
	if _, ok := successionDefinitionFor(raw.Command); ok {
		return authorizeSuccession(state, raw, principal)
	}
	auth := GovernanceAuthorizer{State: state}
	if raw.Command == RulesSet {
		return authorizeRule(auth, raw, principal)
	}
	if raw.Command == LineupSet {
		return authorizeLineupSet(state, raw, principal)
	}
	role, err := auth.Allow(raw.League, principal, raw.Command)
	if err != nil {
		return "", err
	}
	if raw.Command == Revert {
		if err := authorizeReversal(auth, raw.Args.Entry, principal, records); err != nil {
			return "", err
		}
	}
	return role, nil
}

// authorizeLineupSet lets an owner set the lineup of the franchise they hold and no other: a commissioner
// changes someone else's lineup only by the CM-029 correction, which is recorded with a reason.
func authorizeLineupSet(state governance.State, raw RawCommand, principal string) (string, error) {
	change, err := DecodeLineupChange(raw.Args, raw.Command)
	if err != nil {
		return "", err
	}
	seat, ok := state.Franchise(governance.FranchiseID(change.Franchise))
	if !ok || principal == "" || seat.Seat != governance.Principal(principal) {
		return "", fmt.Errorf("you may set a lineup only for your own franchise")
	}
	if !seat.Abilities["lineups"] {
		return "", fmt.Errorf("the commissioner has turned off lineup changes for your franchise")
	}
	return "owner", nil
}

func authorizeRedemption(state governance.State, raw RawCommand, principal string) (string, error) {
	if raw.Args.People.Principal != principal {
		return "", fmt.Errorf("%s", "Other owners act from their own app; this desktop can only act as you")
	}
	invite, ok := state.Invite(raw.Args.People.ID)
	if !ok || invite.Redeemer != governance.Principal(principal) {
		return "", fmt.Errorf("that invite is not available to you")
	}
	return "owner", nil
}

func authorizeReversal(auth GovernanceAuthorizer, entry, principal string, records []Record) error {
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Receipt.Entry != entry {
			continue
		}
		if r.Command.Command == Revert {
			entry = r.Command.Args.Entry
			continue
		}
		var err error
		switch {
		case r.Command.Command == LineupSet || r.Command.Command == DraftPick:
			return nil // reversing needs only the reversal permission itself; the owner's seat is not asked for
		case OwnerWaiverAction(r.Command.Command) != "":
			_, err = auth.Allow(r.Command.League, principal, WaiverProcess)
		case r.Command.Command == RulesSet:
			_, err = authorizeRule(auth, r.Command, principal)
		default:
			_, err = auth.Allow(r.Command.League, principal, r.Command.Command)
		}
		if err != nil {
			return fmt.Errorf("reversing this change needs its original permission: %w", err)
		}
		return nil
	}
	return fmt.Errorf("that recorded change is not in this league")
}

// checkRoleChangesSomething refuses to give a role the person already holds or take one they do not.
// Preview and execute share it, so a change that changes nothing is never offered.
func checkRoleChangesSomething(raw RawCommand, records []Record) error {
	if (raw.Command != roleGrant && raw.Command != roleRevoke) || raw.Args.People == nil {
		return nil
	}
	state, err := GovernanceState(records)
	if err != nil {
		return err
	}
	held := state.HasRole(governance.Principal(raw.Args.People.Principal), raw.Args.People.Role)
	if held && raw.Command == roleGrant {
		return Refused(errors.New("this person already holds that role. Nothing would change"))
	}
	if !held && raw.Command == roleRevoke {
		return Refused(errors.New("this person does not hold that role. Nothing would change"))
	}
	return nil
}

func CheckGovernanceAppend(raw RawCommand, principal string, records []Record, at time.Time) error {
	if err := checkRoleChangesSomething(raw, records); err != nil {
		return err
	}
	switch raw.Command {
	case Revert:
		if err := checkScoreCompensation(raw, records); err != nil {
			return err
		}
	case PlayerPosition, PlayerPositionClear, ScorePlayer, ScoreFranchise:
		return checkScoreAppend(raw, records)
	case StandingsRecordAdjust, StandingsOverride, StandingsOverrideClear:
		return checkStandingsAppend(raw, records)
	}
	trial := make([]Record, len(records), len(records)+1)
	copy(trial, records)
	trial = append(trial, Record{Command: raw, Principal: principal,
		Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: at}})
	if _, err := GovernanceState(trial); err != nil {
		return Refused(fmt.Errorf("people change refused: %w", err))
	}
	if _, ok := successionDefinitionFor(raw.Command); !ok {
		return nil
	}
	before, err := GovernanceState(records)
	if err != nil {
		return err
	}
	if err := takeoverWaitedLongest(before, raw, at); err != nil {
		return Refused(fmt.Errorf("people change refused: %w", err))
	}
	return nil
}
