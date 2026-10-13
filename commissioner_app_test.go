package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type testCommissionerSession struct{ principal string }

func (s testCommissionerSession) Principal() (string, error) { return s.principal, nil }

func commissionerTestApp(t *testing.T) *App {
	t.Helper()
	p, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"))
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
	app := &App{ctx: t.Context(), history: store, started: make(chan struct{}),
		commissionerSession: testCommissionerSession{"creator"}}
	close(app.started)
	return app
}

func executeCommissionerTest(t *testing.T, app *App, raw commissioner.RawCommand) commissioner.Receipt {
	t.Helper()
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := app.TargetCommissionerExecute(string(body))
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func TestCommissionerWire(t *testing.T) {
	app := commissionerTestApp(t)
	raw := commissioner.RawCommand{League: "native.wire", Command: commissioner.Create,
		Args: commissioner.RawArgs{Name: "Our league"}, Reason: "Start our own league", Request: "create",
		Effective: "now", Visibility: "league"}
	first := executeCommissionerTest(t, app, raw)
	raw.Command, raw.Args = commissioner.Revert, commissioner.RawArgs{Entry: first.Entry}
	raw.Expected, raw.Request, raw.Reason, raw.Visibility = 1, "private", "Private correction", "private"
	private := executeCommissionerTest(t, app, raw)
	commish, err := app.TargetCommissionerLedger(raw.League)
	if err != nil || len(commish.Entries) != 2 || commish.Entries[0].ID != private.Entry {
		t.Fatalf("commissioner newest first: %+v %v", commish, err)
	}
	app.commissionerSession = testCommissionerSession{"owner"}
	owner, err := app.TargetCommissionerLedger(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.MarshalIndent(owner, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "Private correction") || strings.Contains(string(body), private.Entry) ||
		len(owner.Entries) != 1 {
		t.Fatalf("private data sent over owner IPC: %s", body)
	}
	path := "frontend/src/app/commissioner/wire.testdata.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(held)) != string(body) {
		t.Fatal("commissioner wire is stale")
	}
	raw.Expected, raw.Request, raw.Visibility = 2, "unauthorized", "league"
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.TargetCommissionerExecute(string(encoded)); err == nil {
		t.Fatal("owner acted as creator through IPC")
	}
	if _, err := app.TargetCommissionerExecute(strings.Replace(string(encoded), `"expected":2`,
		`"principal":"creator","expected":2`, 1)); err == nil {
		t.Fatal("frontend switched principal")
	}
	leagues, err := app.TargetCommissionerLeagues()
	if err != nil || len(leagues) != 1 || leagues[0].Name != "Our league" {
		t.Fatalf("leagues: %+v %v", leagues, err)
	}
}

func TestLocalCommissionerSessionStableAndNotFranchise(t *testing.T) {
	first, err := (localCommissionerSession{"installation"}).Principal()
	if err != nil {
		t.Fatal(err)
	}
	second, err := (localCommissionerSession{"installation"}).Principal()
	if err != nil || first != second || !strings.HasPrefix(first, "local.") {
		t.Fatalf("local identity: %s %v", second, err)
	}
	app := &App{dataDir: "installation"}
	principal, err := app.commissionerPrincipal()
	if err != nil || principal != first {
		t.Fatalf("production port: %s %v", principal, err)
	}
	if _, err := (&App{}).commissionerPrincipal(); err == nil {
		t.Fatal("missing installation fabricated identity")
	}
}
