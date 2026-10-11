package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

func (a *App) contractTeamCap(ctx context.Context, mine string, f envelope.ContractFacts) (contractCap, error) {
	m, _, used := a.league.ContractsView()
	players := []state.MirrorPlayer{}
	for _, p := range m.Players {
		if p.FranchiseID == mine {
			players = append(players, p)
		}
	}
	m.Players = players
	s := state.WhatIfSession{Mine: mine, Snapshot: m, Used: used[mine], Generation: "contract-preview:" + mine + ":" + fmt.Sprint(time.Now().UnixNano()), TaxiPercent: a.rulebook.TaxiCapPercent(), IRPercent: a.rulebook.IRCapPercent()}
	if text := a.rulebook.GetSalaryCap(); text != "" {
		capAmount, err := domain.ParseMoneyMillions(text)
		if err != nil {
			return contractCap{}, fmt.Errorf("contract cap: %w", err)
		}
		s.Cap = &capAmount
	}
	store, err := a.loadWhatIf(ctx, s)
	if err != nil {
		return contractCap{}, err
	}
	s.SeedUsed, _ = store.CapUsed(mine)
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		return contractCap{}, fmt.Errorf("contract coordinator: %w", err)
	}
	if err := coord.ProjectContract(ctx, f.Player.String(), f.Terms.Cells); err != nil {
		return contractCap{}, fmt.Errorf("contract projection: %w", err)
	}
	// Extend the reporting horizon without changing the held baseline commitments.
	horizon := f.Terms.Cells[len(f.Terms.Cells)-1].Season
	view, err := a.whatIfViewThrough(ctx, s, store, false, horizon)
	if err != nil {
		return contractCap{}, err
	}
	out := contractCap{Figures: []envelope.ContractFigure{}, Cap: s.Cap, After: map[int]domain.Money{}}
	for _, row := range view.Figures {
		out.Figures = append(out.Figures, envelope.ContractFigure{Label: row.Label, Before: row.Before, After: row.After, Change: row.Change})
	}
	for _, p := range players {
		if err := whatIfCommitted(ctx, store, p, m.Season, out.After); err != nil {
			return contractCap{}, err
		}
	}
	projected, _ := store.CapUsed(mine)
	out.After[m.Season] = s.Used + projected - s.SeedUsed
	return out, nil
}

// contractCap is the team cap table after an Act, and each season's commitments after it.
type contractCap struct {
	Figures []envelope.ContractFigure
	Cap     *domain.Money
	After   map[int]domain.Money
}

// contractCapReason judges each season the Act adds money to by that season's commitments after the Act.
func contractCapReason(f envelope.ContractFacts, capAmount domain.Money, after map[int]domain.Money) string {
	held := map[int]domain.Money{}
	for _, c := range f.Baseline {
		held[c.Season] = c.Salary
	}
	for _, c := range f.Terms.Cells {
		if c.Salary > held[c.Season] && after[c.Season] > capAmount {
			return fmt.Sprintf("This move exceeds the team's %d salary cap.", c.Season)
		}
	}
	return ""
}

func (a *App) contractCapFacts(ctx context.Context, mine string, f *envelope.ContractFacts) error {
	if len(f.Terms.Cells) == 0 || slices.Contains(f.Reasons, "This contract cannot take this Act.") {
		return nil
	}
	team, err := a.contractTeamCap(ctx, mine, *f)
	if err != nil {
		return err
	}
	f.TeamFigures = team.Figures
	if team.Cap == nil {
		f.Reasons = append(f.Reasons, "Not checked: team salary cap; no cap setting is held")
		return nil
	}
	if last := f.Terms.Cells[len(f.Terms.Cells)-1].Season; last > f.Season {
		f.Reasons = append(f.Reasons, fmt.Sprintf("Not checked: salary caps after %d; later seasons are judged against the held %d cap", f.Season, f.Season))
	}
	if reason := contractCapReason(*f, *team.Cap, team.After); reason != "" {
		f.Reasons = append(f.Reasons, reason)
	}
	return nil
}
