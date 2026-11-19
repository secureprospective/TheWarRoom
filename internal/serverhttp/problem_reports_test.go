package serverhttp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedProblemReportsMemberSubmissionAndCommissionerList(t *testing.T) {
	commissionerClient := workflowClient(t)
	owner := hostedWorkflowClient{t: t, f: commissionerClient.f,
		bearer:   commissionerClient.f.login(t, "owner", "private-password"),
		football: commissionerClient.football}
	submit := func(text string) commissioner.ProblemReportReceipt {
		t.Helper()
		arguments := fmt.Sprintf(`{"league":"native.hosted","text":%q,"route":"control/league","appRelease":42,"viewportWidth":1440}`, text)
		response := owner.invoke(commissioner.ProblemReportCreate, "execute", []byte(arguments), http.StatusOK)
		var receipt commissioner.ProblemReportReceipt
		if err := json.Unmarshal(response.Data, &receipt); err != nil {
			t.Fatal(err)
		}
		if receipt.Message != "Sent — thanks" || strings.Contains(string(response.Data), text) {
			t.Fatalf("member receipt exposes report or lacks acknowledgement: %s", response.Data)
		}
		return receipt
	}
	submit("first member report")
	submit("second member report")

	response := commissionerClient.invoke(commissioner.ProblemReportsRead, "read",
		[]byte(`{"league":"native.hosted"}`), http.StatusOK)
	var reading commissioner.ProblemReportsReading
	if err := json.Unmarshal(response.Data, &reading); err != nil {
		t.Fatal(err)
	}
	if reading.League != "native.hosted" || len(reading.Reports) != 2 ||
		reading.Reports[0].Text != "second member report" || reading.Reports[1].Text != "first member report" {
		t.Fatalf("commissioner report reading is not newest first: %+v", reading)
	}
	newest := reading.Reports[0]
	if newest.Principal == "" || !strings.HasPrefix(newest.ID, "feedback.") || newest.Route != "control/league" ||
		newest.AppRelease != 42 || newest.ViewportWidth != 1440 || newest.UserAgent == "" || newest.CreatedAt.IsZero() {
		t.Fatalf("server report metadata missing: %+v", newest)
	}
	owner.invoke(commissioner.ProblemReportsRead, "read", []byte(`{"league":"native.hosted"}`),
		http.StatusConflict)
}

func TestHostedProblemReportsRejectEleventhMemberReportInOneHour(t *testing.T) {
	client := workflowClient(t)
	member := hostedWorkflowClient{t: t, f: client.f,
		bearer: client.f.login(t, "owner", "private-password"), football: client.football}
	for index := 1; index <= 10; index++ {
		arguments := fmt.Sprintf(`{"league":"native.hosted","text":"member report %d","route":"home/overview","appRelease":42,"viewportWidth":1440}`, index)
		member.invoke(commissioner.ProblemReportCreate, "execute", []byte(arguments), http.StatusOK)
	}
	arguments := []byte(`{"league":"native.hosted","text":"eleventh member report","route":"home/overview","appRelease":42,"viewportWidth":1440}`)
	raw := workflowRequest(t, commissioner.ProblemReportCreate, "execute", string(arguments), member.football)
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	status, response := member.f.send(t, serverhttp.WorkflowPath, string(body), member.bearer, "")
	if status != http.StatusTooManyRequests || !strings.Contains(string(response), "10 reports in the past hour") {
		t.Fatalf("eleventh report status=%d body=%q", status, response)
	}
}
