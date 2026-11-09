package composition

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/commissioner/commissionertest"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/mflref/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type testCommissionerSession struct{ principal string }

func (s testCommissionerSession) Principal() (string, error) { return s.principal, nil }

func commissionerTestApp(t *testing.T) *App {
	t.Helper()
	store := commissionertest.History(t)
	app := &App{marketHeld: &marketapp.Held{}, raceHeld: &raceapp.Held{}, core: &core.State{Ctx: t.Context(), History: store, Started: make(chan struct{}),
		CommissionerSession: testCommissionerSession{"creator"}}}
	close(app.core.Started)
	return app
}

func executeCommissionerTest(t *testing.T, app *App, raw commissioner.RawCommand) commissioner.Receipt {
	t.Helper()
	return commissionertest.Execute(t, newCommissionerAPI(app).TargetCommissionerExecute, raw)
}

func historicalLeague(t *testing.T, app *App, raw commissioner.RawCommand) commissioner.Receipt {
	t.Helper()
	if raw.Command != commissioner.Create {
		t.Fatal("historical fixture needs a Create")
	}
	principal, err := app.commissionerPrincipal()
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := app.core.History.ExecuteCommissioner(t.Context(), raw, principal, app.core.CommissionerAuthorizer)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

type lineupRulesSource struct{ cfg league.RawConfig }

func (s lineupRulesSource) Fetch(context.Context) (league.RawConfig, error) { return s.cfg, nil }

func lineupTestRules(t *testing.T, a *App, bad bool) {
	t.Helper()
	body, err := os.ReadFile("internal/mflref/lineup/testdata/league.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		League struct {
			Starters struct {
				Count     string                         `json:"count"`
				IOP       string                         `json:"iop_starters"`
				IDP       string                         `json:"idp_starters"`
				Positions []struct{ Name, Limit string } `json:"position"`
			} `json:"starters"`
		} `json:"league"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	s := raw.League.Starters
	cfg := league.RawConfig{Source: "test:real league", Starters: league.Starters{
		Count: s.Count, IOPStarters: s.IOP, IDPStarters: s.IDP,
	}}
	for _, p := range s.Positions {
		cfg.Starters.Positions = append(cfg.Starters.Positions, league.PositionLimit{Name: p.Name, Limit: p.Limit})
	}
	if bad {
		cfg.Starters.Count = "unreadable"
	}
	if err := a.core.Rulebook.Initialize(a.core.Ctx, lineupRulesSource{cfg: cfg}); err != nil {
		t.Fatal(err)
	}
}

func lineupTestSnapshot(t *testing.T, a *App) []string {
	t.Helper()

	body, err := os.ReadFile("internal/mflref/lineup/testdata/directory-0025.json")
	if err != nil {
		t.Fatal(err)
	}
	var players []struct{ ID, Name, Position string }
	if err := json.Unmarshal(body, &players); err != nil {
		t.Fatal(err)
	}
	snap := snapshot.Snapshot{}
	snap.Franchises.Value = []snapshot.Franchise{{ID: "0025"}}
	roster := snapshot.Roster{FranchiseID: "0025"}
	for _, p := range players {
		pid, err := playerid.New(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		pos, _ := normalize.PositionFromMFL(p.Position)
		snap.Players.Value = append(snap.Players.Value, snapshot.Player{ID: pid, Position: pos, Name: p.Name})
	}
	body, err = os.ReadFile("internal/mflref/lineup/testdata/roster-0025.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Players []struct{ ID, Status string } `json:"player"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	for _, p := range raw.Players {
		pid, err := playerid.New(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		roster.Players = append(roster.Players, snapshot.RosterPlayer{
			ID: pid, RosterStatus: domain.RosterStatus(p.Status),
		})
	}

	ids := []string{
		"16150", "13133", "16617", "15754", "15761", "16195", "16641", "15798", "16846",
		"16264", "16694", "16734", "16303", "16230", "14892", "13813", "15836", "16267",
		"13322", "16460", "15850",
	}

	snap.Rosters.Value = []snapshot.Roster{roster}
	a.core.TargetSnapshot, a.core.HasTargetSnapshot = snap, true
	return ids
}

func lineupDraftApp(t *testing.T) (*App, *seasonTransport, []string) {
	t.Helper()
	a, transport, _, _ := isolatedSeasonTestApp(t)
	a.core.MyFranchise = "0025"
	lineupTestRules(t, a, false)
	ids := lineupTestSnapshot(t, a)
	now := time.Date(2026, 10, 7, 15, 0, 0, 0, time.UTC)
	a.core.MovesNow = func() time.Time { return now }
	a.core.MovesLog = func(string) {}
	a.core.MovesChanged = func(context.Context) {}
	a.core.Week = leagueweek.Week{Number: 5, Games: []leagueweek.Game{{Kickoff: now.Add(24 * time.Hour)}}}
	a.refreshSeasonLineups(a.core.Ctx)
	a.core.SeasonLineups.FetchedAt = now.Add(-time.Minute)
	transport.requests = nil
	return a, transport, ids
}

func quoteApp(t *testing.T) *legalityFixture {
	t.Helper()
	a := nativeSeedTestApp(t)
	path := filepath.Join(t.TempDir(), "quote.db")
	store, pools := openDirectoryHistory(t, path)
	a.core.History = store
	snap := state.MirrorSnapshot{Season: 2026}
	for _, p := range []struct {
		id, franchise string
		salary        domain.Money
		year          int
		status        domain.ContractStatus
	}{{"0099", "0002", 1_000_000_000, 2029, domain.CStatusUFA}, {"0098", "0002", 800_000_000, 2026, domain.CStatusUFA},
		{"0097", "0002", 50_000_000, 2026, domain.CStatusRFA}, {"0096", "0003", 500_000_000, 2028, domain.CStatusUFA},
		{"0095", "0003", 400_000_000, 2028, domain.CStatusUFA}, {"0094", "0003", 300_000_000, 2028, domain.CStatusUFA}} {
		snap.Players = append(snap.Players, state.MirrorPlayer{MFLID: p.id, FranchiseID: p.franchise,
			RosterStatus: domain.RosterActive, Salary: p.salary, ContractYear: p.year, ContractStatus: p.status})
	}
	if _, err := a.core.League.Replace(t.Context(), snap); err != nil {
		t.Fatal(err)
	}
	var raws []players.RawPlayer
	for id, position := range map[string]string{"0099": "QB", "0098": "QB", "0097": "WR", "0096": "QB",
		"0095": "QB", "0094": "QB"} {
		raws = append(raws, players.RawPlayer{ID: id, Name: "Player, " + id, Position: position, Team: "AAA", DraftYear: "2018"})
	}
	lookup, err := normalize.NewLookup(raws)
	if err != nil {
		t.Fatal(err)
	}
	a.core.Lookup, a.core.HasLookup = lookup, true
	seedNativeTest(t, a, createNativeSeedLeague(t, a, commissioner.PresetLegacy))
	f := &legalityFixture{app: a, path: path, pools: pools}
	f.set(t, "calendar.boundary", `"start_of_day"`, "now")
	f.openWindow(t, "contract-test", []leaguerules.Action{"cut", "buyout", "tag", "extension", "restructure",
		"rfa_tender", "resign"})
	return f
}

func (f *legalityFixture) openWindow(t *testing.T, id string, opens []leaguerules.Action) {
	t.Helper()
	houses, err := newCommissionerAPI(f.app).TargetNativeHouseRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	args := commissioner.RawWindowArgs{ID: id, Start: &commissioner.RawDate{Month: 10, Day: 1},
		End: &commissioner.RawDate{Month: 10, Day: 31}, Opens: opens}
	executeCommissionerTest(t, f.app, commissioner.RawCommand{League: legalityLeague,
		Command: commissioner.WindowSet, Args: commissioner.RawArgs{Window: &args}, Expected: houses.Revision,
		Request: fmt.Sprintf("window.%d", houses.Revision), Reason: "Open the contract desk",
		Effective: "now", Visibility: "league"})
}

func openDirectoryHistory(t *testing.T, path string) (*history.Store, *db.Pools) {
	t.Helper()
	p, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(p, reg)
	store.SetClock(func() time.Time { return time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC) })
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return store, p
}

const legalityLeague = "native.seed"

type legalityFixture struct {
	app   *App
	path  string
	pools *db.Pools
}

func (f *legalityFixture) set(t *testing.T, field, value, effective string) commissioner.Receipt {
	t.Helper()
	houses, err := newCommissionerAPI(f.app).TargetNativeHouseRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	return executeCommissionerTest(t, f.app, commissioner.RawCommand{League: legalityLeague,
		Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: field, Value: json.RawMessage(value)},
		Request: fmt.Sprintf("legality.%d", houses.Revision), Expected: houses.Revision,
		Reason: "Our league's ruling", Effective: effective, Visibility: "league"})
}

