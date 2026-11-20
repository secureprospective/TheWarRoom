package commissioner

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	CorrectRoster      = "correct.roster"
	CorrectTransaction = "correct.transaction"
	CorrectProvenance  = "correct.provenance"
	CorrectPick        = "correct.pick"
	RuleException      = "rule.exception"
)

// CorrectionContract carries explicit contract terms for a roster correction.
type CorrectionContract struct {
	ContractMissing    bool                       `json:"contractMissing,omitempty"`
	Salary             domain.Money               `json:"salaryCents,omitempty"`
	ExpirationYear     int                        `json:"expirationYear,omitempty"`
	ContractStatus     domain.ContractStatus      `json:"contractStatus,omitempty"`
	ContractInfo       string                     `json:"contractInfo,omitempty"`
	HeldContractStatus string                     `json:"heldContractStatus,omitempty"`
	ContractSchedule   []nativecheck.SeasonAmount `json:"contractSchedule,omitempty"`
}

func (c CorrectionContract) Validate() error {
	if c.Salary < 0 || c.ExpirationYear < 0 || c.ExpirationYear > lastSeason {
		return fmt.Errorf("a corrected contract needs nonnegative salary and a valid expiration year")
	}
	return (NativePlayer{
		ContractMissing:    c.ContractMissing,
		Salary:             c.Salary,
		ExpirationYear:     c.ExpirationYear,
		ContractStatus:     c.ContractStatus,
		ContractInfo:       c.ContractInfo,
		HeldContractStatus: c.HeldContractStatus,
		ContractSchedule:   c.ContractSchedule,
	}).validateContract()
}

// RawCorrection names one asset or one automated decision; unrecorded contracts stay missing.
type RawCorrection struct {
	Franchise string              `json:"franchise,omitempty"`
	Player    string              `json:"player,omitempty"`
	Action    string              `json:"action,omitempty"`
	Status    domain.RosterStatus `json:"status,omitempty"`
	Contract  *CorrectionContract `json:"contract,omitempty"`
	Charge    *NativeCharge       `json:"charge,omitempty"`
	Entry     string              `json:"entry,omitempty"`
	Via       string              `json:"via,omitempty"`
	History   *PickEvidence       `json:"history,omitempty"`
	Pick      *CorrectionPick     `json:"pick,omitempty"`
	Owner     string              `json:"owner,omitempty"`
	Rule      string              `json:"rule,omitempty"`
	Scope     *ExceptionScope     `json:"scope,omitempty"`
	Until     string              `json:"until,omitempty"`
}

type CorrectionPick struct {
	Slot     int    `json:"slot,omitempty"`
	Season   int    `json:"season"`
	Round    int    `json:"round"`
	Original string `json:"original"`
	Player   string `json:"player,omitempty"`
	Skipped  bool   `json:"skipped,omitempty"`
	Add      bool   `json:"add,omitempty"`
	Clear    bool   `json:"clear,omitempty"`
	Remove   bool   `json:"remove,omitempty"`
}

type ExceptionScope struct {
	Franchise   string `json:"franchise,omitempty"`
	Player      string `json:"player,omitempty"`
	Transaction string `json:"transaction,omitempty"`
}

func correctionCommand(command string) bool {
	return slices.Contains([]string{
		CorrectRoster, CorrectTransaction, CorrectProvenance, CorrectPick, RuleException,
	}, command)
}

func decodeCorrection(a RawArgs, command string) (RawCorrection, error) {
	rest := a
	rest.Correction = nil
	if a.Correction == nil || !reflect.DeepEqual(rest, RawArgs{}) {
		return RawCorrection{}, fmt.Errorf("send only the details of this correction")
	}
	return *a.Correction, a.Correction.Validate(command)
}

