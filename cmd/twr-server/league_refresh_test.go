package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leagueload"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type cliRefreshFixture struct {
	dir     string
	dbPath  string
	pinPath string
	league  string
	creds   accounts.RawCredentials
	export  mflsource.Export
	now     time.Time
}

func newCLIRefreshFixture(t *testing.T) *cliRefreshFixture {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "server.db")
	pinPath := filepath.Join(dir, "football.sha256")
	if err := prepareFirstLoad(dbPath); err != nil {
		t.Fatal(err)
	}
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	export.Schedule, err = leagueschedule.Parse([]byte(`{"schedule":{"weeklySchedule":[` +
		`{"week":"1","matchup":{"franchise":[` +
		`{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}},` +
		`{"week":"2","matchup":{"franchise":[` +
		`{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}}` +
		`]}}`))
	if err != nil {
		t.Fatal(err)
	}
	f := &cliRefreshFixture{
		dir:     dir,
		dbPath:  dbPath,
		pinPath: pinPath,
		league:  leaguemigration.AppLeagueID("14432", 2026),
		creds:   accounts.RawCredentials{Username: "christopher", Password: "a-long-private-password"},
		export:  export,
		now:     time.Now().UTC(),
	}
	held := heldLeague{
		MFLLeague: "14432",
		Source: func(leaguerules.RuleSet) (leaguemigration.Source, error) {
			return &migrationtest.Source{Export: f.export}, nil
		},
		Football: f.buildFootball,
	}
	var out strings.Builder
	if err := loadHeld(t.Context(), dbPath, held, pinPath, f.creds, &out); err != nil {
		t.Fatalf("initial league-load: %v", err)
	}
	f.now = time.Now().UTC().Add(time.Minute)
	return f
}

func (f *cliRefreshFixture) buildFootball(_ context.Context, season int) ([]byte, string, error) {
	list := make([]players.RawPlayer, 0, len(f.export.Players))
	for id := range f.export.Players {
		list = append(list, players.RawPlayer{
			ID: id, Name: "Player " + id, Position: "WR", Team: "DAL", DraftYear: "2020",
		})
	}
	at := time.Date(season, 9, 1, 0, 0, 0, 0, time.UTC)
	weeks := []leagueweek.Week{
		{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(240 * time.Hour)}}},
		{Number: 2, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(408 * time.Hour)}}},
	}
	return leagueload.BuildFootball(list, weeks, season, f.now)
}

func (f *cliRefreshFixture) held() heldRefresh {
	return heldRefresh{
		Source: func(_ string, _ leaguerules.RuleSet) (leaguemigration.Source, error) {
			return &migrationtest.Source{Export: f.export}, nil
		},
		Football: f.buildFootball,
		Now:      func() time.Time { return f.now },
	}
}

