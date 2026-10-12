package main

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

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
	if trade.Title != "1 offer to you" || trade.Node != "trade" || trade.Workspace != "trade-desk" ||
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