func (r RawCorrection) Validate(command string) error {
	expected := RawCorrection{}
	switch command {
	case CorrectRoster:
		var err error
		expected, err = r.validateRoster()
		if err != nil {
			return err
		}
	case CorrectTransaction:
		expected.Entry = r.Entry
		if blank(r.Entry) {
			return fmt.Errorf("choose the recorded transaction to reverse")
		}
	case CorrectProvenance:
		// The franchise names whose copy is meant; a league with a player pool per side requires it.
		expected.Player, expected.Via, expected.Franchise = r.Player, r.Via, r.Franchise
		if !slices.Contains([]string{"held", "draft", "waiver", "trade", "free_agent", "commissioner"}, r.Via) {
			return fmt.Errorf(
				"choose how the player was acquired: held, draft, waiver, trade, free_agent or commissioner")
		}
	case CorrectPick:
		expected.Pick, expected.Owner, expected.History = r.Pick, r.Owner, r.History
		if err := r.validatePick(); err != nil {
			return err
		}

	case RuleException:
		expected.Rule, expected.Scope, expected.Until = r.Rule, r.Scope, r.Until
		if err := r.validateRuleException(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("choose an asset correction or rule exception")
	}
	if !reflect.DeepEqual(r, expected) {
		return fmt.Errorf("send only the details this correction needs")
	}
	return r.validatePlayer(command)
}

func (r RawCorrection) validateRoster() (RawCorrection, error) {
	if !strings.HasPrefix(r.Franchise, "franchise.") {
		return RawCorrection{}, fmt.Errorf("choose a franchise and whether to add or remove the player")
	}
	expected := RawCorrection{Franchise: r.Franchise, Action: r.Action}
	switch r.Action {
	case "add":
		expected.Player = r.Player
		if r.Contract != nil {
			if err := r.Contract.Validate(); err != nil {
				return RawCorrection{}, err
			}
			expected.Contract = r.Contract
		}
	case "remove":
		expected.Player = r.Player
	case "status":
		expected.Player, expected.Status = r.Player, r.Status
		statuses := []domain.RosterStatus{domain.RosterActive, domain.RosterIR, domain.RosterTaxi}
		if !slices.Contains(statuses, r.Status) {
			return RawCorrection{}, fmt.Errorf("choose active roster, injured reserve or taxi")
		}
	case "contract":
		expected.Player, expected.Contract = r.Player, r.Contract
		if r.Contract == nil {
			return RawCorrection{}, fmt.Errorf("supply the corrected contract details")
		}
		if err := r.Contract.Validate(); err != nil {
			return RawCorrection{}, err
		}
	case "adjustment", "remove_adjustment":
		if err := r.validateCharge(); err != nil {
			return RawCorrection{}, err
		}
		if r.Action == "remove_adjustment" {
			expected.Charge = &NativeCharge{ID: r.Charge.ID, Franchise: r.Franchise}
		} else {
			expected.Charge = r.Charge
		}
	default:
		return RawCorrection{}, fmt.Errorf("choose a franchise and whether to add or remove the player")
	}
	return expected, nil
}

func (r RawCorrection) validateCharge() error {
	if r.Charge == nil || r.Charge.ID == "" || r.Charge.Franchise != r.Franchise {
		return fmt.Errorf("supply a salary adjustment with an id matching the franchise")
	}
	if r.Action == "remove_adjustment" {
		return nil
	}
	badKind := r.Charge.Kind != "" && r.Charge.Kind != deadMoney
	if badKind || r.Charge.Season < 0 || r.Charge.Season > lastSeason {
		return fmt.Errorf("a corrected salary adjustment has an invalid kind or season")
	}
	return nil
}

func (r RawCorrection) validatePlayer(command string) error {
	if command != CorrectRoster && command != CorrectProvenance {
		return nil
	}
	if command == CorrectRoster && (r.Action == "adjustment" || r.Action == "remove_adjustment") {
		return nil
	}
	id, err := playerid.New(r.Player)
	if err != nil || id.String() != r.Player {
		return fmt.Errorf("choose a recorded player")
	}
	return nil
}

func (r RawCorrection) validateRuleException() error {
	if r.Scope == nil {
		return fmt.Errorf("choose a franchise, player or transaction scope")
	}
	count := 0
	for _, id := range []string{r.Scope.Franchise, r.Scope.Player, r.Scope.Transaction} {
		if id != "" {
			count++
		}
	}
	if strings.HasPrefix(r.Rule, "calendar.") && r.Scope.Franchise == "" {
		return fmt.Errorf("a calendar exception needs a franchise scope")
	}
	if count != 1 && (count != 2 || r.Scope.Franchise == "" || r.Scope.Player == "" ||
		strings.HasPrefix(r.Rule, "calendar.")) {
		return fmt.Errorf("choose exactly one exception scope, or a franchise and player together")
	}
	if !supportedExceptionRule(r.Rule) {
		return fmt.Errorf("choose an automated roster, salary-cap or calendar rule")
	}
	if r.Until != "" {
		until, err := time.Parse(time.RFC3339Nano, r.Until)
		if err != nil || until.IsZero() {
			return fmt.Errorf("choose the exact instant the exception expires")
		}
	}
	return nil
}

func CheckCorrectionAppend(raw RawCommand, records []Record, now time.Time) error {
	if !correctionCommand(raw.Command) {
		return nil
	}
	c, err := decodeCorrection(raw.Args, raw.Command)
	if err != nil {
		return Refused(err)
	}
	if raw.Command == RuleException {
		return Refused(checkExceptionTarget(c, records, now))
	}
	if raw.Command == CorrectTransaction {
		if why := rolloverReversalRefusal(records, c.Entry); why != "" {
			return Refused(fmt.Errorf("%s", why))
		}
		if err := reversibleEntry(raw.League, records, c.Entry); err != nil {
			return Refused(err)
		}
	}
	trial := append(slices.Clone(records), Record{Command: raw,
		Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: now}})
	_, err = NativeAssetsReading(raw.League, trial)
	return Refused(err)
}

func checkExceptionTarget(c RawCorrection, records []Record, now time.Time) error {
	if c.Until != "" {
		until, err := time.Parse(time.RFC3339Nano, c.Until)
		if err != nil {
			return fmt.Errorf("read exception expiry: %w", err)
		}
		if !now.Before(until) {
			return fmt.Errorf("the exception must expire after now")
		}
	}
	assets, err := NativeAssetsReading(records[0].Command.League, records)
	if err != nil {
		return err
	}
	if !assets.Seeded {
		return fmt.Errorf("record the starting assets before granting an exception")
	}
	return checkExceptionScope(c, records, assets.Snapshot)
}

