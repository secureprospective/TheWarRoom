// Package commissioner defines the app-owned command boundary and its replayable receipts.
package commissioner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	Create = "commish.league.create"
	Revert = "commish.correction.revert"
	Stream = "commissioner"

	SourceNative    = "native"
	SourceMFLImport = "mfl_import"
)

// Refusal marks a league change the league's own rules turned down, as distinct from a failure to read
// or record it. Each layer wraps the cause with its own context for the log; the owner reads Reason.
type Refusal struct{ err error }

// Refused marks err as a refusal. A nil err stays nil, and an error already marked is returned as is.
func Refused(err error) error {
	var r *Refusal
	if err == nil || errors.As(err, &r) {
		return err
	}
	return &Refusal{err: err}
}

func (r *Refusal) Error() string { return r.err.Error() }
func (r *Refusal) Unwrap() error { return r.err }

// Reason is the innermost message, the sentence that says why, starting with a capital letter.
func (r *Refusal) Reason() string {
	cause := r.err
	for next := errors.Unwrap(cause); next != nil; next = errors.Unwrap(cause) {
		cause = next
	}
	text := cause.Error()
	if text == "" {
		return text
	}
	first, size := utf8.DecodeRuneInString(text)
	return string(unicode.ToUpper(first)) + text[size:]
}

// Session supplies identity independently of IPC arguments and franchise selection.
type Session interface{ Principal() (string, error) }

// Authorizer runs inside the append transaction. Implementations must not write or
// reenter the ledger. Command is its canonical id, not a storage record.
type Authorizer interface {
	Allow(league, principal, command string) (string, error)
}

type Definition struct{ Gravity string }

func DefinitionFor(command string) (Definition, error) {
	switch command {
	case TradeOffer, TradeAccept, TradeDecline, TradeWithdraw, TradeRecuse, DraftStart, DraftPick, TitleSet:
		return Definition{Gravity: "G1"}, nil
	case DraftSkip, DraftAssign, DraftInsert, DraftChange, DraftTimer, DraftRevert,
		DraftStartupOpen, ReserveException, Create, Revert, NativeSeed, SeasonRollover, CalendarException,
		ScoreFranchise, StandingsRecordAdjust, StandingsOverride, StandingsOverrideClear,
		LineupCorrect, BracketAdvance,
		CorrectRoster, CorrectTransaction, CorrectProvenance, CorrectPick, RuleException,
		TradeReview, TradeAppeal, TradeAppealRule, TradeOverride:
		return Definition{Gravity: "G3"}, nil
	case DraftArchive, DraftOrderAuto, DraftOrderSet, DraftOrderPattern, DraftOrderTiebreak,
		RosterReserve, RosterCut, WaiverClaim, WaiverProcess, WaiverOrderSet, WaiverOrderAuto, WaiverOverride,
		BidPlace, BidRaise, BidWithdraw, BidsProcess, BidsOverride,
		ContractEdit, CapAdjust, CapRelief, WindowSet, WindowMove, WindowRemove, ContractActionSet,
		ScheduleGenerate, ScheduleEdit, SchedulePublish, BracketBuild, BracketEdit, BracketSeed,
		BracketPresetSave, HistoryLink, PlayerPosition, PlayerPositionClear, ScorePlayer, StatApprove,
		StatReject, StatCorrect, LineupSet:
		return Definition{Gravity: "G2"}, nil
	default:
		if d, ok := peopleDefinitionFor(command); ok {
			return Definition{Gravity: d.gravity}, nil
		}
		if d, ok := noteDefinitionFor(command); ok {
			return Definition{Gravity: d.gravity}, nil
		}
		return Definition{}, fmt.Errorf("this commissioner command is not available yet")
	}
}

