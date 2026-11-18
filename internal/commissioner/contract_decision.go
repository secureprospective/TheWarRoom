package commissioner

import (
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const ContractDecisionSet = "contract.decision.set"

// RawContractDecision is one owner-selected, rule-priced action on an expiring native contract.
type RawContractDecision struct {
	Franchise   string       `json:"franchise"`
	Player      string       `json:"player"`
	Action      string       `json:"action"`
	Years       int          `json:"years,omitempty"`
	Level       int          `json:"level,omitempty"`
	MoveCents   domain.Money `json:"moveCents,omitempty"`
	AmountCents domain.Money `json:"amountCents,omitempty"`
}

func (r RawCommand) validateContractDecisionPlacement() error {
	if r.Visibility != "private" || r.Effective != "now" {
		return fmt.Errorf("contract decisions are private owner choices that take effect now")
	}
	return nil
}

func (r RawContractDecision) Validate() error {
	id, err := playerid.New(r.Player)
	if err != nil || id.String() != r.Player || r.Franchise == "" || len(r.Franchise) > 200 {
		return fmt.Errorf("choose a player on your franchise")
	}
	if !slices.Contains([]string{"tag", "extension", "rfa_tender", "resign"}, r.Action) {
		return fmt.Errorf("choose a rule-defined expiring-contract option")
	}
	if r.Years < 0 || r.Years > 99 || r.Level < 0 || r.Level > 99 || r.MoveCents < 0 || r.AmountCents < 0 {
		return fmt.Errorf("choose valid contract terms")
	}
	return nil
}

// ContractDecisionOutcome is fixed by the append transaction, never by the owner's request.
type ContractDecisionOutcome struct {
	Before   NativePlayer        `json:"before"`
	After    NativePlayer        `json:"after"`
	Pricing  nativecheck.Pricing `json:"pricing"`
	Season   int                 `json:"season"`
	Deadline time.Time           `json:"deadline"`
}

func (a RawArgs) validateContractDecision() error {
	other := a
	other.ContractDecision, other.ContractDecisionOutcome = nil, nil
	if a.ContractDecision == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return fmt.Errorf("send only the player's contract decision")
	}
	if err := a.ContractDecision.Validate(); err != nil {
		return err
	}
	if a.ContractDecisionOutcome != nil {
		if err := a.ContractDecisionOutcome.Validate(); err != nil {
			return err
		}
		if a.ContractDecisionOutcome.Before.ID != a.ContractDecision.Player ||
			a.ContractDecisionOutcome.Before.Franchise != a.ContractDecision.Franchise ||
			a.ContractDecisionOutcome.After.ID != a.ContractDecision.Player ||
			a.ContractDecisionOutcome.After.Franchise != a.ContractDecision.Franchise {
			return fmt.Errorf("the recorded contract result does not match its player")
		}
	}
	return nil
}

func (r ContractDecisionOutcome) Validate() error {
	if r.Season < 1900 || r.Season > 9999 || r.Deadline.IsZero() || r.Before.ID == "" || r.Before.Franchise == "" ||
		r.After.ID != r.Before.ID || r.After.Franchise != r.Before.Franchise ||
		r.Pricing.Refusal != "" || len(r.Pricing.Needs) != 0 || r.Pricing.Removed ||
		r.Pricing.Salary == nil || len(r.Pricing.Schedule) == 0 ||
		r.After.ExpirationYear != r.Pricing.Schedule[len(r.Pricing.Schedule)-1].Season {
		return fmt.Errorf("a recorded contract decision has no checked result")
	}
	if err := r.Before.validateContract(); err != nil {
		return err
	}
	return r.After.validateContract()
}

// LatestContractDecision finds the effective choice for one player and season. Replacing a choice is
// another ledger event; replay projects only the latest active event in that decision series.
func LatestContractDecision(records []Record, franchise, player string, season int) (Record, bool) {
	active := Replay(records).Active
	var latest Record
	found := false
	for _, record := range records {
		decision := record.Command.Args.ContractDecision
		if record.Command.Command != ContractDecisionSet || decision == nil ||
			!active[record.Receipt.Entry] || decision.Franchise != franchise || decision.Player != player ||
			record.Command.Args.ContractDecisionOutcome == nil ||
			record.Command.Args.ContractDecisionOutcome.Season != season {
			continue
		}
		latest, found = record, true
	}
	return latest, found
}

// ContractDecisionBaseline recovers the contract before the current decision series, if one exists.
func ContractDecisionBaseline(records []Record, franchise, player string, season int) (NativePlayer, bool) {
	latest, ok := LatestContractDecision(records, franchise, player, season)
	if !ok || latest.Command.Args.ContractDecisionOutcome == nil {
		return NativePlayer{}, false
	}
	return latest.Command.Args.ContractDecisionOutcome.Before, true
}
