package serverhttp_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedLeagueActivityMemberAndCommissionerReads(t *testing.T) {
	commissionerClient := workflowClient(t)
	noticeText := "Public activity notice"
	commissionerClient.run(commissionerClient.command(commissioner.NoticePublish,
		commissioner.RawArgs{Note: &commissioner.RawNote{Text: noticeText}}, "activity.notice"))
	arguments := []byte(`{"league":"native.hosted","filter":{"kind":"notice","limit":100}}`)
	read := func(client hostedWorkflowClient) commissioner.LeagueActivity {
		t.Helper()
		response := client.invoke("league.activity.read", "read", arguments, http.StatusOK)
		var activity commissioner.LeagueActivity
		if err := json.Unmarshal(response.Data, &activity); err != nil {
			t.Fatal(err)
		}
		return activity
	}
	for label, client := range map[string]hostedWorkflowClient{
		"commissioner": commissionerClient,
		"member":       {t: t, f: commissionerClient.f, bearer: commissionerClient.f.login(t, "owner", "private-password"), football: commissionerClient.football},
	} {
		activity := read(client)
		if activity.League != "native.hosted" || len(activity.Events) != 1 || activity.Events[0].Kind != commissioner.ActivityNotice ||
			activity.Events[0].Summary != noticeText {
			t.Errorf("%s activity = %+v", label, activity)
		}
		if strings.Contains(activity.Events[0].Summary, "Disposable workflow test ruling") {
			t.Errorf("%s saw the commissioner reason: %+v", label, activity.Events[0])
		}
	}

	for _, arguments := range []string{
		`{"league":"native.hosted","filter":{"kind":"bids"}}`,
		`{"league":"native.hosted","filter":{"limit":101}}`,
		`{"league":"native.hosted","filter":{"unexpected":true}}`,
	} {
		commissionerClient.invoke("league.activity.read", "read", []byte(arguments), http.StatusBadRequest)
	}

	token := commissionerClient.f.agentToken(t, commissionerClient.bearer, []string{"commish.jobs.read"}, "read")
	agentArguments := []byte(`{"league":"native.hosted"}`)
	workflow := workflowRequest(t, "league.activity.read", "read", string(agentArguments), commissionerClient.football)
	request := serverhttp.RawNativeRequest{Protocol: 1, Workflow: &workflow}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	commissionerClient.f.post(t, serverhttp.WorkflowPath, string(body), token.Bearer, "", http.StatusConflict)
}
