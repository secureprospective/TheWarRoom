package commissioner

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	TradeOffer      = "trade.offer"
	TradeAccept     = "trade.accept"
	TradeDecline    = "trade.decline"
	TradeWithdraw   = "trade.withdraw"
	TradeReview     = "trade.review"
	TradeRecuse     = "trade.recuse"
	TradeAppeal     = "trade.appeal"
	TradeAppealRule = "trade.appeal.rule"
	TradeOverride   = "trade.override"
)

// TradeAsset identifies exactly one player or an unspent pick, moving between two parties.
type TradeAsset struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Player   string `json:"player,omitempty"`
	Season   int    `json:"season,omitempty"`
	Round    int    `json:"round,omitempty"`
	Original string `json:"original,omitempty"`
}

type RawTradeArgs struct {
	ID        string       `json:"id,omitempty"`
	Franchise string       `json:"franchise,omitempty"`
	Parties   []string     `json:"parties,omitempty"`
	Assets    []TradeAsset `json:"assets,omitempty"`
	Decision  string       `json:"decision,omitempty"`
}

func TradeCommand(command string) bool {
	return slices.Contains([]string{TradeOffer, TradeAccept, TradeDecline, TradeWithdraw, TradeReview,
		TradeRecuse, TradeAppeal, TradeAppealRule, TradeOverride}, command)
}

func OwnerTradeAction(command string) leaguerules.Action {
	if slices.Contains([]string{TradeOffer, TradeAccept, TradeDecline, TradeWithdraw}, command) {
		return "trade"
	}
	return ""
}

func (a RawArgs) validateTrade(command string) error {
	other := a
	other.Trade, other.TradeOutcome = nil, nil
	if a.Trade == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return fmt.Errorf("send only this trade's details")
	}
	return a.Trade.Validate(command)
}

func tradeNeedsDecision(command string) bool {
	return slices.Contains([]string{TradeReview, TradeAppealRule, TradeOverride}, command)
}

func tradeNeedsFranchise(command string) bool {
	return OwnerTradeAction(command) != "" || command == TradeAppeal
}

func (a RawTradeArgs) Validate(command string) error {
	if command == TradeOffer {
		return a.validateOffer()
	}
	kept := RawTradeArgs{ID: a.ID}
	if tradeNeedsFranchise(command) {
		kept.Franchise = a.Franchise
	}
	if tradeNeedsDecision(command) {
		kept.Decision = a.Decision
	}
	if blank(a.ID) || !reflect.DeepEqual(a, kept) {
		return fmt.Errorf("choose a recorded trade and only this action's details")
	}
	if tradeNeedsFranchise(command) && blank(a.Franchise) {
		return fmt.Errorf("choose your franchise")
	}
	if tradeNeedsDecision(command) && !slices.Contains([]string{"approve", "veto"}, a.Decision) {
		return fmt.Errorf("choose approve or veto")
	}
	return nil
}

func (a RawTradeArgs) validateOffer() error {
	if a.ID != "" || a.Decision != "" || blank(a.Franchise) || len(a.Parties) < 2 || len(a.Assets) == 0 {
		return fmt.Errorf("choose your franchise, the parties and the assets offered")
	}
	seen := map[string]bool{}
	for _, p := range a.Parties {
		if blank(p) || seen[p] {
			return fmt.Errorf("list each trade party once")
		}
		seen[p] = true
	}
	if !seen[a.Franchise] {
		return fmt.Errorf("your franchise must be a party")
	}
	for _, asset := range a.Assets {
		if !seen[asset.From] || !seen[asset.To] || asset.From == asset.To {
			return fmt.Errorf("assets must move between the trade parties")
		}
		if err := asset.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (r RawCommand) validateTradePlacement() error {
	if r.Effective != "now" || r.Visibility != "league" {
		return fmt.Errorf("trade actions take effect now and are visible to every owner")
	}
	return nil
}

func (a TradeAsset) Validate() error {
	if a.Player != "" {
		if _, err := canonicalPlayer(a.Player); err != nil {
			return err
		}
		if a.Season != 0 || a.Round != 0 || a.Original != "" {
			return fmt.Errorf("choose a player or a pick, not both")
		}
	} else if a.Season < 1 || a.Season > 9999 || a.Round < 1 || blank(a.Original) {
		return fmt.Errorf("identify each pick by season, round and original franchise")
	}
	return nil
}
