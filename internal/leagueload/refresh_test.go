package leagueload

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type refreshHarness struct {
	server    *history.Store
	league    string
	commish   string
	export    mflsource.Export
	football  string
	now       time.Time
	extraPlay []players.RawPlayer
}

func newRefreshHarness(t *testing.T) *refreshHarness {
	t.Helper()
	workspace, server := store(t, "workspace.db"), store(t, "server.db")
	h := &refreshHarness{
		server:  server,
		commish: "acct." + uuid.NewString(),
		now:     migrationtest.ReadAt,
	}
	clock := func() time.Time { return h.now }
	workspace.SetClock(clock)
	server.SetClock(clock)
	export := migrationtest.Synthetic("Synthetic NFL", migrationtest.Specs()["8 teams"])
	export.Schedule = mustSchedule(t)
	h.export = export
	res, err := Load(t.Context(), Request{
		MFLLeague:    export.LeagueID,
		Commissioner: governancePrincipal(h.commish),
		Workspace:    workspace,
		Server:       func(context.Context, string) (*history.Store, error) { return server, nil },
		Source: func(leaguerules.RuleSet) (leaguemigration.Source, error) {
			return &migrationtest.Source{Export: h.export}, nil
		},
		Football: h.buildFootball,
		Now:      clock,
	})
	if err != nil {
		t.Fatal(err)
	}
	h.league = res.League
	h.football = res.Football
	h.now = h.now.Add(time.Hour)
	return h
}

func (h *refreshHarness) buildFootball(_ context.Context, season int) ([]byte, string, error) {
	list := make([]players.RawPlayer, 0, len(h.export.Players)+len(h.extraPlay))
	for id := range h.export.Players {
		list = append(list, players.RawPlayer{
			ID: id, Name: "Player " + id, Position: "WR", Team: "DAL", DraftYear: "2020",
		})
	}
	list = append(list, h.extraPlay...)
	at := time.Date(season, 9, 1, 0, 0, 0, 0, time.UTC)
	weeks := []leagueweek.Week{
		{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(240 * time.Hour)}}},
		{Number: 2, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(408 * time.Hour)}}},
	}
	return BuildFootball(list, weeks, season, h.now)
}

func mustSchedule(t *testing.T) []leagueschedule.RawScheduleWeek {
	t.Helper()
	sched, err := leagueschedule.Parse([]byte(`{"schedule":{"weeklySchedule":[` +
		`{"week":"1","matchup":{"franchise":[` +
		`{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}},` +
		`{"week":"2","matchup":{"franchise":[` +
		`{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}}` +
		`]}}`))
	if err != nil {
		t.Fatal(err)
	}
	return sched
}

func (h *refreshHarness) exec(
	t *testing.T, raw commissioner.RawCommand, principal string,
) commissioner.Receipt {
	t.Helper()
	recs, err := leagueservice.Records(t.Context(), h.server, h.league, "test")
	if err != nil {
		t.Fatal(err)
	}
	raw.League = h.league
	raw.Expected = int64(len(recs))
	receipt, err := h.server.ExecuteCommissioner(t.Context(), raw, principal, nil)
	if err != nil {
		t.Fatalf("execute %s: %v", raw.Command, err)
	}
	h.now = h.now.Add(time.Minute)
	return receipt
}

func (h *refreshHarness) source(_ string, _ leaguerules.RuleSet) (leaguemigration.Source, error) {
	return &migrationtest.Source{Export: h.export}, nil
}

