package serverhttp_test

import (
	"encoding/json"
	"reflect"
	"regexp"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type hostedWorkflowClient struct {
	t             *testing.T
	f             *hostedFixture
	bearer, proof string
	football      string
	previews      map[string][]commissioner.PreviewField
}

func workflowClient(t *testing.T) hostedWorkflowClient {
	t.Helper()
	f := newHostedFixture(t, true)
	bearer, proof := f.directCommissioner(t)
	raw := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		t.Fatal(err)
	}
	var response serverhttp.WorkflowResponse
	if err := json.Unmarshal(f.post(t, serverhttp.WorkflowPath, string(body), bearer, proof, 200), &response); err != nil {
		t.Fatal(err)
	}
	return hostedWorkflowClient{t, f, bearer, proof, response.Football, map[string][]commissioner.PreviewField{}}
}
func (c hostedWorkflowClient) invoke(command, mode string, args []byte, status int) serverhttp.WorkflowResponse {
	c.t.Helper()
	raw := workflowRequest(c.t, command, mode, string(args), c.football)
	body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
	if err != nil {
		c.t.Fatal(err)
	}
	result := c.f.post(c.t, serverhttp.WorkflowPath, string(body), c.bearer, c.proof, status)
	var response serverhttp.WorkflowResponse
	if status == 200 {
		if err := json.Unmarshal(result, &response); err != nil {
			c.t.Fatal(err)
		}
	}
	return response
}
func (c hostedWorkflowClient) command(command string, args commissioner.RawArgs, key string) commissioner.RawCommand {
	c.t.Helper()
	var assets commissioner.NativeAssets
	if err := json.Unmarshal(c.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200).Data, &assets); err != nil {
		c.t.Fatal(err)
	}
	visibility := "league"
	if commissioner.OwnerBidAction(command) != "" || command == commissioner.WaiverClaim {
		visibility = "private"
	}
	return commissioner.RawCommand{League: "native.hosted", Command: command, Args: args, Expected: assets.Revision, Request: key, Reason: "Disposable workflow test ruling", Effective: "now", Visibility: visibility}
}
func (c hostedWorkflowClient) run(raw commissioner.RawCommand) commissioner.Receipt {
	c.t.Helper()
	body, err := json.Marshal(raw)
	if err != nil {
		c.t.Fatal(err)
	}
	var preview commissioner.Preview
	if err := json.Unmarshal(c.invoke(raw.Command, "preview", body, 200).Data, &preview); err != nil {
		c.t.Fatal(err)
	}
	c.previews[raw.Command] = append(c.previews[raw.Command], preview.Fields...)
	if len(preview.Blockers) > 0 {
		c.t.Fatalf("%s blocked: %v", raw.Command, preview.Blockers)
	}
	one := c.invoke(raw.Command, "execute", body, 200)
	again := c.invoke(raw.Command, "execute", body, 200)
	if string(one.Data) != string(again.Data) {
		c.t.Fatal("retry appended a second action")
	}
	var receipt commissioner.Receipt
	if err := json.Unmarshal(one.Data, &receipt); err != nil {
		c.t.Fatal(err)
	}
	return receipt
}

