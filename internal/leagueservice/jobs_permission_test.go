package leagueservice

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func TestCommissionerJobsReaderLeavesDeputyScopeClosed(t *testing.T) {
	fixture := newDueFixture(t)
	if err := commissionerJobsReader(recordsFromDueFixture(t, fixture), "creator"); err != nil {
		t.Fatalf("commissioner could not read jobs: %v", err)
	}
	if err := commissionerJobsReader(recordsFromDueFixture(t, fixture), "owner.one"); err == nil {
		t.Fatal("franchise owner read commissioner jobs")
	}
	raw := fixture.raw(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "deputy.one", Role: governance.CoCommissioner,
	}})
	raw.Visibility = "private"
	if _, err := fixture.store.ExecuteCommissionerWithFacts(t.Context(), raw, "creator", nil, fixture.facts, time.UTC); err != nil {
		t.Fatal(err)
	}
	if err := commissionerJobsReader(recordsFromDueFixture(t, fixture), "deputy.one"); err == nil {
		t.Fatal("deputy read all jobs before its visibility policy is set")
	}
}

func TestTargetCommissionerJobsUsesHeldScheduleWithoutPlayerFacts(t *testing.T) {
	fixture := newDueFixture(t)
	kickoff := fixture.clock.Now().Add(24 * time.Hour)
	service := viewServiceWithFacts(t, fixture, "creator", nil, []leagueweek.Week{{Number: 5,
		Games: []leagueweek.Game{{Teams: [2]string{"KC", "BUF"}, Kickoff: kickoff}}}})
	jobs, err := service.TargetCommissionerJobs(dueLeague)
	if err != nil {
		t.Fatal(err)
	}
	job, found := jobForLeagueService(jobs.Jobs, "vacant.franchise.03")
	if !found || job.Identity.Week != 5 || job.Closes != kickoff.UTC().Format(time.RFC3339) {
		t.Fatalf("job builder did not use the server-held schedule: %+v", job)
	}
}

func jobForLeagueService(jobs []commissioner.Job, id string) (commissioner.Job, bool) {
	for _, job := range jobs {
		if job.ID == id {
			return job, true
		}
	}
	return commissioner.Job{}, false
}

func recordsFromDueFixture(t *testing.T, fixture *dueFixture) []commissioner.Record {
	t.Helper()
	events := fixture.events(t)
	records := make([]commissioner.Record, len(events))
	for i, event := range events {
		if event.Payload.Commissioner == nil {
			t.Fatalf("event %d is not a commissioner record", i)
		}
		records[i] = *event.Payload.Commissioner
	}
	return records
}
