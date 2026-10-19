package commissioner

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// The four states a job can be in, in the order the landing lists them.
const (
	JobLive     = "live"
	JobUpcoming = "upcoming"
	JobBlocked  = "blocked"
	JobUnbuilt  = "unbuilt"

	unbuiltWords     = "Not built yet"
	rolloverRunWords = "This season's rollover has been run; the next one has no date yet."
)

// Job is one thing the commissioner works through. Opens and Closes are RFC 3339 instants, empty when
// the state has none; Controls are the census rows the job opens.
type Job struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	State    string   `json:"state"`
	Words    string   `json:"words"`
	Opens    string   `json:"opens"`
	Closes   string   `json:"closes"`
	Controls []string `json:"controls"`
}
type Jobs struct {
	League   string `json:"league"`
	Name     string `json:"name"`
	Revision int64  `json:"revision"`
	Jobs     []Job  `json:"jobs"`
}

// contractWindows are the spring windows that decide each expiring contract (spec §16.6, J4).
func contractWindows() []struct{ id, name string } {
	return []struct{ id, name string }{
		{"contracts", "contract window"}, {"rfa_tenders", "RFA tender window"},
		{"ufa", "UFA bidding window"}, {"resign", "re-signing window"},
	}
}

func unbuilt(id, name string, controls ...string) Job {
	return Job{ID: id, Name: name, State: JobUnbuilt, Words: unbuiltWords, Controls: controls}
}

// JobsReading is public like the rules: every owner may read what the commissioner is working on.
func JobsReading(league string, records []Record, at time.Time) (Jobs, error) {
	if len(records) == 0 {
		return Jobs{}, fmt.Errorf("this app league has not been created")
	}
	rules, err := resolveRecords(records)
	if err != nil {
		return Jobs{}, err
	}
	jobs := jobsFor(rules, ruleCalendar{rules: rules.Calendar, records: records}, at)
	recorded, err := ledgerJobs(league, records, rules, at)
	if err != nil {
		return Jobs{}, err
	}
	jobs = append(jobs, recorded...)
	sortJobs(jobs)
	return Jobs{League: league, Name: Replay(records).Name, Revision: int64(len(records)), Jobs: jobs}, nil
}

// jobsFor works every job out from the rules and the calendar, then puts them in the order the
// landing shows: live first, then upcoming, blocked and not built, each group keeping the spec's order.
func jobsFor(rules leaguerules.RuleSet, cal JobCalendar, at time.Time) []Job {
	var jobs []Job
	if setup, open := setupJob(rules); open {
		jobs = append(jobs, setup)
	}
	jobs = append(jobs, rolloverJob(cal, at),
		contractSeasonJob(cal, at),
		unbuilt("notifications", "Notifications and refresh schedule", "CM-051", "CM-054"))
	sortJobs(jobs)
	return jobs
}

func sortJobs(jobs []Job) {
	rank := map[string]int{JobLive: 0, JobUpcoming: 1, JobBlocked: 2, JobUnbuilt: 3}
	slices.SortStableFunc(jobs, func(a, b Job) int {
		if c := cmp.Compare(rank[a.State], rank[b.State]); c != 0 {
			return c
		}
		// A live job that closes sooner leads; one with no closing time follows those that close.
		if a.State == JobLive && a.Closes != b.Closes {
			if a.Closes == "" || b.Closes == "" {
				return cmp.Compare(b.Closes, a.Closes)
			}
			return cmp.Compare(a.Closes, b.Closes)
		}
		if a.State == JobUpcoming {
			return cmp.Compare(a.Opens, b.Opens)
		}
		return 0
	})
}

// setupJob is live while any rule still Needs a ruling; open is false once every rule is set.
func setupJob(rules leaguerules.RuleSet) (job Job, open bool) {
	fields, values := leaguerules.Catalogue(), leaguerules.ReadSettings(rules)
	var missing []leaguerules.FieldPath
	for i, f := range fields {
		if values[i].State == "Needs a ruling" {
			missing = append(missing, f.Path)
		}
	}
	return setupFor(missing)
}

// setupFor builds the setup job from the rules that still need a ruling.
func setupFor(missing []leaguerules.FieldPath) (job Job, open bool) {
	if len(missing) == 0 {
		return Job{}, false
	}
	owners := map[string]bool{}
	for _, path := range missing {
		if owner, err := ownerFor(path); err == nil {
			owners[owner.control] = true
		}
	}
	noun := "rules still need"
	if len(missing) == 1 {
		noun = "rule still needs"
	}
	controls := make([]string, 0, len(owners))
	for id := range owners {
		controls = append(controls, id)
	}
	slices.Sort(controls)
	return Job{ID: "setup", Name: "League setup", State: JobLive, Controls: controls,
		Words: fmt.Sprintf("%d league %s a ruling.", len(missing), noun)}, true
}