func TestRefreshDryRunCommitAndIdempotency(t *testing.T) {
	h := newRefreshHarness(t)
	capVal := json.RawMessage(strconv.FormatInt(210000000000, 10))
	h.exec(t, commissioner.RawCommand{
		Command: commissioner.RulesSet, Request: "edit-cap",
		Reason: "Commissioner adjusts cap", Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Field: "cap.amount", Value: capVal},
	}, h.commish)

	// Mutate MFL export: contract edit + completed week 1 MFL result.
	h.export.Rosters = slices.Clone(h.export.Rosters)
	h.export.Rosters[0].Salary = "12.50"
	var err error
	h.export.Results, err = weeklyresults.Parse([]byte(`{"allWeeklyResults":{"weeklyResults":[` +
		`{"week":"1","matchup":{"regularSeason":"1","franchise":[` +
		`{"id":"0001","isHome":"1","score":"110.50","result":"W"},` +
		`{"id":"0002","isHome":"0","score":"98.25","result":"L"}]}}` +
		`]}}`))
	if err != nil {
		t.Fatal(err)
	}

	beforeRecs, err := leagueservice.Records(t.Context(), h.server, h.league, "before")
	if err != nil {
		t.Fatal(err)
	}
	dry, err := Refresh(t.Context(), RefreshRequest{
		League:       h.league,
		Commissioner: governancePrincipal(h.commish),
		Server:       h.server,
		CurrentPin:   h.football,
		DryRun:       true,
		Source:       h.source,
		Football:     h.buildFootball,
		Now:          func() time.Time { return h.now },
	})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if !dry.DryRun || dry.Summary.Contracts != 1 || !dry.Summary.ScheduleUpdated ||
		dry.Summary.RecordedWeeks != 1 || dry.Summary.CutoverWeek != 2 {
		t.Fatalf("unexpected dry summary: %+v", dry.Summary)
	}
	if dry.LeagueName != "Synthetic NFL" {
		t.Fatalf("LeagueName = %q, want Synthetic NFL", dry.LeagueName)
	}
	afterDryRecs, err := leagueservice.Records(t.Context(), h.server, h.league, "after-dry")
	if err != nil {
		t.Fatal(err)
	}
	if len(afterDryRecs) != len(beforeRecs) {
		t.Fatalf("dry run modified ledger: %d -> %d", len(beforeRecs), len(afterDryRecs))
	}

	// Real commit applies the exact same summary and preserves the commissioner cap edit.
	applied, err := Refresh(t.Context(), RefreshRequest{
		League:       h.league,
		Commissioner: governancePrincipal(h.commish),
		Server:       h.server,
		CurrentPin:   h.football,
		DryRun:       false,
		Source:       h.source,
		Football:     h.buildFootball,
		Now:          func() time.Time { return h.now },
	})
	if err != nil {
		t.Fatalf("commit refresh: %v", err)
	}
	if applied.DryRun || applied.Summary != dry.Summary {
		t.Fatalf("applied summary %+v != dry summary %+v", applied.Summary, dry.Summary)
	}
	afterRecs, err := leagueservice.Records(t.Context(), h.server, h.league, "after-commit")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := commissioner.RulesNow(afterRecs, h.now)
	if err != nil {
		t.Fatal(err)
	}
	if rules.Cap.Amount.Value != 210000000000 {
		t.Fatalf("commissioner cap edit overwritten: %d", rules.Cap.Amount.Value)
	}

	// Second run later with same mirror is a 0-change no-op (even with a newer clock).
	h.now = h.now.Add(2 * time.Hour)
	again, err := Refresh(t.Context(), RefreshRequest{
		League:       h.league,
		Commissioner: governancePrincipal(h.commish),
		Server:       h.server,
		CurrentPin:   applied.Football,
		DryRun:       false,
		Source:       h.source,
		Football:     h.buildFootball,
		Now:          func() time.Time { return h.now },
	})
	if err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if again.Commands != 0 || again.FootballChanged || again.Football != applied.Football {
		t.Fatalf("second run was not a no-op: %+v", again)
	}
}