func nativeSeedTestApp(t *testing.T) *App {
	t.Helper()
	a := commissionerTestApp(t)
	p, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "mirror.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	a.core.Rulebook = rulebook.New(p)
	cfg := league.RawConfig{CurrentSeason: 2026}
	for i := 32; i >= 1; i-- {
		cfg.Franchises = append(cfg.Franchises, league.Franchise{ID: fmt.Sprintf("%04d", i),
			Name: fmt.Sprintf("Held team %d", i), Abbrev: fmt.Sprintf("H%d", i), Division: "division.1"})
	}
	if err := a.core.Rulebook.Initialize(t.Context(), lineupRulesSource{cfg: cfg}); err != nil {
		t.Fatal(err)
	}
	a.core.League = state.NewMirror(p, a.core.Rulebook)
	if err := a.core.League.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	a.core.Season = 2026
	snap := state.MirrorSnapshot{Season: a.core.Season, Adjustments: []state.Adjustment{
		{ID: "held-credit", FranchiseID: "0002", Amount: -12345, Description: "Held cap credit"}}}
	for i, status := range []domain.RosterStatus{domain.RosterActive, domain.RosterIR, domain.RosterTaxi} {
		snap.Players = append(snap.Players, state.MirrorPlayer{MFLID: fmt.Sprintf("%04d", 99+i),
			FranchiseID: "0002", RosterStatus: status, Salary: domain.Money(12345600 + i),
			ContractYear: 2028 + i, ContractStatus: domain.CStatusUFA, ContractInfo: "Held contract"})
	}
	if _, err := a.core.League.Replace(t.Context(), snap); err != nil {
		t.Fatal(err)
	}
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	a.core.SeasonDraftResults = heldSeasonFeed[[]draftreports.Pick]{FetchedAt: at,
		Value: []draftreports.Pick{{Round: 1, Slot: 1, Franchise: "0002", Player: &id},
			{Round: 1, Slot: 2, Franchise: "0001", Skipped: true}}}
	a.core.SeasonFuturePicks = heldSeasonFeed[[]draftreports.FuturePick]{FetchedAt: at,
		Value: []draftreports.FuturePick{{Year: 2027, Round: 2, Original: "0001", Owner: "0002"}}}
	return a
}

