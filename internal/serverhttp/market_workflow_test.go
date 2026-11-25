package serverhttp_test

import (
	"encoding/json"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
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
func TestHostedNativeAssetsReadIncludesRuleBasedOpenBenchSlots(t *testing.T) {
	client := workflowClient(t)
	response := client.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200)
	var assets commissioner.NativeAssets
	if err := json.Unmarshal(response.Data, &assets); err != nil {
		t.Fatal(err)
	}
	if assets.OpenBenchSlots["franchise.01"] != 97 || assets.OpenBenchSlots["franchise.02"] != 99 {
		t.Fatalf("hosted open bench slots = %v; want 97 and 99 under the held 100-player limit",
			assets.OpenBenchSlots)
	}
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
func (c hostedWorkflowClient) refuse(raw commissioner.RawCommand, want string) []byte {
	c.t.Helper()
	body, err := json.Marshal(raw)
	if err != nil {
		c.t.Fatal(err)
	}
	workflow := workflowRequest(c.t, raw.Command, "execute", string(body), c.football)
	request, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &workflow})
	if err != nil {
		c.t.Fatal(err)
	}
	response := c.f.post(c.t, serverhttp.WorkflowPath, string(request), c.bearer, c.proof, 409)
	if want != "" && !strings.Contains(string(response), want) {
		c.t.Fatalf("%s refusal = %q, want %q", raw.Command, response, want)
	}
	return response
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

func TestHostedDraftQueueIsSeatedPrivateAndReplacedAsAWhole(t *testing.T) {
	commissionerClient := workflowClient(t)
	owner := commissionerClient
	owner.bearer = owner.f.login(t, "owner", "private-password")
	owner.proof = ""
	owner.f.footballGone.Store(true)
	token, err := owner.f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: commissionerClient.bearer}, accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.01", Commands: []string{"native.directory.read", "draft.queue.read", "draft.queue.set"},
		Modes: []string{"read", "execute"},
	})
	if err != nil {
		t.Fatal(err)
	}
	agent := commissionerClient
	agent.bearer, agent.proof = token.Bearer, ""
	agent.invoke("draft.queue.set", "execute", []byte(`{"league":"native.hosted","players":["0101"]}`), 200)
	agentRead := agent.invoke("draft.queue.read", "read", []byte(`{"league":"native.hosted"}`), 200)
	var agentQueue struct {
		Players []string `json:"players"`
	}
	if err := json.Unmarshal(agentRead.Data, &agentQueue); err != nil || !slices.Equal(agentQueue.Players, []string{"0101"}) {
		t.Fatalf("agent private queue = %+v, %v", agentQueue, err)
	}
	set := owner.invoke("draft.queue.set", "execute", []byte(`{"league":"native.hosted","players":["0099","0100"]}`), 200)
	var saved struct {
		Players []string `json:"players"`
	}
	if err := json.Unmarshal(set.Data, &saved); err != nil || !slices.Equal(saved.Players, []string{"0099", "0100"}) {
		t.Fatalf("saved queue = %+v, %v", saved, err)
	}

	commissionerRead := commissionerClient.invoke("draft.queue.read", "read", []byte(`{"league":"native.hosted"}`), 200)
	var hidden struct {
		Players []string `json:"players"`
	}
	if err := json.Unmarshal(commissionerRead.Data, &hidden); err != nil || !slices.Equal(hidden.Players, []string{"0101"}) {
		t.Fatalf("commissioner queue escaped its own seated franchise: %+v, %v", hidden, err)
	}
	other := owner
	other.bearer = owner.f.login(t, "partner", "private-password")
	other.proof = ""
	if _, err := owner.f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: other.bearer}, accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.01",
		Commands: []string{"draft.queue.read", "draft.queue.set"}, Modes: []string{"read", "execute"},
	}); err == nil {
		t.Fatal("another franchise delegated access to the owner's draft queue")
	}
	otherRead := other.invoke("draft.queue.read", "read", []byte(`{"league":"native.hosted"}`), 200)
	if err := json.Unmarshal(otherRead.Data, &hidden); err != nil || len(hidden.Players) != 0 {
		t.Fatalf("another franchise read the queue: %+v, %v", hidden, err)
	}
	other.invoke("draft.queue.set", "execute",
		[]byte(`{"league":"native.hosted","franchise":"franchise.02","players":["0101"]}`), 400)
	ownerRead := owner.invoke("draft.queue.read", "read", []byte(`{"league":"native.hosted"}`), 200)
	if err := json.Unmarshal(ownerRead.Data, &saved); err != nil || !slices.Equal(saved.Players, []string{"0099", "0100"}) {
		t.Fatalf("rejected cross-franchise request changed owner's queue: %+v, %v", saved, err)
	}
	set = owner.invoke("draft.queue.set", "execute", []byte(`{"league":"native.hosted","players":["0100","0099"]}`), 200)
	if err := json.Unmarshal(set.Data, &saved); err != nil || !slices.Equal(saved.Players, []string{"0100", "0099"}) {
		t.Fatalf("reordered queue = %+v, %v", saved, err)
	}
}

