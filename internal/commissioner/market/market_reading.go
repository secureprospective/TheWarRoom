package market

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Choice contains only public words and the target accepted by the named command.
type Choice struct {
	ID    string `json:"id"`
	Words string `json:"words"`
}

type Market struct {
	League   string   `json:"league"`
	Revision int64    `json:"revision"`
	Windows  []Choice `json:"windows"`
	Claims   []Choice `json:"claims"`
	Players  []Choice `json:"players"`
	Bids     []Choice `json:"bids"`
	Trades   []Choice `json:"trades"`
}

func marketCommand(command string) bool {
	switch command {
	case commissioner.WaiverProcess, commissioner.WaiverOverride, commissioner.BidsProcess, commissioner.BidsOverride, commissioner.TradeReview, commissioner.TradeAppealRule, commissioner.TradeOverride:
		return true
	}
	return false
}

// ChoicesReading uses the command's own permission, never an owner's private reading.
// Records from another league are excluded before both authorization and replay.
func ChoicesReading(league, command, principal string, records []commissioner.Record, at time.Time,
	zone *time.Location, facts nativecheck.Facts) (Market, error) {
	out := Market{League: league, Windows: []Choice{}, Claims: []Choice{},
		Players: []Choice{}, Bids: []Choice{}, Trades: []Choice{}}
	if !marketCommand(command) {
		return out, fmt.Errorf("choose a commissioner waiver, bid or trade action")
	}
	held := make([]commissioner.Record, 0, len(records))
	for _, r := range records {
		if r.Command.League == league {
			held = append(held, r)
		}
	}
	out.Revision = int64(len(held))
	if _, err := commissioner.AuthorizeRecords(commissioner.RawCommand{League: league, Command: command}, principal, held); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return out, err
	}
	f, err := commissioner.NewFrame(league, held, at, facts, zone, at)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return out, err
	}
	return marketReadingFrame(f, out, command, principal)
}

func marketReadingFrame(f commissioner.Frame, out Market, command, principal string) (Market, error) {
	league, held, at, zone, facts := f.League, f.Records, f.At, f.Zone, f.Facts
	assets, state := f.Assets, f.Book
	if !assets.Seeded {
		return out, nil
	}
	if err := f.ReadRules(); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return out, err
	}
	rules := f.Rules
	directory, err := commissioner.NativeDirectoryReading(league, held, at)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return out, err
	}
	mode, modeErr := nativecheck.BidMode(rules.Bids, leaguerules.UFAProcess)
	viewer := bidViewer{open: modeErr == nil && mode == leaguerules.Open, zone: zone, teams: directory.Seats(), facts: facts}
	switch command {
	case commissioner.WaiverProcess, commissioner.WaiverOverride:
		err = out.readWaivers(state, rules.Waivers, viewer, at)
	case commissioner.BidsProcess, commissioner.BidsOverride:
		out.readBids(state.Bids, viewer, bidStandFor(rules), at)
	default:
		governance, authErr := commissioner.GovernanceState(held)
		if authErr != nil {
			//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
			return out, authErr
		}
		x := tradeExecution{records: held, state: governance, now: at, principal: principal}
		out.readTrades(state.Trades, viewer, command, x)
	}
	return out, err
}

func (out *Market) readTrades(trades []commissioner.TradeState, v bidViewer, command string, x tradeExecution) {
	for _, t := range trades {
		candidate := t.Clone()
		raw := commissioner.RawCommand{Command: command, Args: commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: t.ID, Decision: "veto"}}}
		switch command {
		case commissioner.TradeReview:
			if x.review(raw, &candidate) != nil {
				continue
			}
		case commissioner.TradeAppealRule:
			if x.ruleAppeal(raw, &candidate) != nil {
				continue
			}
		case commissioner.TradeOverride:
			if t.Status != "approved" && t.Status != "vetoed" && t.Status != "appealed" && !x.stalled(&candidate) {
				continue
			}
		}
		names := make([]string, 0, len(t.Parties))
		for _, id := range t.Parties {
			names = append(names, v.teams.Of(id).Name)
		}
		out.Trades = append(out.Trades, Choice{ID: t.ID, Words: strings.Join(names, " / ") + " — " + t.Words})
	}
}
