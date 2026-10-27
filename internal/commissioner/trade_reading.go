package commissioner

import (
	"fmt"
	"time"
)

// NativeTradeOffer is public to league owners, including every vote, recusal, appeal and override reason.
type NativeTradeOffer struct {
	TradeState
	Expires string `json:"expires"`
}

type NativeTrades struct {
	League   string             `json:"league"`
	Viewer   string             `json:"viewer"`
	Revision int64              `json:"revision"`
	Rules    []RuleLine         `json:"rules"`
	Offers   []NativeTradeOffer `json:"offers"`
	Notes    []string           `json:"notes"`
}

func NativeTradesReading(league, viewer, principal string, records []Record, now time.Time, zone *time.Location) (NativeTrades, error) {
	if err := AuthorizeFranchiseOwner(records, principal, viewer); err != nil {
		return NativeTrades{}, err
	}
	f, err := NewFrame(league, records, now, nil, zone)
	if err != nil {
		return NativeTrades{}, err
	}
	return nativeTradesFrame(f, viewer)
}

func nativeTradesFrame(f Frame, viewer string) (NativeTrades, error) {
	league, records, now, zone := f.League, f.Records, f.At, f.Zone
	assets, state := f.Assets, f.Book
	if err := f.ReadRules(); err != nil {
		return NativeTrades{}, err
	}
	rules := f.Rules
	out := NativeTrades{League: league, Viewer: viewer, Revision: int64(len(records)), Offers: []NativeTradeOffer{}, Notes: []string{},
		Rules: ruleLines(rules, "trades.review", "trades.dotSeats", "trades.decideAt", "trades.futureYears", "trades.rounds",
			"trades.rosterGate", "trades.offerExpiry", "trades.maximumParties")}
	if !assets.Seeded {
		out.Notes = append(out.Notes, "Record the starting rosters and picks before trading")
	}
	if _, err := resolveTradePolicy(rules.Trades); err != nil {
		out.Notes = append(out.Notes, err.Error())
	}
	for _, offer := range state.Trades {
		t := offer.clone()
		if t.Status == "offered" && !now.Before(t.ExpiresAt) {
			t.Status, t.Words = "expired", "This offer expired before every party accepted"
		}
		t.Previous = ""
		t.OfferedAt, t.ExpiresAt = t.OfferedAt.In(zone), t.ExpiresAt.In(zone)
		for _, votes := range [][]TradeVote{t.Votes, t.Recusals, t.Appeals, t.Overrides} {
			for i := range votes {
				votes[i].At = votes[i].At.In(zone)
			}
		}
		out.Offers = append(out.Offers, NativeTradeOffer{TradeState: t, Expires: t.ExpiresAt.Format("Jan 2, 2006 at 3:04 PM MST")})
	}
	return out, nil
}

func tradeLabel(r Record) string {
	labels := map[string]string{TradeOffer: "Offered a trade", TradeAccept: "Accepted a trade", TradeDecline: "Declined a trade",
		TradeWithdraw: "Withdrew a trade offer", TradeReview: "Reviewed a trade", TradeRecuse: "Sat out a trade review",
		TradeAppeal: "Appealed a trade veto", TradeAppealRule: "Ruled on a trade appeal", TradeOverride: "Overrode a trade decision"}
	label := labels[r.Command.Command]
	if r.Command.Reason != "" {
		label = fmt.Sprintf("%s: %s", label, r.Command.Reason)
	}
	return label
}
