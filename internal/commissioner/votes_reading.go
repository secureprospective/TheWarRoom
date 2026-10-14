package commissioner

import (
	"fmt"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

type PeopleVoteCount struct {
	Choice  string `json:"choice"`
	Ballots int    `json:"ballots"`
}

type PeopleVote struct {
	ID              string            `json:"id"`
	Status          string            `json:"status"`
	BallotsCast     int               `json:"ballotsCast"`
	TotalFranchises int               `json:"totalFranchises"`
	VotesRequired   int               `json:"votesRequired"`
	Requirement     string            `json:"requirement"`
	EndsAt          time.Time         `json:"endsAt"`
	Candidates      []string          `json:"candidates"`
	Counts          []PeopleVoteCount `json:"counts"`
	VotedFranchises []string          `json:"votedFranchises,omitempty"`
}

func readVotes(state governance.State, records []Record, total int, private bool) []PeopleVote {
	ids := make(map[string]bool)
	var at time.Time
	for _, r := range records {
		at = r.Receipt.RecordedAt
		if p := r.Command.Args.People; p != nil && p.ID != "" {
			ids[p.ID] = true
		}
	}
	sorted := make([]string, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	sort.Strings(sorted)
	out := []PeopleVote{}
	for _, id := range sorted {
		v, ok := state.Vote(id)
		if !ok || v.Closed {
			continue
		}
		out = append(out, readVote(state, v, total, private, at))
	}
	return out
}

func readVote(state governance.State, v governance.Vote, total int, private bool, at time.Time) PeopleVote {
	out := PeopleVote{ID: v.ID, BallotsCast: len(v.Ballots), TotalFranchises: total,
		VotesRequired: governance.VoteOutThreshold(total), Requirement: "Yes ballots needed to remove the commissioner",
		Status: "Commissioner removal voting is open", EndsAt: v.EndsAt,
		Candidates: []string{}, Counts: []PeopleVoteCount{}}
	if v.Election {
		out.Status = state.ElectionStatus(v.ID, at)
		out.VotesRequired = governance.ElectionThreshold(total)
		out.Requirement = "Ballots for one candidate needed to win outright"
		if v.Runoff {
			out.VotesRequired = 0
			out.Requirement = "Most ballots wins; tied leaders need a tie ruling"
		}
	}
	counts := make(map[string]int)
	for _, p := range v.Candidates {
		out.Candidates = append(out.Candidates, string(p))
		counts[string(p)] = 0
	}
	for f, p := range v.Ballots {
		counts[string(p)]++
		if private {
			out.VotedFranchises = append(out.VotedFranchises, string(f))
		}
	}
	for choice, count := range counts {
		out.Counts = append(out.Counts, PeopleVoteCount{choice, count})
	}
	sort.Slice(out.Counts, func(i, j int) bool { return out.Counts[i].Choice < out.Counts[j].Choice })
	sort.Strings(out.Candidates)
	sort.Strings(out.VotedFranchises)
	if !v.Election {
		out.Status = fmt.Sprintf("Commissioner removal voting is open: %d yes ballots; %d needed", counts["yes"],
			out.VotesRequired)
	}
	return out
}

func successionStatus(state governance.State, records []Record) string {
	if state.ElectionNeeded() {
		return "A commissioner election is needed"
	}
	active := Replay(records).Active
	pending := false
	for _, r := range records {
		if !active[r.Receipt.Entry] {
			continue
		}
		p := r.Command.Args.People
		if p == nil {
			continue
		}
		switch r.Command.Command {
		case "commish.succession.stepdown":
			pending = p.Successor == ""
		case "commish.vote.close":
			v, ok := state.Vote(p.ID)
			if ok && v.Closed && v.Passed {
				pending = !v.Election
			}
		case "commish.succession.accept", "commish.succession.decline", "commish.succession.takeover":
			pending = false
		}
	}
	if pending {
		return "The co-commissioner has taken over; acceptance is pending"
	}
	return "The commissioner is serving; no succession is pending"
}
