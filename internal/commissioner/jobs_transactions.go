package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// transactionJobs reports the native queues, not an imported league's pending moves.
func transactionJobs(assets NativeAssets, state waiverState, rules leaguerules.RuleSet, at time.Time) []Job {
	waivers := Job{ID: "waivers", Name: "Waiver processing", State: JobLive, Controls: []string{"CM-035"}}
	bids := Job{ID: "bids", Name: "Bid processing", State: JobLive, Controls: []string{"CM-036"}}
	if !assets.Seeded {
		waivers.State, bids.State = JobBlocked, JobBlocked
		waivers.Words, bids.Words = "Record starting assets first.", "Record starting assets first."
		return []Job{waivers, bids}
	}
	fillWaiverJob(&waivers, state.Windows, rules.Waivers, at)
	fillBidJob(&bids, state.Bids, rules.Bids, at)
	return []Job{waivers, bids}
}

func fillWaiverJob(job *Job, windows []waiverWindow, rules leaguerules.WaiversRules, at time.Time) {
	if rules.Duration.State != leaguerules.IsKnown {
		job.State, job.Words = JobBlocked, "Needs a ruling: how long waivers last."
		return
	}
	pending := 0
	for _, window := range windows {
		if !window.Processed {
			pending++
			jobClose(job, window.CutAt.Add(rules.Duration.Value), at)
		}
	}
	job.Words = fmt.Sprintf("Waiver windows awaiting processing: %d. Closed windows are processed automatically.", pending)
}

func fillBidJob(job *Job, book bidBook, rules leaguerules.BidsRules, at time.Time) {
	if rules.StandFor.State != leaguerules.IsKnown || rules.StandFor.Value <= 0 {
		job.State, job.Words = JobBlocked, "Needs a ruling: how long a bid must stand to win."
		return
	}
	if rules.UFA.State != leaguerules.IsKnown {
		job.State, job.Words = JobBlocked, "Needs a ruling: whether free-agent bids are blind or open."
		return
	}
	// This reading is public: not even a count or a deadline may expose a blind auction.
	if rules.UFA.Value == leaguerules.Blind {
		job.Words = "Blind bids close automatically; the commissioner settles ties."
		return
	}
	pending := 0
	for _, auction := range book.Auctions {
		if !auction.Decided && len(auction.liveBids()) > 0 {
			pending++
			jobClose(job, auction.standingAt().Add(rules.StandFor.Value), at)
		}
	}
	job.Words = fmt.Sprintf("Players with bids: %d. Untied bids close automatically; the commissioner settles ties.", pending)
}

func jobClose(job *Job, closeAt, now time.Time) {
	if !closeAt.After(now) {
		return
	}
	closing := closeAt.UTC().Format(time.RFC3339)
	if job.Closes == "" || closing < job.Closes {
		job.Closes = closing
	}
}
