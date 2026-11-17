package serverhttp_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestHostedFranchiseHistoryMemberAudienceAndOutsider(t *testing.T) {
	c := workflowClient(t)
	other := c
	other.bearer, other.proof = c.f.login(t, "owner", "private-password"), ""
	cut := other.run(other.command(commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0102", Franchise: "franchise.02"}}, "history.cut"))
	secret := other.run(other.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "6001", Franchise: "franchise.02", Salary: 500000000, Years: 1}}, "history.sealed"))
	const scope = `{"league":"native.hosted","franchise":"franchise.02","season":2026}`
	read := func(client hostedWorkflowClient, scope string) commissioner.Ledger {
		t.Helper()
		var page commissioner.Ledger
		if err := json.Unmarshal(client.invoke("franchise.moves.history.read", "read", []byte(scope), 200).Data, &page); err != nil {
			t.Fatal(err)
		}
		return page
	}
	has := func(page commissioner.Ledger, id string) bool {
		return slices.ContainsFunc(page.Entries, func(e commissioner.Entry) bool { return e.ID == id })
	}
	if own := read(other, scope); !has(own, cut.Entry) || !has(own, secret.Entry) {
		t.Fatalf("owner missing own history: %+v", own)
	}
	member := c
	member.bearer, member.proof = c.f.login(t, "partner", "private-password"), ""
	if public := read(member, scope); !has(public, cut.Entry) || has(public, secret.Entry) {
		t.Fatalf("LEDGER audience: %+v", public)
	}
	if page := read(other, `{"league":"native.hosted","franchise":"franchise.02","season":2025}`); len(page.Entries) != 0 {
		t.Fatal("season page contains another season")
	}
	other.invoke("moves.read", "read", []byte(`{"league":"native.hosted","franchise":"franchise.01"}`), 409)
	code, err := c.f.enrollment.Issue(t.Context(), "native.hosted", "outsider")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.f.enrollment.Redeem(t.Context(), accounts.RawEnrollment{Code: code.Code, Username: "outsider", Password: "private-password"}); err != nil {
		t.Fatal(err)
	}
	outsider := c
	outsider.bearer, outsider.proof = c.f.login(t, "outsider", "private-password"), ""
	outsider.invoke("franchise.moves.history.read", "read", []byte(scope), 403)
	c.invoke("franchise.moves.history.read", "read", []byte(`{"league":"native.hosted","franchise":"franchise.02","season":0}`), 400)
}