func (f *cliRefreshFixture) withStore(
	t *testing.T, fn func(files stateFiles, store *history.Store, principal string),
) {
	t.Helper()
	files, err := openStateFiles(t.Context(), f.dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()
	store, err := leagueStore(t.Context(), files, f.league)
	if err != nil {
		t.Fatal(err)
	}
	acct, err := commissionerSignIn(t.Context(), files, f.creds)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := acct.Principal()
	if err != nil {
		t.Fatal(err)
	}
	fn(files, store, string(principal))
}

func execOnStore(
	t *testing.T, store *history.Store, league string, raw commissioner.RawCommand, principal string,
) commissioner.Receipt {
	t.Helper()
	recs, err := leagueservice.Records(t.Context(), store, league, "test")
	if err != nil {
		t.Fatal(err)
	}
	raw.League = league
	raw.Expected = int64(len(recs))
	receipt, err := store.ExecuteCommissioner(t.Context(), raw, principal, nil)
	if err != nil {
		t.Fatalf("execute %s: %v", raw.Command, err)
	}
	return receipt
}

func TestLeagueRefreshEndToEndPreservesEditsAndIsIdempotent(t *testing.T) {
	f := newCLIRefreshFixture(t)
	ownerAcct := "acct." + uuid.NewString()
	editedName := "Commissioner Custom Team"
	var commishPrincipal string
	f.withStore(t, func(_ stateFiles, store *history.Store, principal string) {
		commishPrincipal = principal
		// Seat commissioner on franchise.25, claim franchise.02, edit cap rule, edit franchise.03 name.
		execOnStore(t, store, f.league, commissioner.RawCommand{
			Command: "commish.seat.replace", Request: "seat-commish-25",
			Reason: "Commissioner takes franchise 25", Effective: "now", Visibility: "league",
			Args: commissioner.RawArgs{People: &commissioner.RawPeople{
				Franchise: "franchise.25", Principal: principal,
			}},
		}, principal)
		execOnStore(t, store, f.league, commissioner.RawCommand{
			Command: "commish.seat.replace", Request: "seat-owner-02",
			Reason: "Owner claims franchise 02", Effective: "now", Visibility: "league",
			Args: commissioner.RawArgs{People: &commissioner.RawPeople{
				Franchise: "franchise.02", Principal: ownerAcct,
			}},
		}, principal)
		execOnStore(t, store, f.league, commissioner.RawCommand{
			Command: commissioner.RulesSet, Request: "edit-cap-rule",
			Reason: "Commissioner cap rule edit", Effective: "now", Visibility: "league",
			Args: commissioner.RawArgs{
				Field: "cap.amount", Value: json.RawMessage(strconv.FormatInt(225000000000, 10)),
			},
		}, principal)
		execOnStore(t, store, f.league, commissioner.RawCommand{
			Command: "commish.franchise.edit", Request: "edit-fr-03",
			Reason: "Commissioner renames team", Effective: "now", Visibility: "league",
			Args: commissioner.RawArgs{People: &commissioner.RawPeople{
				Franchise: "franchise.03", Name: editedName, Abbreviation: "CCT",
			}},
		}, principal)
	})

	// Mutate MFL: change a contract, add a salary adjustment, and complete week 1 MFL results.
	f.export.Rosters = slices.Clone(f.export.Rosters)
	f.export.Rosters[0].Salary = "14.25"
	f.export.Adjustments = append(slices.Clone(f.export.Adjustments), mflsource.Adjustment{
		ID: "adj-mfl-1", Franchise: "0001", Amount: "1.50", Description: "MFL cap penalty",
	})
	var err error
	f.export.Results, err = weeklyresults.Parse([]byte(`{"allWeeklyResults":{"weeklyResults":[` +
		`{"week":"1","matchup":{"regularSeason":"1","franchise":[` +
		`{"id":"0001","isHome":"1","score":"121.40","result":"W"},` +
		`{"id":"0002","isHome":"0","score":"102.10","result":"L"}]}}` +
		`]}}`))
	if err != nil {
		t.Fatal(err)
	}

	var beforeCount int
	f.withStore(t, func(_ stateFiles, store *history.Store, _ string) {
		recs, err := leagueservice.Records(t.Context(), store, f.league, "before")
		if err != nil {
			t.Fatal(err)
		}
		beforeCount = len(recs)
	})

	// Dry run changes nothing and prints owner-readable summary counts.
	var dryOut strings.Builder
	opts := refreshOptions{
		dbPath: f.dbPath, league: f.league, pinPath: f.pinPath, dryRun: true,
	}
	if err := refreshHeld(t.Context(), opts, f.held(), f.creds, &dryOut); err != nil {
		t.Fatalf("dry-run refresh: %v", err)
	}
	dryText := dryOut.String()
	t.Logf("Example dry-run output:\n%s", dryText)
	for _, want := range []string{
		"Dry run for Legacy NFL (2026 season)",
		"Contract updates: 1",
		"Salary cap adjustments: 1",
		"Schedule and MFL scores: updated (1 completed week; MFL scores count through week 1)",
		"Open trade offers blocking refresh: none",
	} {
		if !strings.Contains(dryText, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, dryText)
		}
	}
	if strings.Contains(dryText, f.league) || strings.Contains(dryText, "franchise.") {
		t.Fatalf("dry-run output exposed raw IDs without names:\n%s", dryText)
	}
	f.withStore(t, func(_ stateFiles, store *history.Store, _ string) {
		recs, err := leagueservice.Records(t.Context(), store, f.league, "after-dry")
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != beforeCount {
			t.Fatalf("dry run wrote ledger entries: %d -> %d", beforeCount, len(recs))
		}
	})

	// Real commit applies changes and prints the same summary lines.
	var commitOut strings.Builder
	opts.dryRun = false
	if err := refreshHeld(t.Context(), opts, f.held(), f.creds, &commitOut); err != nil {
		t.Fatalf("commit refresh: %v", err)
	}
	commitText := commitOut.String()
	if summaryBody(dryText) != summaryBody(commitText) {
		t.Fatalf("dry-run summary != commit summary:\n--- dry ---\n%s\n--- commit ---\n%s",
			dryText, commitText)
	}

	// Verify 0 asset diffs, MFL scores count, and commissioner edits + seats preserved.
	f.withStore(t, func(_ stateFiles, store *history.Store, _ string) {
		recs, err := leagueservice.Records(t.Context(), store, f.league, "verify")
		if err != nil {
			t.Fatal(err)
		}
		app, err := leaguemigration.ReadApp(f.league, recs, f.now)
		if err != nil {
			t.Fatal(err)
		}
		seats := app.Directory.Seats()
		if seats.Of("franchise.25").Owner != commishPrincipal {
			t.Fatalf("franchise.25 owner = %q, want %q", seats.Of("franchise.25").Owner, commishPrincipal)
		}
		if seats.Of("franchise.02").Owner != ownerAcct {
			t.Fatalf("franchise.02 owner = %q, want %q", seats.Of("franchise.02").Owner, ownerAcct)
		}
		if seats.Of("franchise.03").Name != editedName {
			t.Fatalf("franchise.03 name = %q, want %q", seats.Of("franchise.03").Name, editedName)
		}
		rules, err := commissioner.RulesNow(recs, f.now)
		if err != nil {
			t.Fatal(err)
		}
		if rules.Cap.Amount.Value != 225000000000 {
			t.Fatalf("cap rule overwritten: %d", rules.Cap.Amount.Value)
		}
		recorded := commissioner.RecordedResults(recs, 2026)
		if recorded == nil || recorded.CutoverWeek != 2 || len(recorded.Teams) != 2 {
			t.Fatalf("recorded MFL results = %+v", recorded)
		}
		plan, err := mflsource.Prepare(f.export)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range leaguemigration.Compare(plan, app) {
			if strings.HasPrefix(d.Field, "player ") ||
				strings.HasPrefix(d.Field, "roster ") ||
				strings.HasPrefix(d.Field, "contract ") ||
				strings.HasPrefix(d.Field, "salary adjustment ") ||
				strings.HasPrefix(d.Field, "draft pick ") {
				t.Fatalf("unexpected remaining asset diff: %+v", d)
			}
		}
	})

	// Second run is a 0-change no-op.
	var secondOut strings.Builder
	f.now = f.now.Add(2 * time.Hour)
	if err := refreshHeld(t.Context(), opts, f.held(), f.creds, &secondOut); err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if !strings.Contains(secondOut.String(), "already up to date") {
		t.Fatalf("second run output = %q", secondOut.String())
	}
}

