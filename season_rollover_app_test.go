package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// rolloverView is everything a GM can read of the league that the season rollover governs (census CM-027).
type rolloverView struct {
	Season        [5]int // roster check, rules, directory, assets, calendar year read
	Roster        string
	ReserveRows   int
	Pending       []string
	Statuses      map[string]domain.RosterStatus
	Limits        string
	WaitingLimits int
	Deadline      time.Time
	WindowYear    int
	Picks         []commissioner.NativePick
}

func rolloverAppFixture(t *testing.T) *legalityFixture {
	t.Helper()
	named := func(id, name string) snapshot.Player {
		pid, err := playerid.New(id)
		if err != nil {
			t.Fatal(err)
		}
		return snapshot.Player{ID: pid, Name: name}
	}
	f := legalityAppWith(t, map[string]int{"3001": 2026, "1005": 2026},
		[]snapshot.Player{named("3001", "Taxi, Terry"), named("1005", "Starter, Sam")})
	f.set(t, "calendar.boundary", `"start_of_day"`, "now")
	f.set(t, "calendar.tradeDeadline", `"2026-11-09T05:00:00Z"`, "now")
	f.set(t, "roster.limits", `{"min":18,"max":60}`, "next_rollover")
	f.set(t, "roster.limits", `{"min":18,"max":90}`, "next_rollover")
	return f
}

func (f *legalityFixture) rolloverView(t *testing.T) rolloverView {
	t.Helper()
	check, err := f.app.TargetNativeRosterCheck(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := f.app.TargetCommissionerRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	directory, err := f.app.TargetNativeDirectory(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	assets := readNativeTest(t, f.app, legalityLeague)
	view := rolloverView{Roster: f.roster(t, "franchise.02").Words, ReserveRows: len(f.reserves(t).Players),
		Statuses: map[string]domain.RosterStatus{}, Picks: assets.Snapshot.Picks}
	view.Season[0], view.Season[1], view.Season[2], view.Season[3] = check.RosterSeason, check.RulesSeason,
		directory.RosterSeason, assets.Snapshot.Season
	for _, p := range assets.Snapshot.Players {
		view.Statuses[p.ID] = p.Status
		if p.Pending {
			view.Pending = append(view.Pending, p.ID+"@"+p.Franchise)
		}
	}
	for _, section := range rules.Sections {
		for _, row := range section.Rows {
			if row.Field == "roster.limits" {
				view.Limits, view.WaitingLimits = string(row.Current), len(row.Pending)
			}
		}
	}
	view.Season[4] = view.Season[3]
	calendar, err := f.app.TargetLeagueCalendar(RawLeagueCalendarScope{League: legalityLeague, Season: view.Season[3],
		LocalZone: "America/Chicago", Franchise: "franchise.02", Action: "trade"})
	if err != nil {
		t.Fatal(err)
	}
	view.Deadline, view.WindowYear = *calendar.Clock.TradeDeadline, calendar.Clock.Windows[0].Start.Year()
	return view
}

func (f *legalityFixture) rolloverCommand(t *testing.T, to int, dry bool) commissioner.RawCommand {
	t.Helper()
	houses, err := f.app.TargetNativeHouseRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	return commissioner.RawCommand{League: legalityLeague, Command: commissioner.SeasonRollover,
		Args:   commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: to, DryRun: dry}},
		Reason: "Turn the league year", Expected: houses.Revision,
		Request: fmt.Sprintf("rollover.%d.%v", houses.Revision, dry), Effective: "now", Visibility: "league"}
}

// TestSeasonRolloverGovernsEveryReading is the wiring proof (spec 16.10) for CM-027: a dry run changes
// nothing; the real run moves the roster check, the rules, the directory, the assets and the calendar to
// the new season under rulings 9a, 9b and 9c; the compensating revert brings every reading back; and the
// same holds after the database is physically reopened.
func TestSeasonRolloverGovernsEveryReading(t *testing.T) {
	t.Parallel()
	f := rolloverAppFixture(t)
	var first rolloverView
	for pass, name := range []string{"live", "after reopening the database"} {
		if pass == 1 {
			f.reopen(t)
		}
		before := f.rolloverView(t)
		if pass == 0 {
			first = before
			assertBeforeRollover(t, before)
		} else if !reflect.DeepEqual(before, first) {
			t.Fatalf("%s: the readings differ from the first before-state:\n%+v\n%+v", name, before, first)
		}
		preview := executeCommissionerTest(t, f.app, f.rolloverCommand(t, 2027, true))
		if preview.Plan == nil || preview.Entry != "" || preview.Plan.Blocked {
			t.Fatalf("%s: the dry run returned %+v", name, preview)
		}
		if got := f.rolloverView(t); !reflect.DeepEqual(got, before) {
			t.Fatalf("%s: the dry run changed a reading:\n%+v\n%+v", name, got, before)
		}
		act := executeCommissionerTest(t, f.app, f.rolloverCommand(t, 2027, false))
		if act.Plan == nil || act.Plan.BeforeDigest != preview.Plan.BeforeDigest {
			t.Fatalf("%s: the real run and the dry run were made from different states", name)
		}
		assertAfterRollover(t, name, before, f.rolloverView(t), act.Plan)
		f.revert(t, act)
		if got := f.rolloverView(t); !reflect.DeepEqual(got, before) {
			t.Fatalf("%s: the compensating revert did not restore the readings:\n%+v\n%+v", name, got, before)
		}
	}
}