func TestHostedBidsWaiversTradeCorrectionAndRollover(t *testing.T) {
	c := workflowClient(t)
	bid := c.run(c.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "6001", Franchise: "franchise.01", Salary: 500000000, Years: 1}}, "bid.once"))
	bid = c.run(c.command(commissioner.BidRaise, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Bid: bid.Entry, Franchise: "franchise.01", Salary: 600000000, Years: 1}}, "raise.once"))
	c.run(c.command(commissioner.BidWithdraw, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Bid: bid.Entry, Franchise: "franchise.01"}}, "withdraw.once"))
	other := c
	other.bearer, other.proof = c.f.login(t, "owner", "private-password"), ""
	cut := other.run(other.command(commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0102", Franchise: "franchise.02"}}, "cut.once"))
	c.run(c.command(commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: cut.Entry, Franchise: "franchise.01"}}, "claim.once"))
	trade := c.run(c.command(commissioner.TradeOffer, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{Franchise: "franchise.01", Parties: []string{"franchise.01", "franchise.02"}, Assets: []commissioner.TradeAsset{{From: "franchise.01", To: "franchise.02", Player: "0101"}}}}, "offer.once"))
	c.run(c.command(commissioner.TradeWithdraw, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: trade.Entry, Franchise: "franchise.01"}}, "trade.withdraw.once"))
	partner := c
	partner.bearer, partner.proof = c.f.login(t, "partner", "private-password"), ""
	offer := other.run(other.command(commissioner.TradeOffer, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{Franchise: "franchise.02", Parties: []string{"franchise.02", "franchise.03"}, Assets: []commissioner.TradeAsset{{From: "franchise.02", To: "franchise.03", Season: 2027, Round: 1, Original: "franchise.02"}}}}, "pick.offer.once"))
	partner.run(partner.command(commissioner.TradeAccept, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer.Entry, Franchise: "franchise.03"}}, "accept.once"))
	c.run(c.command(commissioner.TradeReview, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer.Entry, Decision: "approve"}}, "review.once"))
	delta := leaguerules.Decimal(15000)
	c.run(c.command(commissioner.ScoreFranchise, commissioner.RawArgs{Score: &commissioner.RawScoreChange{Franchise: "franchise.01", Season: 2026, Week: 1, Delta: &delta}}, "score.once"))
	c.run(c.command(commissioner.StandingsRecordAdjust, commissioner.RawArgs{Standings: &commissioner.RawStandingsChange{Franchise: "franchise.01", Season: 2026, Wins: 1}}, "record.once"))
	fresh := history.New(c.f.pools, &measures.Registry{})
	replayStandings, err := fresh.NativeStandings(t.Context(), "native.hosted", 2026)
	if err != nil {
		t.Fatal(err)
	}
	var hostedStandings commissioner.NativeStandings
	if err := json.Unmarshal(c.invoke("standings.read", "read", []byte(`{"league":"native.hosted","season":2026}`), 200).Data, &hostedStandings); err != nil {
		t.Fatal(err)
	}
	if hostedStandings.Override == nil || hostedStandings.Override.Records["franchise.01"].Wins != 1 || !reflect.DeepEqual(hostedStandings.Override, replayStandings.Override) || !reflect.DeepEqual(hostedStandings.Computed, replayStandings.Computed) {
		t.Fatal("correction replay disagrees with hosted standings")
	}
	c.run(c.command(commissioner.SeasonRollover, commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2027}}, "rollover.once"))
	requirePlainPreviews(t, c.previews)
	events, err := c.f.history.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	records := make([]commissioner.Record, 0, len(events))
	for _, e := range events {
		records = append(records, *e.Payload.Commissioner)
	}
	replay, err := commissioner.NativeAssetsReading("native.hosted", records)
	if err != nil {
		t.Fatal(err)
	}
	current, err := c.f.history.NativeAssets(t.Context(), "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	nextBid := c.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "6001", Franchise: "franchise.01", Salary: 500000000, Years: 1}}, "old.evidence")
	body, err := json.Marshal(nextBid)
	if err != nil {
		t.Fatal(err)
	}
	c.invoke(nextBid.Command, "execute", body, 409)
	preserved, err := c.f.history.NativeAssets(t.Context(), "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	if preserved.Revision != current.Revision {
		t.Fatal("old-season evidence appended a bid after rollover")
	}
	if replay.Snapshot.Season != 2027 || !reflect.DeepEqual(replay.Snapshot, current.Snapshot) {
		t.Fatal("rollover replay disagrees with hosted assets")
	}
}

