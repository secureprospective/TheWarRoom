package commissioner

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// autoReturnTeams identifies franchises with a standing automatic return in the ledger.
func autoReturnTeams(records []Record) map[string]bool {
	active := Replay(records).Active
	teams := map[string]bool{}
	for _, r := range records {
		if active[r.Receipt.Entry] && r.Principal == ReserveClockPrincipal && r.Command.Command == RosterReserve && r.Command.Args.Waiver != nil {
			teams[r.Command.Args.Waiver.Franchise] = true
		}
	}
	return teams
}

func countedRoster(seed *RawNativeSnapshot, rules leaguerules.RosterRules, franchise string) int {
	n := 0
	for _, p := range seed.Players {
		if p.Franchise == franchise && (p.Status == domain.RosterActive || !rules.ReserveExcluded.Value) {
			n++
		}
	}
	return n
}

func rosterOverfull(seed *RawNativeSnapshot, rules leaguerules.RosterRules, franchise string) bool {
	return rules.ReserveExcluded.State == leaguerules.IsKnown && rules.Limits.State == leaguerules.IsKnown &&
		rules.Limits.Value.Max.State == leaguerules.IsKnown && countedRoster(seed, rules, franchise) > rules.Limits.Value.Max.Value
}

// mustCutJobs remain visible in the commissioner's jobs/activity view until humans fix the roster.
// There is deliberately no deadline, acknowledgement or automatic-cut operation.
func mustCutJobs(assets NativeAssets, records []Record, rules leaguerules.RuleSet) []Job {
	jobs := []Job{}
	if !assets.Seeded {
		return jobs
	}
	teams := autoReturnTeams(records)
	for _, f := range assets.Snapshot.Franchises {
		if !teams[f.ID] || !rosterOverfull(assets.Snapshot, rules.Roster, f.ID) {
			continue
		}
		jobs = append(jobs, Job{ID: "must_cut." + f.ID, Name: "Team activity: must cut — " + f.Name,
			State: JobLive, Controls: []string{"CM-003", "CM-010", "CM-011"},
			Words: fmt.Sprintf("%s has %d counted players; maximum %d after an automatic reserve return. Unresolved team activity: owner cuts or commissioner acts. No deadline; no automatic cut.",
				f.Name, countedRoster(assets.Snapshot, rules.Roster, f.ID), rules.Roster.Limits.Value.Max.Value)})
	}
	return jobs
}

// allowDebtReduction permits successive cuts when more than one player must leave. All other
// roster and cap checks remain in force; acquiring or returning a player manually gets no exemption.
func AllowDebtReduction(before, after *RawNativeSnapshot, records []Record, rules leaguerules.RuleSet, franchise string) leaguerules.RuleSet {
	if autoReturnTeams(records)[franchise] && rosterOverfull(before, rules.Roster, franchise) &&
		countedRoster(after, rules.Roster, franchise) < countedRoster(before, rules.Roster, franchise) && rosterOverfull(after, rules.Roster, franchise) {
		rules.Roster.Limits.Value.Max.Value = countedRoster(after, rules.Roster, franchise)
	}
	return rules
}
