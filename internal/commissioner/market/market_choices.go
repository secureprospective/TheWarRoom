package market

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func (out *Market) readWaivers(state commissioner.MarketBook, rules leaguerules.WaiversRules,
	v bidViewer, at time.Time) error {
	policy, err := nativecheck.NewWaiverPolicy(rules)
	if err != nil {
		return fmt.Errorf("read commissioner waiver rules: %w", err)
	}
	for _, w := range state.Windows {
		name := v.name(w.Player.ID)
		if w.Processed {
			team := "No team awarded"
			if w.Winner != "" {
				team = "Awarded to " + v.teams.Of(w.Winner).Name
			}
			for _, claim := range w.Claims {
				out.Claims = append(out.Claims, Choice{ID: claim.ID,
					Words: name + " — claim by " + v.teams.Of(claim.Franchise).Name + " — " + team})
			}
			continue
		}
		closes, err := policy.Close(w.CutAt)
		if err != nil {
			return fmt.Errorf("read commissioner waiver close: %w", err)
		}
		status := "Clock running"
		if !at.Before(closes) {
			status = "Ready to process"
		}
		words := fmt.Sprintf("%s — cut by %s — closes %s — %s", name, v.teams.Of(w.Player.Franchise).Name,
			closes.In(v.zone).Format(bidTimeLayout), status)
		out.Windows = append(out.Windows, Choice{ID: w.ID, Words: words})
	}
	return nil
}

func (out *Market) readBids(book commissioner.BidBook, v bidViewer,
	stand leaguerules.Value[time.Duration], at time.Time) {
	for i := range book.Auctions {
		a := &book.Auctions[i]
		name := v.name(a.Player)
		if a.Side != "" {
			name = a.Side + ": " + name
		}
		if a.Decided {
			if book.Latest(a.Side, a.Player) != a {
				continue
			}
			for _, bid := range a.Bids {
				words := fmt.Sprintf("%s — %s — %s", name, v.teams.Of(bid.Franchise).Name,
					nativecheck.ContractWords(bid.Salary, bid.Years))
				out.Bids = append(out.Bids, Choice{ID: bid.ID, Words: words})
			}
			continue
		}
		if len(a.LiveBids()) == 0 {
			continue
		}
		closeAt, words := bidClose(stand, a.StandingAt(), v.zone)
		if !v.open && (closeAt == nil || at.Before(*closeAt)) {
			continue
		}
		if closeAt != nil {
			if at.Before(*closeAt) {
				words = "Clock running — " + words
			} else {
				words = "Ready for a decision"
			}
		}
		if closeAt != nil && !at.Before(*closeAt) {
			words = decidedBidWords(a.LiveBids())
		}

		out.Players = append(out.Players, Choice{ID: a.Player, Words: name + " — " + words})
	}
}

func decidedBidWords(live []commissioner.BidRecord) string {
	lead, ok := commissioner.LeaderOf(live)
	if !ok {
		return "Ready for a decision"
	}
	tied := 0
	for _, b := range live {
		if b.Points == lead.Points {
			tied++
		}
	}
	if tied > 1 {
		return "Tie to settle"
	}
	return "Ready for a decision"
}