func TestHostedBlindBidIsInvisibleEvenToCommissioner(t *testing.T) {
	c := workflowClient(t)
	other := c
	other.bearer, other.proof = c.f.login(t, "owner", "private-password"), ""
	reads := []struct{ command, args string }{
		{"bids.read", `{"league":"native.hosted","franchise":"franchise.01"}`},
		{"commish.market.read", `{"league":"native.hosted","command":"bids.process"}`},
		{"commish.market.read", `{"league":"native.hosted","command":"bids.award.override"}`},
		{"native.assets.read", `{"league":"native.hosted"}`},
		{"standings.read", `{"league":"native.hosted","season":2026}`},
	}
	before := make([]serverhttp.WorkflowResponse, len(reads))
	for i, r := range reads {
		before[i] = c.invoke(r.command, "read", []byte(r.args), 200)
	}
	other.run(other.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "6001", Franchise: "franchise.02", Salary: 500000000, Years: 1}}, "hidden.bid"))
	for i, r := range reads {
		after := c.invoke(r.command, "read", []byte(r.args), 200)
		if !reflect.DeepEqual(before[i], after) {
			t.Fatalf("hidden bid changed commissioner reading %s", r.command)
		}
	}
	c.invoke("bids.read", "read", []byte(`{"league":"native.hosted","franchise":"franchise.02"}`), 409)
	var own commissioner.NativeBids
	if err := json.Unmarshal(other.invoke("bids.read", "read", []byte(`{"league":"native.hosted","franchise":"franchise.02"}`), 200).Data, &own); err != nil {
		t.Fatal(err)
	}
	if len(own.Players) != 1 || own.Players[0].YourBid == nil || own.Players[0].Leader != nil {
		t.Fatal("owner cannot read just their private bid")
	}
}

// requirePlainPreviews pins the owner-facing confirm wording: names rather than ledger, account or player
// ids, no stored status codes, and counts that agree with their nouns.
func requirePlainPreviews(t *testing.T, previews map[string][]commissioner.PreviewField) {
	t.Helper()
	want := []struct{ command, label, before, after string }{
		{commissioner.BidPlace, "Bids for player 6001", "None recorded", "1 bid awaiting a decision"},
		{commissioner.RosterCut, "Waivers for Other team receiver", "None recorded", "Waiting for processing"},
		{commissioner.RosterCut, "Practice team 2 cap charges", "None recorded", "Dead money for released player Other team receiver: $700,000.00"},
		{commissioner.RosterCut, "Other team receiver contract", "$1,000,000.00 salary; contract ends in 2028", "None recorded"},
		{commissioner.TradeOffer, "This trade, between Practice team 1 and Practice team 2", "None recorded", "Waiting for every party to accept"},
		{commissioner.TradeAccept, "This trade, between Practice team 2 and Practice team 3", "Waiting for every party to accept", "Waiting for trade review"},
		{commissioner.TradeReview, "This trade, between Practice team 2 and Practice team 3", "Waiting for trade review", "Approved; the assets have changed teams; 1 review vote"},
		{commissioner.TradeReview, "Season 2027, round 1, Practice team 2 pick", "Practice team 2; not used", "Practice team 3; not used"},
		{commissioner.StandingsRecordAdjust, "Record set by hand for Practice team 1", "None recorded", "1 win, 0 losses, 0 ties"},
	}
	for _, w := range want {
		if !slices.ContainsFunc(previews[w.command], func(f commissioner.PreviewField) bool {
			return f.Label == w.label && f.Before == w.before && f.After == w.after
		}) {
			t.Errorf("%s preview lacks %q: %q -> %q; got %+v", w.command, w.label, w.before, w.after, previews[w.command])
		}
	}
	raw := regexp.MustCompile(`\b(act|acct|proposed)\.|\b1 (bids|review votes|wins)\b|\(slot 0\)|\b(offered|review|approved|withdrawn);|; $`)
	for command, fields := range previews {
		for _, f := range fields {
			if f.Label == "League ledger" {
				continue
			}
			for _, words := range []string{f.Label, f.Before, f.After} {
				if raw.MatchString(words) {
					t.Errorf("%s preview shows an internal id, status code or wrong plural: %q", command, words)
				}
			}
		}
	}
}
