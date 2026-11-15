package serverhttp_test

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedWorkflowAdmissionParity(t *testing.T) {
	publisher := serverhttp.NewPublisher()
	f := newVersionedHostedFixture(t, serverhttp.NewWorkflowVersions(serverhttp.PublisherVersionSource{Publisher: publisher}), true, true)
	commissionerBearer, proof := f.directCommissioner(t)
	ownerBearer := f.login(t, "owner", "private-password")
	reads := []struct{ name, command, arguments string }{
		{"directory", "native.directory.read", `{"league":"native.hosted"}`},
		{"week", "week.read", `{"league":"native.hosted","franchise":"franchise.02","season":2026,"localZone":"UTC","at":""}`},
		{"schedule", "schedule.read", `{"league":"native.hosted"}`},
		{"results", "results.read", `{"league":"native.hosted","season":2026,"week":1}`},
		{"standings", "standings.read", `{"league":"native.hosted","season":2026}`},
		{"odds", "odds.read", `{"league":"native.hosted","season":2026}`},
		{"bracket", "bracket.read", `{"league":"native.hosted","season":2026}`},
		{"projections", "projections.read", `{"league":"native.hosted","season":2026,"week":1,"players":"0099,0100"}`},
		{"own-lineup", "lineup.read", `{"league":"native.hosted","franchise":"franchise.02","week":1,"at":""}`},
		{"other-lineup", "lineup.read", `{"league":"native.hosted","franchise":"franchise.01","week":1,"at":""}`},
	}
	for _, role := range []struct{ name, bearer, proof string }{{"owner", ownerBearer, ""}, {"commissioner", commissionerBearer, proof}} {
		for _, read := range reads {
			t.Run(role.name+"/"+read.name, func(t *testing.T) {
				status := 200
				if role.name == "owner" && read.name == "other-lineup" {
					status = 409
				}
				raw := workflowRequest(t, read.command, "read", read.arguments, "")
				reply := taggedWorkflow(t, f, raw, role.bearer, role.proof, status)
				if status == 200 && (reply.Tags == nil || len(reply.Data) == 0) {
					t.Fatal("reading lost tags or data")
				}
			})
		}
	}
	_, counts := f.tally.Snapshot()
	if got := counts["lineup.read read"]; got.Calls != 4 || got.Refused != 1 {
		t.Fatalf("refusals not counted exactly once: %+v", got)
	}
}
