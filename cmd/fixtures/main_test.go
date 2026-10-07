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
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type seedRules struct{}

func (seedRules) Fetch(context.Context) (league.RawConfig, error) {
	return league.RawConfig{SalaryCapAmount: "125", Franchises: []league.Franchise{{ID: "0001", Name: "Real Team"}}, IncludeTaxiWithSalary: "50", IncludeIRWithSalary: "0"}, nil
}

func seedLeague(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()
	pools, err := db.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	}()
	rb := rulebook.New(pools)
	if err := rb.Initialize(ctx, seedRules{}); err != nil {
		t.Fatal(err)
	}
	mirror := state.NewMirror(pools, rb)
	if err := mirror.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = mirror.Replace(ctx, state.MirrorSnapshot{Season: 2026, Players: []state.MirrorPlayer{{MFLID: "0531", FranchiseID: "0001", Salary: 100000000, ContractYear: 2027, ContractStatus: domain.CStatusUFA, RosterStatus: domain.RosterTaxi}}})
	if err != nil {
		t.Fatal(err)
	}
}

func seedArchive(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()
	pools, err := db.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	}()
	reg, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	hs := history.New(pools, reg)
	if err := hs.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		url, body string
		hour      int
	}{
		{"/2026/export?TYPE=players&L=" + ingestion.LeagueID, `{"players":{"player":{"id":"0531","name":"Older","position":"QB","team":"BUF"}}}`, 1},
		{"/2026/export?TYPE=players&L=" + ingestion.LeagueID, `{"players":{"player":{"id":"0531","name":"Newest","position":"QB","team":"BUF","status":"R"}}}`, 2},
		{"/2025/export?TYPE=players&L=" + ingestion.LeagueID, `{"players":{"player":{"id":"0531","name":"Wrong season","position":"QB"}}}`, 3},
		{"/2026/export?TYPE=players&L=" + ingestion.LeagueID, `{"players":{"player":{"id":"0531","position":""}}}`, 4},
	} {
		var gz bytes.Buffer
		zw := gzip.NewWriter(&gz)
		if _, err := zw.Write([]byte(row.body)); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(row.body))
		err := hs.Record(ctx, archive.Fetch{URL: row.url, Status: 200, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(row.body)), Gzip: gz.Bytes(), FetchedAt: time.Date(2026, 10, 5, row.hour, 0, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunUsesStoresAndLeavesSnapshotsUntouched(t *testing.T) {
	dir := t.TempDir()
	leaguePath, historyPath := filepath.Join(dir, "league.db"), filepath.Join(dir, "history.db")
	whatifPath := filepath.Join(dir, "whatif.db")
	seedLeague(t, leaguePath)
	seedArchive(t, historyPath)
	emptyDB(t, whatifPath)
	before := map[string][]byte{}
	for _, path := range []string{leaguePath, historyPath, whatifPath} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		before[path] = body
	}
	out := filepath.Join(dir, "out")
	args := []string{"-db", leaguePath, "-history", historyPath, "-whatif", whatifPath, "-out", out}
	if err := run(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	var snap snapshot.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatal(err)
	}
	if len(snap.Players.Value) != 1 || snap.Players.Value[0].Name != "Newest" || snap.Players.Value[0].ID.String() != "0531" {
		t.Fatal(snap.Players)
	}
	if snap.Franchises.Value[0].CapUsed == nil || *snap.Franchises.Value[0].CapUsed != 50000000 {
		t.Fatal("did not use rulebook's taxi discount", snap.Franchises)
	}
	for _, p := range []snapshot.Provenance{snap.League.Provenance, snap.Franchises.Provenance, snap.Rosters.Provenance, snap.Players.Provenance} {
		if p.Kind != "fixture" {
			t.Fatal(p)
		}
	}
	if snap.Players.Provenance.Freshness.FetchedAt != "2026-10-05T02:00:00Z" {
		t.Fatal(snap.Players.Provenance)
	}
	assertFixtureClock(t, filepath.Join(out, "clock.json"))
	if err := run(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(filepath.Join(out, "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, again) {
		t.Fatal("fixture export is not deterministic")
	}
	for path, want := range before {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Fatalf("wrote snapshot %s", path)
		}
	}
}

func TestRunOfflineWithoutActiveRulebook(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.db")
	emptyDB(t, path)
	err := run(context.Background(),
		[]string{"-db", path, "-history", path, "-whatif", path, "-out", filepath.Join(dir, "out")})
	if err == nil || !strings.Contains(err.Error(), "fixtures run offline") {
		t.Fatalf("offline gate = %v", err)
	}
}

func emptyDB(t *testing.T, path string) {
	t.Helper()
	pools, err := db.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
}

// An empty what-if store seeds its genesis phase from the mirror, as the app does at startup.
func assertFixtureClock(t *testing.T, path string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var clock clockReading
	if err := json.Unmarshal(body, &clock); err != nil {
		t.Fatal(err)
	}
	if clock.Provenance.Kind != "fixture" || clock.Value.Phase != domain.PhaseOffseason ||
		len(clock.Value.Deadlines) != 0 || len(clock.Value.Windows) != 7 {
		t.Fatal(clock)
	}
}
