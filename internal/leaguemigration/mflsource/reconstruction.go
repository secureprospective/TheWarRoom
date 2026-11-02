package mflsource

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// TransactionReplayCommands reconstructs the state before the first supplied move.
// Baseline repairs and pre-observation moves form one ordinary atomic batch, so
// readers never see a half-reconstructed present roster.
func TransactionReplayCommands(raw transactions.RawTransactions, league, reason string, seed commissioner.RawNativeSnapshot, franchises, players map[string]string, draft ...[]draftreports.Pick) ([][]commissioner.RawCommand, error) {
	groups, err := TransactionCommands(raw, league, reason, seed.Season, franchises, players)
	if err != nil {
		return nil, err
	}
	rows, err := transactions.ToTransactions(raw)
	if err != nil {
		return nil, fmt.Errorf("read reconstruction evidence: %w", err)
	}
	slices.SortStableFunc(rows, func(a, b leaguefeed.Transaction) int { return a.Time.Compare(b.Time) })
	before := cloneReconstructionSeed(seed)
	if err := annotateHistoricalGroups(groups, seed, draft); err != nil {
		return nil, err
	}
	earliest := seed.RosterAt
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		if !row.Time.Before(seed.RosterAt) {
			continue
		}
		earliest = row.Time
		inverse, err := transactionCommands(reverseTransaction(row), league, reason, seed.Season, franchises, players)
		if err != nil {
			return nil, err
		}
		for _, command := range inverse {
			annotateHistoricalPick(command.Args.Correction, seed, draft)
			if err := reconstructAsset(&before, seed, command.Args.Correction); err != nil {
				return nil, fmt.Errorf("reconstruct %s at %s: %w", row.Kind, row.Time.Format(time.RFC3339Nano), err)
			}
		}
	}
	if earliest.Equal(seed.RosterAt) {
		return groups, nil
	}
	baseline, err := baselineCorrections(league, reason, seed, before, earliest.Add(-time.Nanosecond))
	if err != nil {
		return nil, err
	}
	history := baseline
	after := [][]commissioner.RawCommand{}
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		at, err := group[0].EffectiveTime(seed.RosterAt)
		if err != nil {
			return nil, fmt.Errorf("date reconstruction: %w", err)
		}
		if at.Before(seed.RosterAt) {
			history = append(history, group...)
		} else {
			after = append(after, group)
		}
	}
	return append([][]commissioner.RawCommand{history}, after...), nil
}

func reverseTransaction(row leaguefeed.Transaction) leaguefeed.Transaction {
	reverse := func(c *leaguefeed.RosterChange) *leaguefeed.RosterChange {
		if c == nil {
			return nil
		}
		return &leaguefeed.RosterChange{In: c.Out, Out: c.In}
	}
	row.AddsDrops, row.IR, row.Taxi = reverse(row.AddsDrops), reverse(row.IR), reverse(row.Taxi)
	if row.Trade != nil {
		trade := *row.Trade
		trade.Gave, trade.CounterpartyGave = trade.CounterpartyGave, trade.Gave
		row.Trade = &trade
	}
	return row
}

func reconstructAsset(before *commissioner.RawNativeSnapshot, observed commissioner.RawNativeSnapshot, c *commissioner.RawCorrection) error {
	if c.Pick != nil {
		if c.History != nil {
			for i := range before.PickHistory {
				p := &before.PickHistory[i]
				if p.Pick == *c.Pick {
					p.Owner = c.Owner
					return nil
				}
			}
			before.PickHistory = append(before.PickHistory, commissioner.HistoricalPick{Pick: *c.Pick, Owner: c.Owner, Evidence: *c.History})
			return nil
		}
		return reconstructPick(before, c)
	}
	index := before.PlayerOf(c.Franchise, c.Player)
	switch c.Action {
	case "remove":
		if index < 0 {
			return fmt.Errorf("player %s is not on %s after the observed acquisition", c.Player, c.Franchise)
		}
		before.Players = slices.Delete(before.Players, index, index+1)
	case "add":
		if before.OnSide(before.SideOf(c.Franchise), c.Player) {
			return fmt.Errorf("player %s is already rostered before undoing the drop", c.Player)
		}
		p := commissioner.NativePlayer{ID: c.Player, ContractMissing: true}
		if i := observed.PlayerOn(before.SideOf(c.Franchise), c.Player); i >= 0 {
			p = observed.Players[i]
		}
		p.Franchise, p.Status = c.Franchise, domain.RosterActive
		before.Players = append(before.Players, p)
	case "status":
		if index < 0 {
			return fmt.Errorf("reserve player %s is not on %s", c.Player, c.Franchise)
		}
		before.Players[index].Status = c.Status
	}
	return nil
}

