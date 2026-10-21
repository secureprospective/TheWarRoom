package composition

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func westCoast(t *testing.T) *time.Location {
	t.Helper()
	zone, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	return zone
}

// gateProbe is a registered native owner action; ran records each franchise it was run for.
type gateProbe struct {
	registry []NativeOwnerAction
	ran      []string
}

func newGateProbe(action leaguerules.Action) *gateProbe {
	p := &gateProbe{}
	p.registry = append(nativeOwnerActions(), NativeOwnerAction{Name: "probe", Opens: action,
		Run: func(_ context.Context, _ *App, permit commissioner.Permit, _ json.RawMessage) (commissioner.Receipt, error) {
			p.ran = append(p.ran, permit.Franchise())
			return commissioner.Receipt{}, nil
		}})
	return p
}

// try runs the probe as one franchise and says whether it ran, and what the gate told the owner if not.
func (p *gateProbe) try(t *testing.T, app *App, league, franchise string) (bool, string) {
	t.Helper()
	before := len(p.ran)
	_, err := app.runNativeAction(p.registry, league, franchise, "probe", nil)
	ran := len(p.ran) == before+1
	if ran != (err == nil) {
		t.Fatal("the action ran or was refused inconsistently", ran, err)
	}
	if err != nil {
		var refused *commissioner.Refusal
		if !errors.As(err, &refused) {
			t.Fatal("a closed gate is a refusal the owner can read", err)
		}
		return false, refused.Reason()
	}
	return true, ""
}

func windowCommandFrom(raw commissioner.RawCommand, command, request string, expected int64,
	w commissioner.RawWindowArgs) commissioner.RawCommand {
	raw.Command, raw.Request, raw.Expected = command, request, expected
	raw.Args = commissioner.RawArgs{Window: &w}
	return raw
}

func date(month, day int) *commissioner.RawDate { return &commissioner.RawDate{Month: month, Day: day} }

func TestWindowMoveRedatesTheGateRevertRestart(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	app.core.LocalZone = westCoast(t)
	raw := seedCalendarTest(t, app)
	probe := newGateProbe("ufa_bid")
	if ran, why := probe.try(t, app, raw.League, "franchise.01"); !ran {
		t.Fatal("the window is open on May 5", why)
	}
	moved := executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowMove, "move", 4,
		commissioner.RawWindowArgs{ID: "ufa", Start: date(5, 8)}))
	for range 2 {
		ran, why := probe.try(t, app, raw.League, "franchise.01")
		want := "This action opens with the ufa window on May 7, 2026 at 9:00 PM PDT."
		if ran || why != want {
			t.Fatalf("a moved window must re-date the decision: ran=%v %q", ran, why)
		}
		restart()
	}
	writeWindowWire(t, calendarOwnerRead(t, app))
	revert := raw
	revert.Command, revert.Args = commissioner.Revert, commissioner.RawArgs{Entry: moved.Entry}
	revert.Expected, revert.Request = 5, "restore-move"
	executeCommissionerTest(t, app, revert)
	for range 2 {
		if ran, why := probe.try(t, app, raw.League, "franchise.01"); !ran {
			t.Fatal("the compensated move must reopen the action", why)
		}
		restart()
	}
}

func TestWindowSetAndRemoveGateAnAction(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	app.core.LocalZone = westCoast(t)
	raw := seedCalendarTest(t, app)
	probe := newGateProbe("cut")
	if ran, _ := probe.try(t, app, raw.League, "franchise.01"); ran {
		t.Fatal("no window opens a cut yet")
	}
	set := executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowSet, "set", 4,
		commissioner.RawWindowArgs{ID: "cuts", Start: date(6, 1), End: date(6, 5),
			Opens: []leaguerules.Action{"cut", "buyout"}}))
	if again := executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowSet, "set", 4,
		commissioner.RawWindowArgs{ID: "cuts", Start: date(6, 1), End: date(6, 5),
			Opens: []leaguerules.Action{"cut", "buyout"}})); again.Entry != set.Entry {
		t.Fatal("a retried set appended twice")
	}
	restart()
	if ran, why := probe.try(t, app, raw.League, "franchise.01"); !ran {
		t.Fatal("the new window opens a cut on June 2", why)
	}
	removed := executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowRemove, "remove", 5,
		commissioner.RawWindowArgs{ID: "cuts"}))
	restart()
	if ran, _ := probe.try(t, app, raw.League, "franchise.01"); ran {
		t.Fatal("a removed window must close the action")
	}
	revert := raw
	revert.Command, revert.Args = commissioner.Revert, commissioner.RawArgs{Entry: removed.Entry}
	revert.Expected, revert.Request = 6, "restore-remove"
	executeCommissionerTest(t, app, revert)
	restart()
	if ran, why := probe.try(t, app, raw.League, "franchise.01"); !ran {
		t.Fatal("the compensated removal must restore the window", why)
	}
}

