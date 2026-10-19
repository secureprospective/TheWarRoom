package commissioner

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type fakeCalendar struct {
	rollover time.Time
	windows  map[string]WindowDates
}

func (f fakeCalendar) RolloverAt() (time.Time, string) {
	if f.rollover.IsZero() {
		return time.Time{}, rolloverWords
	}
	return f.rollover, ""
}
func (f fakeCalendar) Window(id string) (WindowDates, bool) {
	dates, ok := f.windows[id]
	return dates, ok
}

func day(month time.Month, d int) time.Time { return time.Date(2027, month, d, 5, 0, 0, 0, time.UTC) }

func springWindows() map[string]WindowDates {
	return map[string]WindowDates{
		"contracts":   {Start: day(3, 1), Close: day(4, 30)},
		"rfa_tenders": {Start: day(5, 1), Close: day(5, 3)},
		"ufa":         {Start: day(5, 3), Close: day(5, 10)},
		"resign":      {Start: day(5, 10), Close: day(5, 15)},
	}
}

func byID(t *testing.T, jobs []Job, id string) Job {
	t.Helper()
	for _, j := range jobs {
		if j.ID == id {
			return j
		}
	}
	t.Fatalf("job %s is missing from %+v", id, jobs)
	return Job{}
}

func TestContractSeasonIsLiveUpcomingOrBlockedFromItsDates(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	cal := fakeCalendar{windows: springWindows()}
	live := byID(t, jobsFor(rules, cal, day(5, 2)), "contract_season")
	if live.State != JobLive || live.Closes != "2027-05-03T05:00:00Z" || live.Opens != "" ||
		live.Words != "The RFA tender window is open." {
		t.Fatalf("inside the RFA window the job is live until it closes: %+v", live)
	}
	// Two windows share the 3 May instant: the job leads with the one that closes first.
	if shared := byID(t, jobsFor(rules, cal, day(5, 3)), "contract_season"); shared.Words != "The UFA bidding window is open." {
		t.Fatalf("at the shared instant the later window is the open one: %+v", shared)
	}
	before := byID(t, jobsFor(rules, cal, day(2, 1)), "contract_season")
	if before.State != JobUpcoming || before.Opens != "2027-03-01T05:00:00Z" || before.Closes != "" ||
		before.Words != "The contract window opens next." {
		t.Fatalf("before the first window the job is upcoming: %+v", before)
	}
	gap := byID(t, jobsFor(rules, cal, day(5, 16)), "contract_season")
	if gap.State != JobBlocked || !strings.Contains(gap.Words, "no dates yet") {
		t.Fatalf("after every window has closed the next season has no dates: %+v", gap)
	}
}

func TestContractSeasonBlockedSaysWhichWindowsHaveNoDates(t *testing.T) {
	windows := springWindows()
	delete(windows, "ufa")
	delete(windows, "resign")
	job := byID(t, jobsFor(leaguerules.LegacyNFL(), fakeCalendar{windows: windows}, day(5, 2)), "contract_season")
	want := "These windows have no dates yet: UFA bidding window, re-signing window. The commissioner sets them."
	if job.State != JobBlocked || job.Words != want || job.Closes != "" || job.Opens != "" {
		t.Fatalf("an undated window blocks the job in words: %+v", job)
	}
	windows = springWindows()
	windows["ufa"] = WindowDates{Start: day(5, 10), Close: day(5, 3)}
	job = byID(t, jobsFor(leaguerules.LegacyNFL(), fakeCalendar{windows: windows}, day(5, 2)), "contract_season")
	if job.State != JobBlocked || !strings.HasPrefix(job.Words, "The UFA bidding window closes before it opens") {
		t.Fatalf("a window that closes before it opens blocks the job: %+v", job)
	}
}

func TestRolloverIsBlockedUntilDatedThenUpcomingThenLive(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	none := byID(t, jobsFor(rules, fakeCalendar{}, day(1, 1)), "rollover")
	if none.State != JobBlocked || none.Words != "The next rollover has no date yet; the commissioner sets it." {
		t.Fatalf("an undated rollover is blocked in the commissioner's words: %+v", none)
	}
	soon := byID(t, jobsFor(rules, fakeCalendar{rollover: day(3, 1)}, day(1, 1)), "rollover")
	if soon.State != JobUpcoming || soon.Opens != "2027-03-01T05:00:00Z" {
		t.Fatalf("a dated rollover still to come is upcoming from its date: %+v", soon)
	}
	due := byID(t, jobsFor(rules, fakeCalendar{rollover: day(3, 1)}, day(3, 1)), "rollover")
	if due.State != JobLive || due.Opens != "" {
		t.Fatalf("at its date the rollover is work for the commissioner: %+v", due)
	}
}

func TestOnlyNotificationsAreNotBuiltYet(t *testing.T) {
	jobs := jobsFor(leaguerules.LegacyNFL(), fakeCalendar{windows: springWindows()}, day(5, 2))
	for _, j := range jobs {
		if (j.State == JobUnbuilt) != (j.ID == "notifications") {
			t.Fatalf("only notifications and the refresh schedule are unbuilt: %+v", j)
		}
	}
	job := byID(t, jobs, "notifications")
	if job.Words != "Not built yet" || job.Opens != "" || job.Closes != "" {
		t.Fatalf("an unbuilt job never claims to work: %+v", job)
	}
}