func assertBeforeRollover(t *testing.T, v rolloverView) {
	t.Helper()
	if v.Season != [5]int{2026, 2026, 2026, 2026, 2026} || v.Deadline.Year() != 2026 || v.WindowYear != 2026 ||
		v.Roster != "Active roster 82 of 80: 2 over" || v.ReserveRows != 5 || len(v.Pending) != 0 ||
		v.WaitingLimits != 2 || strings.Contains(v.Limits, "90") {
		t.Fatalf("before the rollover: %+v", v)
	}
}

func assertAfterRollover(t *testing.T, name string, before, after rolloverView, plan *commissioner.RolloverPlan) {
	t.Helper()
	if after.Season != [5]int{2027, 2027, 2027, 2027, 2027} || after.Deadline.Year() != 2027 || after.WindowYear != 2027 {
		t.Fatalf("%s: a reading did not move to the new season: %+v", name, after)
	}
	for id, status := range after.Statuses { // ruling 9a
		if status != domain.RosterActive {
			t.Fatalf("%s: player %s is still on %s", name, id, status)
		}
	}
	if after.ReserveRows != 0 || after.Roster != "Within 18 to 90" { // 9a through the roster check
		t.Fatalf("%s: roster check after the rollover: %q, %d reserve rows", name, after.Roster, after.ReserveRows)
	}
	if !reflect.DeepEqual(after.Pending, []string{"1005@franchise.02", "3001@franchise.02"}) &&
		!reflect.DeepEqual(after.Pending, []string{"3001@franchise.02", "1005@franchise.02"}) { // 9b
		t.Fatalf("%s: expiring players are not kept and marked pending: %v", name, after.Pending)
	}
	if !strings.Contains(after.Limits, `"Max":90`) || after.WaitingLimits != 0 { // the newest waiting change
		t.Fatalf("%s: the waiting roster limits: %s, %d still waiting", name, after.Limits, after.WaitingLimits)
	}
	if !reflect.DeepEqual(after.Picks, before.Picks) { // 9c
		t.Fatalf("%s: picks were issued or changed at rollover", name)
	}
	lines := strings.Join(plan.Lines, "\n")
	for _, want := range []string{"Taxi, Terry is pending a contract decision, not released.",
		"Starter, Sam is pending a contract decision, not released.", "Taxi, Terry returns from taxi",
		"No draft picks are issued at rollover."} {
		if !strings.Contains(lines, want) {
			t.Fatalf("%s: the plan does not say %q", name, want)
		}
	}
	if len(plan.Pending) != 2 || !after.Deadline.After(before.Deadline) {
		t.Fatalf("%s: plan pending %d, deadline %v -> %v", name, len(plan.Pending), before.Deadline, after.Deadline)
	}
}

func TestSeasonRolloverRefusalsThroughTheApp(t *testing.T) {
	t.Parallel()
	f := rolloverAppFixture(t)
	for _, to := range []int{2026, 2028} {
		body, err := json.Marshal(f.rolloverCommand(t, to, false))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.app.TargetCommissionerExecute(string(body)); err == nil ||
			!strings.Contains(err.Error(), "Move only from season 2026 to season 2027") {
			t.Fatalf("to %d: %v", to, err)
		}
	}
	f.app.core.CommissionerSession = testCommissionerSession{"owner"}
	body, err := json.Marshal(f.rolloverCommand(t, 2027, false))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.TargetCommissionerExecute(string(body)); err == nil {
		t.Fatal("an owner rolled the league over")
	}
	if got := f.rolloverView(t).Season; got != [5]int{2026, 2026, 2026, 2026, 2026} {
		t.Fatalf("a refused rollover moved the league: %v", got)
	}
}
