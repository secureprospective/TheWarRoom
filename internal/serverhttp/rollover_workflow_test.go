package serverhttp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// refreshReads is the desktop's whole-league refresh for one franchise; one failed reading fails it.
// Lineup is left out: the desktop already shows a lineup without evidence as unavailable.
func (c hostedWorkflowClient) refreshReads(franchise string) map[string]serverhttp.WorkflowResponse {
	c.t.Helper()
	readings := map[string]serverhttp.WorkflowResponse{"native.assets.read": c.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200)}
	var assets commissioner.NativeAssets
	if err := json.Unmarshal(readings["native.assets.read"].Data, &assets); err != nil {
		c.t.Fatal(err)
	}
	season := assets.Snapshot.Season
	for _, r := range []struct{ command, args string }{
		{"native.reserve.check", `{"league":"native.hosted"}`},
		{"draft.read", fmt.Sprintf(`{"league":"native.hosted","franchise":%q,"season":0,"at":"0001-01-01T00:00:00Z"}`, franchise)},
		{"commish.rules.read", `{"league":"native.hosted"}`},
		{"bids.read", fmt.Sprintf(`{"league":"native.hosted","franchise":%q}`, franchise)},
		{"waivers.read", fmt.Sprintf(`{"league":"native.hosted","franchise":%q,"at":%q}`, franchise, time.Now().UTC().Format(time.RFC3339Nano))},
		{"trades.read", fmt.Sprintf(`{"league":"native.hosted","viewer":%q}`, franchise)},
		{"results.read", fmt.Sprintf(`{"league":"native.hosted","season":%d,"week":1}`, season)},
		{"standings.read", fmt.Sprintf(`{"league":"native.hosted","season":%d}`, season)},
	} {
		readings[r.command] = c.invoke(r.command, "read", []byte(r.args), 200)
	}
	return readings
}

func TestHostedRolloverAfterWorkerBidAwardKeepsEveryReadingOpen(t *testing.T) {
	c := workflowClient(t)
	owner := c
	owner.bearer, owner.proof = c.f.login(t, "owner", "private-password"), ""
	owner.run(owner.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "6001", Franchise: "franchise.02", Salary: 500000000, Years: 1}}, "award.bid"))

	ctx, stop := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); c.f.worker.Run(ctx, func(error) {}) }()
	t.Cleanup(func() { stop(); <-done })
	c.f.skew.Store(int64(3 * time.Minute))
	c.f.worker.Wake()
	for deadline := time.Now().Add(10 * time.Second); ; {
		assets, err := c.f.history.NativeAssets(t.Context(), "native.hosted")
		if err != nil {
			t.Fatal(err)
		}
		if slices.ContainsFunc(assets.Snapshot.Players, func(p commissioner.NativePlayer) bool { return p.ID == "6001" && p.Franchise == "franchise.02" }) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the due worker never awarded the bid")
		}
		time.Sleep(20 * time.Millisecond)
	}
	stop()
	<-done

	rollover := c.command(commissioner.SeasonRollover, commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2027}}, "rollover.after.award")
	c.run(rollover)
	body, err := json.Marshal(rollover)
	if err != nil {
		t.Fatal(err)
	}
	receipt := c.invoke(rollover.Command, "execute", body, 200)
	requirePlainRollover(t, c.previews[commissioner.SeasonRollover])
	// The practice draft starts two seconds after provisioning; read it running, however fast the test is.
	time.Sleep(2 * time.Second)
	readings := c.refreshReads("franchise.01")
	fresh := c
	fresh.bearer, fresh.proof = c.f.login(t, "owner", "private-password"), ""
	fresh.refreshReads("franchise.02")
	var bids commissioner.NativeBids
	if err := json.Unmarshal(readings["bids.read"].Data, &bids); err != nil {
		t.Fatal(err)
	}
	if want := "Bids for the 2027 season open once the server operator selects football players and schedule for that season."; !slices.Contains(bids.Notes, want) {
		t.Fatalf("new-season bids do not say why bidding waits: %q", bids.Notes)
	}
	out, err := json.MarshalIndent(map[string]any{"receipt": receipt, "readings": readings}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkHostedWire(t, "../../frontend/src/app/commissioner/rollover.wire.testdata.json", out)
}

// requirePlainRollover pins the owner-facing rollover wording: the awarded free agent by name, the free-agent
// route in plain words, dates in the league's time zone, and no build-status words.
func requirePlainRollover(t *testing.T, fields []commissioner.PreviewField) {
	t.Helper()
	after := make([]string, 0, len(fields))
	for _, f := range fields {
		after = append(after, f.After)
	}
	for _, want := range []string{
		"Practice free agent: contract years remaining change from 1 to 0.",
		"Practice free agent is pending a contract decision, not released. The app cannot say yet whether he becomes a restricted or unrestricted free agent: no rookie draft round is recorded for his contract.",
		"The rookie draft window opens Fri, Jan 1, 2027 at 12:00 AM CST.",
		"Salaries stay as recorded.",
	} {
		if !slices.Contains(after, want) {
			t.Errorf("rollover preview lacks %q; got %q", want, after)
		}
	}
	deadline := regexp.MustCompile(`^Trade deadline takes effect: (Mon|Tue|Wed|Thu|Fri|Sat|Sun), [A-Z][a-z]{2} \d{1,2}, 2027 at \d{1,2}:\d{2} [AP]M C[SD]T\.$`)
	if !slices.ContainsFunc(after, deadline.MatchString) {
		t.Errorf("rollover preview lacks a 2027 trade deadline in league time; got %q", after)
	}
	plain := regexp.MustCompile(`no name recorded|RFA|UFA|not built yet| UTC\b`)
	for _, words := range after {
		if plain.MatchString(words) {
			t.Errorf("rollover preview is not plain English: %q", words)
		}
	}
}
