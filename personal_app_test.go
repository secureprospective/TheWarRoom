package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

func holdPersonalBody(t *testing.T, a *App, export, body string) {
	t.Helper()
	var gz bytes.Buffer
	writer := gzip.NewWriter(&gz)
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(body))
	if err := a.core.History.Record(a.core.Ctx, archive.Fetch{Source: "league_archive", Status: 200,
		URL:    "https://fixture.invalid/2026/export?L=14432&TYPE=" + export,
		SHA256: hex.EncodeToString(hash[:]), Size: int64(len(body)), Gzip: gz.Bytes(), FetchedAt: a.core.MovesNow()}); err != nil {
		t.Fatal(err)
	}
}

func TestIRViolationAndHeldEligibilityBacklog(t *testing.T) {
	t.Parallel()
	a, transport, _ := tradeDraftApp(t)
	a.core.TargetSnapshot.Rosters.Provenance.Freshness.State = domain.FreshLive
	a.core.SeasonInjuries.FetchedAt = a.core.MovesNow()

	for i := range a.core.TargetSnapshot.Rosters.Value[0].Players {
		p := &a.core.TargetSnapshot.Rosters.Value[0].Players[i]
		p.RosterStatus = domain.RosterActive
		if i < 2 {
			p.RosterStatus = domain.RosterIR
			a.core.SeasonInjuries.Value = append(a.core.SeasonInjuries.Value, leaguefeed.Injury{ID: p.ID, Status: "IR-PUP"})
		}
	}
	r, err := a.TargetAlerts("0025")
	_, unavailable := alertKinds(r)
	if err != nil || strings.Contains(strings.Join(unavailable, ","), "ir") {
		t.Fatalf("held eligibility unavailable: %+v %v", r, err)
	}
	a.core.SeasonInjuries.Value[0].Status = "Questionable"
	a.core.SeasonInjuries.Value[1].Status = "Suspended"
	r, err = a.TargetAlerts("0025")
	if err != nil {
		t.Fatal(err)
	}
	violations := 0
	for _, alert := range r.Alerts {
		if alert.Kind != "ir" {
			continue
		}
		violations++
		if alert.Detail == "" || alert.PlayerID == "" || alert.Title == "" || alert.Node != "hq" {
			t.Fatalf("IR reason and route: %+v", alert)
		}
	}
	if violations != 2 {
		t.Fatalf("one per player, activation eligibility is not classification: %+v", r)
	}
	a.core.SeasonInjuries.RefreshError = "offline"
	r, err = a.TargetAlerts("0025")
	if err != nil {
		t.Fatal(err)
	}
	for _, alert := range r.Alerts {
		if alert.Kind == "ir" {
			t.Fatalf("stale classification asserted: %+v", alert)
		}
	}
	if len(transport.requests) != 0 {
		t.Fatal("alerts called MFL")
	}
}

