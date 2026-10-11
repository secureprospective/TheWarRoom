package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

func (a *App) contractTeamFigures(ctx context.Context, mine string, f envelope.ContractFacts) ([]envelope.ContractFigure, error) {
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
			return nil, fmt.Errorf("contract cap: %w", err)
		}
		s.Cap = &capAmount
	}
	store, err := a.loadWhatIf(ctx, s)
	if err != nil {
		return nil, err
	}
	s.SeedUsed, _ = store.CapUsed(mine)
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("contract coordinator: %w", err)
	}
	if err := coord.ProjectContract(ctx, f.Player.String(), f.Terms.Cells); err != nil {
		return nil, fmt.Errorf("contract projection: %w", err)
	}
	// Extend the reporting horizon without changing the held baseline commitments.
	horizon := f.Terms.Cells[len(f.Terms.Cells)-1].Season
	view, err := a.whatIfViewThrough(ctx, s, store, false, horizon)
	if err != nil {
		return nil, err
	}
	out := []envelope.ContractFigure{}
	for _, row := range view.Figures {
		out = append(out, envelope.ContractFigure{Label: row.Label, Before: row.Before, After: row.After, Change: row.Change})
	}
	return out, nil
}

func contractCapReason(figures []envelope.ContractFigure) string {
	for _, row := range figures {
		if row.Label == "Room" && strings.Contains(row.After, "-") {
			return "This move exceeds the team's salary cap."
		}
	}
	return ""
}

func (a *App) contractCapFacts(ctx context.Context, mine string, f *envelope.ContractFacts) error {
	if len(f.Terms.Cells) == 0 || slices.Contains(f.Reasons, "This contract cannot take this Act.") {
		return nil
	}
	figures, err := a.contractTeamFigures(ctx, mine, *f)
	if err != nil {
		return err
	}
	f.TeamFigures = figures
	if len(figures) > 0 && figures[0].Before == "Unavailable" {
		f.Reasons = append(f.Reasons, "Not checked: team salary cap; no cap setting is held")
	}
	if reason := contractCapReason(figures); reason != "" {
		f.Reasons = append(f.Reasons, reason)
	}
	return nil
}