func TestLeagueRefreshStopsOnOpenTradeOffers(t *testing.T) {
	f := newCLIRefreshFixture(t)
	owner1, owner2 := "acct."+uuid.NewString(), "acct."+uuid.NewString()
	var f1, f2, team1, team2 string
	var beforeCount int
	f.withStore(t, func(_ stateFiles, store *history.Store, principal string) {
		enableServerTrading(t, store, f.league, principal)
		recs, err := leagueservice.Records(t.Context(), store, f.league, "read")
		if err != nil {
			t.Fatal(err)
		}
		app, err := leaguemigration.ReadApp(f.league, recs, f.now)
		if err != nil {
			t.Fatal(err)
		}
		var p1, p2 commissioner.NativePick
		for _, pk := range app.Snapshot.Picks {
			if pk.Season <= app.Snapshot.Season || pk.Original == "" || pk.Player != "" || pk.Skipped {
				continue
			}
			if p1.Original == "" {
				p1 = pk
			} else if pk.Owner != p1.Owner {
				p2 = pk
				break
			}
		}
		if p1.Original == "" || p2.Original == "" {
			t.Fatal("missing two future picks for trade offer test")
		}
		f1, f2 = p1.Owner, p2.Owner
		seats := app.Directory.Seats()
		team1, team2 = seats.Of(f1).Name, seats.Of(f2).Name
		for i, seat := range [][2]string{{f1, owner1}, {f2, owner2}} {
			execOnStore(t, store, f.league, commissioner.RawCommand{
				Command: "commish.seat.replace", Request: "seat-" + strconv.Itoa(i),
				Reason: "Seat owner", Effective: "now", Visibility: "league",
				Args: commissioner.RawArgs{People: &commissioner.RawPeople{
					Franchise: seat[0], Principal: seat[1],
				}},
			}, principal)
		}
		execOnStore(t, store, f.league, commissioner.RawCommand{
			Command: commissioner.TradeOffer, Request: "open-offer-1",
			Reason: "Offer trade", Effective: "now", Visibility: "league",
			Args: commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{
				Franchise: f1,
				Parties:   []string{f1, f2},
				Assets: []commissioner.TradeAsset{
					{
						From: f1, To: f2,
						Season: p1.Season, Round: p1.Round, Original: p1.Original,
					},
					{
						From: f2, To: f1,
						Season: p2.Season, Round: p2.Round, Original: p2.Original,
					},
				},
			}},
		}, owner1)
		recs, err = leagueservice.Records(t.Context(), store, f.league, "after-offer")
		if err != nil {
			t.Fatal(err)
		}
		beforeCount = len(recs)
	})

	for _, dryRun := range []bool{true, false} {
		var out strings.Builder
		opts := refreshOptions{dbPath: f.dbPath, league: f.league, dryRun: dryRun}
		err := refreshHeld(t.Context(), opts, f.held(), f.creds, &out)
		if err == nil {
			t.Fatalf("dryRun=%t: expected open trade stop error", dryRun)
		}
		text := out.String()
		if !strings.Contains(text, team1) || !strings.Contains(text, team2) {
			t.Fatalf("dryRun=%t: output missing team names %q/%q:\n%s", dryRun, team1, team2, text)
		}
		if strings.Contains(text, f1) || strings.Contains(text, f2) ||
			strings.Contains(text, f.league) {
			t.Fatalf("dryRun=%t: output leaked raw IDs:\n%s", dryRun, text)
		}
	}

	f.withStore(t, func(_ stateFiles, store *history.Store, _ string) {
		recs, err := leagueservice.Records(t.Context(), store, f.league, "check")
		if err != nil {
			t.Fatal(err)
		}
		if len(recs) != beforeCount {
			t.Fatalf("stopped refresh modified ledger: %d -> %d", beforeCount, len(recs))
		}
	})
}