func rulesWindows(t *testing.T, app *App, league string) json.RawMessage {
	t.Helper()
	rules, err := newCommissionerAPI(app).TargetCommissionerRules(league)
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range rules.Sections {
		for _, row := range section.Rows {
			if row.Field == "calendar.windows" {
				return row.Current
			}
		}
	}
	t.Fatal("the rules desk has no windows row")
	return nil
}

func TestWindowCommandsAndRulesSetNeverDisagree(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	raw := seedCalendarTest(t, app)
	// A second league reaches the same windows through the named commands alone.
	other := raw
	other.League, other.Request, other.Expected = "native.byverbs", "create", 0
	other.Command = commissioner.Create
	other.Args = commissioner.RawArgs{Name: "By commands", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}
	executeCommissionerTest(t, app, other)
	other.Command = commissioner.RulesSet
	for i, s := range []struct{ field, value string }{{"calendar.timezone", `"America/New_York"`},
		{"calendar.boundary", `"start_of_day"`}} {
		other.Args = commissioner.RawArgs{Field: s.field, Value: json.RawMessage(s.value)}
		other.Request, other.Expected = s.field, int64(i+1)
		executeCommissionerTest(t, app, other)
	}
	executeCommissionerTest(t, app, windowCommandFrom(other, commissioner.WindowSet, "rfa", 3,
		commissioner.RawWindowArgs{ID: "rfa", Start: date(5, 1), End: date(5, 3),
			Opens: []leaguerules.Action{"rfa_tender"}}))
	executeCommissionerTest(t, app, windowCommandFrom(other, commissioner.WindowSet, "ufa", 4,
		commissioner.RawWindowArgs{ID: "ufa", Start: date(5, 3), End: date(5, 10),
			Opens: []leaguerules.Action{"ufa_bid"}}))
	read := func(league string) commissioner.Calendar {
		got, err := newCommissionerAPI(app).TargetLeagueCalendar(RawLeagueCalendarScope{League: league, Season: 2026,
			LocalZone: "UTC", Franchise: "franchise.01", Action: "ufa_bid"})
		if err != nil {
			t.Fatal(err)
		}
		got.League, got.Revision = "", 0
		return got
	}
	same := func(when string) {
		t.Helper()
		if !reflect.DeepEqual(read(raw.League), read(other.League)) ||
			string(rulesWindows(t, app, raw.League)) != string(rulesWindows(t, app, other.League)) {
			t.Fatalf("%s: the two doors disagree\n%s\n%s", when, rulesWindows(t, app, raw.League),
				rulesWindows(t, app, other.League))
		}
	}
	same("set through rules.set and through window.set")
	// One league moves through the verb, the other through a whole-value rules.set.
	executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowMove, "move", 4,
		commissioner.RawWindowArgs{ID: "ufa", Start: date(5, 4)}))
	other.Command, other.Request, other.Expected = commissioner.RulesSet, "whole", 5
	other.Args = commissioner.RawArgs{Field: "calendar.windows", Value: json.RawMessage(`{
		"rfa":{"Start":{"Month":5,"Day":1},"End":{"Month":5,"Day":3},"Opens":["rfa_tender"]},
		"ufa":{"Start":{"Month":5,"Day":4},"End":{"Month":5,"Day":10},"Opens":["ufa_bid"]}}`)}
	executeCommissionerTest(t, app, other)
	same("moved through window.move and through rules.set")
	restart()
	same("after reopening the database")
	// A verb after a whole-value rules.set builds on it.
	executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowRemove, "remove", 5,
		commissioner.RawWindowArgs{ID: "rfa"}))
	other.Args = commissioner.RawArgs{Field: "calendar.windows", Value: json.RawMessage(`{
		"ufa":{"Start":{"Month":5,"Day":4},"End":{"Month":5,"Day":10},"Opens":["ufa_bid"]}}`)}
	other.Request, other.Expected = "whole-again", 6
	executeCommissionerTest(t, app, other)
	same("removed through window.remove and through rules.set")
	if strings.Contains(string(rulesWindows(t, app, raw.League)), "rfa") {
		t.Fatal("the removed window is still in force")
	}
}

