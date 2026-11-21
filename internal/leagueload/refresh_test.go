package leagueload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
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

func TestRefreshPreservesOfficialStatsIdentitiesAndScheduleWeeks(t *testing.T) {
	h := newRefreshHarness(t)
	body, err := h.server.Body(t.Context(), h.football)
	if err != nil {
		t.Fatal(err)
	}
	var enriched history.RawFootballEvidence
	if err := json.Unmarshal(body, &enriched); err != nil {
		t.Fatal(err)
	}

	// Pick one player from the mirror and enrich with GSIS/PFR/Sleeper/NFLRosterSeasons/Designation/CareerGames.
	var targetID string
	for id := range enriched.Players {
		if targetID == "" || id < targetID {
			targetID = id
		}
	}
	seasons := []int{2024, 2025, 2026}
	target := enriched.Players[targetID]
	target.Identity = "00-0023459"
	target.PFR = "RodgAa00"
	target.Sleeper = "4046"
	target.NFLRosterSeasons = &seasons
	target.Facts.Designation = leaguerules.Out
	target.Facts.DesignationHeld = true
	target.Facts.CareerGames = 42
	target.Facts.CareerGamesHeld = true
	enriched.Players[targetID] = target

	// Also add a non-mirror player (e.g., retired/historical player) that must not be dropped.
	enriched.Players["9899"] = history.RawFootballPlayer{
		Identity: "00-0023853",
		Facts:    nativecheck.PlayerFacts{Name: "Historical Kicker", Position: leaguerules.PK, Team: "BAL"},
	}

	// Mark week 1 final and add week 3 (as refresh-nfl-schedule does).
	enriched.Weeks[0].Games[0].Final = true
	at := time.Date(enriched.Season, 9, 1, 0, 0, 0, 0, time.UTC)
	enriched.Weeks = append(enriched.Weeks, leagueweek.Week{
		Number: 3,
		Games:  []leagueweek.Game{{Teams: [2]string{"KC", "BUF"}, Kickoff: at.Add(576 * time.Hour)}},
	})

	// Attach real split nflverse official stats (week part imported separately).
	weeksCSV, err := os.ReadFile("../fantasyscore/source/testdata/weeks.csv")
	if err != nil {
		t.Fatal(err)
	}
	weekLines := strings.Split(strings.TrimSpace(string(weeksCSV)), "\n")
	weeklyStr := strings.ReplaceAll(strings.Join(weekLines[:3], "\n")+"\n", ",2025,", ",2026,")
	playsStr := "game_id,play_id,season,week,season_type,passer_player_id,receiver_player_id," +
		"rusher_player_id,passing_yards,receiving_yards,rushing_yards,touchdown,td_player_id," +
		"td_team,return_team,interception,punt_attempt,kickoff_attempt,play_deleted,two_point_attempt\n" +
		"2025_01_PIT_NYJ,1,2026,1,REG,00-0023459,,,40,NA,NA,0,,,,0,0,0,0,0\n" +
		"2025_01_BAL_BUF,1,2026,1,REG,,,,NA,NA,NA,0,,,,0,0,0,0,0\n"
	partBody, err := json.Marshal(history.RawFootballWeek{
		Version: 1, Season: enriched.Season, Week: 1, Weekly: weeklyStr, Plays: playsStr,
	})
	if err != nil {
		t.Fatal(err)
	}
	partSum := sha256.Sum256(partBody)
	partDigest := hex.EncodeToString(partSum[:])
	if err := h.server.ImportFootballWeek(t.Context(), partBody, partDigest); err != nil {
		t.Fatal(err)
	}
	enriched.Stats = []history.RawFootballStats{{
		AsOf:  enriched.AsOf.Add(-time.Minute),
		Parts: []string{partDigest},
	}}
	enrichedBody, err := json.Marshal(enriched)
	if err != nil {
		t.Fatal(err)
	}
	enrichedSum := sha256.Sum256(enrichedBody)
	enrichedDigest := hex.EncodeToString(enrichedSum[:])
	if err := h.server.ImportFootball(t.Context(), enrichedBody, enrichedDigest); err != nil {
		t.Fatal(err)
	}
	h.football = enrichedDigest

	// 1. Refresh with unchanged mirror must recognize football is unchanged and preserve enrichedDigest.
	unchanged, err := Refresh(t.Context(), RefreshRequest{
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
		t.Fatalf("refresh unchanged mirror: %v", err)
	}
	if unchanged.FootballChanged || unchanged.Football != enrichedDigest {
		t.Fatalf("unchanged mirror overwrote enriched football: changed=%t got=%s want=%s",
			unchanged.FootballChanged, unchanged.Football, enrichedDigest)
	}

	// 2. Add a new player in the mirror: refresh merges mirror update while preserving all server data.
	h.extraPlay = []players.RawPlayer{{
		ID: "9950", Name: "Mirror New Rookie", Position: "RB", Team: "KC", DraftYear: "2026",
	}}
	h.now = h.now.Add(time.Hour)
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
		t.Fatalf("refresh with new mirror player: %v", err)
	}
	if !applied.FootballChanged || applied.Football == enrichedDigest {
		t.Fatalf("expected updated merged football digest: %+v", applied)
	}
	mergedBody, err := h.server.Body(t.Context(), applied.Football)
	if err != nil {
		t.Fatal(err)
	}
	var merged history.RawFootballEvidence
	if err := json.Unmarshal(mergedBody, &merged); err != nil {
		t.Fatal(err)
	}
	if _, ok := merged.Players["9950"]; !ok {
		t.Fatal("new mirror player 9950 missing from merged football evidence")
	}
	if _, ok := merged.Players["9899"]; !ok {
		t.Fatal("non-mirror player 9899 was lost during refresh")
	}
	gotTarget := merged.Players[targetID]
	if gotTarget.Identity != "00-0023459" || gotTarget.PFR != "RodgAa00" || gotTarget.Sleeper != "4046" ||
		gotTarget.NFLRosterSeasons == nil || !slices.Equal(*gotTarget.NFLRosterSeasons, seasons) ||
		!gotTarget.Facts.DesignationHeld || gotTarget.Facts.Designation != leaguerules.Out ||
		!gotTarget.Facts.CareerGamesHeld || gotTarget.Facts.CareerGames != 42 {
		t.Fatalf("enriched player metadata lost during refresh: %+v", gotTarget)
	}
	if len(merged.Stats) != 1 || !slices.Equal(merged.Stats[0].Parts, []string{partDigest}) {
		t.Fatalf("official nflverse stats lost during refresh: %+v", merged.Stats)
	}
	pinned, err := h.server.LoadFootball(t.Context(), applied.Football)
	if err != nil {
		t.Fatalf("load merged football evidence: %v", err)
	}
	heldStats, err := pinned.CanonicalPlayerWeekStats(
		t.Context(), source.RawNativeScope{Player: targetID, Season: enriched.Season, Week: 1},
	)
	if err != nil || len(heldStats) != 1 || heldStats[0].Week == nil {
		t.Fatalf("official stats not readable after refresh: %+v %v", heldStats, err)
	}
	if len(merged.Weeks) != 3 || !merged.Weeks[0].Games[0].Final || merged.Weeks[2].Number != 3 {
		t.Fatalf("final week 1 or server-held week 3 lost during refresh: %+v", merged.Weeks)
	}

	// 3. Running again with the merged pin (or empty CurrentPin resolved from server store) is a no-op.
	h.now = h.now.Add(time.Hour)
	for _, pin := range []string{applied.Football, ""} {
		again, err := Refresh(t.Context(), RefreshRequest{
			League:       h.league,
			Commissioner: governancePrincipal(h.commish),
			Server:       h.server,
			CurrentPin:   pin,
			DryRun:       false,
			Source:       h.source,
			Football:     h.buildFootball,
			Now:          func() time.Time { return h.now },
		})
		if err != nil || again.FootballChanged || again.Football != applied.Football {
			t.Fatalf("second refresh (pin=%q) was not a no-op: %+v %v", pin, again, err)
		}
	}
}