func createNativeSeedLeague(t *testing.T, a *App, preset string) RawNativeSeedRequest {
	t.Helper()
	source := commissioner.SourceMFLImport
	if preset == commissioner.PresetEmpty {
		source = commissioner.SourceNative
	}
	raw := commissioner.RawCommand{League: "native.seed", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Seed league", Preset: preset, Source: source, Season: 2026, FranchiseCount: 32},
		Reason: "Create the league", Request: "create", Effective: "now", Visibility: "league"}
	historicalLeague(t, a, raw)
	return RawNativeSeedRequest{League: raw.League, Expected: 1, Request: "held.start", Reason: "Start from held data"}
}

func seedNativeTest(t *testing.T, a *App, input RawNativeSeedRequest) commissioner.Receipt {
	t.Helper()
	receipt, err := newCommissionerAPI(a).TargetNativeSeed(input)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func readNativeTest(t *testing.T, a *App, leagueID string) commissioner.NativeAssets {
	t.Helper()
	assets, err := newCommissionerAPI(a).TargetNativeAssets(leagueID)
	if err != nil {
		t.Fatal(err)
	}
	return assets
}

type seasonTransport struct {
	requests []string
	keyed    []string
	weeks    []string
	starts   []time.Time
	now      time.Time
	bodies   map[string]string
	failure  string
}

func (s *seasonTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	export := r.URL.Query().Get("TYPE")
	if export == "league" {
		return seasonResponse(`{"league":{"baseURL":"https://www47.myfantasyleague.com"}}`), nil
	}
	s.requests = append(s.requests, export)
	if r.URL.Query().Get("APIKEY") != "" {
		s.keyed = append(s.keyed, export)
	}
	s.weeks = append(s.weeks, r.URL.Query().Get("W"))
	s.starts = append(s.starts, s.now)
	if s.failure == export {
		return nil, errors.New("synthetic offline")
	}
	if id := r.URL.Query().Get("BRACKET_ID"); id != "" {
		export += "_" + id
	}
	return seasonResponse(s.bodies[export]), nil
}

func seasonResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}
}

