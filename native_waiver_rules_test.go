package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const waiverPolicyLeague = "native.waiver-policy"

func createWaiverPolicyLeague(t *testing.T, app *App, preset string) commissioner.RawCommand {
	t.Helper()
	raw := commissioner.RawCommand{League: waiverPolicyLeague, Command: commissioner.Create,
		Args:    commissioner.RawArgs{Name: "Waiver policy league", Preset: preset, FranchiseCount: 3, Season: 2026},
		Request: "create", Effective: "now", Visibility: "league", Reason: "Start this league"}
	executeCommissionerTest(t, app, raw)
	return raw
}

func readWaiverPolicy(t *testing.T, app *App) (nativecheck.WaiverPolicy, []commissioner.RuleRow) {
	t.Helper()
	reading, err := app.TargetCommissionerRules(waiverPolicyLeague)
	if err != nil {
		t.Fatal(err)
	}
	var rows []commissioner.RuleRow
	for _, section := range reading.Sections {
		for _, row := range section.Rows {
			if row.Field == "waivers.trigger" || row.Field == "waivers.duration" || row.Field == "waivers.method" {
				rows = append(rows, row)
			}
		}
	}
	if len(rows) != 3 {
		t.Fatalf("waiver catalogue has %d rows", len(rows))
	}
	records, err := app.leagueRecords(context.Background(), waiverPolicyLeague)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := commissioner.RulesNow(records, app.timeNow())
	if err != nil {
		t.Fatal(err)
	}
	policy, err := nativecheck.NewWaiverPolicy(rules.Waivers)
	if err != nil {
		t.Fatal(err)
	}
	return policy, rows
}

func TestWaiverDurationCommandDatesPolicyRevertReopen(t *testing.T) {
	at := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	app, reopen := calendarTestApp(t, at)
	raw := createWaiverPolicyLeague(t, app, commissioner.PresetLegacy)
	assertClose := func(hours int, words string) {
		t.Helper()
		policy, rows := readWaiverPolicy(t, app)
		closeAt, err := policy.Close(at.Add(-time.Hour))
		if err != nil || !closeAt.Equal(at.Add(time.Duration(hours-1)*time.Hour)) {
			t.Fatalf("duration %d closes %s: %v", hours, closeAt, err)
		}
		if rows[1].Field != "waivers.duration" || rows[1].Value != words {
			t.Fatalf("duration reading: %+v", rows)
		}
	}
	assertClose(24, "1 day")
	raw.Command, raw.Request, raw.Expected = commissioner.RulesSet, "duration", 1
	raw.Args = commissioner.RawArgs{Field: "waivers.duration", Value: json.RawMessage(`172800000000000`)}
	change := executeCommissionerTest(t, app, raw)
	for range 2 {
		assertClose(48, "2 days")
		reopen()
	}
	revertEntry(t, app, raw, change.Entry, "restore-duration", 2)
	for range 2 {
		assertClose(24, "1 day")
		reopen()
	}
}

func TestWaiverMethodCommandChangesPolicyAwardRevertReopen(t *testing.T) {
	at := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	app, reopen := calendarTestApp(t, at)
	raw := createWaiverPolicyLeague(t, app, commissioner.PresetLegacy)
	priority := nativecheck.WaiverPriority{Standings: []string{"franchise.03", "franchise.02", "franchise.01"},
		Rolling: []string{"franchise.01", "franchise.02", "franchise.03"}}
	assertAward := func(winner, method string, order []string) {
		t.Helper()
		policy, rows := readWaiverPolicy(t, app)
		award, err := policy.Award(priority, []string{"franchise.01", "franchise.03"})
		if err != nil || award.Winner != winner || !slices.Equal(award.Order, order) {
			t.Fatalf("award %+v: %v; want %s %v", award, err, winner, order)
		}
		if rows[0].Value != "Cut" || rows[2].Value != method {
			t.Fatalf("trigger/method reading: %+v", rows)
		}
	}
	assertAward("franchise.03", "Reverse standings", priority.Standings)
	raw.Command, raw.Request, raw.Expected = commissioner.RulesSet, "method", 1
	raw.Args = commissioner.RawArgs{Field: "waivers.method", Value: json.RawMessage(`"rolling"`)}
	change := executeCommissionerTest(t, app, raw)
	for range 2 {
		assertAward("franchise.01", "Rolling", []string{"franchise.02", "franchise.03", "franchise.01"})
		reopen()
	}
	revertEntry(t, app, raw, change.Entry, "restore-method", 2)
	for range 2 {
		assertAward("franchise.03", "Reverse standings", priority.Standings)
		reopen()
	}
}

func TestEmptyLeagueDoesNotBorrowLegacyWaiverPolicy(t *testing.T) {
	at := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	app, reopen := calendarTestApp(t, at)
	createWaiverPolicyLeague(t, app, commissioner.PresetEmpty)
	for range 2 {
		records, err := app.leagueRecords(context.Background(), waiverPolicyLeague)
		if err != nil {
			t.Fatal(err)
		}
		rules, err := commissioner.RulesNow(records, at)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := nativecheck.NewWaiverPolicy(rules.Waivers); err == nil {
			t.Fatal("empty league borrowed the Legacy waiver policy")
		}
		reopen()
	}
}

func TestNativeWaiverPolicyWire(t *testing.T) {
	at := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	app, _ := calendarTestApp(t, at)
	createWaiverPolicyLeague(t, app, commissioner.PresetLegacy)
	_, rows := readWaiverPolicy(t, app)
	body, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("internal", "commissioner", "testdata", "native-waiver-policy.wire.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(want) {
		t.Fatal("waiver policy wire differs; regenerate with TWR_WRITE_WIRE=1")
	}
}
