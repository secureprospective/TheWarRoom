package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func (a *App) byeAlerts(r *AlertReading, mine string, now time.Time) {
	a.targetSnapshotMu.Lock()
	snap := a.targetSnapshot
	a.targetSnapshotMu.Unlock()
	a.weekMu.Lock()
	week, failed := a.week, a.weekRefreshError
	a.weekMu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
	defer cancel()
	byes := archivedOnly(ctx, a, nflschedule.ByeExport, true, nflschedule.ParseByes)
	rules, err := lineup.ParseRules(a.rulebook.ActiveConfig().Starters)
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
	players, names, byeMap := byeRoster(snap, mine, byes.value)
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
		r.Alerts = append(r.Alerts, byeAlert(short, players, names, byeMap, now, lock,
			byes.provenance("MFL nflByeWeeks", "Bye weeks")))
	}
}

func byeInputsNote(snap snapshot.Snapshot, mine string, byes heldSeasonFeed[[]nflschedule.RawBye],
	week leagueweek.Week) string {
	if byes.fetchedAt.IsZero() || len(byes.value) == 0 || byes.refreshError != "" {
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
		feed.bodies = a.history.ReferenceBodies
		rows, _, found, err := readArchivedFeed(ctx, a.season, feed)
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

func byeAlert(short lineup.ByeShortfall, players []lineup.ByePlayer, names []string, byes map[string]int,
	now, lock time.Time, provenance snapshot.Provenance) Alert {
	positions := map[domain.Position]string{domain.PosQB: "quarterback", domain.PosRB: "running back",
		domain.PosWR: "wide receiver", domain.PosTE: "tight end", domain.PosK: "kicker",
		domain.PosDE: "defensive end", domain.PosDT: "defensive tackle", domain.PosLB: "linebacker",
		domain.PosCB: "cornerback", domain.PosS: "safety"}
	position := positions[short.Position]
	title := fmt.Sprintf("Week %d: %s shortfall", short.Week, position)
	if short.Available == 0 {
		title = fmt.Sprintf("Week %d: no %s available", short.Week, position)
	}
	onBye := []string{}
	for i, p := range players {
		if p.Active && p.Position == short.Position && byes[p.Team] == short.Week {
			name := names[i]
			if name == "" {
				name = "Player name unavailable"
			}
			onBye = append(onBye, name)
		}
	}
	detail := fmt.Sprintf("%d available; needs %d. Checks position minimums only.", short.Available, short.Required)
	if len(onBye) > 0 {
		detail += " On bye: " + strings.Join(onBye, ", ") + "."
	}
	return Alert{Kind: "bye", Urgency: alertUrgency(now, lock), Title: title, Detail: detail,
		Node: "hq", Workspace: "lineup-and-roster", At: lock, Provenance: provenance}
}
