package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type AlertReading struct {
	Alerts      []Alert       `json:"alerts"`
	Unavailable []Unavailable `json:"unavailable"`
}

type Alert struct {
	Kind       string              `json:"kind"`
	Urgency    string              `json:"urgency"`
	Title      string              `json:"title"`
	Detail     string              `json:"detail"`
	Node       string              `json:"node"`
	Workspace  string              `json:"workspace"`
	At         time.Time           `json:"at"`
	Provenance snapshot.Provenance `json:"provenance"`
	PlayerID   string              `json:"playerId,omitempty"`
	MFLURL     string              `json:"mflUrl,omitempty"`
}

type Unavailable struct {
	Kind string `json:"kind"`
	Note string `json:"note"`
}

// TargetAlerts reads held state only; waivers are absent by league rule (gap-closure §8).
func (a *App) TargetAlerts(franchiseID string) (AlertReading, error) {
	saved, err := a.TargetLineup(franchiseID)
	if err != nil {
		return AlertReading{}, fmt.Errorf("target alerts: %w", err)
	}
	trades, err := a.TargetTrades(franchiseID)
	if err != nil {
		return AlertReading{}, fmt.Errorf("target alerts: %w", err)
	}
	now := a.core.MovesNow()
	r := AlertReading{Alerts: []Alert{}, Unavailable: []Unavailable{}}
	a.irAlerts(&r, franchiseID)
	a.lineupAlert(&r, saved, now)
	a.byeAlerts(&r, franchiseID, now)
	r.tradeAlert(trades, now)
	slices.SortFunc(r.Alerts, func(a, b Alert) int { return controlapp.CompareAlerts(controlapp.Alert(a), controlapp.Alert(b)) })
	return r, nil
}

func (a *App) lineupAlert(r *AlertReading, saved LineupReading, now time.Time) {
	a.core.WeekMu.Lock()
	week, failed := a.core.Week, a.core.WeekRefreshError
	a.core.WeekMu.Unlock()
	if failed != "" || len(week.Games) == 0 {
		note := joinNote("lineup lock unknown", failed)
		r.Unavailable = append(r.Unavailable, Unavailable{Kind: "lineup", Note: note})
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
		r.Unavailable = append(r.Unavailable, Unavailable{Kind: "lineup", Note: reason})
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
	r.Alerts = append(r.Alerts, Alert{
		Kind: "lineup", Urgency: controlapp.AlertUrgency(now, lock), Title: "Check your lineup",
		Detail: strings.Join(problems, "; "), Node: "hq", Workspace: "lineup-and-roster",
		At: lock, Provenance: saved.Provenance,
	})
}

func (r *AlertReading) tradeAlert(trades TradeReading, now time.Time) {
	if reason := envelope.TradeFeedReason(trades.Provenance); reason != "" {
		r.Unavailable = append(r.Unavailable, Unavailable{Kind: "trades", Note: reason})
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
	r.Alerts = append(r.Alerts, Alert{
		Kind: "trades", Urgency: controlapp.AlertUrgency(now, soonest), Title: controlapp.OffersTitle(count),
		Detail: "Review pending trade offers", Node: "trade", Workspace: "trade-desk",
		At: soonest, Provenance: trades.Provenance,
	})
}

func (a *App) irAlerts(r *AlertReading, franchiseID string) {
	a.core.TargetSnapshotMu.Lock()
	snap, loaded := a.core.TargetSnapshot, a.core.HasTargetSnapshot
	a.core.TargetSnapshotMu.Unlock()
	injuries := a.heldInjuries()
	if !loaded || snap.Rosters.Provenance.Freshness.State == domain.FreshFail ||
		snap.Rosters.Provenance.Freshness.State == domain.FreshStale || injuries.Provenance.Freshness.State != domain.FreshLive {
		r.Unavailable = append(r.Unavailable, Unavailable{Kind: "ir", Note: "IR eligibility not checked; injuries or roster unavailable or stale"})
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
			r.Alerts = append(r.Alerts, Alert{Kind: "ir", Urgency: "U0", Title: "Possible IR violation: " + name,
				Detail: decision.Reason, Node: "hq", Workspace: "lineup-and-roster", PlayerID: id,
				MFLURL: target.URL, Provenance: injuries.Provenance})
		}
	}
}