func TestHostedTradeDraftPocketIsPrivateAndTokenScoped(t *testing.T) {
	commissionerClient := workflowClient(t)
	teamA := commissionerClient
	teamB := commissionerClient
	teamB.bearer = teamB.f.login(t, "owner", "private-password")
	teamB.proof = ""

	first := teamA.invoke(leagueservice.TradeDraftSave, "execute", []byte(`{"league":"native.hosted","partner":"franchise.02","give":["0100"],"get":["pick:2027:1:franchise.02"],"label":"Window deal"}`), 200)
	var saved leagueservice.SavedTradeDraft
	if err := json.Unmarshal(first.Data, &saved); err != nil || len(saved.ID) != 32 || saved.Partner != "franchise.02" || saved.Label != "Window deal" || saved.At == "" {
		t.Fatalf("saved proposal = %+v, %v", saved, err)
	}
	partnerResult := teamB.invoke(leagueservice.TradeDraftSave, "execute", []byte(`{"league":"native.hosted","partner":"franchise.01","give":["0101"],"get":[],"label":"Partner proposal"}`), 200)
	var partnerDraft leagueservice.SavedTradeDraft
	if err := json.Unmarshal(partnerResult.Data, &partnerDraft); err != nil || partnerDraft.ID == "" {
		t.Fatalf("partner proposal = %+v, %v", partnerDraft, err)
	}
	read := func(client hostedWorkflowClient, want int) []leagueservice.SavedTradeDraft {
		t.Helper()
		response := client.invoke(leagueservice.TradeDraftRead, "read", []byte(`{"league":"native.hosted"}`), 200)
		var drafts leagueservice.TradeDraftList
		if err := json.Unmarshal(response.Data, &drafts); err != nil || len(drafts.Drafts) != want {
			t.Fatalf("saved proposals = %+v, %v; wanted %d", drafts, err, want)
		}
		return drafts.Drafts
	}
	if drafts := read(teamA, 1); drafts[0].ID != saved.ID || drafts[0].Give[0] != "0100" || drafts[0].Get[0] != "pick:2027:1:franchise.02" {
		t.Fatalf("owner proposal was not reopened intact: %+v", drafts)
	}
	if drafts := read(teamB, 1); drafts[0].Label != "Partner proposal" || drafts[0].ID == saved.ID {
		t.Fatalf("partner cannot read its own proposal: %+v", drafts)
	}
	deleteArgs := func(id string) []byte {
		t.Helper()
		body, err := json.Marshal(leagueservice.RawTradeDraftDelete{League: "native.hosted", ID: id})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	teamB.invoke(leagueservice.TradeDraftDelete, "execute", deleteArgs(saved.ID), 409)
	if drafts := read(teamA, 1); drafts[0].ID != saved.ID {
		t.Fatalf("partner deleted owner's proposal: %+v", drafts)
	}

	token, err := teamA.f.tokens.Issue(t.Context(), accounts.RawSession{Bearer: teamA.bearer}, accounts.RawTokenScope{
		League: "native.hosted", Franchise: "franchise.01",
		Commands: []string{"native.directory.read", leagueservice.TradeDraftRead, leagueservice.TradeDraftSave, leagueservice.TradeDraftDelete},
		Modes:    []string{"read", "execute"},
	})
	if err != nil {
		t.Fatal(err)
	}
	agent := commissionerClient
	agent.bearer, agent.proof = token.Bearer, ""
	if drafts := read(agent, 1); drafts[0].ID != saved.ID {
		t.Fatalf("owner-scoped token read another team's proposal: %+v", drafts)
	}
	agent.invoke(leagueservice.TradeDraftRead, "read", []byte(`{"league":"native.hosted","franchise":"franchise.02"}`), 400)
	agent.invoke(leagueservice.TradeDraftDelete, "execute", deleteArgs(partnerDraft.ID), 409)
	if drafts := read(teamB, 1); drafts[0].ID != partnerDraft.ID {
		t.Fatalf("team A token deleted team B proposal: %+v", drafts)
	}
	if drafts := read(teamA, 1); drafts[0].ID != saved.ID {
		t.Fatalf("spoofed franchise read changed the owner's proposal: %+v", drafts)
	}
	deleted := teamA.invoke(leagueservice.TradeDraftDelete, "execute", deleteArgs(saved.ID), 200)
	var receipt leagueservice.TradeDraftDeleted
	if err := json.Unmarshal(deleted.Data, &receipt); err != nil || receipt.ID != saved.ID {
		t.Fatalf("owner deletion = %+v, %v", receipt, err)
	}
	if drafts := read(teamA, 0); len(drafts) != 0 {
		t.Fatalf("owner deletion did not remove the proposal: %+v", drafts)
	}
}

func TestHostedRosterAddIsSeatedPreviewedAndReceipted(t *testing.T) {
	commissionerClient := workflowClient(t)
	owner := commissionerClient
	owner.bearer, owner.proof = owner.f.login(t, "owner", "private-password"), ""
	before := owner.command(commissioner.RosterAdd, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "6001", Franchise: "franchise.02"}}, "owner.add").Expected
	receipt := owner.run(owner.command(commissioner.RosterAdd, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "6001", Franchise: "franchise.02"}}, "owner.add"))
	if receipt.Entry == "" || receipt.Revision == 0 || receipt.Revision == before {
		t.Fatalf("free-agent add receipt = %+v, before revision %d", receipt, before)
	}
	var assets commissioner.NativeAssets
	if err := json.Unmarshal(owner.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200).Data, &assets); err != nil {
		t.Fatal(err)
	}
	if assets.Revision != receipt.Revision {
		t.Fatalf("receipt revision %d does not match landed assets %d", receipt.Revision, assets.Revision)
	}
	if index := assets.Snapshot.PlayerOf("franchise.02", "6001"); index < 0 || assets.Snapshot.Players[index].Salary <= 0 {
		t.Fatalf("owner add did not land a contracted player: %+v", assets.Snapshot.Players)
	}
	// roster.add is owner-only: a commissioner changes another team's roster through a correction.
	commissionerClient.refuse(commissionerClient.command(commissioner.RosterAdd,
		commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "6002", Franchise: "franchise.02"}}, "commissioner.add"), "")
	if err := json.Unmarshal(owner.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200).Data, &assets); err != nil {
		t.Fatal(err)
	}
	if assets.Snapshot.PlayerOf("franchise.02", "6002") >= 0 {
		t.Fatal("commissioner added a free agent to another owner's franchise")
	}
	beforeOther := assets.Revision
	beforeOtherSnapshot, err := json.Marshal(assets.Snapshot)
	if err != nil {
		t.Fatal(err)
	}

	other := commissionerClient
	other.bearer, other.proof = other.f.login(t, "partner", "private-password"), ""
	refused := other.command(commissioner.RosterAdd, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "6003", Franchise: "franchise.02"}}, "other.owner.add")
	other.refuse(refused, "")
	if err := json.Unmarshal(owner.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200).Data, &assets); err != nil {
		t.Fatal(err)
	}
	afterOtherSnapshot, err := json.Marshal(assets.Snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if assets.Revision != beforeOther || string(beforeOtherSnapshot) != string(afterOtherSnapshot) ||
		assets.Snapshot.PlayerOf("franchise.02", "6003") >= 0 {
		t.Fatalf("another owner's refused add changed league state: revision %d -> %d", beforeOther, assets.Revision)
	}
}

