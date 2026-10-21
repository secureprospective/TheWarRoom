package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func alertKinds(r controlapp.AlertReading) (alerts, unavailable []string) {
	for _, a := range r.Alerts {
		alerts = append(alerts, a.Kind)
	}
	for _, u := range r.Unavailable {
		unavailable = append(unavailable, u.Kind)
	}
	return alerts, unavailable
}

func TestTargetAlerts(t *testing.T) {
	t.Parallel()
	a, _, _ := tradeDraftApp(t)
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	r, err := a.controlAPI().TargetAlerts("0025")
	if err != nil {
		t.Fatal(err)
	}
	alerts, unavailable := alertKinds(r)
	if !slices.Equal(alerts, []string{"trades"}) || !slices.Equal(unavailable, []string{"ir", "bye"}) {
		t.Fatalf("legal lineup, one offer: %v %v", alerts, unavailable)
	}
	trade := r.Alerts[0]
	if trade.Title != "1 offer to you" || trade.Node != "trade" || trade.Workspace != "trade-desk" ||
		!trade.At.Equal(a.core.MovesNow().Add(time.Hour)) {
		t.Fatalf("trade alert: %+v", trade)
	}
	body, _ := json.Marshal(r)
	if strings.Contains(strings.ToLower(string(body)), "waiver") {
		t.Fatalf("waivers are absent by league rule: %s", body)
	}

	rows := a.core.SeasonLineups.Value.Franchises
	for i := range rows {
		if rows[i].Franchise == "0025" {
			rows[i].Starters = rows[i].Starters[:20]
		}
	}
	r, err = a.controlAPI().TargetAlerts("0025")
	if alerts, _ = alertKinds(r); err != nil || !slices.Equal(alerts, []string{"lineup", "trades"}) {
		t.Fatalf("partial lineup: %v %v", alerts, err)
	}
	lineup := r.Alerts[0]
	lock := a.core.MovesNow().Add(24 * time.Hour)
	if lineup.Title != "Check your lineup" || lineup.Node != "hq" || !lineup.At.Equal(lock) ||
		lineup.Detail == "" || r.Alerts[1].Urgency > lineup.Urgency {
		t.Fatalf("lineup alert: %+v", r.Alerts)
	}

	a.core.Week = leagueweek.Week{Number: 5, Games: []leagueweek.Game{{Kickoff: a.core.MovesNow().Add(-time.Hour)}}}
	r, err = a.controlAPI().TargetAlerts("0025")
	if alerts, _ = alertKinds(r); err != nil || slices.Contains(alerts, "lineup") {
		t.Fatalf("locked week still alerts: %v %v", alerts, err)
	}

	a.core.SeasonPendingTrades.RefreshError = "offline"
	r, err = a.controlAPI().TargetAlerts("0025")
	alerts, unavailable = alertKinds(r)
	if err != nil || slices.Contains(alerts, "trades") || !slices.Contains(unavailable, "trades") {
		t.Fatalf("failed pendingTrades: %v %v %v", alerts, unavailable, err)
	}
}

func byeFixtureApp(t *testing.T) (*App, *seasonTransport) {
	t.Helper()
	a, transport, _ := tradeDraftApp(t)
	a.core.TargetSnapshot.Rosters.Provenance.Freshness.State = domain.FreshLive
	for i := range a.core.TargetSnapshot.Players.Value {
		p := &a.core.TargetSnapshot.Players.Value[i]
		p.Team = "NYJ"
		if p.Position == domain.PosK {
			p.Team, p.Name = "DET", "Bates"
		}
		if p.Position == domain.PosQB {
			p.Team = "PIT"
		}
	}
	a.core.Week.Number = 6
	holdPersonalBody(t, a, "nflByeWeeks", `{"nflByeWeeks":{"team":[
		{"id":"DET","bye_week":"6"},{"id":"PIT","bye_week":"8"},{"id":"NYJ","bye_week":"9"}]}}`)
	return a, transport
}

func onlyByeAlerts(r controlapp.AlertReading) []controlapp.Alert {
	return slices.DeleteFunc(slices.Clone(r.Alerts), func(a controlapp.Alert) bool { return a.Kind != "bye" })
}

