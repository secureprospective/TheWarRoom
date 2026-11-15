package commissioner

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func shellLedgerApp(t *testing.T) (*ledgerFixture, string) {
	t.Helper()
	app := commissionerTestApp(t)
	raw := commissioner.RawCommand{League: "native.shell", Command: commissioner.Create,
		Args: commissioner.RawArgs{Name: "Our league", Preset: commissioner.PresetEmpty, FranchiseCount: 4,
			Season: 2026},
		Reason: "Start", Request: "create", Effective: "now", Visibility: "league"}
	// Keep this pre-auto-start ledger fixture unchanged; new creation is covered separately.
	if _, err := app.history.ExecuteCommissioner(t.Context(), raw, app.session, nil); err != nil {
		t.Fatal(err)
	}
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

func TestCommissionerLedgerFiltersRejectBadInput(t *testing.T) {
	t.Parallel()
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
