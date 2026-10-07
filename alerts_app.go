package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
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
	now := a.movesNow()
	r := AlertReading{Alerts: []Alert{}, Unavailable: []Unavailable{{
		Kind: "ir", Note: "NFL injury status not yet read (MFL injuries export)",
	}}}
	a.lineupAlert(&r, saved, now)
	r.tradeAlert(trades, now)
	slices.SortFunc(r.Alerts, func(a, b Alert) int {
		if order := cmp.Compare(b.Urgency, a.Urgency); order != 0 {
			return order
		}
		if a.At.IsZero() != b.At.IsZero() {
			if a.At.IsZero() {
				return 1
			}
			return -1
		}
		if order := a.At.Compare(b.At); order != 0 {
			return order
		}
		return cmp.Compare(a.Kind, b.Kind)
	})
	return r, nil
}

func (a *App) lineupAlert(r *AlertReading, saved LineupReading, now time.Time) {
	a.weekMu.Lock()
	week, failed := a.week, a.weekRefreshError
	a.weekMu.Unlock()
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
		problems = append(problems, problem.Message)
	}
	r.Alerts = append(r.Alerts, Alert{
		Kind: "lineup", Urgency: alertUrgency(now, lock), Title: "Check your lineup",
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
		Kind: "trades", Urgency: alertUrgency(now, soonest), Title: offersTitle(count),
		Detail: "Review pending trade offers", Node: "trade", Workspace: "trade-desk-and-offers",
		At: soonest, Provenance: trades.Provenance,
	})
}

func offersTitle(count int) string {
	if count == 1 {
		return "1 offer to you"
	}
	return fmt.Sprintf("%d offers to you", count)
}

func alertUrgency(now, at time.Time) string {
	reading := leagueclock.Clock(now, leagueclock.Inputs{Events: []leagueclock.Event{{
		Status: "PLANNED", At: at,
	}}})
	return string(reading.Deadlines[0].Urgency)
}
