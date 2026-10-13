package main

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func byeFixtureApp(t *testing.T) (*App, *seasonTransport) {
	t.Helper()
	a, transport, _ := tradeDraftApp(t)
	a.targetSnapshot.Rosters.Provenance.Freshness.State = domain.FreshLive
	for i := range a.targetSnapshot.Players.Value {
		p := &a.targetSnapshot.Players.Value[i]
		p.Team = "NYJ"
		if p.Position == domain.PosK {
			p.Team, p.Name = "DET", "Bates"
		}
		if p.Position == domain.PosQB {
			p.Team = "PIT"
		}
	}
	a.week.Number = 6
	holdPersonalBody(t, a, "nflByeWeeks", `{"nflByeWeeks":{"team":[
		{"id":"DET","bye_week":"6"},{"id":"PIT","bye_week":"8"},{"id":"NYJ","bye_week":"9"}]}}`)
	return a, transport
}

func onlyByeAlerts(r AlertReading) []Alert {
	return slices.DeleteFunc(slices.Clone(r.Alerts), func(a Alert) bool { return a.Kind != "bye" })
}

func TestByeAlertsHeldWeekSixAndUnavailable(t *testing.T) {
	a, transport := byeFixtureApp(t)
	r, err := a.TargetAlerts("0025")
	if err != nil {
		t.Fatal(err)
	}
	byes := onlyByeAlerts(r)
	lock := a.movesNow().Add(24 * time.Hour)
	if len(byes) != 1 || byes[0].Title != "Week 6: no kicker available" ||
		!strings.Contains(byes[0].Detail, "Bates") || !strings.Contains(byes[0].Detail, "0 available; needs 1") ||
		byes[0].At != lock || byes[0].Urgency != alertUrgency(a.movesNow(), lock) ||
		byes[0].Node != "hq" || byes[0].Workspace != "lineup-and-roster" {
		t.Fatalf("week six shortfall: %+v", byes)
	}
	if len(transport.requests) != 0 {
		t.Fatal("bye alerts fetched from MFL")
	}
	a.history = nil
	r, err = a.TargetAlerts("0025")
	_, unavailable := alertKinds(r)
	if err != nil || len(onlyByeAlerts(r)) != 0 || !slices.Contains(unavailable, "bye") {
		t.Fatalf("bye data not held: %+v %v", r, err)
	}
}

func TestByeAlertsHorizonAndMissingRoster(t *testing.T) {
	a, _ := byeFixtureApp(t)
	for _, week := range []int{5, 7, 8, 10} {
		a.week.Number = week
		r, err := a.TargetAlerts("0025")
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
			if !slices.ContainsFunc(byes, func(a Alert) bool {
				return a.Title == "Week 8: no quarterback available"
			}) {
				t.Fatalf("week eight quarterback: %+v", byes)
			}
		}
	}
	a.week.Number = 6
	a.targetSnapshot.Rosters.Value = nil
	r, err := a.TargetAlerts("0025")
	_, unavailable := alertKinds(r)
	if err != nil || len(onlyByeAlerts(r)) != 0 || !slices.Contains(unavailable, "bye") {
		t.Fatalf("roster not held: %+v %v", r, err)
	}
}

func TestByeAlertsMissingIdentityAndLock(t *testing.T) {
	a, _ := byeFixtureApp(t)
	a.week = leagueweek.Week{Number: 6}
	r, err := a.TargetAlerts("0025")
	_, unavailable := alertKinds(r)
	byes := onlyByeAlerts(r)
	if err != nil || len(byes) != 1 || !byes[0].At.IsZero() || byes[0].Urgency != "U0" ||
		!slices.Contains(unavailable, "bye") {
		t.Fatalf("unknown lock: %+v %v", r, err)
	}
	a.targetSnapshot.Players.Value = nil
	r, err = a.TargetAlerts("0025")
	_, unavailable = alertKinds(r)
	if err != nil || len(onlyByeAlerts(r)) != 0 || !slices.Contains(unavailable, "bye") {
		t.Fatalf("unknown position/team: %+v %v", r, err)
	}
}

func TestByeAlertNextWeekHeldLock(t *testing.T) {
	a, _ := byeFixtureApp(t)
	a.week.Number = 5
	lock := a.movesNow().Add(48 * time.Hour)
	holdPersonalBody(t, a, "nflSchedule&W=6", fmt.Sprintf(`{"nflSchedule":{"week":"6","matchup":{
		"kickoff":"%d","gameSecondsRemaining":"3600","team":[{"id":"NYJ"},{"id":"PIT"}]}}}`, lock.Unix()))
	r, err := a.TargetAlerts("0025")
	byes := onlyByeAlerts(r)
	if err != nil || len(byes) != 1 || byes[0].At != lock ||
		byes[0].Urgency != alertUrgency(a.movesNow(), lock) {
		t.Fatalf("next week's held lock: %+v %v", byes, err)
	}
}
