package serverhttp_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedRoleNoOpReturnsPreviewReason(t *testing.T) {
	c := workflowClient(t)
	raw := c.command("commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: c.f.principal, Role: governance.Commissioner,
	}}, "role.noop")
	raw.Visibility = "private"
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Command: &raw})
	if err != nil {
		t.Fatal(err)
	}
	result := c.f.post(t, serverhttp.PreviewPath, string(body), c.bearer, c.proof, http.StatusOK)
	var preview commissioner.Preview
	if err := json.Unmarshal(result, &preview); err != nil {
		t.Fatal(err)
	}
	want := "This person already holds that role. Nothing would change"
	if len(preview.Blockers) != 1 || preview.Blockers[0] != want {
		t.Fatalf("preview blockers=%q, want %q", preview.Blockers, want)
	}
	result = c.f.post(t, serverhttp.ExecutePath, string(body), c.bearer, c.proof, http.StatusConflict)
	if got := strings.TrimSpace(string(result)); got != want {
		t.Fatalf("execute refusal=%q, want %q", got, want)
	}
}

func TestHostedRoleGrantToAccountWithoutSeatOrRole(t *testing.T) {
	c := workflowClient(t)
	code, err := c.f.enrollment.Issue(t.Context(), "native.hosted", "unseated")
	if err != nil {
		t.Fatal(err)
	}
	account, err := c.f.enrollment.Redeem(t.Context(), accounts.RawEnrollment{
		Code: code.Code, Username: "unseated", Password: "private-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	person, err := account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	records, err := c.f.history.SharedCommissionerRecords(t.Context(), "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		t.Fatal(err)
	}
	for _, franchise := range []governance.FranchiseID{"franchise.01", "franchise.02", "franchise.03"} {
		if state.HoldsSeat(franchise, person) {
			t.Fatal("new account already holds a seat")
		}
	}
	for _, role := range []governance.Role{governance.Commissioner, governance.CoCommissioner, governance.DOT} {
		if state.HasRole(person, role) {
			t.Fatal("new account already holds a role")
		}
	}
	raw := c.command("commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: string(person), Role: governance.CoCommissioner,
	}}, "unseated.role")
	raw.Visibility = "private"
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Command: &raw})
	if err != nil {
		t.Fatal(err)
	}
	c.f.post(t, serverhttp.ExecutePath, string(body), c.bearer, c.proof, 200)
	records, err = c.f.history.SharedCommissionerRecords(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	state, err = commissioner.GovernanceState(records)
	if err != nil || !state.HasRole(person, governance.CoCommissioner) {
		t.Fatalf("unseated account grant missing: %v", err)
	}
}

func TestHostedRoleGrantRequiresKnownAccountAndOldGrantReplays(t *testing.T) {
	c := workflowClient(t)
	owner, err := c.f.accounts.Credential(t.Context(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	person, err := owner.Principal()
	if err != nil {
		t.Fatal(err)
	}
	raw := c.command("commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{Principal: strings.TrimPrefix(string(person), "acct."), Role: governance.CoCommissioner}}, "bad.role")
	raw.Visibility = "private"
	request := func(raw commissioner.RawCommand) []byte {
		t.Helper()
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Command: &raw})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	if err := raw.ValidatePreview(); err != nil {
		t.Fatalf("invalid test command: %v", err)
	}
	before, err := c.f.history.SharedCommissionerRecords(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	result := c.f.post(t, serverhttp.ExecutePath, string(request(raw)), c.bearer, c.proof, 409)
	if !strings.Contains(string(result), "No account on this server matches that person") {
		t.Fatalf("wrong refusal: %s", result)
	}
	after, err := c.f.history.SharedCommissionerRecords(t.Context(), raw.League)
	if err != nil || len(after) != len(before) {
		t.Fatalf("rejected grant changed ledger: %d -> %d, %v", len(before), len(after), err)
	}
	// A well-formed account ID absent from the sign-in store is refused too.
	raw.Args.People.Principal = "acct.11111111-1111-4111-8111-111111111111"
	result = c.f.post(t, serverhttp.ExecutePath, string(request(raw)), c.bearer, c.proof, 409)
	if !strings.Contains(string(result), "No account on this server matches that person") {
		t.Fatalf("wrong refusal: %s", result)
	}
	raw.Args.People.Principal = string(person)
	raw.Request = "good.role"
	c.f.post(t, serverhttp.ExecutePath, string(request(raw)), c.bearer, c.proof, 200)
	records, err := c.f.history.SharedCommissionerRecords(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil || !state.HasRole(person, governance.CoCommissioner) {
		t.Fatalf("owner grant missing: %v", err)
	}
	// Bypass hosted admission to represent a ledger written by an older server.
	raw.Args.People.Principal = "acct.11111111-1111-4111-8111-111111111111"
	raw.Expected = int64(len(records))
	raw.Request = "old.role"
	if _, err := c.f.history.ExecuteCommissioner(t.Context(), raw, c.f.principal, nil); err != nil {
		t.Fatal(err)
	}
	records, err = c.f.history.HeldCommissionerRecords(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	state, err = commissioner.GovernanceState(records)
	if err != nil || !state.HasRole("acct.11111111-1111-4111-8111-111111111111", governance.CoCommissioner) {
		t.Fatalf("old grant did not replay: %v", err)
	}
	// An old phantom role does not make its nonexistent account valid for another grant.
	raw.Args.People.Role = governance.DOT
	raw.Expected = int64(len(records))
	raw.Request = "phantom.role"
	result = c.f.post(t, serverhttp.ExecutePath, string(request(raw)), c.bearer, c.proof, 409)
	if !strings.Contains(string(result), "No account on this server matches that person") {
		t.Fatalf("phantom account accepted: %s", result)
	}
}