// rolloverJob is blocked until the commissioner dates the rollover, upcoming until that date, and live
// once it has passed: the commissioner previews the rollover and runs it.
func rolloverJob(cal JobCalendar, at time.Time) Job {
	job := Job{ID: "rollover", Name: "Rollover", Controls: []string{"CM-027"}}
	date, reason := cal.RolloverAt()
	switch {
	case reason != "":
		job.State, job.Words = JobBlocked, reason
	case at.Before(date):
		job.State, job.Words, job.Opens = JobUpcoming, "The rollover is dated; preview it before it runs.",
			date.UTC().Format(time.RFC3339)
	default:
		job.State, job.Words = JobLive, "The rollover date has passed; preview the rollover and run it."
	}
	return job
}

func contractSeasonJob(cal JobCalendar, at time.Time) Job {
	job := Job{ID: "contract_season", Name: "Contract season", Controls: []string{"CM-023", "CM-009"}}
	type named struct {
		name string
		WindowDates
	}
	var dated []named
	var undated []string
	for _, w := range contractWindows() {
		dates, ok := cal.Window(w.id)
		if !ok {
			undated = append(undated, w.name)
			continue
		}
		if !dates.Close.After(dates.Start) {
			job.State, job.Words = JobBlocked, "The "+w.name+" closes before it opens; the commissioner fixes its dates."
			return job
		}
		dated = append(dated, named{w.name, dates})
	}
	if len(undated) > 0 {
		job.State = JobBlocked
		job.Words = "These windows have no dates yet: " + strings.Join(undated, ", ") +
			". The commissioner sets them."
		return job
	}
	var open, next *named
	for i, w := range dated {
		if !at.Before(w.Start) && at.Before(w.Close) && (open == nil || w.Close.Before(open.Close)) {
			open = &dated[i]
		}
		if at.Before(w.Start) && (next == nil || w.Start.Before(next.Start)) {
			next = &dated[i]
		}
	}
	switch {
	case open != nil:
		job.State, job.Words, job.Closes = JobLive, "The "+open.name+" is open.", open.Close.UTC().Format(time.RFC3339)
	case next != nil:
		job.State, job.Words, job.Opens = JobUpcoming, "The "+next.name+" opens next.", next.Start.UTC().Format(time.RFC3339)
	default:
		job.State = JobBlocked
		job.Words = "The next contract season has no dates yet; the commissioner sets them."
	}
	return job
}

// WindowDates is when a league window opens and closes.
type WindowDates struct{ Start, Close time.Time }

// JobCalendar is where the jobs get their dates. Today it reads the dates recorded in the league's
// rules (ruleCalendar). An adapter over the native ActionCalendar can replace it without changing jobs.
type JobCalendar interface {
	// RolloverAt is the dated rollover; reason says, in plain words, why there is none.
	RolloverAt() (at time.Time, reason string)
	// Window is the dated opening and closing of one league window; ok is false when either has no date.
	Window(id string) (dates WindowDates, ok bool)
}

// ruleCalendar reads the resolved rules' calendar values: only an exact recorded instant counts as a
// date, because a month and day alone do not say which season they belong to. The records say whether
// the dated rollover has already been run.
type ruleCalendar struct {
	rules   leaguerules.CalendarRules
	records []Record
}

func (c ruleCalendar) RolloverAt() (time.Time, string) {
	if c.rules.Rollover.State != leaguerules.IsKnown {
		return time.Time{}, rolloverWords
	}
	if rolloverRunSinceDated(c.records) {
		return time.Time{}, rolloverRunWords
	}
	return c.rules.Rollover.Value.UTC(), ""
}

// rolloverRunSinceDated says whether a rollover was run after the rollover date in force was set.
func rolloverRunSinceDated(records []Record) bool {
	active := Replay(records).Active
	var dated time.Time
	ran := false
	for _, r := range records {
		switch {
		case !active[r.Receipt.Entry]:
		case r.Command.Command == RulesSet && r.Command.Args.Field == rolloverField:
			dated, ran = r.Receipt.RecordedAt, false
		case recordedRollover(r) && r.Receipt.RecordedAt.After(dated):
			ran = true
		}
	}
	return ran
}

func (c ruleCalendar) Window(id string) (WindowDates, bool) {
	if c.rules.Windows.State != leaguerules.IsKnown {
		return WindowDates{}, false
	}
	w, found := c.rules.Windows.Value[id]
	if !found || w.DatedStart.State != leaguerules.IsKnown || w.DatedEnd.State != leaguerules.IsKnown {
		return WindowDates{}, false
	}
	return WindowDates{Start: w.DatedStart.Value.UTC(), Close: w.DatedEnd.Value.UTC()}, true
}
