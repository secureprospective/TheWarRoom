package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
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

func peopleWireLeague(t *testing.T, app *App) commissioner.RawCommand {
	t.Helper()
	raw := commissioner.RawCommand{League: "native.people", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "People league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 4},
		Reason: "Create our league", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Visibility = "commish.seat.replace", "league"
	for i, owner := range []string{"owner-one", "owner-two", "owner-three"} {
		raw.Args = commissioner.RawArgs{People: &commissioner.RawPeople{
			Franchise: fmt.Sprintf("franchise.%02d", i+1), Principal: owner}}
		raw.Expected, raw.Request = int64(i+1), fmt.Sprintf("seat.%d", i)
		executeCommissionerTest(t, app, raw)
	}
	raw.Command, raw.Visibility = "commish.role.grant", "private"
	raw.Args = commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "private-deputy", Role: governance.CoCommissioner}}
	raw.Expected, raw.Request = 4, "grant"
	executeCommissionerTest(t, app, raw)
	raw.Visibility = "league"
	for i, step := range []struct{ who, command, choice string }{
		{"private-deputy", "commish.succession.voteout.open", ""}, {"owner-two", "commish.vote.cast", "yes"}} {
		app.commissionerSession = testCommissionerSession{step.who}
		raw.Command, raw.Expected, raw.Request = step.command, int64(5+i), fmt.Sprintf("vote.%d", i)
		raw.Visibility = map[bool]string{true: "private", false: "league"}[step.choice != ""]
		raw.Args = commissioner.RawArgs{People: &commissioner.RawPeople{ID: "removal", Principal: step.who,
			Commissioner: "creator", Choice: step.choice}}
		if step.choice != "" {
			raw.Args.People.Commissioner = ""
		}
		executeCommissionerTest(t, app, raw)
	}
	app.commissionerSession = testCommissionerSession{"creator"}
	return raw
}

func TestCommissionerPeopleWire(t *testing.T) {
	t.Parallel()
	app := commissionerTestApp(t)
	raw := peopleWireLeague(t, app)
	views := map[string]commissioner.People{}
	for _, principal := range []string{"creator", "private-deputy", "owner-one", "stranger"} {
		app.commissionerSession = testCommissionerSession{principal}
		people, err := app.TargetCommissionerPeople(raw.League)
		if err != nil || people.Revision != 7 || people.Seats[0].Owner != "owner-one" {
			t.Fatalf("IPC reading: %+v %v", people, err)
		}
		body, err := json.Marshal(people)
		if err != nil {
			t.Fatal(err)
		}
		if principal == "owner-one" || principal == "stranger" {
			if strings.Contains(string(body), "private-deputy") || len(people.Roles) != 0 || people.MayChangeRules {
				t.Fatalf("owner people wire leaks private role: %s", body)
			}
		} else if len(people.Roles) != 2 || !people.MayChangeRules {
			t.Fatalf("session role missing from people wire: %s", body)
		}
		views[principal] = people
	}
	if !views["creator"].MayGrantRoles || views["private-deputy"].MayGrantRoles ||
		views["owner-one"].YourRole != "Owner" || views["stranger"].YourRole != "Visitor" ||
		views["owner-one"].YourFranchise != "franchise.01" || views["creator"].YourFranchise != "" ||
		views["stranger"].YourFranchise != "" {
		t.Fatalf("the reading does not say what each principal may start: %+v", views)
	}
	app.commissionerSession = testCommissionerSession{"creator"}
	raw.Command, raw.Expected, raw.Request = "commish.role.grant", 7, "second.cc"
	raw.Visibility = "private"
	raw.Args = commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "owner-two", Role: governance.CoCommissioner}}
	encoded, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	_, refusal := app.TargetCommissionerExecute(string(encoded))
	// The owner reads the reason alone, without the context each layer adds for the log.
	if refusal == nil || refusal.Error() != "This league already has a co-commissioner: revoke them first" {
		t.Fatalf("a second co-commissioner must be refused in the owner's words: %v", refusal)
	}
	body, err := json.MarshalIndent(map[string]any{"commissioner": views["creator"],
		"owner": views["owner-one"], "secondCoCommissionerRefusal": refusal.Error()}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/people.wire.testdata.json", body, "people")
	if _, err := app.TargetCommissionerPeople("legacy"); err == nil {
		t.Fatal("imported league admitted to native people reading")
	}
}

// checkWire holds a Go-written fixture the frontend parser reads; TWR_WRITE_WIRE=1 rewrites it.
func checkWire(t *testing.T, path string, body []byte, name string) {
	t.Helper()
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
		t.Fatalf("commissioner %s wire is stale", name)
	}
}

func TestCommissionerStaleVersionIsRefusedWithTheWordsTheFormWatchesFor(t *testing.T) {
	t.Parallel()
	app := commissionerTestApp(t)
	raw := peopleWireLeague(t, app)
	raw.Command, raw.Visibility = commissioner.RulesSet, "league"
	raw.Args = commissioner.RawArgs{Field: "roster.limits", Value: json.RawMessage(`{"min":20,"max":70}`)}
	raw.Expected, raw.Request, raw.Effective = 2, "stale", "next_rollover"
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.TargetCommissionerExecute(string(body)); err == nil ||
		!strings.HasPrefix(err.Error(), "League version is ") ||
		!strings.Contains(err.Error(), "reread before changing it") {
		t.Fatalf("a stale version must say the league changed: %v", err)
	}
}