func checkExceptionScope(c RawCorrection, records []Record, snapshot *RawNativeSnapshot) error {
	switch {
	case c.Scope.Player != "" && c.Scope.Franchise != "":
		if snapshot.PlayerOf(c.Scope.Franchise, c.Scope.Player) < 0 {
			return fmt.Errorf("choose a player on that franchise's recorded roster")
		}
	case c.Scope.Franchise != "":
		hasFranchise := slices.ContainsFunc(snapshot.Franchises, func(f NativeFranchise) bool {
			return f.ID == c.Scope.Franchise
		})
		if !hasFranchise {
			return fmt.Errorf("choose a franchise in this app league")
		}
	case c.Scope.Player != "":
		side, err := snapshot.TargetSide(c.Scope.Franchise)
		if err != nil {
			return err
		}
		if !snapshot.OnSide(side, c.Scope.Player) {
			return fmt.Errorf("choose a recorded roster player")
		}
	case c.Scope.Transaction != "":
		if !slices.ContainsFunc(records, func(r Record) bool { return r.Receipt.Entry == c.Scope.Transaction }) {
			return fmt.Errorf("choose a recorded transaction")
		}
	}
	return nil
}

func (r RawCorrection) validatePick() error {
	if err := r.validatePickIdentity(); err != nil {
		return err
	}
	if r.History != nil {
		return r.validatePickHistory()
	}
	return r.Pick.validateMutation()
}

func (r RawCorrection) validatePickIdentity() error {
	if r.Pick == nil || r.Pick.Season < 1920 || r.Pick.Season > 9999 ||
		r.Pick.Round < 1 || r.Pick.Round > 100 || !strings.HasPrefix(r.Owner, "franchise.") {
		return fmt.Errorf("choose the draft season, round, pick and corrected owner")
	}
	if r.Pick.Original == "" {
		if r.Pick.Slot < 1 || r.Pick.Slot > 1000 {
			return fmt.Errorf("choose the current draft slot")
		}
		return nil
	}
	if r.Pick.Slot != 0 || !strings.HasPrefix(r.Pick.Original, "franchise.") {
		return fmt.Errorf("choose an original franchise or a current draft slot, not both")
	}
	return nil
}

func (r RawCorrection) validatePickHistory() error {
	if err := r.History.Validate(); err != nil {
		return err
	}
	if r.Pick.hasMutation() {
		return fmt.Errorf("historical pick ownership cannot carry selection or pick changes")
	}
	return nil
}

func (p CorrectionPick) hasMutation() bool {
	return p.Player != "" || p.Skipped || p.Add || p.Clear || p.Remove
}

func (p CorrectionPick) hasSelection() bool {
	return p.Player != "" || p.Skipped
}

func (p CorrectionPick) validateMutation() error {
	if p.Remove && (p.Add || p.Clear || p.hasSelection()) {
		return fmt.Errorf("a removed pick cannot also carry selection or add flags")
	}
	if p.Clear && (p.Add || p.hasSelection()) {
		return fmt.Errorf("a cleared pick cannot also carry a selection or add flag")
	}
	return p.validateSelection()
}

func (p CorrectionPick) validateSelection() error {
	if p.Player != "" && p.Skipped {
		return fmt.Errorf("a draft pick cannot be both selected and skipped")
	}
	if (p.hasSelection() || p.Clear) && p.Original != "" {
		return fmt.Errorf("only a current draft slot can carry a selection")
	}
	if p.Player != "" {
		id, err := playerid.New(p.Player)
		if err != nil || id.String() != p.Player {
			return fmt.Errorf("choose a valid selected player")
		}
	}
	return nil
}

// reversibleEntry is the rule for which recorded entry a commissioner may reverse: an active cut, add, waiver
// award or draft pick, or an owner's free-agent bid that is still standing. Owners cannot withdraw a bid
// (a bid is a bid); a mistaken one is removed only by this recorded correction.
func reversibleEntry(league string, records []Record, entry string) error {
	index := slices.IndexFunc(records, func(r Record) bool { return r.Receipt.Entry == entry })
	if index < 0 || !Replay(records).Active[entry] {
		return fmt.Errorf("choose one active recorded cut, add, waiver award, draft pick or standing bid")
	}
	switch records[index].Command.Command {
	case RosterCut, RosterAdd, WaiverProcess, WaiverOverride, DraftPick:
		return nil
	case BidPlace, BidRaise:
		_, market, err := nativeAssetsState(league, records)
		if err != nil {
			return err
		}
		a, j, err := market.Bids.Find(entry)
		if err != nil || a.Decided || !a.Bids[j].Live() {
			return fmt.Errorf("only a bid that is still standing can be reversed; correct a decided award instead")
		}
		return nil
	}
	return fmt.Errorf("choose one active recorded cut, add, waiver award, draft pick or standing bid")
}