func TestRefreshDoesNotImportFootballWhenBatchFails(t *testing.T) {
	h := newRefreshHarness(t)
	h.extraPlay = []players.RawPlayer{{
		ID: "9960", Name: "Uncommitted Rookie", Position: "WR", Team: "DAL", DraftYear: "2026",
	}}
	req := RefreshRequest{
		League:       h.league,
		Commissioner: governancePrincipal(h.commish),
		Server:       h.server,
		CurrentPin:   h.football,
		Source:       h.source,
		Football:     h.buildFootball,
		Now:          func() time.Time { return h.now },
	}
	fb, err := prepareMirrorFootball(t.Context(), req, h.export.Season)
	if err != nil {
		t.Fatal(err)
	}
	if !fb.changed {
		t.Fatal("expected changed football evidence for batch failure test")
	}
	sibling := h.server.Sibling()
	sibling.SetLineupEnvironment(fb.env)
	badCmd := commissioner.RawCommand{
		League:     h.league,
		Command:    commissioner.CorrectRoster,
		Request:    "mfl-refresh.bad-expected",
		Reason:     "Simulated stale revision",
		Effective:  "now",
		Visibility: "league",
		Expected:   999999,
		Args: commissioner.RawArgs{Correction: &commissioner.RawCorrection{
			Franchise: "franchise.01", Player: "999999", Action: "remove",
		}},
	}
	err = commitServerRefresh(t.Context(), req, sibling, fb, []commissioner.RawCommand{badCmd})
	if err == nil || !strings.Contains(err.Error(), "nothing was changed") {
		t.Fatalf("expected batch failure error, got: %v", err)
	}
	if _, loadErr := h.server.LoadFootball(t.Context(), fb.digest); !errors.Is(loadErr, history.ErrNoBody) {
		t.Fatalf("football evidence was imported before ledger.Batch failed: loadErr=%v", loadErr)
	}
}
