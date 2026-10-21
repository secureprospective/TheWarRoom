package composition

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func jobsLeague(t *testing.T, app *App, league, preset string) commissioner.RawCommand {
	t.Helper()
	raw := commissioner.RawCommand{League: league, Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Jobs league", Preset: preset, FranchiseCount: 4, Season: 2026},
		Reason: "Start", Request: "create", Effective: "now", Visibility: "league"}
	executeCommissionerTest(t, app, raw)
	return raw
}

// setRule records one rule change that takes effect now, and returns the next expected version.
func setRule(t *testing.T, app *App, raw commissioner.RawCommand, field, value string) commissioner.RawCommand {
	t.Helper()
	raw.Command, raw.Request, raw.Reason = commissioner.RulesSet, "set."+field, "Agreed by the owners"
	raw.Args = commissioner.RawArgs{Field: field, Value: json.RawMessage(value)}
	executeCommissionerTest(t, app, raw)
	raw.Expected++
	return raw
}

// datedWindows is the league's own windows reading with exact dates added to the spring windows.
func datedWindows(t *testing.T, app *App, league string) string {
	t.Helper()
	rules, err := newCommissionerAPI(app).TargetCommissionerRules(league)
	if err != nil {
		t.Fatal(err)
	}
	var windows map[string]map[string]any
	for _, s := range rules.Sections {
		for _, r := range s.Rows {
			if r.Field == "calendar.windows" {
				if err := json.Unmarshal(r.Current, &windows); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	for id, dates := range map[string][2]string{
		"contracts":   {"2027-03-01T05:00:00Z", "2027-05-01T04:00:00Z"},
		"rfa_tenders": {"2027-05-01T04:00:00Z", "2027-05-03T04:00:00Z"},
		"ufa":         {"2027-05-03T04:00:00Z", "2027-05-10T04:00:00Z"},
		"resign":      {"2027-05-10T04:00:00Z", "2027-05-15T04:00:00Z"},
	} {
		windows[id]["DatedStart"], windows[id]["DatedEnd"] = dates[0], dates[1]
	}
	body, err := json.Marshal(windows)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestCommissionerJobsWire(t *testing.T) {
	t.Parallel()
	app := commissionerTestApp(t)
	at := func(rfc string) {
		parsed, err := time.Parse(time.RFC3339, rfc)
		if err != nil {
			t.Fatal(err)
		}
		app.core.History.SetClock(func() time.Time { return parsed })
	}
	views := map[string]commissioner.Jobs{}
	read := func(name, league string) commissioner.Jobs {
		reading, err := newCommissionerAPI(app).TargetCommissionerJobs(league)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		views[name] = reading
		return reading
	}
	state := func(reading commissioner.Jobs, id string) commissioner.Job {
		for _, j := range reading.Jobs {
			if j.ID == id {
				return j
			}
		}
		t.Fatalf("no %s job in %+v", id, reading.Jobs)
		return commissioner.Job{}
	}
	jobsLeague(t, app, "native.empty", commissioner.PresetEmpty)
	empty := read("empty", "native.empty")
	if setup := state(empty, "setup"); setup.State != commissioner.JobLive || empty.Jobs[0].ID != "setup" {
		t.Fatalf("a new empty league starts in setup: %+v", empty.Jobs)
	}
	raw := jobsLeague(t, app, "native.legacy", commissioner.PresetLegacy)
	fresh := read("fresh", "native.legacy")
	if j := state(fresh, "contract_season"); j.State != commissioner.JobBlocked ||
		!strings.HasPrefix(j.Words, "These windows have no dates yet: contract window, RFA tender window,") {
		t.Fatalf("undated windows block the contract season: %+v", j)
	}
	if j := state(fresh, "rollover"); j.State != commissioner.JobBlocked ||
		j.Words != "The next rollover has no date yet; the commissioner sets it." {
		t.Fatalf("an undated rollover is blocked: %+v", j)
	}
	raw.Expected = 1
	raw = setRule(t, app, raw, "calendar.windows", datedWindows(t, app, "native.legacy"))
	raw = setRule(t, app, raw, "calendar.rollover", `"2027-03-01T05:00:00Z"`)
	at("2027-02-01T12:00:00Z")
	read("before", "native.legacy")
	if j := state(views["before"], "contract_season"); j.State != commissioner.JobUpcoming ||
		j.Opens != "2027-03-01T05:00:00Z" {
		t.Fatalf("before the first window the season is upcoming: %+v", j)
	}
	at("2027-05-02T12:00:00Z")
	live := read("live", "native.legacy")
	if j := state(live, "contract_season"); j.State != commissioner.JobLive || j.Closes != "2027-05-03T04:00:00Z" ||
		live.Jobs[0].ID != "contract_season" {
		t.Fatalf("inside the RFA window the season is live and leads: %+v", live.Jobs)
	}
	if j := state(live, "rollover"); j.State != commissioner.JobLive {
		t.Fatalf("a rollover whose date has passed is the commissioner's to run: %+v", j)
	}
	if _, err := newCommissionerAPI(app).TargetCommissionerJobs("mfl.123"); err == nil || !strings.Contains(err.Error(), "choose an app league") {
		t.Fatalf("an imported league has no commissioner jobs: %v", err)
	}
	if _, err := newCommissionerAPI(app).TargetCommissionerJobs("native.missing"); err == nil {
		t.Fatal("a league that was never created has no jobs")
	}
	body, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "frontend/src/app/commissioner/jobs.wire.testdata.json", body, "jobs")
}
