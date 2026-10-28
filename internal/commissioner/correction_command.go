package commissioner

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	CorrectRoster      = "correct.roster"
	CorrectTransaction = "correct.transaction"
	CorrectProvenance  = "correct.provenance"
	CorrectPick        = "correct.pick"
	RuleException      = "rule.exception"
)

// RawCorrection names one asset or one automated decision; contracts are never invented by a correction.
type RawCorrection struct {
	Franchise string          `json:"franchise,omitempty"`
	Player    string          `json:"player,omitempty"`
	Action    string          `json:"action,omitempty"`
	Entry     string          `json:"entry,omitempty"`
	Via       string          `json:"via,omitempty"`
	Pick      *CorrectionPick `json:"pick,omitempty"`
	Owner     string          `json:"owner,omitempty"`
	Rule      string          `json:"rule,omitempty"`
	Scope     *ExceptionScope `json:"scope,omitempty"`
	Until     string          `json:"until,omitempty"`
}

type CorrectionPick struct {
	Season   int    `json:"season"`
	Round    int    `json:"round"`
	Original string `json:"original"`
}

type ExceptionScope struct {
	Franchise   string `json:"franchise,omitempty"`
	Player      string `json:"player,omitempty"`
	Transaction string `json:"transaction,omitempty"`
}

func correctionCommand(command string) bool {
	return slices.Contains([]string{CorrectRoster, CorrectTransaction, CorrectProvenance, CorrectPick, RuleException}, command)
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
		expected.Player, expected.Franchise, expected.Action = r.Player, r.Franchise, r.Action
		if !slices.Contains([]string{"add", "remove"}, r.Action) || !strings.HasPrefix(r.Franchise, "franchise.") {
			return fmt.Errorf("choose a franchise and whether to add or remove the player")
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
			return fmt.Errorf("choose how the player was acquired: held, draft, waiver, trade, free_agent or commissioner")
		}
	case CorrectPick:
		expected.Pick, expected.Owner = r.Pick, r.Owner
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

func (r RawCorrection) validatePlayer(command string) error {
	if command != CorrectRoster && command != CorrectProvenance {
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
		index := slices.IndexFunc(records, func(r Record) bool { return r.Receipt.Entry == c.Entry })
		if index < 0 || !Replay(records).Active[c.Entry] ||
			!slices.Contains([]string{RosterCut, WaiverProcess, WaiverOverride, DraftPick}, records[index].Command.Command) {
			return Refused(fmt.Errorf("choose one active recorded cut, waiver award or draft pick"))
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
		if !slices.ContainsFunc(snapshot.Franchises, func(f NativeFranchise) bool { return f.ID == c.Scope.Franchise }) {
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
	if r.Pick == nil || r.Pick.Season < 1920 || r.Pick.Season > 9999 || r.Pick.Round < 1 ||
		!strings.HasPrefix(r.Pick.Original, "franchise.") || !strings.HasPrefix(r.Owner, "franchise.") {
		return fmt.Errorf("choose the draft season, round, original franchise and corrected owner")
	}
	return nil
}