func TestWindowCommandAuthorityAndSafety(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	app, _ := calendarTestApp(t, at)
	raw := seedCalendarTest(t, app)
	grant := raw
	grant.Command, grant.Visibility, grant.Request, grant.Expected = "commish.role.grant", "private", "grant", 4
	grant.Args = commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "deputy", Role: governance.CoCommissioner}}
	executeCommissionerTest(t, app, grant)
	seat := raw
	seat.Command, seat.Request, seat.Expected = "commish.seat.replace", "seat", 5
	seat.Args = commissioner.RawArgs{People: &commissioner.RawPeople{Franchise: "franchise.01", Principal: "owner"}}
	executeCommissionerTest(t, app, seat)
	move := windowCommandFrom(raw, commissioner.WindowMove, "move", 6,
		commissioner.RawWindowArgs{ID: "ufa", Start: date(5, 4)})
	send := func(c commissioner.RawCommand) error {
		body, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		_, err = newCommissionerAPI(app).TargetCommissionerExecute(string(body))
		return err
	}
	for _, who := range []string{"owner", "outsider"} {
		app.core.CommissionerSession = testCommissionerSession{who}
		if send(move) == nil {
			t.Fatalf("%s changed a league window", who)
		}
	}
	app.core.CommissionerSession = testCommissionerSession{"deputy"}
	for name, bad := range map[string]func(*commissioner.RawCommand){
		"unknown window":  func(c *commissioner.RawCommand) { c.Args.Window.ID = "nothing" },
		"impossible day":  func(c *commissioner.RawCommand) { c.Args.Window.Start = date(2, 30) },
		"no start":        func(c *commissioner.RawCommand) { c.Args.Window.Start = nil },
		"private":         func(c *commissioner.RawCommand) { c.Visibility = "private" },
		"later":           func(c *commissioner.RawCommand) { c.Effective = "next_rollover" },
		"move with opens": func(c *commissioner.RawCommand) { c.Args.Window.Opens = []leaguerules.Action{"cut"} },
		"stale version":   func(c *commissioner.RawCommand) { c.Expected = 3 },
		"lowered gravity": func(c *commissioner.RawCommand) { c.Gravity = "G1" },
		"other args":      func(c *commissioner.RawCommand) { c.Args.Name = "x" },
	} {
		trial := move
		window := *move.Args.Window
		trial.Args.Window = &window
		bad(&trial)
		if send(trial) == nil {
			t.Fatalf("accepted an unsafe window change: %s", name)
		}
	}
	if calendarOwnerRead(t, app).Revision != 6 {
		t.Fatal("a refused window change wrote to the ledger")
	}
	if err := send(move); err != nil {
		t.Fatal("a co-commissioner may move a window", err)
	}
	set := windowCommandFrom(raw, commissioner.WindowSet, "empty", 7, commissioner.RawWindowArgs{ID: "x", Start: date(6, 1)})
	if send(set) == nil {
		t.Fatal("a window that opens nothing was accepted")
	}
}

func TestReversalCannotOrphanALaterWindowChange(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	raw := seedCalendarTest(t, app)
	set := executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowSet, "set", 4,
		commissioner.RawWindowArgs{ID: "extra", Start: date(6, 1), Opens: []leaguerules.Action{"cut"}}))
	executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowMove, "move", 5,
		commissioner.RawWindowArgs{ID: "extra", Start: date(6, 2)}))
	revert := raw
	revert.Command, revert.Args = commissioner.Revert, commissioner.RawArgs{Entry: set.Entry}
	revert.Expected, revert.Request = 6, "undo-set"
	body, err := json.Marshal(revert)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newCommissionerAPI(app).TargetCommissionerExecute(string(body)); err == nil ||
		!strings.Contains(err.Error(), "extra window does not exist") {
		t.Fatal("reversing the window a later move depends on must be refused", err)
	}
	restart()
	if calendarOwnerRead(t, app).Revision != 6 {
		t.Fatal("the refused reversal wrote to the ledger")
	}
}

func writeWindowWire(t *testing.T, reading commissioner.Calendar) {
	t.Helper()
	body, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("internal", "leagueclock", "testdata", "window-move-wire.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil || strings.TrimSpace(string(held)) != string(body) {
		t.Fatal("window wire changed; regenerate with TWR_WRITE_WIRE=1", err)
	}
}

func TestWindowMoveKeepsNoStaleDatedStart(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	app, restart := calendarTestApp(t, at)
	app.core.LocalZone = westCoast(t)
	raw := seedCalendarTest(t, app)
	raw.Args = commissioner.RawArgs{Field: "calendar.windows", Value: json.RawMessage(`{
		"ufa":{"Start":{"Month":5,"Day":3},"DatedStart":"2026-05-03T04:00:00Z","End":{"Month":5,"Day":10},
		"Opens":["ufa_bid"]}}`)}
	raw.Command, raw.Request, raw.Expected = commissioner.RulesSet, "dated", 4
	executeCommissionerTest(t, app, raw)
	probe := newGateProbe("ufa_bid")
	if ran, why := probe.try(t, app, raw.League, "franchise.01"); !ran {
		t.Fatal("the dated window is open on May 5", why)
	}
	executeCommissionerTest(t, app, windowCommandFrom(raw, commissioner.WindowMove, "move", 5,
		commissioner.RawWindowArgs{ID: "ufa", Start: date(5, 8)}))
	restart()
	if ran, why := probe.try(t, app, raw.League, "franchise.01"); ran ||
		!strings.Contains(why, "May 7, 2026 at 9:00 PM PDT") {
		t.Fatal("a dated opening must not outrank the moved day", ran, why)
	}
}
