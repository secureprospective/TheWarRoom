package leagueload

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func store(t *testing.T, name string) *history.Store {
	t.Helper()
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), name), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	s := history.New(pools, registry)
	if err := s.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return s
}

func hold(t *testing.T, s *history.Store, rawURL string, body []byte, at time.Time) {
	t.Helper()
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	if _, err := w.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if err := s.Record(t.Context(), archive.Fetch{URL: rawURL, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(body)),
		Gzip: gz.Bytes(), Status: 200, FetchedAt: at}); err != nil {
		t.Fatal(err)
	}
}

// The held copy answers whichever MFL host and credential the reader uses; the newest copy wins;
// a read the mirror does not hold fails without any network attempt; only GET is answered.
func TestReplayAnswersOnlyHeldReads(t *testing.T) {
	mirror := Mirror{store(t, "mirror.db")}
	at := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	hold(t, mirror.Store, "https://www47.myfantasyleague.com/2026/export?APIKEY=REDACTED&JSON=1&L=14432&TYPE=rosters", []byte(`{"old":1}`), at)
	hold(t, mirror.Store, "https://www47.myfantasyleague.com/2026/export?APIKEY=REDACTED&JSON=1&L=14432&TYPE=rosters", []byte(`{"new":1}`), at.Add(time.Hour))
	hold(t, mirror.Store, "https://www47.myfantasyleague.com/2026/export?JSON=1&L=14432&TYPE=rosters&W=3", []byte(`{"week":3}`), at.Add(2*time.Hour))
	client := &http.Client{Transport: archive.Replay{Lookup: mirror.Lookup}}
	get := func(u string) (string, error) {
		resp, err := client.Get(u)
		if err != nil {
			return "", err
		}
		defer func() { _ = resp.Body.Close() }()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return buf.String(), nil
	}
	body, err := get("https://api.myfantasyleague.com/2026/export?TYPE=rosters&L=14432&JSON=1")
	if err != nil || body != `{"new":1}` {
		t.Fatalf("held read = %q, %v", body, err)
	}
	if _, err := get("https://api.myfantasyleague.com/2026/export?TYPE=rosters&L=99999&JSON=1"); !errors.Is(err, archive.ErrNotHeld) {
		t.Fatalf("unheld read = %v, want ErrNotHeld", err)
	}
	if resp, err := client.Post("https://api.myfantasyleague.com/2026/import?TYPE=rosters&L=14432", "text/plain", nil); err == nil {
		_ = resp.Body.Close()
		t.Fatal("replay answered a write")
	}
}

func legacyFootball(t *testing.T) func(context.Context, int) ([]byte, string, error) {
	t.Helper()
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	var list []players.RawPlayer
	for id := range export.Players {
		list = append(list, players.RawPlayer{ID: id, Name: "Player " + id, Position: "WR", Team: "DAL", DraftYear: "2020"})
	}
	list = append(list, players.RawPlayer{ID: "9998", Name: "Team unit", Position: "Def", Team: "DAL"})
	return func(_ context.Context, season int) ([]byte, string, error) {
		at := time.Date(season, 9, 1, 0, 0, 0, 0, time.UTC)
		weeks := []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(240 * time.Hour)}}}}
		return BuildFootball(list, weeks, season, at)
	}
}