func TestLeagueRefreshCLIFlagsAndRealMirror(t *testing.T) {
	f := newCLIRefreshFixture(t)
	mirrorPath := buildRealLegacyMirror(t, f.export)
	goodInput := consoleInputJSON(t, f.creds.Username, f.creds.Password)

	// Reject missing required flags.
	if err := runLeagueRefresh(t.Context(), []string{"-db", f.dbPath},
		strings.NewReader(goodInput), &strings.Builder{}); err == nil {
		t.Fatal("accepted missing -mirror and -league")
	}

	// Real CLI dry-run against a read-only SQLite mirror.
	var dryOut strings.Builder
	args := []string{
		"-db", f.dbPath, "-mirror", mirrorPath, "-league", f.league,
		"-football-pin", f.pinPath, "-dry-run",
	}
	if err := runLeagueRefresh(t.Context(), args, strings.NewReader(goodInput), &dryOut); err != nil {
		t.Fatalf("runLeagueRefresh -dry-run: %v", err)
	}
	if !strings.Contains(dryOut.String(), "Dry run for Legacy NFL (2026 season)") {
		t.Fatalf("unexpected CLI dry-run output:\n%s", dryOut.String())
	}

	// Real CLI commit against the same mirror, then idempotent second run.
	commitArgs := []string{
		"-db", f.dbPath, "-mirror", mirrorPath, "-league", f.league, "-football-pin", f.pinPath,
	}
	var commitOut strings.Builder
	if err := runLeagueRefresh(
		t.Context(), commitArgs, strings.NewReader(goodInput), &commitOut,
	); err != nil {
		t.Fatalf("runLeagueRefresh commit: %v", err)
	}
	if summaryBody(dryOut.String()) != summaryBody(commitOut.String()) {
		t.Fatalf("CLI dry-run != commit summary:\n--- dry ---\n%s\n--- commit ---\n%s",
			dryOut.String(), commitOut.String())
	}

	// Reject wrong password.
	wrongInput := consoleInputJSON(t, f.creds.Username, "wrong-value-long-enough-for-test")
	if err := runLeagueRefresh(
		t.Context(), args, strings.NewReader(wrongInput), &strings.Builder{},
	); err == nil {
		t.Fatal("accepted wrong commissioner password")
	}
}

