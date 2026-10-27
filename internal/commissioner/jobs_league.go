package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const startingAssetsWords = "Record starting assets first."

// ledgerJobs are the jobs whose state comes from the league's recorded changes, not only its rules.
func ledgerJobs(league string, records []Record, rules leaguerules.RuleSet, at time.Time) ([]Job, error) {
	assets, state, err := nativeAssetsState(league, records)
	if err != nil {
		return nil, err
	}
	jobs := transactionJobs(assets, state, rules, at)
	jobs = append(jobs, mustCutJobs(assets, RecordsThrough(records, at), rules)...)
	return append(jobs,
		draftJob(assets, RecordsThrough(records, at), at),
		seededJob(assets, Job{ID: "stat_queue", Name: "Stat approval queue", Controls: []string{"CM-031", "CM-032"},
			Words: "Statistic changes from the held files wait in the queue for approval."}),
		tradeReviewJob(assets, state.Trades),
		playoffJob(records, rules, at),
		seededJob(assets, Job{ID: "in_season", Name: "In-season operations",
			Controls: []string{"CM-028", "CM-029", "CM-030", "CM-033", "CM-034", "CM-038", "CM-039"},
			Words:    "Corrections, overrides and exceptions are open when needed; each needs a recorded reason."}),
		Job{ID: "communications", Name: "League notices and news", State: JobLive, Controls: []string{"CM-052", "CM-053"},
			Words: "Official notices and What's New posts go to the league from here."}), nil
}

// seededJob is live once the league's starting assets are recorded; until then it waits for them.
func seededJob(assets NativeAssets, job Job) Job {
	job.State = JobLive
	if !assets.Seeded {
		job.State, job.Words = JobBlocked, startingAssetsWords
	}
	return job
}

// draftJob follows the next rookie draft nobody has closed: undated, waiting for its start, or under way.
func draftJob(assets NativeAssets, through []Record, at time.Time) Job {
	job := Job{ID: "rookie_draft", Name: "Rookie draft",
		Controls: []string{"CM-040", "CM-041", "CM-042", "CM-043", "CM-044", "CM-045"}}
	if !assets.Seeded {
		job.State, job.Words = JobBlocked, startingAssetsWords
		return job
	}
	season, ok := draftSeasonPending(assets, through)
	if !ok {
		job.State, job.Words = JobBlocked, "No rookie draft is left to run; closing a draft issues the next picks."
		return job
	}
	start, dated := draftStartFor(through, season)
	switch {
	case !dated:
		job.State, job.Words = JobBlocked, fmt.Sprintf("The %d rookie draft has no start time; the commissioner sets it.", season)
	case at.Before(start):
		job.State, job.Words, job.Opens = JobUpcoming, fmt.Sprintf("The %d rookie draft starts at its set time.", season),
			start.UTC().Format(time.RFC3339)
	default:
		job.State, job.Words = JobLive, fmt.Sprintf("The %d rookie draft is under way.", season)
	}
	return job
}

// tradeReviewJob counts the trades waiting on the DOT and the appeals waiting on a commissioner.
func tradeReviewJob(assets NativeAssets, trades []TradeState) Job {
	job := Job{ID: "dot_review", Name: "DOT review and appeal", State: JobLive, Controls: []string{"CM-014", "CM-037"}}
	if !assets.Seeded {
		job.State, job.Words = JobBlocked, startingAssetsWords
		return job
	}
	review, appealed := 0, 0
	for _, t := range trades {
		switch t.Status {
		case "review":
			review++
		case "appealed":
			appealed++
		}
	}
	job.Words = fmt.Sprintf("Trades awaiting review: %d. Appeals awaiting a ruling: %d.", review, appealed)
	return job
}

// playoffJob is live once this season's bracket is built, and upcoming before that.
func playoffJob(records []Record, rules leaguerules.RuleSet, at time.Time) Job {
	job := Job{ID: "playoffs", Name: "Playoffs", Controls: []string{"CM-016", "CM-046", "CM-047"}}
	p := rules.Playoffs
	if p.StartWeek.State != leaguerules.IsKnown || p.FinalWeek.State != leaguerules.IsKnown {
		job.State, job.Words = JobBlocked, "Needs a ruling: which weeks the playoffs are."
		return job
	}
	season, err := SeasonAt(records, at)
	active := Replay(records).Active
	for _, r := range records {
		if err == nil && r.Command.Command == BracketBuild && active[r.Receipt.Entry] && r.Command.Args.Bracket != nil &&
			r.Command.Args.Bracket.Season == season && !at.Before(r.Receipt.RecordedAt) {
			job.State, job.Words = JobLive, fmt.Sprintf("The %d bracket is built; seed a slot or advance a team by hand when needed.", season)
			return job
		}
	}
	job.State = JobUpcoming
	job.Words = fmt.Sprintf("Weeks %d to %d. The bracket is built from the final standings.", p.StartWeek.Value, p.FinalWeek.Value)
	return job
}
