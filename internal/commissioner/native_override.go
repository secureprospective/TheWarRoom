package commissioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	ContractEdit     = "commish.contract.edit"
	CapAdjust        = "commish.cap.adjust"
	CapRelief        = "commish.cap.relief"
	ReserveException = "commish.reserve.exception"
)

// RawNativeOverride is the typed value of a native correction. Money is exact cents.
type RawNativeOverride struct {
	Player    string        `json:"player,omitempty"`
	Franchise string        `json:"franchise,omitempty"`
	Years     *int          `json:"years,omitempty"`
	Salary    *domain.Money `json:"salaryCents,omitempty"`
	Amount    *domain.Money `json:"amountCents,omitempty"`
	Season    int           `json:"season,omitempty"`
	Bucket    string        `json:"bucket,omitempty"`
	Until     string        `json:"until,omitempty"`
}

func decodeOverride(command string, body json.RawMessage) (RawNativeOverride, error) {
	var raw RawNativeOverride
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("read native correction: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return raw, fmt.Errorf("send exactly one native correction")
	}
	return raw, raw.Validate(command)
}

func (r RawNativeOverride) Validate(command string) error {
	if err := r.validateTarget(command); err != nil {
		return err
	}
	if command != ContractEdit && (r.Years != nil || r.Salary != nil) {
		return fmt.Errorf("contract details belong only to a contract correction")
	}
	if command != ReserveException && (r.Bucket != "" || r.Until != "") {
		return fmt.Errorf("reserve details belong only to a reserve exception")
	}
	return r.validateChange(command)
}

func (r RawNativeOverride) validateTarget(command string) error {
	if command == CapAdjust {
		if r.Player != "" || strings.TrimSpace(r.Franchise) == "" || r.Season < 1 || r.Season > 9999 {
			return fmt.Errorf("choose a franchise and season for the salary adjustment")
		}
		return nil
	}
	id, err := playerid.New(r.Player)
	if err != nil || id.String() != r.Player || r.Franchise != "" || r.Season != 0 {
		return fmt.Errorf("choose a recorded player for this correction")
	}
	return nil
}

func (r RawNativeOverride) validateChange(command string) error {
	switch command {
	case ContractEdit:
		if r.Years == nil || *r.Years < 1 || *r.Years > 9999 || r.Salary == nil || *r.Salary < 0 || r.Amount != nil {
			return fmt.Errorf("give a positive contract length and a nonnegative salary in cents")
		}
	case CapAdjust:
		if r.Amount == nil {
			return fmt.Errorf("give the signed salary adjustment in cents")
		}
	case CapRelief:
		if r.Amount == nil || *r.Amount < 0 {
			return fmt.Errorf("give nonnegative salary relief in cents")
		}
	case ReserveException:
		return r.validateException()
	default:
		return fmt.Errorf("choose a native correction")
	}
	return nil
}

func (r RawNativeOverride) validateException() error {
	if r.Amount != nil || (r.Bucket != "ir" && r.Bucket != "taxi") {
		return fmt.Errorf("choose IR or taxi for this exception")
	}
	if r.Until == "" {
		return nil
	}
	at, err := time.Parse(time.RFC3339Nano, r.Until)
	if err != nil || at.IsZero() {
		return fmt.Errorf("choose when the reserve exception expires")
	}
	return nil
}

func (a RawArgs) validateOverride(command string) error {
	if a.Name != "" || a.Entry != "" || a.People != nil || a.Field != "" || a.Preset != "" ||
		a.FranchiseCount != 0 || len(a.Seed) != 0 || len(a.Franchises) != 0 || a.Native != nil {
		return fmt.Errorf("send only the native correction as its typed value")
	}
	_, err := decodeOverride(command, a.Value)
	return err
}

func (r RawCommand) validateOverridePlacement() error {
	if r.Effective != "now" || r.Visibility != "league" || strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("native corrections take effect now, with a reason visible to every owner")
	}
	return nil
}

// PendingTradeCapCheck is the escalation seam for native trade review. No trade ledger exists yet.
func PendingTradeCapCheck(_ RawCommand, _ []Record) (bool, string) {
	return false, "No pending trades"
}

func CheckOverrideAppend(raw RawCommand, records []Record, now time.Time) error {
	switch raw.Command {
	case ContractEdit, CapAdjust, CapRelief, ReserveException:
		if changed, _ := PendingTradeCapCheck(raw, records); changed {
			return Refused(fmt.Errorf("this correction changes a pending trade; trade review is required"))
		}
		if err := checkOverrideTarget(raw, records, now); err != nil {
			return Refused(err)
		}
	case Revert:
		trial := append(append([]Record{}, records...), Record{Command: raw,
			Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: now}})
		if _, err := NativeAssetsReading(raw.League, trial); err != nil {
			return Refused(err)
		}
	}
	return nil
}

func checkOverrideTarget(raw RawCommand, records []Record, now time.Time) error {
	assets, err := NativeAssetsReading(raw.League, records)
	if err != nil {
		return err
	}
	if !assets.Seeded {
		return fmt.Errorf("record the starting assets before correcting them")
	}
	value, err := decodeOverride(raw.Command, raw.Args.Value)
	if err != nil {
		return err
	}
	if value.Until != "" {
		until, parseErr := time.Parse(time.RFC3339Nano, value.Until)
		if parseErr != nil {
			return fmt.Errorf("read exception expiry: %w", parseErr)
		}
		if !now.Before(until) {
			return fmt.Errorf("the exception must expire after now")
		}
	}
	trial := append(append([]Record{}, records...), Record{Command: raw,
		Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: now}})
	_, err = NativeAssetsReading(raw.League, trial)
	return err
}

func overrideLabel(r Record) string {
	switch r.Command.Command {
	case ContractEdit:
		return "Corrected a player's contract"
	case CapAdjust:
		return "Recorded a franchise salary adjustment"
	case CapRelief:
		return "Granted salary relief for a player"
	case ReserveException:
		return "Granted a player reserve exception"
	}
	return "Corrected recorded league assets"
}