// The Legacy NFL as held lands on the server active and writable by its commissioner; the workspace
// copy is retired; the football evidence is pinned on the server.
func TestLoadMovesTheLeagueActiveOntoTheServer(t *testing.T) {
	workspace, server := store(t, "workspace.db"), store(t, "server.db")
	now := func() time.Time { return migrationtest.ReadAt }
	workspace.SetClock(now)
	server.SetClock(now)
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	commish := "acct." + uuid.NewString()
	result, err := Load(t.Context(), Request{MFLLeague: "14432", Commissioner: governancePrincipal(commish), Workspace: workspace, Server: func(context.Context, string) (*history.Store, error) { return server, nil },
		Source: func(leaguerules.RuleSet) (leaguemigration.Source, error) {
			return &migrationtest.Source{Export: export}, nil
		}, Football: legacyFootball(t), Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if result.League != leaguemigration.AppLeagueID("14432", 2026) || result.Franchises != 32 || result.Football == "" {
		t.Fatalf("result = %+v", result)
	}
	status, err := server.TransferStatus(t.Context(), result.League)
	if err != nil || !status.Writable {
		t.Fatalf("server status = %+v, %v", status, err)
	}
	if moved, err := workspace.TransferStatus(t.Context(), result.League); err != nil || moved.Writable {
		t.Fatalf("workspace still writable: %+v, %v", moved, err)
	}
	leagues, err := server.NativeLeagues(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	listed := false
	for _, l := range leagues {
		listed = listed || l.ID == result.League
	}
	if !listed {
		t.Fatalf("server does not list %s: %+v", result.League, leagues)
	}
	if _, err := server.LoadFootball(t.Context(), result.Football); err != nil {
		t.Fatalf("football not pinned on server: %v", err)
	}
	head, err := server.CommittedHead(t.Context(), result.League, commissioner.Stream)
	if err != nil {
		t.Fatal(err)
	}
	write := commissioner.RawCommand{League: result.League, Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: "cap.amount", Value: json.RawMessage(strconv.FormatInt(200000000000, 10))},
		Expected: head, Request: "server.write", Reason: "Commissioner writes on the server", Effective: "now", Visibility: "league"}
	if _, err := server.ExecuteCommissioner(t.Context(), write, commish, nil); err != nil {
		t.Fatalf("commissioner write on server: %v", err)
	}
	if _, err := server.ExecuteCommissioner(t.Context(), write, "acct."+uuid.NewString(), nil); err == nil {
		t.Fatal("a stranger wrote to the loaded league")
	}
}

// A source failure records nothing anywhere: no workspace league, nothing on the server.
func TestLoadFailureLeavesTheServerEmpty(t *testing.T) {
	workspace, server := store(t, "workspace.db"), store(t, "server.db")
	now := func() time.Time { return migrationtest.ReadAt }
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	_, err = Load(t.Context(), Request{MFLLeague: "14432", Commissioner: governancePrincipal("acct." + uuid.NewString()), Workspace: workspace, Server: func(context.Context, string) (*history.Store, error) { return server, nil },
		Source: func(leaguerules.RuleSet) (leaguemigration.Source, error) {
			return &migrationtest.Source{Export: export, FailAt: "rosters"}, nil
		}, Football: legacyFootball(t), Now: now})
	if err == nil {
		t.Fatal("load succeeded without rosters")
	}
	if leagues, err := server.NativeLeagues(t.Context()); err != nil || len(leagues) != 0 {
		t.Fatalf("server leagues after failure = %+v, %v", leagues, err)
	}
}

// Football evidence keeps people only, with their team and draft year, and no invented statistics.
func TestBuildFootballKeepsPeopleOnly(t *testing.T) {
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	list := []players.RawPlayer{{ID: "0531", Name: "Kept, QB", Position: "QB", Team: "KCC", DraftYear: "2017"},
		{ID: "0532", Name: "Undrafted, LB", Position: "LB", Team: "FA", DraftYear: "1970"},
		{ID: "0533", Name: "Chiefs", Position: "Def", Team: "KCC"}, {ID: "0534", Name: "Coach", Position: "Coach", Team: "KCC"}}
	weeks := []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"KCC", "BAL"}, Kickoff: at}}}}
	body, digest, err := BuildFootball(list, weeks, 2026, at)
	if err != nil {
		t.Fatal(err)
	}
	var raw history.RawFootballEvidence
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if digest != hex.EncodeToString(sum[:]) || len(raw.Players) != 2 || len(raw.Stats) != 0 {
		t.Fatalf("players = %d stats = %d", len(raw.Players), len(raw.Stats))
	}
	qb, lb := raw.Players["0531"].Facts, raw.Players["0532"].Facts
	if qb.Team != "KCC" || !qb.DraftYearHeld || qb.DraftYear != 2017 || lb.DraftYearHeld || lb.Team != "FA" {
		t.Fatalf("facts = %+v / %+v", qb, lb)
	}
}

func governancePrincipal(p string) governance.Principal { return governance.Principal(p) }
