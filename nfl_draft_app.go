package main

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
)

func (a *App) TargetNFLDraft() (Report, error) {
	if err := a.ready(); err != nil {
		return Report{}, fmt.Errorf("NFL draft: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	held := archivedOnly(ctx, a, "players", false, func(body []byte) ([]players.RawPlayer, error) {
		rows, err := players.Parse(ctx, body)
		if err != nil {
			return nil, fmt.Errorf("NFL draft: players: %w", err)
		}
		if !slices.ContainsFunc(rows, func(p players.RawPlayer) bool { return p.DraftRound != "" && p.DraftPick != "" }) {
			return nil, fmt.Errorf("NFL draft: detailed selection fields not held")
		}
		return rows, nil
	})
	return nflDraftReport(a.season, held), nil
}

func nflDraftReport(season int, held heldSeasonFeed[[]players.RawPlayer]) Report {
	report := reportShell("M-087", fmt.Sprintf("NFL draft · %d", season),
		"NFL draft year, round and pick are read from the held detailed MFL players export, not the fantasy league's draftResults. "+
			"Only selections for this season with both round and pick are shown; undrafted and missing selections are not invented.",
		held, time.Local, "Player", "Position", "NFL team", "Round", "Pick")
	report.Source = "Source: held MFL players (DETAILS=1) archive"
	rows := slices.Clone(held.value)
	slices.SortFunc(rows, func(a, b players.RawPlayer) int {
		ar, _ := strconv.Atoi(a.DraftRound)
		br, _ := strconv.Atoi(b.DraftRound)
		if order := cmp.Compare(ar, br); order != 0 {
			return order
		}
		ap, _ := strconv.Atoi(a.DraftPick)
		bp, _ := strconv.Atoi(b.DraftPick)
		if order := cmp.Compare(ap, bp); order != 0 {
			return order
		}
		return cmp.Compare(a.ID, b.ID)
	})
	for _, p := range rows {
		year, _ := strconv.Atoi(p.DraftYear)
		round, _ := strconv.Atoi(p.DraftRound)
		pick, _ := strconv.Atoi(p.DraftPick)
		if year != season || round == 0 || pick == 0 {
			continue
		}
		report.Rows = append(report.Rows, ReportRow{Cells: []string{p.Name, p.Position, p.Team, p.DraftRound, p.DraftPick}})
	}
	if report.Status == "ok" && len(report.Rows) == 0 {
		report.Status, report.Note = "empty", "No NFL selections for this season are held."
	}
	return report
}