func seasonFixture(t *testing.T, feed, name string) string {
	t.Helper()
	body, err := os.ReadFile("internal/ingestion/" + feed + "/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func seasonGzFixture(t *testing.T, feed, name string) string {
	t.Helper()
	zr, err := gzip.NewReader(strings.NewReader(seasonFixture(t, feed, name)))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func isolatedSeasonTestApp(t *testing.T) (*App, *seasonTransport, *[]time.Duration, *int) {
	t.Helper()
	a := targetTestApp(t)
	transport := &seasonTransport{
		now: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC),
		bodies: map[string]string{
			"transactions":     seasonFixture(t, "transactions", "transactions.json"),
			"liveScoring":      seasonFixture(t, "livescoring", "liveScoring-w5.json"),
			"pendingTrades":    `{"pendingTrades":{}}`,
			"messageBoard":     `{"messageBoard":{}}`,
			"polls":            `{"polls":{}}`,
			"calendar":         `{"calendar":{}}`,
			"tradeBait":        `{"tradeBaits":{}}`,
			"draftResults":     seasonFixture(t, "draftreports", "results-2026.json"),
			"futureDraftPicks": seasonFixture(t, "draftreports", "future-2026.json"),
			"projectedScores":  strings.Replace(seasonFixture(t, "marketfeeds", "projectedScores.json"), `"week":"6"`, `"week":"5"`, 1),
			"playerRanks":      seasonFixture(t, "marketfeeds", "playerRanks.json"),
			"adp":              seasonFixture(t, "marketfeeds", "adp.json"),
			"aav":              seasonFixture(t, "marketfeeds", "aav.json"),
			"weeklyResults":    seasonGzFixture(t, "weeklyresults", "weeklyResults-2026.json.gz"),
			"playoffBrackets":  seasonFixture(t, "playoffs", "playoffBrackets-2026.json"),
			"playoffBracket_1": seasonFixture(t, "playoffs", "playoffBracket_1-2026.json"),
			"playoffBracket_4": seasonFixture(t, "playoffs", "playoffBracket_4-2026.json"),
			"playoffBracket_5": seasonFixture(t, "playoffs", "playoffBracket_5-2026.json"),
			"leagueStandings":  seasonFixture(t, "leaguestandings", "leagueStandings-2026.json"),
		},
	}
	var err error
	a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(transport),
		mfl.WithKeySource(func(context.Context) (mflkey.Key, error) { return "", mflkey.ErrNoKey }))
	if err != nil {
		t.Fatal(err)
	}
	pauses := []time.Duration{}
	a.core.SeasonPause = func(ctx context.Context, d time.Duration) error {
		pauses = append(pauses, d)
		transport.now = transport.now.Add(d)
		return ctx.Err()
	}
	events := 0
	a.core.SeasonChanged = func(context.Context) { events++ }
	t.Cleanup(func() { a.core.WeekWorkers.Wait() })
	return a, transport, &pauses, &events
}

func targetTestApp(t *testing.T) *App {
	t.Helper()
	return targetTestAppAt(t, filepath.Join(t.TempDir(), "league.db"))
}

func targetTestAppAt(t *testing.T, path string) *App {
	t.Helper()
	ctx := context.Background()
	p, err := db.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := personal.New(p).Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	rb := rulebook.New(p)
	mirror := state.NewMirror(p, rb)
	if err := mirror.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := mirror.Replace(ctx, state.MirrorSnapshot{
		Season:  2026,
		Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001", RosterStatus: domain.RosterActive}},
	}); err != nil {
		t.Fatal(err)
	}
	historyDir := t.TempDir()
	seed, err := os.ReadFile(suiteFixturePath(t, "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(historyDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := root.WriteFile("history.db", seed, 0o600); err != nil {
		t.Fatal(err)
	}
	hp, err := db.Open(ctx, filepath.Join(historyDir, "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := hp.Close(); err != nil {
			t.Error(err)
		}
	})
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	hs := history.New(hp, reg)
	moveStore := moves.New(p)
	if err := moveStore.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	a := NewApp()
	a.core.MyFranchise = "0001"
	a.core.Pools = p
	a.core.HistPools = hp
	a.core.Moves = moveStore
	a.core.Ctx = ctx
	a.core.Season = 2026
	a.core.League = mirror
	a.core.Rulebook = rb
	a.core.History = hs
	close(a.core.Started)
	return a
}

func recordTargetPlayers(t *testing.T, hs *history.Store) {
	t.Helper()
	body := []byte(`{"players":{"player":{"id":"0042","name":"Archived","position":"QB","team":"BUF"}}}`)
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err := zw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	err := hs.Record(context.Background(), archive.Fetch{
		URL:       "/2026/export?TYPE=players&L=" + ingestion.LeagueID,
		Status:    200,
		SHA256:    hex.EncodeToString(sum[:]),
		Size:      int64(len(body)),
		Gzip:      gz.Bytes(),
		FetchedAt: time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
}