type RawArgs struct {
	Correction     *RawCorrection      `json:"correction,omitempty"`
	Trade          *RawTradeArgs       `json:"trade,omitempty"`
	TradeOutcome   *TradeState         `json:"tradeOutcome,omitempty"`
	Draft          *RawDraftArgs       `json:"draft,omitempty"`
	Waiver         *RawWaiverArgs      `json:"waiver,omitempty"`
	WaiverOutcome  *WaiverOutcome      `json:"waiverOutcome,omitempty"`
	Bid            *RawBidArgs         `json:"bid,omitempty"`
	BidOutcome     *BidOutcome         `json:"bidOutcome,omitempty"`
	DraftOutcome   *DraftOutcome       `json:"draftOutcome,omitempty"`
	Name           string              `json:"name,omitempty"`
	Entry          string              `json:"entry,omitempty"`
	Franchises     []string            `json:"franchises,omitempty"`
	People         *RawPeople          `json:"people,omitempty"`
	Note           *RawNote            `json:"note,omitempty"`
	Field          string              `json:"field,omitempty"`
	Value          json.RawMessage     `json:"value,omitempty"`
	Source         string              `json:"source,omitempty"`
	Preset         string              `json:"preset,omitempty"`
	FranchiseCount int                 `json:"franchiseCount,omitempty"`
	Seed           []SeedValue         `json:"seed,omitempty"`
	Native         *RawNativeSnapshot  `json:"native,omitempty"`
	Exception      *RawWindowException `json:"exception,omitempty"`
	Schedule       *RawScheduleArgs    `json:"schedule,omitempty"`
	Window         *RawWindowArgs      `json:"window,omitempty"`
	Override       *RawNativeOverride  `json:"override,omitempty"`
	Score          *RawScoreChange     `json:"score,omitempty"`
	Standings      *RawStandingsChange `json:"standings,omitempty"`
	Stat           *RawStatChange      `json:"stat,omitempty"`
	Lineup         *RawLineupChange    `json:"lineup,omitempty"`
	Rollover       *RawRolloverArgs    `json:"rollover,omitempty"`
	Bracket        *RawBracketChange   `json:"bracket,omitempty"`
	Title          *RawTitleChange     `json:"title,omitempty"`
	Link           *RawHistoryLink     `json:"link,omitempty"`
	Season         int                 `json:"season,omitempty"`
	ContractAction *RawContractAction  `json:"contractAction,omitempty"`
}

// validateDetails refuses arguments that belong to another command.
func (a RawArgs) validateDetails(command string) error {
	if a.decodedDetails() || a.ownedDetails(command) || a.foreignRuleArgs(command) {
		return fmt.Errorf("send only the details of this league action")
	}
	return nil
}

// decodedDetails says whether any argument that only a typed decoder may read is present here.
func (a RawArgs) decodedDetails() bool {
	return a.Correction != nil || a.scoringDetails() || a.marketDetails() || a.Schedule != nil || a.Window != nil || a.Override != nil ||
		a.Draft != nil || a.DraftOutcome != nil || a.ContractAction != nil
}

// scoringDetails covers the arguments of the score, standings, statistics, lineup and playoff commands.
func (a RawArgs) scoringDetails() bool {
	return a.Score != nil || a.Standings != nil || a.Stat != nil || a.Lineup != nil || a.Bracket != nil ||
		a.Title != nil || a.Link != nil
}

// marketDetails covers the arguments of the waiver and bid commands, including the decisions the app writes.
func (a RawArgs) marketDetails() bool {
	return a.Waiver != nil || a.WaiverOutcome != nil || a.Bid != nil || a.BidOutcome != nil || a.Trade != nil || a.TradeOutcome != nil
}

// ownedDetails says whether an argument that belongs to one command is sent with another.
func (a RawArgs) ownedDetails(command string) bool {
	_, isNote := noteDefinitionFor(command)
	return (command != SeasonRollover && a.Rollover != nil) || (command != NativeSeed && a.Native != nil) ||
		(a.Note != nil && !isNote) || (command != CalendarException && a.Exception != nil)
}

// validateDecoded checks the commands that decode their own typed argument; handled is false for any other.
func (a RawArgs) validateDecoded(command string) (handled bool, err error) {
	switch {
	case correctionCommand(command):
		_, err = decodeCorrection(a, command)
	case isScoreCommand(command):
		_, err = DecodeScoreChange(a, command)
	case isStandingsCommand(command):
		_, err = DecodeStandingsChange(a, command)
	case isStatCommand(command):
		_, err = DecodeStatChange(a, command)
	case isLineupCommand(command):
		_, err = DecodeLineupChange(a, command)
	case draftCommand(command):
		err = a.validateDraft(command)
	case WaiverCommand(command):
		err = a.validateWaiver(command)
	case TradeCommand(command):
		err = a.validateTrade(command)
	case BidCommand(command):
		err = a.validateBid(command)
	case command == ContractActionSet:
		err = a.validateContractAction()
	case playoffCommand(command):
		err = a.validatePlayoff(command)
	default:
		return false, nil
	}
	return true, err
}

func (a RawArgs) Validate(command string) error {
	if handled, err := a.validateDecoded(command); handled {
		return err
	}
	if scheduleCommand(command) {
		return a.validateSchedule(command)
	}
	if windowCommand(command) {
		return a.validateWindow(command)
	}
	if overrideCommand(command) {
		_, err := decodeOverride(command, a)
		return err
	}
	if err := a.validateDetails(command); err != nil {
		return err
	}
	switch command {
	case SeasonRollover:
		return a.validateRollover()
	case CalendarException:
		return a.validateException()
	case NativeSeed:
		return a.validateNativeSeed()
	case RulesSet:
		return a.validateRule()
	case Create:
		return a.validateCreation()
	case Revert:
		if strings.TrimSpace(a.Entry) == "" || a.Name != "" || a.People != nil {
			return fmt.Errorf("choose the recorded change to reverse")
		}
	default:
		return a.validateOther(command)
	}
	return nil
}

