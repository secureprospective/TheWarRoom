package commissioner

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	RosterCut       = "roster.cut"
	WaiverClaim     = "waivers.claim"
	WaiverProcess   = "waivers.process"
	WaiverOrderSet  = "waivers.order.set"
	WaiverOrderAuto = "waivers.order.auto"
	WaiverOverride  = "waivers.award.override"
)

// RawWaiverArgs names recorded native assets, never prices or an owner identity.
type RawWaiverArgs struct {
	Player    string   `json:"player,omitempty"`
	Franchise string   `json:"franchise,omitempty"`
	Window    string   `json:"window,omitempty"`
	Claim     string   `json:"claim,omitempty"`
	Order     []string `json:"order,omitempty"`
}

// WaiverOutcome is computed under the append lock; IPC cannot supply a decision.
type WaiverClaimResult struct {
	ID      string `json:"id"`
	Refusal string `json:"refusal,omitempty"`
}

type WaiverOutcome struct {
	Claims     []WaiverClaimResult `json:"claims,omitempty"`
	PriorOrder []string            `json:"priorOrder,omitempty"`
	CloseAt    time.Time           `json:"closeAt,omitempty"`
	Player     *NativePlayer       `json:"player,omitempty"`
	Dead       domain.Money        `json:"deadCents,omitempty"`
	Season     int                 `json:"season"`
	Winner     string              `json:"winner,omitempty"`
	Order      []string            `json:"order,omitempty"`
	Source     string              `json:"source,omitempty"`
	ClearsDead bool                `json:"clearsDead"`
}

func WaiverCommand(command string) bool {
	return slices.Contains([]string{RosterCut, WaiverClaim, WaiverProcess,
		WaiverOrderSet, WaiverOrderAuto, WaiverOverride}, command)
}

func OwnerWaiverAction(command string) leaguerules.Action {
	switch command {
	case RosterCut:
		return "cut"
	case WaiverClaim:
		return "ufa_bid"
	}
	return ""
}

func (a RawArgs) validateWaiver(command string) error {
	other := a
	other.Waiver, other.WaiverOutcome = nil, nil
	if a.Waiver == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return fmt.Errorf("send only the details of this waiver action")
	}
	return a.Waiver.Validate(command)
}

func (r RawWaiverArgs) Validate(command string) error {
	switch command {
	case RosterCut:
		id, err := playerid.New(r.Player)
		if err != nil || id.String() != r.Player || strings.TrimSpace(r.Franchise) == "" {
			return fmt.Errorf("choose a player on your roster")
		}
	case WaiverClaim:
		if r.Window == "" || r.Franchise == "" {
			return fmt.Errorf("choose a waiver player and your franchise")
		}
	case WaiverProcess:
		if r.Window == "" {
			return fmt.Errorf("choose a waiver window to process")
		}
	case WaiverOrderSet:
		if len(r.Order) == 0 {
			return fmt.Errorf("list every franchise in priority order")
		}
	case WaiverOverride:
		if r.Claim == "" || r.Franchise == "" {
			return fmt.Errorf("choose a recorded claim and its winning franchise")
		}
	}
	return r.validateWaiverShape(command)
}

func (r RawWaiverArgs) validateWaiverShape(command string) error {
	expected := RawWaiverArgs{}
	switch command {
	case RosterCut:
		expected.Player, expected.Franchise = r.Player, r.Franchise
	case WaiverClaim:
		expected.Window, expected.Franchise = r.Window, r.Franchise
	case WaiverProcess:
		expected.Window = r.Window
	case WaiverOrderSet:
		expected.Order = r.Order
	case WaiverOverride:
		expected.Claim, expected.Franchise = r.Claim, r.Franchise
	}
	if !reflect.DeepEqual(r, expected) {
		return fmt.Errorf("send only the details of this waiver action")
	}
	return nil
}

func (r RawCommand) validateWaiverPlacement() error {
	visibility := "league"
	if r.Command == WaiverClaim {
		visibility = "private"
	}
	if r.Effective != "now" || r.Visibility != visibility {
		return fmt.Errorf("waiver actions take effect now; claims stay private and other changes are public")
	}
	return nil
}

func AuthorizeWaiverOwner(records []Record, principal, franchise string) error {
	state, err := GovernanceState(records)
	if err != nil {
		return err
	}
	seat, ok := state.Franchise(governance.FranchiseID(franchise))
	if !ok || seat.Seat == "" || seat.Seat != governance.Principal(principal) {
		return fmt.Errorf("you may act only for your own franchise")
	}
	return nil
}

func waiverLabel(command string) string {
	switch command {
	case RosterCut:
		return "Released a roster player to waivers"
	case WaiverClaim:
		return "Recorded a private waiver claim"
	case WaiverProcess:
		return "Processed a closed waiver window"
	case WaiverOrderSet:
		return "Set the waiver priority by hand"
	case WaiverOrderAuto:
		return "Restored waiver priority from recorded standings"
	case WaiverOverride:
		return "Corrected a waiver award"
	}
	return "Recorded a waiver change"
}

func (r RawCommand) nativePlacement() (bool, error) {
	if WaiverCommand(r.Command) {
		return true, r.validateWaiverPlacement()
	}
	if overrideCommand(r.Command) {
		return true, r.validateOverridePlacement()
	}
	return false, nil
}
