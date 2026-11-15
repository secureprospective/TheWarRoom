package serverhttp_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// cachedRead is one league-state reading the reply cache keeps (serverhttp cachedReadingOperation).
type cachedRead struct{ path, body string }

func cachedReads(t *testing.T) []cachedRead {
	t.Helper()
	league := `{"protocol":1,"league":"native.hosted"}`
	out := []cachedRead{{serverhttp.LedgerPath, league}, {serverhttp.PeoplePath, league}}
	for _, command := range []string{"commish.rules.read", "commish.corrections.read", "native.assets.read", "native.reserve.check", "native.roster.check"} {
		raw := workflowRequest(t, command, "read", `{"league":"native.hosted"}`, "")
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, cachedRead{serverhttp.WorkflowPath, string(body)})
	}
	return out
}

// requireFresh reads every cached reading twice through the cached server (miss, then hit) and once through
// the uncached twin, for each viewer, and requires the same status and bytes every time.
func requireFresh(t *testing.T, f *hostedFixture, stage string, viewers map[string][2]string) map[string][]byte {
	t.Helper()
	seen := map[string][]byte{}
	for name, v := range viewers {
		for _, read := range cachedReads(t) {
			wantStatus, want := f.sendOn(t, f.fresh, read.path, read.body, v[0], v[1])
			for pass := range 2 {
				status, got := f.sendOn(t, f.server, read.path, read.body, v[0], v[1])
				if status != wantStatus || !bytes.Equal(got, want) {
					t.Fatalf("%s: %s %s pass %d: cached %d %.300s\nfresh %d %.300s", stage, name, read.body, pass, status, got, wantStatus, want)
				}
			}
			if read.path == serverhttp.LedgerPath {
				seen[name] = want
			}
		}
	}
	return seen
}

// TestReplyCacheMatchesFreshReadings: at every stage (a private bid, a clock jump, a season rollover) each
// viewer's cached replies are byte-identical to an uncached server's, and one owner never gets another's.
func TestReplyCacheMatchesFreshReadings(t *testing.T) {
	c := workflowClient(t)
	owner, partner := c, c
	owner.bearer, owner.proof = c.f.login(t, "owner", "private-password"), ""
	partner.bearer, partner.proof = c.f.login(t, "partner", "private-password"), ""
	viewers := map[string][2]string{"commissioner": {c.bearer, c.proof}, "owner": {owner.bearer, ""}, "partner": {partner.bearer, ""}}

	requireFresh(t, c.f, "start", viewers)
	owner.run(owner.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "6001", Franchise: "franchise.02", Salary: 500000000, Years: 1}}, "cache.private.bid"))
	ledgers := requireFresh(t, c.f, "private bid", viewers)
	if bytes.Equal(ledgers["owner"], ledgers["partner"]) {
		t.Fatal("the private bid must make the owners' ledgers differ, or isolation is untested")
	}
	c.f.skew.Store(int64(48 * time.Hour))
	requireFresh(t, c.f, "two days later", viewers)
	c.run(c.command(commissioner.SeasonRollover, commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2027}}, "cache.rollover"))
	requireFresh(t, c.f, "rollover", viewers)
}

// TestReplyCacheStillAdmits: a signed-out or refused caller never gets a reply a signed-in viewer cached.
func TestReplyCacheStillAdmits(t *testing.T) {
	c := workflowClient(t)
	for _, read := range cachedReads(t) {
		c.f.post(t, read.path, read.body, c.bearer, c.proof, 200)
		if status, _ := c.f.send(t, read.path, read.body, "", ""); status != 401 {
			t.Fatalf("%s: signed-out status %d after a cached read", read.body, status)
		}
		if status, _ := c.f.send(t, read.path, read.body, "not-a-session", ""); status != 401 && status != 503 {
			t.Fatalf("%s: unknown session status %d after a cached read", read.body, status)
		}
	}
}