func TestRefreshStopsOnOpenTradeWithFranchiseNames(t *testing.T) {
	h := newRefreshHarness(t)
	enableTrading(t, h)
	recs, err := leagueservice.Records(t.Context(), h.server, h.league, "picks")
	if err != nil {
		t.Fatal(err)
	}
	app, err := leaguemigration.ReadApp(h.league, recs, h.now)
	if err != nil {
		t.Fatal(err)
	}
	pick1, pick2 := findTwoTradePicks(t, app.Snapshot.Picks, app.Snapshot.Season)
	f1, f2 := pick1.Owner, pick2.Owner
	owner1, owner2 := "acct."+uuid.NewString(), "acct."+uuid.NewString()
	for i, seat := range [][2]string{{f1, owner1}, {f2, owner2}} {
		h.exec(t, commissioner.RawCommand{
			Command: "commish.seat.replace", Request: "seat-" + strconv.Itoa(i),
			Reason: "Seat owner", Effective: "now", Visibility: "league",
			Args: commissioner.RawArgs{People: &commissioner.RawPeople{
				Franchise: seat[0], Principal: seat[1],
			}},
		}, h.commish)
	}
	h.exec(t, commissioner.RawCommand{
		Command: commissioner.TradeOffer, Request: "offer-1",
		Reason: "Offer trade", Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{
			Franchise: f1,
			Parties:   []string{f1, f2},
			Assets: []commissioner.TradeAsset{
				{
					From: f1, To: f2,
					Season: pick1.Season, Round: pick1.Round, Original: pick1.Original,
				},
				{
					From: f2, To: f1,
					Season: pick2.Season, Round: pick2.Round, Original: pick2.Original,
				},
			},
		}},
	}, owner1)

	seats := app.Directory.Seats()
	name1, name2 := seats.Of(f1).Name, seats.Of(f2).Name
	for _, dryRun := range []bool{true, false} {
		_, err := Refresh(t.Context(), RefreshRequest{
			League:       h.league,
			Commissioner: governancePrincipal(h.commish),
			Server:       h.server,
			CurrentPin:   h.football,
			DryRun:       dryRun,
			Source:       h.source,
			Football:     h.buildFootball,
			Now:          func() time.Time { return h.now },
		})
		var stopped *StoppedTradesError
		if !errors.As(err, &stopped) || len(stopped.Offers) != 1 {
			t.Fatalf("dryRun=%t: want StoppedTradesError with 1 offer, got %v", dryRun, err)
		}
		if !slices.Contains(stopped.Offers[0].Franchises, name1) ||
			!slices.Contains(stopped.Offers[0].Franchises, name2) {
			t.Fatalf("stopped offer missing names: %+v (want %q, %q)", stopped.Offers[0], name1, name2)
		}
		if strings.Contains(err.Error(), f1) || strings.Contains(err.Error(), f2) {
			t.Fatalf("error leaked raw franchise IDs: %v", err)
		}
	}
}

func enableTrading(t *testing.T, h *refreshHarness) {
	t.Helper()
	for _, rule := range [][2]string{
		{"trades.review", `"commissioner"`},
		{"trades.decideAt", `1`},
		{"trades.futureYears", `3`},
		{"trades.rounds", `{"min":1,"max":5}`},
		{"trades.rosterGate", `false`},
		{"trades.offerExpiry", `172800000000000`},
		{"trades.maximumParties", `2`},
		{"cap.enforcement", `"hard"`},
		{"cap.quantum", `100000`},
		{"calendar.timezone", `"UTC"`},
		{"calendar.tradeDeadline", `"2026-12-01T00:00:00Z"`},
	} {
		h.exec(t, commissioner.RawCommand{
			Command: commissioner.RulesSet, Request: "rule-" + rule[0],
			Reason: "Configure trade rule", Effective: "now", Visibility: "league",
			Args: commissioner.RawArgs{Field: rule[0], Value: json.RawMessage(rule[1])},
		}, h.commish)
	}
	h.exec(t, commissioner.RawCommand{
		Command: commissioner.WindowSet, Request: "open-trades",
		Reason: "Open trade window", Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Window: &commissioner.RawWindowArgs{
			ID: "trades", Start: &commissioner.RawDate{Month: 1, Day: 1},
			End:   &commissioner.RawDate{Month: 12, Day: 31},
			Opens: []leaguerules.Action{"trade"},
		}},
	}, h.commish)
}

func findTwoTradePicks(
	t *testing.T, picks []commissioner.NativePick, season int,
) (commissioner.NativePick, commissioner.NativePick) {
	t.Helper()
	var p1, p2 commissioner.NativePick
	for _, pk := range picks {
		if pk.Season <= season || pk.Original == "" || pk.Player != "" || pk.Skipped {
			continue
		}
		if p1.Original == "" {
			p1 = pk
			continue
		}
		if pk.Owner != p1.Owner {
			p2 = pk
			return p1, p2
		}
	}
	t.Fatal("could not find two future picks owned by distinct franchises")
	return p1, p2
}
