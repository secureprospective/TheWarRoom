package main

import (
	"context"
	"fmt"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func (a *App) byeAlerts(r *AlertReading, mine string, now time.Time) {
	a.core.TargetSnapshotMu.Lock()
	snap := a.core.TargetSnapshot
	a.core.TargetSnapshotMu.Unlock()
	a.core.WeekMu.Lock()
	week, failed := a.core.Week, a.core.WeekRefreshError
	a.core.WeekMu.Unlock()
	ctx, cancel := context.WithTimeout(a.core.Ctx, m4Timeout)
	defer cancel()
	byes := archivedOnly(ctx, a, nflschedule.ByeExport, true, nflschedule.ParseByes)
	rules, err := lineup.ParseRules(a.core.Rulebook.ActiveConfig().Starters)
	note := byeInputsNote(snap, mine, byes, week)
	if err != nil || a.rulesFreshness().State == FreshFail {
		note = "Bye-week check unavailable: lineup rules not held."
	}
	if failed != "" {
		note = "Bye-week check unavailable: current NFL week could not be checked."
	}
	if note != "" {
		r.Unavailable = append(r.Unavailable, Unavailable{Kind: "bye", Note: note})
		return
	}
	players, names, byeMap := byeRoster(snap, mine, byes.Value)
	for _, p := range players {
		_, known := byeMap[p.Team]
		if p.Active && (!known || p.Position == "" || p.Position == domain.PosFlag) {
			r.Unavailable = append(r.Unavailable, Unavailable{Kind: "bye",
				Note: "Bye-week check unavailable: an active player's position, NFL team or bye week is missing."})
			return
		}
	}
	for _, short := range lineup.ByeShortfalls(rules, players, byeMap) {
		if short.Week < week.Number || short.Week > week.Number+1 {
			continue
		}
		lock := a.byeWeekLock(ctx, week, short.Week, now)
		if lock.IsZero() {
			r.Unavailable = append(r.Unavailable, Unavailable{Kind: "bye",
				Note: fmt.Sprintf("Week %d lineup lock unavailable; check Franchise HQ for bye shortfalls.", short.Week)})
		}
		r.Alerts = append(r.Alerts, Alert(controlapp.ByeAlert(short, players, names, byeMap, now, lock,
			byes.Provenance("MFL nflByeWeeks", "Bye weeks"))))
	}
}

func byeInputsNote(snap snapshot.Snapshot, mine string, byes heldSeasonFeed[[]nflschedule.RawBye],
	week leagueweek.Week) string {
	if byes.FetchedAt.IsZero() || len(byes.Value) == 0 || byes.RefreshError != "" {
		return "Bye-week check unavailable: NFL bye weeks not held."
	}
	if week.Number == 0 {
		return "Bye-week check unavailable: current NFL week not held."
	}
	for _, roster := range snap.Rosters.Value {
		if roster.FranchiseID == mine && snap.Rosters.Provenance.Freshness.State != FreshFail &&
			snap.Rosters.Provenance.Freshness.State != FreshStale {
			return ""
		}
	}
	return "Bye-week check unavailable: your roster not held or out of date."
}

func byeRoster(snap snapshot.Snapshot, mine string, byes []nflschedule.RawBye) ([]lineup.ByePlayer, []string, map[string]int) {
	identities := map[string]snapshot.Player{}
	for _, p := range snap.Players.Value {
		identities[p.ID.String()] = p
	}
	byeMap := map[string]int{}
	for _, b := range byes {
		byeMap[b.Team] = b.ByeWeek()
	}
	players, names := []lineup.ByePlayer{}, []string{}
	for _, roster := range snap.Rosters.Value {
		if roster.FranchiseID != mine {
			continue
		}
		for _, p := range roster.Players {
			identity := identities[p.ID.String()]
			players = append(players, lineup.ByePlayer{Position: identity.Position, Team: identity.Team,
				Active: p.RosterStatus == domain.RosterActive})
			names = append(names, identity.Name)
		}
	}
	return players, names, byeMap
}

func (a *App) byeWeekLock(ctx context.Context, current leagueweek.Week, week int, now time.Time) time.Time {
	if week != current.Number {
		feed := a.scheduleFeed(week)
		feed.bodies = a.core.History.ReferenceBodies
		rows, _, found, err := readArchivedFeed(ctx, a.core.Season, feed)
		if err != nil || !found || len(rows) == 0 {
			return time.Time{}
		}
		current, err = nflschedule.ToWeek(rows[0], now)
		if err != nil {
			return time.Time{}
		}
	}
	lock, _ := leagueweek.NextLock(now, current)
	return lock
}
