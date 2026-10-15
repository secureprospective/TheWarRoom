package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func shellLedgerApp(t *testing.T) (*App, string) {
	t.Helper()
	app := commissionerTestApp(t)
	raw := commissioner.RawCommand{League: "native.shell", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Our league", Preset: commissioner.PresetEmpty, FranchiseCount: 4},
		Reason: "Start", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, app, raw)
	app.history.SetClock(func() time.Time { return time.Date(2026, 10, 13, 13, 0, 0, 0, time.UTC) })
	raw.Command, raw.Expected, raw.Request, raw.Visibility = "commish.role.grant", 1, "grant", "private"
	raw.Args = commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "secret-deputy", Role: governance.CoCommissioner}}
	executeCommissionerTest(t, app, raw)
	app.history.SetClock(func() time.Time { return time.Date(2026, 10, 13, 14, 0, 0, 0, time.UTC) })
	raw.Command, raw.Expected, raw.Request, raw.Visibility = commissioner.RulesSet, 2, "rule", "league"
	raw.Args = commissioner.RawArgs{Field: "calendar.timezone", Value: json.RawMessage(`"Europe/London"`)}
	changed := executeCommissionerTest(t, app, raw)
	app.history.SetClock(func() time.Time { return time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC) })
	raw.Command, raw.Expected, raw.Request = commissioner.Revert, 3, "undo"
	raw.Args = commissioner.RawArgs{Entry: changed.Entry}
	executeCommissionerTest(t, app, raw)
	return app, raw.League
}

func TestCommissionerLedgerFiltersWire(t *testing.T) {
	app, league := shellLedgerApp(t)
	views := make(map[string]commissioner.Ledger)
	for _, test := range []struct {
		name, body string
		count      int
	}{
		{"all", `{}`, 4},
		{"rules", `{"kind":"rules"}`, 1},
		{"setup", `{"kind":"setup"}`, 1},
		{"corrections", `{"kind":"corrections"}`, 1},
		{"people", `{"kind":"people"}`, 1},
		{"person", `{"person":" CrEaToR "}`, 4},
		{"absent", `{"person":"nobody"}`, 0},
		{"dates", `{"from":"2026-10-13T09:00:00-04:00","before":"2026-10-13T10:00:00-04:00"}`, 1},
		{"combined", `{"kind":"rules","person":"creator","from":"2026-10-13T13:00:00Z","before":"2026-10-13T15:00:00Z"}`, 1},
	} {
		reading, err := app.TargetCommissionerLedgerFiltered(league, test.body)
		if err != nil || len(reading.Entries) != test.count || reading.Revision != 4 {
			t.Fatalf("%s: %+v %v", test.name, reading, err)
		}
		views[test.name] = reading
	}
	app.commissionerSession = testCommissionerSession{"owner"}
	owner, err := app.TargetCommissionerLedgerFiltered(league, `{"kind":"people"}`)
	if err != nil || len(owner.Entries) != 0 || owner.Revision != 4 {
		t.Fatalf("owner received private role rows: %+v %v", owner, err)
	}
	views["owner"] = owner
	body, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/ledger-filters.wire.testdata.json", body, "ledger filters")
	all, err := app.TargetCommissionerLedger(league)
	if err != nil || all.Revision != 4 || len(all.Entries) != 3 {
		t.Fatalf("filtering changed the ledger or leaked its private row: %+v %v", all, err)
	}
}

func TestCommissionerLedgerFiltersRejectBadInput(t *testing.T) {
	app, league := shellLedgerApp(t)
	for _, body := range []string{
		`{"kind":"unknown"}`, `{"person":4}`, `{"principal":"creator"}`, `{} {}`,
		`{"from":"yesterday"}`, `{"before":"tomorrow"}`,
		`{"from":"2026-10-14T00:00:00Z","before":"2026-10-13T00:00:00Z"}`,
		`{"from":"2026-10-13T00:00:00Z","before":"2026-10-13T00:00:00Z"}`,
	} {
		if _, err := app.TargetCommissionerLedgerFiltered(league, body); err == nil {
			t.Fatalf("bad ledger filters accepted: %s", body)
		}
	}
	reading, err := app.TargetCommissionerLedgerFiltered(league, `{"kind":"people","person":"secret-deputy"}`)
	if err != nil || len(reading.Entries) != 0 {
		t.Fatal("person filter must search the author, not private payload details", reading, err)
	}
	if _, err := app.TargetCommissionerLedgerFiltered("mfl.123", `{}`); err == nil ||
		!strings.Contains(err.Error(), "choose an app league") {
		t.Fatal("imported league filter accepted", err)
	}
}