func TestHostedRosterAddRefusesWaiversRosteredPlayersAndClosedWindow(t *testing.T) {
	commissionerClient := workflowClient(t)
	owner := commissionerClient
	owner.bearer, owner.proof = owner.f.login(t, "owner", "private-password"), ""
	owner.run(owner.command(commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "0102", Franchise: "franchise.02"}}, "add.test.cut"))
	waived := owner.command(commissioner.RosterAdd, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "0102", Franchise: "franchise.02"}}, "add.on.waivers")
	owner.refuse(waived, "")
	rostered := owner.command(commissioner.RosterAdd, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "0099", Franchise: "franchise.02"}}, "add.rostered")
	owner.refuse(rostered, "")

	future := time.Now().UTC().AddDate(0, 0, 2)
	closedRules, err := json.Marshal(map[string]any{"future": map[string]any{
		"start": map[string]int{"month": int(future.Month()), "day": future.Day()}, "opens": []string{"ufa_bid"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	commissionerClient.run(commissionerClient.command(commissioner.RulesSet, commissioner.RawArgs{
		Field: "calendar.windows", Value: closedRules,
	}, "add.closed.window"))
	closed := owner.command(commissioner.RosterAdd, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "6001", Franchise: "franchise.02"}}, "add.closed")
	owner.refuse(closed, "")
}

func TestHostedRosterAddRequiresRoomOrAtomicDrop(t *testing.T) {
	commissionerClient := workflowClient(t)
	commissionerClient.run(commissionerClient.command(commissioner.RulesSet, commissioner.RawArgs{
		Field: "roster.limits", Value: json.RawMessage(`{"min":0,"max":3}`),
	}, "add.max.three"))
	owner := commissionerClient
	owner.bearer, owner.proof = owner.f.login(t, "owner", "private-password"), ""
	for _, player := range []string{"6002", "6003"} {
		owner.run(owner.command(commissioner.RosterAdd, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
			Player: player, Franchise: "franchise.02"}}, "fill."+player))
	}
	full := owner.command(commissioner.RosterAdd, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "6004", Franchise: "franchise.02"}}, "add.full")
	owner.refuse(full, "")
	var before commissioner.NativeAssets
	if err := json.Unmarshal(owner.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200).Data, &before); err != nil {
		t.Fatal(err)
	}
	if before.Snapshot.PlayerOf("franchise.02", "6004") >= 0 {
		t.Fatal("full-roster refusal appended the free agent")
	}
	add := owner.command(commissioner.RosterAdd, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{
		Player: "6004", Franchise: "franchise.02", Drop: "0102"}}, "add.with.drop")
	receipt := owner.run(add)
	var after commissioner.NativeAssets
	if err := json.Unmarshal(owner.invoke("native.assets.read", "read", []byte(`{"league":"native.hosted"}`), 200).Data, &after); err != nil {
		t.Fatal(err)
	}
	if receipt.Revision == before.Revision || after.Revision != receipt.Revision ||
		after.Snapshot.PlayerOf("franchise.02", "6004") < 0 || after.Snapshot.PlayerOf("franchise.02", "0102") >= 0 {
		t.Fatalf("combined add/drop was not one landed transaction: receipt=%+v before=%+v after=%+v", receipt, before, after)
	}
	var waivers market.NativeWaivers
	waiverArgs, err := json.Marshal(map[string]string{"league": "native.hosted", "franchise": "franchise.02", "at": time.Now().UTC().Format(time.RFC3339Nano)})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(owner.invoke("waivers.read", "read", waiverArgs, 200).Data, &waivers); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(waivers.Players, func(player market.NativeWaiverPlayer) bool {
		return player.Player == "0102" && player.Window == receipt.Entry
	}) {
		t.Fatalf("atomic drop did not create its waiver window: %+v", waivers.Players)
	}
	moveArgs, err := json.Marshal(map[string]string{"league": "native.hosted", "franchise": "franchise.02"})
	if err != nil {
		t.Fatal(err)
	}
	var moves market.NativeMoves
	if err := json.Unmarshal(owner.invoke("moves.read", "read", moveArgs, 200).Data, &moves); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(moves.Moves, func(move market.NativeMove) bool {
		return move.CorrelationID == receipt.Entry && slices.Equal(move.Spec.Subject.Players, []string{"6004", "0102"})
	}) {
		t.Fatalf("combined add/drop was not reported as one move: %+v", moves.Moves)
	}
}

