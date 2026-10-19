package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func (out *Market) readWaivers(state waiverState, rules leaguerules.WaiversRules,
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
				team = "Awarded to " + v.teams.of(w.Winner).Name
			}
			for _, claim := range w.Claims {
				out.Claims = append(out.Claims, MarketChoice{ID: claim.ID,
					Words: name + " — claim by " + v.teams.of(claim.Franchise).Name + " — " + team})
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
		words := fmt.Sprintf("%s — cut by %s — closes %s — %s", name, v.teams.of(w.Player.Franchise).Name,
			closes.In(v.zone).Format(bidTimeLayout), status)
		out.Windows = append(out.Windows, MarketChoice{ID: w.ID, Words: words})
	}
	return nil
}

func (out *Market) readBids(book bidBook, v bidViewer,
	stand leaguerules.Value[time.Duration], at time.Time) {
	for i := range book.Auctions {
		a := &book.Auctions[i]
		name := v.name(a.Player)
		if a.Decided {
			if book.latest(a.Player) != a {
				continue
			}
			for _, bid := range a.Bids {
				words := fmt.Sprintf("%s — %s — %s", name, v.teams.of(bid.Franchise).Name,
					nativecheck.ContractWords(bid.Salary, bid.Years))
				out.Bids = append(out.Bids, MarketChoice{ID: bid.ID, Words: words})
			}
			continue
		}
		if len(a.liveBids()) == 0 {
			continue
		}
		closeAt, words := bidClose(stand, a.standingAt(), v.zone)
		if closeAt != nil {
			if at.Before(*closeAt) {
				words = "Clock running — " + words
			} else {
				words = "Ready for a decision"
			}
		}
		// Only the reason travels: even the commissioner cannot read an undecided blind bid.
		live := a.liveBids()
		if lead, ok := leaderOf(live); ok && closeAt != nil && !at.Before(*closeAt) {
			tied := 0
			for _, b := range live {
				if b.Points == lead.Points {
					tied++
				}
			}
			if tied > 1 {
				words = "Tie to settle"
			}
		}
		out.Players = append(out.Players, MarketChoice{ID: a.Player, Words: name + " — " + words})
	}
}
