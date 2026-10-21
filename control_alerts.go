package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	tradesapp "github.com/secureprospective/TheWarRoom/internal/app/trades"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// Alerts gathers the alert tray's alerts unsorted; control.API sorts them and words the refusal.
// Each held input is read under its own lock, in the binding's original order.
func (h controlHost) Alerts(franchiseID string) (controlapp.AlertReading, error) {
	a := h.app
	saved, err := a.contracts().TargetLineup(franchiseID)
	if err != nil {
		return controlapp.AlertReading{}, err //nolint:wrapcheck // control.API prefixes the refusal
	}
	trades, err := a.tradesAPI().TargetTrades(franchiseID)
	if err != nil {
		return controlapp.AlertReading{}, err //nolint:wrapcheck // control.API prefixes the refusal
	}
	now := a.core.MovesNow()
	r := controlapp.AlertReading{Alerts: []controlapp.Alert{}, Unavailable: []controlapp.Unavailable{}}
	a.irAlerts(&r, franchiseID)
	a.lineupAlert(&r, saved, now)
	a.byeAlerts(&r, franchiseID, now)
	tradeAlert(&r, trades, now)
	return r, nil
}

func (a *App) lineupAlert(r *controlapp.AlertReading, saved contractsapp.LineupReading, now time.Time) {
	a.core.WeekMu.Lock()
	week, failed := a.core.Week, a.core.WeekRefreshError
	a.core.WeekMu.Unlock()
	if failed != "" || len(week.Games) == 0 {
		note := report.JoinNote("lineup lock unknown", failed)
		r.Unavailable = append(r.Unavailable, controlapp.Unavailable{Kind: "lineup", Note: note})
		return
	}
	if !leagueweek.LastLock(week).After(now) {
		return
	}
	reason := envelope.TradeFeedReason(saved.Provenance)
	if reason == "" && saved.Week != week.Number {
		reason = fmt.Sprintf("saved lineup is week %d, current week is %d", saved.Week, week.Number)
	}
	if reason == "" && saved.RulesSource.Freshness.State == FreshFail {
		reason = saved.RulesSource.Freshness.Note
	}
	if reason != "" {
		r.Unavailable = append(r.Unavailable, controlapp.Unavailable{Kind: "lineup", Note: reason})
		return
	}
	if saved.Check.Legal && saved.Check.Full {
		return
	}
	lock, _ := leagueweek.NextLock(now, week)
	problems := make([]string, 0, len(saved.Check.Problems))
	for _, problem := range saved.Check.Problems {
		problems = append(problems, strings.TrimRight(problem.Message, ".; "))
	}
	r.Alerts = append(r.Alerts, controlapp.Alert{
		Kind: "lineup", Urgency: controlapp.AlertUrgency(now, lock), Title: "Check your lineup",
		Detail: strings.Join(problems, "; "), Node: "hq", Workspace: "lineup-and-roster",
		At: lock, Provenance: saved.Provenance,
	})
}

func tradeAlert(r *controlapp.AlertReading, trades tradesapp.TradeReading, now time.Time) {
	if reason := envelope.TradeFeedReason(trades.Provenance); reason != "" {
		r.Unavailable = append(r.Unavailable, controlapp.Unavailable{Kind: "trades", Note: reason})
		return
	}
	count := 0
	var soonest time.Time
	for _, offer := range trades.Offers {
		if offer.Direction != "to_you" || (!offer.Expires.IsZero() && !offer.Expires.After(now)) {
			continue
		}
		count++
		if !offer.Expires.IsZero() && (soonest.IsZero() || offer.Expires.Before(soonest)) {
			soonest = offer.Expires
		}
	}
	if count == 0 {
		return
	}
	r.Alerts = append(r.Alerts, controlapp.Alert{
		Kind: "trades", Urgency: controlapp.AlertUrgency(now, soonest), Title: controlapp.OffersTitle(count),
		Detail: "Review pending trade offers", Node: "trade", Workspace: "trade-desk",
		At: soonest, Provenance: trades.Provenance,
	})
}

func (a *App) irAlerts(r *controlapp.AlertReading, franchiseID string) {
	a.core.TargetSnapshotMu.Lock()
	snap, loaded := a.core.TargetSnapshot, a.core.HasTargetSnapshot
	a.core.TargetSnapshotMu.Unlock()
	injuries := a.heldInjuries()
	if !loaded || snap.Rosters.Provenance.Freshness.State == domain.FreshFail ||
		snap.Rosters.Provenance.Freshness.State == domain.FreshStale || injuries.Provenance.Freshness.State != domain.FreshLive {
		r.Unavailable = append(r.Unavailable, controlapp.Unavailable{Kind: "ir", Note: "IR eligibility not checked; injuries or roster unavailable or stale"})
		return
	}
	names := map[string]string{}
	for _, p := range snap.Players.Value {
		names[p.ID.String()] = p.Name
	}
	seen := map[string]bool{}
	for _, roster := range snap.Rosters.Value {
		if roster.FranchiseID != franchiseID {
			continue
		}
		for _, p := range roster.Players {
			id := p.ID.String()
			if p.RosterStatus != domain.RosterIR || seen[id] {
				continue
			}
			seen[id] = true
			decision := eligibility.IRClassification(eligibility.Injuries{Status: injuries.Status,
				Fresh: injuries.Provenance.Freshness}, id)
			if decision.Verdict != eligibility.Ineligible {
				continue
			}
			name := names[id]
			if name == "" {
				name = "Player name unavailable"
			}
			target, _ := a.rosterMoveTarget("18")
			r.Alerts = append(r.Alerts, controlapp.Alert{Kind: "ir", Urgency: "U0", Title: "Possible IR violation: " + name,
				Detail: decision.Reason, Node: "hq", Workspace: "lineup-and-roster", PlayerID: id,
				MFLURL: target.URL, Provenance: injuries.Provenance})
		}
	}
}

func (a *App) byeAlerts(r *controlapp.AlertReading, mine string, now time.Time) {
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
	if err != nil || a.core.RulesFreshness().State == FreshFail {
		note = "Bye-week check unavailable: lineup rules not held."
	}
	if failed != "" {
		note = "Bye-week check unavailable: current NFL week could not be checked."
	}
	if note != "" {
		r.Unavailable = append(r.Unavailable, controlapp.Unavailable{Kind: "bye", Note: note})
		return
	}
	players, names, byeMap := byeRoster(snap, mine, byes.Value)
	for _, p := range players {
		_, known := byeMap[p.Team]
		if p.Active && (!known || p.Position == "" || p.Position == domain.PosFlag) {
			r.Unavailable = append(r.Unavailable, controlapp.Unavailable{Kind: "bye",
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
			r.Unavailable = append(r.Unavailable, controlapp.Unavailable{Kind: "bye",
				Note: fmt.Sprintf("Week %d lineup lock unavailable; check Franchise HQ for bye shortfalls.", short.Week)})
		}
		r.Alerts = append(r.Alerts, controlapp.ByeAlert(short, players, names, byeMap, now, lock,
			byes.Provenance("MFL nflByeWeeks", "Bye weeks")))
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
		feed.Bodies = a.core.History.ReferenceBodies
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