func (a RawArgs) foreignRuleArgs(command string) bool {
	return (command != RulesSet && (a.Field != "" || len(a.Value) != 0)) || (command != Create &&
		(a.Source != "" || a.Preset != "" || a.FranchiseCount != 0 || len(a.Seed) != 0 || len(a.Franchises) != 0 || a.Season != 0))
}

func (a RawArgs) validateOther(command string) error {
	if _, isNote := noteDefinitionFor(command); isNote {
		return a.validateNote()
	}
	return a.validatePeople(command)
}

func (a RawArgs) validatePeople(command string) error {
	if _, ok := peopleDefinitionFor(command); !ok {
		return fmt.Errorf("this commissioner command is not available yet")
	}
	if a.People == nil || a.Name != "" || a.Entry != "" {
		return fmt.Errorf("send the details of this people change")
	}
	return a.People.Validate(command)
}

type RawCommand struct {
	League     string  `json:"league"`
	Command    string  `json:"command"`
	Args       RawArgs `json:"args"`
	Reason     string  `json:"reason"`
	Expected   int64   `json:"expected"`
	Request    string  `json:"request"`
	Effective  string  `json:"effective"`
	Visibility string  `json:"visibility"`
	Gravity    string  `json:"gravity,omitempty"`
}

// Validate is the full check a change must pass before it can be recorded.
func (r RawCommand) Validate() error { return r.validate(true) }

// ValidatePreview is Validate without the typed-reason demand, so a dry run can describe a change
// before the reason is written. MissingReason says what is still owed.
func (r RawCommand) ValidatePreview() error { return r.validate(false) }

const reasonRequired = "give a reason for this change"

// MissingReason is a refusal while a change that needs a typed reason has none, and nil otherwise.
func (r RawCommand) MissingReason() error {
	definition, err := r.EffectiveDefinition()
	if err != nil {
		return err
	}
	if definition.Gravity != "G3" || strings.TrimSpace(r.Reason) != "" {
		return nil
	}
	return Refused(errors.New(reasonRequired))
}

func (r RawCommand) validate(requireReason bool) error {
	definition, err := r.EffectiveDefinition()
	if err != nil {
		return err
	}
	if strings.TrimSpace(r.League) != r.League || !strings.HasPrefix(r.League, "native.") || r.League == "native." ||
		strings.TrimSpace(r.Request) == "" || r.Expected < 0 || r.Expected == math.MaxInt64 {
		return fmt.Errorf("choose an app league and a valid request and league version")
	}
	if r.Gravity != "" && r.Gravity != definition.Gravity {
		return fmt.Errorf("this change requires a recorded reason; its safeguards cannot be lowered")
	}
	if requireReason {
		if err := r.MissingReason(); err != nil {
			return err
		}
	}
	// Placement checks can require the same reason; a dry run still describes the proposed act.
	if !requireReason {
		r = r.withPreviewReason(definition)
	}
	if err := r.validatePlacement(); err != nil {
		return err
	}
	if err := r.Args.Validate(r.Command); err != nil {
		return err
	}
	return r.validateSchedulePlacement()
}

// publicNowRule says why a command that is never private and never waits for a rollover must be
// recorded now for the league; it is empty for every other command.
func publicNowRule(command string) string {
	switch {
	case command == SeasonRollover:
		return "the season rollover takes effect now and is visible to every owner"
	case draftCommand(command):
		return "draft changes take effect now and are visible to every owner"
	case command == NativeSeed:
		return "the starting snapshot is recorded now for every owner"
	case windowCommand(command):
		return "window changes take effect now and are visible to every owner"
	case isStandingsCommand(command):
		return "standings overrides take effect now and are shown to the league"
	case isStatCommand(command):
		return "statistic decisions take effect now and are shown to the league"
	case command == CalendarException:
		return "window exceptions take effect now and are visible to every owner"
	case playoffCommand(command):
		return playoffPlacement(command)
	case command == Create:
		return "a new league is created now and recorded for the league"
	}
	if _, isNote := noteDefinitionFor(command); isNote {
		return "notices and league notes are posted now and shown to every owner"
	}
	return ""
}

func (r RawCommand) validatePlacement() error {
	if special, err := r.nativePlacement(); special {
		return err
	}

	if rule := publicNowRule(r.Command); rule != "" && (r.Visibility != "league" || r.Effective != "now") {
		return errors.New(rule)
	}
	if check, ok := r.ownPlacement(); ok {
		return check()
	}
	if r.Visibility != "league" && r.Visibility != "private" {
		return fmt.Errorf("choose who may read this change")
	}
	if r.Effective != "now" && r.Effective != "next_rollover" {
		at, err := time.Parse(time.RFC3339Nano, r.Effective)
		if err != nil || at.IsZero() {
			return fmt.Errorf("choose when this change takes effect")
		}
	}
	return r.validatePeoplePlacement()
}