func TestPersonalHeldReportsAndReadOnlyHandOff(t *testing.T) {
	t.Parallel()
	a, transport, _ := tradeDraftApp(t)
	a.core.TargetSnapshot.Franchises.Value[0].Name = "My franchise"
	if err := personal.New(a.core.Pools).Initialize(a.core.Ctx); err != nil {
		t.Fatal(err)
	}
	player := a.core.TargetSnapshot.Players.Value[0].ID.String()
	if err := a.TargetPersonalChange(personal.RawChange{Mine: "0025", Kind: "watch", Entity: "player", ID: player, Action: "add"}); err != nil {
		t.Fatal(err)
	}
	holdPersonalBody(t, a, "tradeBait", `{"tradeBaits":{"tradeBait":{"franchise_id":"0025","willGiveUp":"`+player+`","inExchangeFor":"Future picks"}}}`)
	a.core.SeasonInjuries.FetchedAt = time.Date(2026, 10, 12, 15, 24, 15, 0, time.UTC)
	r, err := a.TargetPersonal("0025")
	if err != nil || len(r.Players) != 1 || !strings.Contains(r.Players[0].Status, " · as of ") ||
		strings.Contains(r.Players[0].Status, "T15:24:15Z") {
		t.Fatalf("a watched player's status reads in local time: %+v %v", r.Players, err)
	}
	if err != nil || len(r.Entries) != 1 || len(r.Players) != 1 || len(r.Bait.Rows) != 1 ||
		r.Bait.Rows[0].Cells[0] != "My franchise" || r.Bait.Rows[0].Cells[2] != "Future picks" {
		t.Fatalf("held personal reading: %+v %v", r, err)
	}
	var opened string
	a.core.OpenURL = func(_ context.Context, url string) { opened = url }
	a.core.MFLClient, err = mfl.New("fixture.invalid", 10000, mfl.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.TargetPersonalHandOff("bait"); err != nil || !strings.Contains(opened, "&O=133") {
		t.Fatalf("bait handoff: %s %v", opened, err)
	}
	if err := a.TargetPersonalHandOff("draft"); err != nil || !strings.Contains(opened, "&O=129") {
		t.Fatalf("draft handoff: %s %v", opened, err)
	}
	if err := a.TargetPersonalHandOff("submit"); err == nil {
		t.Fatal("invalid handoff accepted")
	}
	if len(transport.requests) != 0 {
		t.Fatal("list submitted or fetched")
	}
	body, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"entries"`) || !strings.Contains(string(body), `"players"`) {
		t.Fatalf("wire: %s", body)
	}
}

func TestCalendarMergeHeldImport(t *testing.T) {
	t.Parallel()
	a, transport, _ := tradeDraftApp(t)
	now := a.core.MovesNow()
	next := strconv.FormatInt(now.Add(72*time.Hour).Unix(), 10)
	holdPersonalBody(t, a, "calendar", `{"calendar":{"event":[`+
		`{"end_time":"","type":"DRAFT_START","title":"Rookie draft","id":"8352818","start_time":"`+
		strconv.FormatInt(now.Add(-24*time.Hour).Unix(), 10)+`"},`+
		`{"end_time":"","type":"TRADE_DEADLINE","title":"Trade deadline","id":"8352819","start_time":"`+next+`"}]}}`)
	clock := snapshot.Sourced[leagueclock.Reading]{Value: leagueclock.Clock(now, leagueclock.Inputs{
		Events: []leagueclock.Event{{ID: "local", Kind: "LINEUP_LOCK", Status: "PLANNED", At: now.Add(time.Hour)}}})}
	a.mergeMFLCalendar(a.core.Ctx, now, &clock)
	ids := []string{}
	for _, d := range clock.Value.Deadlines {
		ids = append(ids, d.ID)
		// The wire contract (frontend data/parse.ts deadline): dated exactly when urgency is not U0.
		if (d.At == nil) != (d.Urgency == leagueclock.U0) {
			t.Fatalf("deadline %s breaks the dated/urgency contract: %+v", d.ID, d)
		}
		if d.ID == "mfl-calendar:8352819" && (!strings.Contains(d.Label, "Trade deadline") || d.Urgency != leagueclock.U1) {
			t.Fatalf("upcoming calendar event: %+v", d)
		}
	}
	if slices.Contains(ids, "mfl-calendar:8352818") || !slices.Contains(ids, "mfl-calendar:8352819") || len(ids) != 2 {
		t.Fatalf("a past calendar event is not a deadline; the upcoming one is: %v", ids)
	}
	if len(transport.requests) != 0 {
		t.Fatalf("calendar merge fetched: %v", transport.requests)
	}
}

func TestRosterPointsFromHeldPeriods(t *testing.T) {
	t.Parallel()
	a, transport, _ := tradeDraftApp(t)
	id := a.core.TargetSnapshot.Players.Value[0].ID.String()
	holdPersonalBody(t, a, "playerScores", `{"playerScores":{"week":"YTD","playerScore":{"id":"`+id+`","score":"134.80"}}}`)
	holdPersonalBody(t, a, "playerScores", `{"playerScores":{"week":"4","playerScore":{"id":"`+id+`","score":"0.00"}}}`)
	holdPersonalBody(t, a, "nflByeWeeks", `{"nflByeWeeks":{"team":{"id":"BUF","bye_week":"7"}}}`)
	reports, err := a.TargetRosterReports("0025")
	if err != nil || len(reports) != 2 {
		t.Fatalf("reports: %+v %v", reports, err)
	}
	name := a.core.TargetSnapshot.Players.Value[0].Name
	found := false
	for _, row := range reports[0].Rows {
		if row.Cells[0] == name {
			found = row.Cells[3] == "134.80" && row.Cells[4] == "0.00"
		}
	}
	if !found {
		t.Fatalf("periods mixed, zero treated as missing or row omitted: %+v", reports[0])
	}
	if len(transport.requests) != 0 {
		t.Fatal("roster report fetched")
	}
}

func TestNFLDraftFromDetailedPlayers(t *testing.T) {
	t.Parallel()
	a, transport, _ := tradeDraftApp(t)
	holdPersonalBody(t, a, "players", `{"players":{"player":[
 {"id":"17462","name":"Mendoza, Fernando","position":"QB","team":"LVR","draft_year":"2026","draft_round":"1","draft_pick":"1"},
 {"id":"17463","name":"Simpson, Ty","position":"QB","team":"LAR","draft_year":"2026","draft_round":"1","draft_pick":"13"},
 {"id":"0816","name":"Old rookie","position":"WR","team":"IND","draft_year":"2025","draft_round":"1","draft_pick":"2"}]}}`)
	r, err := draftAPI(a).TargetNFLDraft()
	if err != nil || len(r.Rows) != 2 || r.Rows[0].Cells[0] != "Mendoza, Fernando" ||
		r.Rows[1].Cells[4] != "13" || r.ID != "M-087" {
		t.Fatalf("NFL draft: %+v %v", r, err)
	}
	if len(transport.requests) != 0 {
		t.Fatal("NFL draft refetched players")
	}
}
