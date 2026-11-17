package serverhttp_test

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedPlayerReadingsRefuseOutsidersAndMissingInputs(t *testing.T) {
	f := newHostedFixture(t, true, true)
	owner := f.login(t, "owner", "private-password")
	code, err := f.enrollment.Issue(t.Context(), "native.hosted", "outsider")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.enrollment.Redeem(t.Context(), accounts.RawEnrollment{Code: code.Code, Username: "outsider", Password: "private-password"}); err != nil {
		t.Fatal(err)
	}
	outsider := f.login(t, "outsider", "private-password")
	for _, command := range []string{"player.usage.read", "player.agearc.read"} {
		t.Run(command, func(t *testing.T) {
			for _, tc := range []struct {
				name, bearer, args string
				status             int
			}{
				{"member missing inputs", owner, `{"league":"native.hosted","player":"0099"}`, 409},
				{"unknown player", owner, `{"league":"native.hosted","player":"99999"}`, 409},
				{"invalid player", owner, `{"league":"native.hosted","player":"99"}`, 400},
				{"outsider", outsider, `{"league":"native.hosted","player":"0099"}`, 403},
				{"anonymous", "", `{"league":"native.hosted","player":"0099"}`, 401},
			} {
				t.Run(tc.name, func(t *testing.T) {
					raw := workflowRequest(t, command, "read", tc.args, "")
					body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
					if err != nil {
						t.Fatal(err)
					}
					status, result := f.send(t, serverhttp.WorkflowPath, string(body), tc.bearer, "")
					if status != tc.status {
						t.Fatalf("status %d, want %d: %s", status, tc.status, result)
					}
				})
			}
		})
	}
}
