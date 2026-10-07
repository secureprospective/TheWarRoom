package main

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func TestTargetPulseNow(t *testing.T) {
	a, transport, _ := lineupDraftApp(t)
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	r, err := a.TargetPulseNow()
	if err != nil || r.Week != 5 || len(r.Matchups) != 16 {
		t.Fatalf("pulse: week %d, %d matchups, %v", r.Week, len(r.Matchups), err)
	}
	saved, err := a.TargetLineup("0025")
	if err != nil {
		t.Fatal(err)
	}
	var mine PulseSide
	for _, m := range r.Matchups {
		if m.Home.FranchiseID == "" || m.Away.FranchiseID == "" || m.Home.FranchiseID == m.Away.FranchiseID {
			t.Fatalf("matchup: %+v", m)
		}
		for _, side := range []PulseSide{m.Home, m.Away} {
			if side.FranchiseID == "0025" {
				mine = side
			}
		}
	}
	if len(mine.Starters) != 21 || mine.YetToPlay != 21 {
		t.Fatalf("0025: %d starters, yet to play %d", len(mine.Starters), mine.YetToPlay)
	}
	for i, p := range mine.Starters {
		if p.ID != saved.Starters[i].ID || p.Name == "" || p.Name == p.ID || p.Position == "" {
			t.Fatalf("starter %d: %+v vs %+v", i, p, saved.Starters[i])
		}
	}
	if len(transport.requests) != 0 {
		t.Fatal("pulse fetched")
	}
	a.seasonLineups.refreshError = "offline"
	failed, err := a.TargetPulseNow()
	body, _ := json.Marshal(failed)
	if err != nil || len(failed.Matchups) != 0 || !strings.Contains(string(body), `"matchups":[]`) {
		t.Fatalf("failed feed: %s %v", body, err)
	}
}

func TestHeldSideNeverNull(t *testing.T) {
	body, err := json.Marshal(heldSide(map[string]PulseSide{}, "0007"))
	if err != nil || !strings.Contains(string(body), `"starters":[]`) {
		t.Fatalf("%s %v", body, err)
	}
}

func alertKinds(r AlertReading) (alerts, unavailable []string) {
	for _, a := range r.Alerts {
		alerts = append(alerts, a.Kind)
	}
	for _, u := range r.Unavailable {
		unavailable = append(unavailable, u.Kind)
	}
	return alerts, unavailable
}

func TestTargetAlerts(t *testing.T) {
	a, _, _ := tradeDraftApp(t)
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	r, err := a.TargetAlerts("0025")
	if err != nil {
		t.Fatal(err)
	}
	alerts, unavailable := alertKinds(r)
	if !slices.Equal(alerts, []string{"trades"}) || !slices.Equal(unavailable, []string{"ir"}) {
		t.Fatalf("legal lineup, one offer: %v %v", alerts, unavailable)
	}
	trade := r.Alerts[0]
	if trade.Title != "1 offer to you" || trade.Node != "trade" || trade.Workspace != "trade-desk-and-offers" ||
		!trade.At.Equal(a.movesNow().Add(time.Hour)) {
		t.Fatalf("trade alert: %+v", trade)
	}
	body, _ := json.Marshal(r)
	if strings.Contains(strings.ToLower(string(body)), "waiver") {
		t.Fatalf("waivers are absent by league rule: %s", body)
	}

	rows := a.seasonLineups.value.Franchises
	for i := range rows {
		if rows[i].Franchise == "0025" {
			rows[i].Starters = rows[i].Starters[:20]
		}
	}
	r, err = a.TargetAlerts("0025")
	if alerts, _ = alertKinds(r); err != nil || !slices.Equal(alerts, []string{"trades", "lineup"}) {
		t.Fatalf("partial lineup: %v %v", alerts, err)
	}
	lineup := r.Alerts[1]
	lock := a.movesNow().Add(24 * time.Hour)
	if lineup.Title != "Check your lineup" || lineup.Node != "hq" || !lineup.At.Equal(lock) ||
		lineup.Detail == "" || r.Alerts[0].Urgency < lineup.Urgency {
		t.Fatalf("lineup alert: %+v", r.Alerts)
	}

	a.week = leagueweek.Week{Number: 5, Games: []leagueweek.Game{{Kickoff: a.movesNow().Add(-time.Hour)}}}
	r, err = a.TargetAlerts("0025")
	if alerts, _ = alertKinds(r); err != nil || slices.Contains(alerts, "lineup") {
		t.Fatalf("locked week still alerts: %v %v", alerts, err)
	}

	a.seasonPendingTrades.refreshError = "offline"
	r, err = a.TargetAlerts("0025")
	alerts, unavailable = alertKinds(r)
	if err != nil || slices.Contains(alerts, "trades") || !slices.Contains(unavailable, "trades") {
		t.Fatalf("failed pendingTrades: %v %v %v", alerts, unavailable, err)
	}
}