// reasonByPlacement says whether a command's own placement refuses a blank reason whatever its gravity, so
// the person confirming it is asked for one from the start rather than told it is optional.
func reasonByPlacement(command string) bool {
	return correctionCommand(command) || overrideCommand(command) || isScoreCommand(command) ||
		command == LineupCorrect
}

// ownPlacement is the placement check of a command family that sets its own timing and audience.
func (r RawCommand) ownPlacement() (func() error, bool) {
	switch {
	case correctionCommand(r.Command):
		return r.validateOverridePlacement, true
	case overrideCommand(r.Command):
		return r.validateOverridePlacement, true
	case isScoreCommand(r.Command):
		return r.validateScorePlacement, true
	case isLineupCommand(r.Command):
		return r.validateLineupPlacement, true
	case ruleCommand(r.Command):
		return r.validateRulePlacement, true
	}
	return nil, false
}

func (r RawCommand) validatePeoplePlacement() error {
	if r.Args.People == nil {
		return nil
	}
	event, err := r.Args.People.event(r.Command)
	if err != nil {
		return err
	}
	if r.Effective != "now" {
		return fmt.Errorf("people changes take effect now")
	}
	private := event.Private()
	if d, ok := successionDefinitionFor(r.Command); ok {
		private = d.private
	}
	if private && r.Visibility != "private" {
		return fmt.Errorf("role changes, invitations and ballots must stay private")
	}
	if !private && r.Visibility != "league" {
		return fmt.Errorf("league-state changes must be visible to owners")
	}
	return nil
}

// Decode rejects identity/role fields and unknown arguments even for direct IPC calls.
func Decode(body string) (RawCommand, error) { return decode(body, true) }

// DecodePreview reads a command for a dry run: every check of Decode except the typed reason.
func DecodePreview(body string) (RawCommand, error) { return decode(body, false) }

func decode(body string, requireReason bool) (RawCommand, error) {
	var raw RawCommand
	decoder := json.NewDecoder(bytes.NewBufferString(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("read commissioner command: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return raw, fmt.Errorf("send exactly one commissioner command")
	}
	if raw.Args.WaiverOutcome != nil || raw.Args.BidOutcome != nil || raw.Args.TradeOutcome != nil {
		return raw, fmt.Errorf("waiver and bid decisions are made by the app")
	}
	if raw.Args.DraftOutcome != nil {
		return raw, fmt.Errorf("draft contracts and releases are fixed by the app")
	}
	raw, err := raw.Prepared()
	if err != nil {
		return raw, err
	}
	if err := raw.validate(requireReason); err != nil {
		return raw, err
	}
	return raw, nil
}

func (r RawCommand) Canonical(principal string) (string, error) {
	r.Args.WaiverOutcome, r.Args.BidOutcome, r.Args.TradeOutcome, r.Args.DraftOutcome = nil, nil, nil, nil
	definition, err := r.EffectiveDefinition()
	if err != nil {
		return "", err
	}
	r.Gravity = definition.Gravity
	if r.Effective != "now" && r.Effective != "next_rollover" {
		at, err := time.Parse(time.RFC3339Nano, r.Effective)
		if err != nil {
			return "", fmt.Errorf("canonical effective time: %w", err)
		}
		r.Effective = at.UTC().Format(time.RFC3339Nano)
	}
	body, err := json.Marshal(struct {
		Command   RawCommand `json:"command"`
		Principal string     `json:"principal"`
	}{r, principal})
	if err != nil {
		return "", fmt.Errorf("encode commissioner command: %w", err)
	}
	return string(body), nil
}

func (r RawCommand) EffectiveDefinition() (Definition, error) {
	definition, err := r.Definition()
	if err != nil {
		return Definition{}, err
	}
	if r.Args.Draft != nil && r.Args.Draft.Running {
		definition.Gravity = "G3"
	}
	if r.Command == ScheduleEdit && r.Args.Schedule != nil && r.Args.Schedule.Published {
		definition.Gravity = "G3"
	}
	if changesAfterKickoff(r.Command) && r.Args.Bracket != nil && r.Args.Bracket.Started {
		definition.Gravity = "G3"
	}
	return definition, nil
}

func (r RawCommand) withPreviewReason(definition Definition) RawCommand {
	if definition.Gravity == "G3" && strings.TrimSpace(r.Reason) == "" {
		r.Reason = "Awaiting your reason"
	}
	return r
}