func TestByeAlertsHeldWeekSixAndUnavailable(t *testing.T) {
	t.Parallel()
	a, transport := byeFixtureApp(t)
	r, err := a.controlAPI().TargetAlerts("0025")
	if err != nil {
		t.Fatal(err)
	}
	byes := onlyByeAlerts(r)
	lock := a.core.MovesNow().Add(24 * time.Hour)
	if len(byes) != 1 || byes[0].Title != "Week 6: no kicker available" ||
		!strings.Contains(byes[0].Detail, "Bates") || !strings.Contains(byes[0].Detail, "0 available; needs 1") ||
		byes[0].At != lock || byes[0].Urgency != controlapp.AlertUrgency(a.core.MovesNow(), lock) ||
		byes[0].Node != "hq" || byes[0].Workspace != "lineup-and-roster" {
		t.Fatalf("week six shortfall: %+v", byes)
	}
	if len(transport.requests) != 0 {
		t.Fatal("bye alerts fetched from MFL")
	}
	a.core.History = nil
	r, err = a.controlAPI().TargetAlerts("0025")
	_, unavailable := alertKinds(r)
	if err != nil || len(onlyByeAlerts(r)) != 0 || !slices.Contains(unavailable, "bye") {
		t.Fatalf("bye data not held: %+v %v", r, err)
	}
}

func TestByeAlertsHorizonAndMissingRoster(t *testing.T) {
	t.Parallel()
	a, _ := byeFixtureApp(t)
	for _, week := range []int{5, 7, 8, 10} {
		a.core.Week.Number = week
		r, err := a.controlAPI().TargetAlerts("0025")
		if err != nil {
			t.Fatal(err)
		}
		byes := onlyByeAlerts(r)
		if week == 10 && len(byes) != 0 {
			t.Fatalf("no shortfall: %+v", byes)
		}
		if week == 5 && (len(byes) != 1 || !strings.Contains(byes[0].Title, "Week 6")) {
			t.Fatalf("next week: %+v", byes)
		}
		if week == 7 || week == 8 {
			if !slices.ContainsFunc(byes, func(a controlapp.Alert) bool {
				return a.Title == "Week 8: no quarterback available"
			}) {
				t.Fatalf("week eight quarterback: %+v", byes)
			}
		}
	}
	a.core.Week.Number = 6
	a.core.TargetSnapshot.Rosters.Value = nil
	r, err := a.controlAPI().TargetAlerts("0025")
	_, unavailable := alertKinds(r)
	if err != nil || len(onlyByeAlerts(r)) != 0 || !slices.Contains(unavailable, "bye") {
		t.Fatalf("roster not held: %+v %v", r, err)
	}
}

func TestByeAlertsMissingIdentityAndLock(t *testing.T) {
	t.Parallel()
	a, _ := byeFixtureApp(t)
	a.core.Week = leagueweek.Week{Number: 6}
	r, err := a.controlAPI().TargetAlerts("0025")
	_, unavailable := alertKinds(r)
	byes := onlyByeAlerts(r)
	if err != nil || len(byes) != 1 || !byes[0].At.IsZero() || byes[0].Urgency != "U0" ||
		!slices.Contains(unavailable, "bye") {
		t.Fatalf("unknown lock: %+v %v", r, err)
	}
	a.core.TargetSnapshot.Players.Value = nil
	r, err = a.controlAPI().TargetAlerts("0025")
	_, unavailable = alertKinds(r)
	if err != nil || len(onlyByeAlerts(r)) != 0 || !slices.Contains(unavailable, "bye") {
		t.Fatalf("unknown position/team: %+v %v", r, err)
	}
}

func TestByeAlertNextWeekHeldLock(t *testing.T) {
	t.Parallel()
	a, _ := byeFixtureApp(t)
	a.core.Week.Number = 5
	lock := a.core.MovesNow().Add(48 * time.Hour)
	holdPersonalBody(t, a, "nflSchedule&W=6", fmt.Sprintf(`{"nflSchedule":{"week":"6","matchup":{
		"kickoff":"%d","gameSecondsRemaining":"3600","team":[{"id":"NYJ"},{"id":"PIT"}]}}}`, lock.Unix()))
	r, err := a.controlAPI().TargetAlerts("0025")
	byes := onlyByeAlerts(r)
	if err != nil || len(byes) != 1 || byes[0].At != lock ||
		byes[0].Urgency != controlapp.AlertUrgency(a.core.MovesNow(), lock) {
		t.Fatalf("next week's held lock: %+v %v", byes, err)
	}
}