func consoleInputJSON(t *testing.T, user, secret string) string {
	t.Helper()
	b, err := json.Marshal(map[string]string{"username": user, "password": secret})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func buildRealLegacyMirror(t *testing.T, export mflsource.Export) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	pools, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	s := history.New(pools, registry)
	if err := s.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..")
	readFixture := func(rel string) []byte {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	at := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	recordBody := func(rawURL string, body []byte) {
		var gz bytes.Buffer
		w := gzip.NewWriter(&gz)
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		if err := s.Record(t.Context(), archive.Fetch{
			URL: rawURL, SHA256: hex.EncodeToString(sum[:]),
			Size: int64(len(body)), Gzip: gz.Bytes(), Status: 200, FetchedAt: at,
		}); err != nil {
			t.Fatal(err)
		}
	}
	base := "https://api.myfantasyleague.com/2026/export?JSON=1&L=14432&TYPE="
	recordBody(base+"league", readFixture("internal/ingestion/league/testdata/league-2026.json"))
	recordBody(base+"rules", readFixture("internal/ingestion/league/testdata/rules-2026.json"))
	recordBody(base+"rosters", readFixture("internal/ingestion/testdata/comms/rosters-2026.json"))
	recordBody(base+"draftResults", readFixture("internal/ingestion/draftreports/testdata/results-2026.json"))
	recordBody(base+"futureDraftPicks", readFixture("internal/ingestion/draftreports/testdata/future-2026.json"))
	recordBody(base+"salaryAdjustments", []byte(`{"salaryAdjustments":{"salaryAdjustment":[]}}`))
	schedJSON := `{"schedule":{"weeklySchedule":[{"week":"1","matchup":{"franchise":[` +
		`{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}}]}}`
	recordBody(base+"schedule", []byte(schedJSON))
	recordBody(base+"transactions", []byte(`{"transactions":{"transaction":[]}}`))
	weeklyJSON := `{"allWeeklyResults":{"weeklyResults":[{"week":"1","matchup":{"regularSeason":"1",` +
		`"franchise":[{"id":"0001","isHome":"1","score":"115.00","result":"W"},` +
		`{"id":"0002","isHome":"0","score":"99.50","result":"L"}]}}]}}`
	recordBody(base+"weeklyResults&W=YTD", []byte(weeklyJSON))
	liveJSON := `{"liveScoring":{"week":"2","matchup":{"franchise":[` +
		`{"id":"0001","isHome":"1","score":"0.00","gameSecondsRemaining":"3600","players":{"player":[]}},` +
		`{"id":"0002","isHome":"0","score":"0.00","gameSecondsRemaining":"3600","players":{"player":[]}}]}}}`
	recordBody(base+"liveScoring", []byte(liveJSON))

	type mflPlayer struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Position  string `json:"position"`
		Team      string `json:"team"`
		DraftYear string `json:"draft_year"`
	}
	plist := make([]mflPlayer, 0, len(export.Players))
	for id := range export.Players {
		plist = append(plist, mflPlayer{
			ID: id, Name: "Player " + id, Position: "WR", Team: "DAL", DraftYear: "2020",
		})
	}
	playersBody, err := json.Marshal(map[string]map[string][]mflPlayer{"players": {"player": plist}})
	if err != nil {
		t.Fatal(err)
	}
	recordBody("https://api.myfantasyleague.com/2026/export?DETAILS=1&JSON=1&TYPE=players", playersBody)
	nflSched := `{"nflSchedule":{"week":"1","matchup":[` +
		`{"kickoff":"1789063200","team":[{"id":"DAL","isHome":"1"},{"id":"NYG","isHome":"0"}]}]}}`
	recordBody("https://api.myfantasyleague.com/2026/export?JSON=1&TYPE=nflSchedule&W=1", []byte(nflSched))
	return path
}

func enableServerTrading(t *testing.T, store *history.Store, league, principal string) {
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
		execOnStore(t, store, league, commissioner.RawCommand{
			Command: commissioner.RulesSet, Request: "rule-" + rule[0],
			Reason: "Configure trade rule", Effective: "now", Visibility: "league",
			Args: commissioner.RawArgs{Field: rule[0], Value: json.RawMessage(rule[1])},
		}, principal)
	}
	execOnStore(t, store, league, commissioner.RawCommand{
		Command: commissioner.WindowSet, Request: "open-trades",
		Reason: "Open trade window", Effective: "now", Visibility: "league",
		Args: commissioner.RawArgs{Window: &commissioner.RawWindowArgs{
			ID: "trades", Start: &commissioner.RawDate{Month: 1, Day: 1},
			End:   &commissioner.RawDate{Month: 12, Day: 31},
			Opens: []leaguerules.Action{"trade"},
		}},
	}, principal)
}

func summaryBody(output string) string {
	_, rest, ok := strings.Cut(output, "\n")
	if !ok {
		return output
	}
	return rest
}