func TestHostedBidsWaiversTradeCorrectionAndRollover(t *testing.T) {
	c := workflowClient(t)
	bid := c.run(c.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "6001", Franchise: "franchise.01", Salary: 500000000, Years: 1}}, "bid.once"))
	bid = c.run(c.command(commissioner.BidRaise, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Bid: bid.Entry, Franchise: "franchise.01", Salary: 600000000, Years: 1}}, "raise.once"))
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
		{"market.wins.read", `{"league":"native.hosted"}`},
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
	var own market.NativeBids
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

func TestHostedResultsUseRequestLeagueAndSeeCommittedScore(t *testing.T) {
	c := workflowClient(t)
	lineup := c.command(commissioner.LineupSet, commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{
		Season: 2026, Week: 1, Franchise: "franchise.01", Starters: []string{"0099"},
	}}, "burst.lineup")
	lineup.Visibility = "private"
	c.run(lineup)
	owner := c
	owner.bearer, owner.proof = c.f.login(t, "owner", "private-password"), ""
	args := []byte(`{"league":"native.hosted","season":2026,"week":1}`)
	readAdjustment := func(want leaguerules.Decimal) {
		t.Helper()
		var results commissioner.NativeResults
		if err := json.Unmarshal(owner.invoke("results.read", "read", args, 200).Data, &results); err != nil {
			t.Fatal(err)
		}
		if results.League != "native.hosted" || results.Season != 2026 || results.Week != 1 {
			t.Fatalf("wrong results scope: %s season %d week %d", results.League, results.Season, results.Week)
		}
		for _, score := range results.Scores {
			if score.Franchise == "franchise.01" {
				if score.Adjustment != want {
					t.Fatalf("score adjustment = %v, want %v", score.Adjustment, want)
				}
				return
			}
		}
		t.Fatal("results omitted franchise.01")
	}
	readAdjustment(0)
	owner.invoke("results.read", "read", []byte(`{"league":"native.other","season":2026,"week":1}`), 403)
	delta := leaguerules.Decimal(15001)
	c.run(c.command(commissioner.ScoreFranchise, commissioner.RawArgs{
		Score: &commissioner.RawScoreChange{Franchise: "franchise.01", Season: 2026, Week: 1, Delta: &delta},
	}, "burst.score"))
	readAdjustment(delta)
}
