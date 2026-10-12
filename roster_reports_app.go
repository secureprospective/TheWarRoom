package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func (a *App) TargetRosterReports(mine string) ([]Report, error) {
	if err := a.ready(); err != nil {
		return nil, fmt.Errorf("roster reports: startup: %w", err)
	}
	snap, err := a.personalSnapshot(mine)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	a.weekMu.Lock()
	lastWeek := a.week.Number - 1
	a.weekMu.Unlock()
	season := scorePeriod(ctx, a, "YTD")
	week := scorePeriod(ctx, a, strconv.Itoa(lastWeek))
	byes := archivedOnly(ctx, a, nflschedule.ByeExport, true, nflschedule.ParseByes)
	rules, err := lineup.ParseRules(a.rulebook.ActiveConfig().Starters)
	stats, bye := rosterReports(snap, mine, season, week, byes, rules)
	if err != nil {
		bye.Note += " Starter bounds unavailable: " + err.Error()
	}
	return []Report{stats, bye}, nil
}

func scorePeriod(ctx context.Context, a *App, period string) heldSeasonFeed[[]ingestion.RawPlayerScore] {
	return archivedOnly(ctx, a, "playerScores", false, func(body []byte) ([]ingestion.RawPlayerScore, error) {
		rows, err := ingestion.ParsePlayerScores(body)
		if err != nil {
			return nil, fmt.Errorf("player scores: parse: %w", err)
		}
		out := []ingestion.RawPlayerScore{}
		for _, r := range rows {
			if r.Week == period {
				out = append(out, r)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("player scores: period %s not held", period)
		}
		return out, nil
	})
}

func rosterReports(snap snapshot.Snapshot, mine string, season, week heldSeasonFeed[[]ingestion.RawPlayerScore],
	byes heldSeasonFeed[[]nflschedule.RawBye], rules lineup.Rules) (Report, Report) {
	stats := reportShell("M-101", "Roster with stats",
		"Season points and last completed week's points come from held playerScores, not estimates. Missing rows remain unavailable.",
		season, time.Local, "Player", "Position", "NFL team", "Season points", "Last week points")
	bye := reportShell("M-154", "Bye weeks",
		"Active players left after each NFL bye are compared with the league's minimum starters at that position. IR and taxi do not count. "+
			"This checks position minima only, not flexible starters, injuries or every legal-lineup constraint.",
		byes, time.Local, "Player", "Position", "NFL team", "Bye week")
	stats.Source = "Source: held MFL playerScores archive"
	bye.Source = "Source: held MFL nflByeWeeks archive"
	stats.Status = "ok"
	if season.fetchedAt.IsZero() || week.fetchedAt.IsZero() {
		stats.Status = "stale"
	}
	if !week.fetchedAt.IsZero() {
		stats.Note += " Last-week scores as of " + reportTime(week.fetchedAt, time.Local) + "."
	}
	if week.fetchedAt.IsZero() {
		stats.Note += " Last-week points unavailable."
	}
	identities := map[string]snapshot.Player{}
	for _, p := range snap.Players.Value {
		identities[p.ID.String()] = p
	}
	byeMap := map[string]int{}
	for _, b := range byes.value {
		byeMap[b.Team] = b.ByeWeek()
	}
	seasonPoints, weekPoints := scoreMap(season.value), scoreMap(week.value)
	active := []lineup.ByePlayer{}
	for _, r := range snap.Rosters.Value {
		if r.FranchiseID != mine {
			continue
		}
		for _, p := range r.Players {
			id := p.ID.String()
			identity := identities[id]
			name := identity.Name
			if name == "" {
				name = "Player name unavailable"
			}
			stats.Rows = append(stats.Rows, ReportRow{Cells: []string{name, string(identity.Position), identity.Team,
				personalScoreText(seasonPoints, id), personalScoreText(weekPoints, id)}})
			w := "Unavailable"
			if week, ok := byeMap[identity.Team]; ok {
				w = strconv.Itoa(week)
			}
			bye.Rows = append(bye.Rows, ReportRow{Cells: []string{name, string(identity.Position), identity.Team, w}})
			active = append(active, lineup.ByePlayer{Position: identity.Position, Team: identity.Team, Active: p.RosterStatus == domain.RosterActive})
		}
	}
	for _, s := range lineup.ByeShortfalls(rules, active, byeMap) {
		bye.Summary = append(bye.Summary, ReportRow{Cells: []string{fmt.Sprintf("Week %d · %s", s.Week, s.Position),
			fmt.Sprintf("%d available; needs %d", s.Available, s.Required)}})
	}
	if len(bye.Summary) == 0 && len(byes.value) > 0 {
		bye.Note = "No position minimum runs short in the held bye-week schedule."
	}
	return stats, bye
}
func scoreMap(rows []ingestion.RawPlayerScore) map[string]string {
	out := map[string]string{}
	for _, r := range rows {
		out[r.ID] = r.Score
	}
	return out
}
func personalScoreText(points map[string]string, id string) string {
	if value, ok := points[id]; ok {
		return value
	}
	return "Unavailable"
}
