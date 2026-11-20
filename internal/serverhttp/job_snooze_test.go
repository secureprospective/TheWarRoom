package serverhttp_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func readJobSnoozes(t *testing.T, f *hostedFixture, bearer string) accounts.JobSnoozesState {
	t.Helper()
	body := f.post(t, serverhttp.AccountPreferencePath,
		layoutRequest(t, "job.snoozes.read", "read", `{}`), bearer, "", 200)
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	var state accounts.JobSnoozesState
	if err := json.Unmarshal(response.Data, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestHostedJobSnoozesArePrivateAndWakeIsAccountScoped(t *testing.T) {
	f := newHostedFixture(t, true)
	commissioner, _ := f.directCommissioner(t)
	owner := f.login(t, "owner", "private-password")
	until := time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano)
	set := layoutRequest(t, "job.snooze.set", "execute",
		`{"league":"league.a","jobId":"trade.review","until":"`+until+`"}`)
	f.post(t, serverhttp.AccountPreferencePath, set, commissioner, "", 200)

	mine := readJobSnoozes(t, f, commissioner)
	if len(mine.Snoozes) != 1 || mine.Snoozes[0].League != "league.a" || mine.Snoozes[0].JobID != "trade.review" {
		t.Fatalf("commissioner snoozes = %+v", mine)
	}
	if other := readJobSnoozes(t, f, owner); len(other.Snoozes) != 0 {
		t.Fatalf("another principal saw private snoozes: %+v", other)
	}

	wake := layoutRequest(t, "job.snooze.wake", "execute",
		`{"league":"league.a","jobId":"trade.review"}`)
	f.post(t, serverhttp.AccountPreferencePath, wake, commissioner, "", 200)
	if got := readJobSnoozes(t, f, commissioner); len(got.Snoozes) != 0 {
		t.Fatalf("wake left snooze active: %+v", got)
	}
	if got := readJobSnoozes(t, f, owner); len(got.Snoozes) != 0 {
		t.Fatalf("wake changed another principal: %+v", got)
	}

	spoof := layoutRequest(t, "job.snooze.set", "execute",
		`{"league":"league.a","jobId":"trade.review","until":"`+until+`","accountId":"attacker"}`)
	f.post(t, serverhttp.AccountPreferencePath, spoof, commissioner, "", 400)
	agent := f.agentToken(t, commissioner, []string{"commish.jobs.read"}, "read")
	f.post(t, serverhttp.AccountPreferencePath,
		layoutRequest(t, "job.snoozes.read", "read", `{}`), agent.Bearer, "", 403)
}