func TestCommissionerWire(t *testing.T) {
	t.Parallel()
	app := commissionerTestApp(t)
	raw := commissioner.RawCommand{League: "native.wire", Command: commissioner.Create,
		Args: commissioner.RawArgs{Name: "Our league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Reason: "Start our own league", Request: "create",
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
	t.Parallel()
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

func TestCommissionerRulesWire(t *testing.T) {
	t.Parallel()
	app := commissionerTestApp(t)
	raw := commissioner.RawCommand{League: "native.rules", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Rules league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 4},
		Reason: "Start our own league", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Reason, raw.Visibility = commissioner.RulesSet, "Owners agreed the roster size", "league"
	raw.Args = commissioner.RawArgs{Field: "roster.limits", Value: json.RawMessage(`{"min":20,"max":70}`)}
	raw.Expected, raw.Request, raw.Effective = 1, "limits", "now"
	recorded := time.Date(2026, 10, 14, 4, 17, 3, 0, time.UTC)
	app.history.SetClock(func() time.Time { return recorded })
	receipt := executeCommissionerTest(t, app, raw)
	raw.Args = commissioner.RawArgs{Field: "cap.amount", Value: json.RawMessage(`12500000000`)}
	raw.Expected, raw.Request, raw.Effective, raw.Reason = 2, "cap", "next_rollover", ""
	executeCommissionerTest(t, app, raw)
	creator, err := app.TargetCommissionerRules(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	app.commissionerSession = testCommissionerSession{"owner"}
	owner, err := app.TargetCommissionerRules(raw.League)
	if err != nil || !reflect.DeepEqual(creator, owner) {
		t.Fatalf("owners read the same public rules: %v", err)
	}
	for _, section := range owner.Sections {
		for _, row := range section.Rows {
			if row.Field == "roster.limits" && (row.ChangedAt != receipt.RecordedAt.Format(time.RFC3339Nano) ||
				row.Citation != "Changed in the league ledger") {
				t.Fatalf("rule source must carry the recorded instant, not a UTC date: %+v", row)
			}
		}
	}
	body, err := json.Marshal(owner)
	if err != nil {
		t.Fatal(err)
	}
	path := "frontend/src/app/commissioner/rules.wire.testdata.json"
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
		t.Fatal("commissioner rules wire is stale")
	}
	for _, league := range []string{"legacy", "native.", " native.rules", "native.missing"} {
		if _, err := app.TargetCommissionerRules(league); err == nil {
			t.Fatalf("%q admitted to the app league rules reading", league)
		}
	}
}

func TestCommissionerLegacyRulesWire(t *testing.T) {
	t.Parallel()
	app := commissionerTestApp(t)
	executeCommissionerTest(t, app, commissioner.RawCommand{League: "native.legacy", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Legacy league", Preset: commissioner.PresetLegacy, Season: 2026},
		Reason: "Start from the Legacy NFL rules", Request: "create", Effective: "now", Visibility: "league"})
	reading, err := app.TargetCommissionerRules("native.legacy")
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(reading)
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/rules.legacy.wire.testdata.json", body, "legacy rules")
}

func TestCommissionerReplacedRulesWire(t *testing.T) {
	t.Parallel()
	app := commissionerTestApp(t)
	raw := commissioner.RawCommand{League: "native.replaced", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Waiting rules", Preset: commissioner.PresetLegacy, Season: 2026, FranchiseCount: 6},
		Reason: "Start", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, app, raw)
	raw.Command, raw.Effective, raw.Reason = commissioner.RulesSet, "next_rollover", ""
	for i, value := range []string{`{"min":20,"max":70}`, `{"min":22,"max":75}`, `{"min":24,"max":78}`} {
		at := time.Date(2026, 10, 14, 4, 17+i, 3, 0, time.UTC)
		app.history.SetClock(func() time.Time { return at })
		raw.Args = commissioner.RawArgs{Field: "roster.limits", Value: json.RawMessage(value)}
		raw.Expected, raw.Request = int64(i+1), fmt.Sprintf("waiting.%d", i)
		executeCommissionerTest(t, app, raw)
	}
	rules, err := app.TargetCommissionerRules(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := app.TargetCommissionerLedger(raw.League)
	if err != nil {
		t.Fatal(err)
	}
	if ledger.Entries[0].ReplacedAt != "" || ledger.Entries[1].ReplacedAt != "2026-10-14T04:19:03Z" ||
		ledger.Entries[2].ReplacedAt != ledger.Entries[1].ReplacedAt {
		t.Fatalf("ledger winners %+v", ledger)
	}
	body, err := json.Marshal(struct {
		Rules  commissioner.Rules  `json:"rules"`
		Ledger commissioner.Ledger `json:"ledger"`
	}{rules, ledger})
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/replaced.wire.testdata.json", body, "replaced rules")
}
