package platform

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

func ValidateMoveURL(target string) error {
	u, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("move URL: %w", err)
	}
	host := strings.ToLower(u.Hostname())
	if u.Scheme != "https" || u.User != nil ||
		(host != "myfantasyleague.com" && !strings.HasSuffix(host, ".myfantasyleague.com")) {
		return fmt.Errorf("move URL: HTTPS MFL host without user info required")
	}
	return nil
}

func MovePredicate(intent string) envelope.Predicate {
	registry := map[string]envelope.Predicate{
		"roster.ir":           envelope.IRPredicate{},
		envelope.CutIntent:    envelope.ReleasePredicate{},
		envelope.BuyoutIntent: envelope.ReleasePredicate{},
		"rfa.tender":          envelope.ContractPredicate{}, "fa.resign": envelope.ContractPredicate{}, "contract.tag": envelope.ContractPredicate{}, "contract.extend": envelope.ContractPredicate{}, "contract.restructure": envelope.ContractPredicate{},
		"roster.taxi":   envelope.TaxiPredicate{},
		"lineup.set":    envelope.LineupPredicate{},
		"trade.accept":  envelope.TradePredicate{},
		"trade.propose": envelope.TradeActPredicate{}, "trade.reject": envelope.TradeActPredicate{}, "trade.revoke": envelope.TradeActPredicate{},
		"bid.submit": envelope.BidPredicate{},
	}
	return registry[intent]
}

func WatchableMove(e envelope.Envelope, listed envelope.State) bool {
	return e.State() == listed && (!envelope.IsContract(e.Receipt().Spec.Intent) || e.State() != envelope.HandedOff)
}

func SkipMoveRefresh(source envelope.Source, sources map[envelope.Source]bool) bool {
	return !sources[source] || (source == envelope.Contracts && sources[envelope.Rosters])
}

func SaveMoveEvidence(before, after envelope.Envelope) bool {
	if before.State() != after.State() {
		return true
	}
	old, next := before.Receipt(), after.Receipt()
	if !envelope.IsTrade(old.Spec.Intent) || old.Spec.Intent == "trade.accept" || len(next.Audit) == 0 {
		return false
	}
	return len(old.Audit) == 0 || old.Audit[len(old.Audit)-1].Note != next.Audit[len(next.Audit)-1].Note
}

func RosterMoveTarget(host string, season int, league, option string) (envelope.Target, []string) {
	if host == "" {
		return envelope.Target{Kind: envelope.Unmapped}, []string{"MFL host not yet known"}
	}
	return envelope.Target{Kind: envelope.Mapped, URL: fmt.Sprintf(
		"https://%s/%d/options?L=%s&O=%s", host, season, url.QueryEscape(league), option)}, nil
}
