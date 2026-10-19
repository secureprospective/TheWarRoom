package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// MarketChoice contains only public words and the target accepted by the named command.
type MarketChoice struct {
	ID    string `json:"id"`
	Words string `json:"words"`
}

type Market struct {
	League   string         `json:"league"`
	Revision int64          `json:"revision"`
	Windows  []MarketChoice `json:"windows"`
	Claims   []MarketChoice `json:"claims"`
	Players  []MarketChoice `json:"players"`
	Bids     []MarketChoice `json:"bids"`
	Trades   []MarketChoice `json:"trades"`
}

func marketCommand(command string) bool {
	switch command {
	case WaiverProcess, WaiverOverride, BidsProcess, BidsOverride, TradeReview, TradeAppealRule, TradeOverride:
		return true
	}
	return false
}

// MarketReading uses the command's own permission, never an owner's private reading.
// Records from another league are excluded before both authorization and replay.
func MarketReading(league, command, principal string, records []Record, at time.Time,
	zone *time.Location, facts nativecheck.Facts) (Market, error) {
	out := Market{League: league, Windows: []MarketChoice{}, Claims: []MarketChoice{},
		Players: []MarketChoice{}, Bids: []MarketChoice{}, Trades: []MarketChoice{}}
	if !marketCommand(command) {
		return out, fmt.Errorf("choose a commissioner waiver, bid or trade action")
	}
	held := make([]Record, 0, len(records))
	for _, r := range records {
		if r.Command.League == league {
			held = append(held, r)
		}
	}
	out.Revision = int64(len(held))
	if _, err := AuthorizeRecords(RawCommand{League: league, Command: command}, principal, held); err != nil {
		return out, err
	}
	assets, state, err := nativeAssetsState(league, held, at)
	if err != nil {
		return out, err
	}
	if !assets.Seeded {
		return out, nil
	}
	rules, err := RulesNow(held, at)
	if err != nil {
		return out, err
	}
	directory, err := NativeDirectoryReading(league, held, at)
	if err != nil {
		return out, err
	}
	viewer := bidViewer{zone: zone, teams: directory.seats(), facts: facts}
	switch command {
	case WaiverProcess, WaiverOverride:
		err = out.readWaivers(state, rules.Waivers, viewer, at)
	case BidsProcess, BidsOverride:
		out.readBids(state.Bids, viewer, bidStandFor(rules), at)
	default:
		governance, authErr := GovernanceState(held)
		if authErr != nil {
			return out, authErr
		}
		x := tradeExecution{records: held, state: governance, now: at, principal: principal}
		out.readTrades(state.Trades, viewer, command, x)
	}
	return out, err
}

func (out *Market) readTrades(trades []TradeState, v bidViewer, command string, x tradeExecution) {
	for _, t := range trades {
		candidate := t.clone()
		raw := RawCommand{Command: command, Args: RawArgs{Trade: &RawTradeArgs{ID: t.ID, Decision: "veto"}}}
		switch command {
		case TradeReview:
			if x.review(raw, &candidate) != nil {
				continue
			}
		case TradeAppealRule:
			if x.ruleAppeal(raw, &candidate) != nil {
				continue
			}
		case TradeOverride:
			if t.Status != "approved" && t.Status != "vetoed" && t.Status != "appealed" && !x.stalled(&candidate) {
				continue
			}
		}
		names := make([]string, 0, len(t.Parties))
		for _, id := range t.Parties {
			names = append(names, v.teams.of(id).Name)
		}
		out.Trades = append(out.Trades, MarketChoice{ID: t.ID, Words: strings.Join(names, " / ") + " — " + t.Words})
	}
}