func TestLiveJobsComeFirstThenUpcomingBlockedAndNotBuilt(t *testing.T) {
	cal := fakeCalendar{windows: springWindows()}
	jobs := jobsFor(leaguerules.LegacyNFL(), cal, day(5, 2))
	var states []string
	for _, j := range jobs {
		states = append(states, j.State)
	}
	rank := map[string]int{JobLive: 0, JobUpcoming: 1, JobBlocked: 2, JobUnbuilt: 3}
	if !slices.IsSortedFunc(states, func(a, b string) int { return rank[a] - rank[b] }) || states[0] != JobLive {
		t.Fatalf("live first, then upcoming, blocked, not built: %v", states)
	}
	if jobs[0].ID != "contract_season" || jobs[1].ID != "setup" {
		t.Fatalf("the live job that closes leads the one with no closing time: %+v", jobs[:2])
	}
	if jobs[2].ID != "rollover" || jobs[2].State != JobBlocked {
		t.Fatalf("the blocked rollover follows the live jobs: %+v", jobs[2])
	}
	soon := byID(t, jobsFor(leaguerules.LegacyNFL(), cal, day(2, 1)), "contract_season")
	if soon.State != JobUpcoming {
		t.Fatalf("a job before its window is upcoming: %+v", soon)
	}
}

func TestSetupIsLiveWhileARuleNeedsARulingThenLeavesTheList(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	setup := byID(t, jobsFor(rules, fakeCalendar{}, day(1, 1)), "setup")
	if setup.State != JobLive || setup.Closes != "" || !strings.HasSuffix(setup.Words, "still need a ruling.") {
		t.Fatalf("setup is live with no closing time: %+v", setup)
	}
	for _, id := range setup.Controls {
		if _, ok := governance.CensusControl(id); !ok {
			t.Fatalf("setup opens an unknown control %s", id)
		}
	}
	if !slices.Contains(setup.Controls, "CM-024") || !slices.Contains(setup.Controls, "CM-025") {
		t.Fatalf("setup opens the rows that own the unset rules: %v", setup.Controls)
	}
	fields, values := leaguerules.Catalogue(), leaguerules.ReadSettings(rules)
	needs := 0
	for i := range fields {
		if values[i].State == "Needs a ruling" {
			needs++
		}
	}
	if needs < 2 || setup.Words != fmt.Sprintf("%d league rules still need a ruling.", needs) {
		t.Fatalf("setup counts the rules that need a ruling (%d): %q", needs, setup.Words)
	}
	one, open := setupFor([]leaguerules.FieldPath{"calendar.boundary"})
	if !open || one.Words != "1 league rule still needs a ruling." || !slices.Equal(one.Controls, []string{"CM-024"}) {
		t.Fatalf("one unset rule reads in the singular and opens its own row: %+v", one)
	}
	if done, open := setupFor(nil); open {
		t.Fatalf("setup leaves the list once every rule is set: %+v", done)
	}
}

func TestEveryJobOpensRealCensusRows(t *testing.T) {
	jobs := jobsFor(leaguerules.LegacyNFL(), fakeCalendar{windows: springWindows()}, day(5, 2))
	for _, j := range jobs {
		if len(j.Controls) == 0 {
			t.Fatalf("job %s opens nothing", j.ID)
		}
		for _, id := range j.Controls {
			if _, ok := governance.CensusControl(id); !ok {
				t.Fatalf("job %s opens an unknown control %s", j.ID, id)
			}
		}
	}
}

func TestRuleCalendarCountsOnlyRecordedInstants(t *testing.T) {
	calendar := leaguerules.LegacyNFL().Calendar
	cal := ruleCalendar{rules: calendar}
	if _, reason := cal.RolloverAt(); reason != rolloverWords {
		t.Fatalf("an unset rollover is the commissioner's to date: %q", reason)
	}
	if _, ok := cal.Window("contracts"); ok {
		t.Fatal("a window with only a month and day has no date for a season")
	}
	windows := calendar.Windows.Value
	w := windows["contracts"]
	w.DatedStart = leaguerules.Known(day(3, 1))
	w.DatedEnd = leaguerules.Known(day(4, 30))
	windows["contracts"] = w
	calendar.Rollover = leaguerules.Known(day(3, 1))
	cal = ruleCalendar{rules: calendar}
	if dates, ok := cal.Window("contracts"); !ok || !dates.Start.Equal(day(3, 1)) || !dates.Close.Equal(day(4, 30)) {
		t.Fatalf("a dated window reads back: %+v %v", dates, ok)
	}
	if at, reason := cal.RolloverAt(); reason != "" || !at.Equal(day(3, 1)) {
		t.Fatalf("a dated rollover reads back: %v %q", at, reason)
	}
	if _, ok := cal.Window("no_such_window"); ok {
		t.Fatal("a window the league does not have has no dates")
	}
	calendar.Windows = leaguerules.Needs[map[string]leaguerules.Window]("test")
	if _, ok := (ruleCalendar{rules: calendar}).Window("contracts"); ok {
		t.Fatal("windows that need a ruling have no dates")
	}
}