func reconstructPick(before *commissioner.RawNativeSnapshot, c *commissioner.RawCorrection) error {
	for i := range before.Picks {
		p := &before.Picks[i]
		if p.Season == c.Pick.Season && p.Round == c.Pick.Round && p.Original == c.Pick.Original && p.Slot == c.Pick.Slot {
			p.Owner = c.Owner
			return nil
		}
	}
	return fmt.Errorf("traded pick %d round %d slot %d from %s has no observation", c.Pick.Season, c.Pick.Round, c.Pick.Slot, c.Pick.Original)
}

func baselineCorrections(league, reason string, seed, before commissioner.RawNativeSnapshot, at time.Time) ([]commissioner.RawCommand, error) {
	body, err := json.Marshal(before)
	if err != nil {
		return nil, fmt.Errorf("identify roster reconstruction: %w", err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(body))
	out := []commissioner.RawCommand{}
	add := func(c commissioner.RawCorrection) {
		command := commissioner.CorrectRoster
		if c.Pick != nil {
			command = commissioner.CorrectPick
		}
		out = append(out, commissioner.RawCommand{League: league, Command: command, Args: commissioner.RawArgs{Correction: &c},
			Request: fmt.Sprintf("transactions.sync.baseline.%s.%d", digest, len(out)), Reason: reason,
			Effective: at.UTC().Format(time.RFC3339Nano), Visibility: "league"})
	}
	baselineRosterCorrections(seed, before, add)
	for i, p := range before.Picks {
		if seed.Picks[i].Owner != p.Owner {
			add(commissioner.RawCorrection{Owner: p.Owner, Pick: &commissioner.CorrectionPick{Season: p.Season, Round: p.Round, Slot: p.Slot, Original: p.Original}})
		}
	}
	for _, p := range before.PickHistory {
		evidence := p.Evidence
		pick := p.Pick
		add(commissioner.RawCorrection{Owner: p.Owner, Pick: &pick, History: &evidence})
	}
	for _, command := range out {
		if err := command.Validate(); err != nil {
			return nil, fmt.Errorf("check roster reconstruction: %w", err)
		}
	}
	return out, nil
}

func baselineRosterCorrections(seed, before commissioner.RawNativeSnapshot, add func(commissioner.RawCorrection)) {
	for _, p := range seed.Players {
		index := before.PlayerOn(seed.SideOf(p.Franchise), p.ID)
		if index < 0 || before.Players[index].Franchise != p.Franchise {
			add(commissioner.RawCorrection{Player: p.ID, Franchise: p.Franchise, Action: "remove"})
		}
	}
	for _, p := range before.Players {
		index := seed.PlayerOn(seed.SideOf(p.Franchise), p.ID)
		if index < 0 || seed.Players[index].Franchise != p.Franchise {
			add(commissioner.RawCorrection{Player: p.ID, Franchise: p.Franchise, Action: "add"})
			index = -1
		}
		if (index < 0 && p.Status != domain.RosterActive) || (index >= 0 && seed.Players[index].Status != p.Status) {
			add(commissioner.RawCorrection{Player: p.ID, Franchise: p.Franchise, Action: "status", Status: p.Status})
		}
	}
}

func cloneReconstructionSeed(seed commissioner.RawNativeSnapshot) commissioner.RawNativeSnapshot {
	seed.Players, seed.Picks = slices.Clone(seed.Players), slices.Clone(seed.Picks)
	seed.PickHistory = slices.Clone(seed.PickHistory)
	return seed
}
