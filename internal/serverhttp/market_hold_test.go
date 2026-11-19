package serverhttp_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func TestHostedCommissionerCanHoldTradeReviewOnceAndCorrectIt(t *testing.T) {
	commissionerClient := workflowClient(t)
	owner := commissionerClient
	owner.bearer, owner.proof = owner.f.login(t, "owner", "private-password"), ""
	partner := commissionerClient
	partner.bearer, partner.proof = partner.f.login(t, "partner", "private-password"), ""

	offer := owner.run(owner.command(commissioner.TradeOffer, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{
		Franchise: "franchise.02", Parties: []string{"franchise.02", "franchise.03"},
		Assets: []commissioner.TradeAsset{{From: "franchise.02", To: "franchise.03", Season: 2027, Round: 1, Original: "franchise.02"}},
	}}, "hold.offer"))
	request := func(raw commissioner.RawCommand) string {
		t.Helper()
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Command: &raw})
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	beforeNotReview, err := commissionerClient.f.history.SharedCommissionerRecords(t.Context(), "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	notReview := commissionerClient.command(commissioner.TradeHold,
		commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer.Entry}}, "hold.not-review")
	commissionerClient.f.post(t, serverhttp.ExecutePath, request(notReview),
		commissionerClient.bearer, commissionerClient.proof, 409)
	afterNotReview, err := commissionerClient.f.history.SharedCommissionerRecords(t.Context(), "native.hosted")
	if err != nil || len(afterNotReview) != len(beforeNotReview) {
		t.Fatalf("hold on a non-review trade changed history: %d -> %d, %v", len(beforeNotReview), len(afterNotReview), err)
	}
	partner.run(partner.command(commissioner.TradeAccept, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{
		ID: offer.Entry, Franchise: "franchise.03",
	}}, "hold.accept"))

	readTrade := func(client hostedWorkflowClient, viewer string) market.NativeTradeOffer {
		t.Helper()
		args, err := json.Marshal(map[string]string{"league": "native.hosted", "viewer": viewer})
		if err != nil {
			t.Fatal(err)
		}
		var trades market.NativeTrades
		if err := json.Unmarshal(client.invoke("trades.read", "read", args, 200).Data, &trades); err != nil {
			t.Fatal(err)
		}
		for _, row := range trades.Offers {
			if row.ID == offer.Entry {
				return row
			}
		}
		t.Fatalf("trade %s missing from %s read: %+v", offer.Entry, viewer, trades.Offers)
		return market.NativeTradeOffer{}
	}
	before := readTrade(owner, "franchise.02")
	if before.Status != "review" || len(before.Holds) != 0 {
		t.Fatalf("accepted trade did not enter unheld review: %+v", before)
	}
	choices, err := json.Marshal(map[string]string{"league": "native.hosted", "command": commissioner.TradeHold})
	if err != nil {
		t.Fatal(err)
	}
	var eligible market.Market
	if err := json.Unmarshal(commissionerClient.invoke("commish.market.read", "read", choices, 200).Data, &eligible); err != nil {
		t.Fatal(err)
	}
	if len(eligible.Trades) != 1 || eligible.Trades[0].ID != offer.Entry {
		t.Fatalf("commissioner hold choices = %+v", eligible.Trades)
	}

	const reason = "Updated injury information needs another review day"
	hold := commissionerClient.command(commissioner.TradeHold,
		commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer.Entry}}, "hold.once")
	hold.Reason = reason
	holdArguments, err := json.Marshal(hold)
	if err != nil {
		t.Fatal(err)
	}
	previewBody := commissionerClient.invoke(commissioner.TradeHold, "preview", holdArguments, 200).Data
	var preview commissioner.Preview
	if err := json.Unmarshal(previewBody, &preview); err != nil {
		t.Fatal(err)
	}
	fields := map[string]commissioner.PreviewField{}
	for _, field := range preview.Fields {
		fields[field.Label] = field
	}
	deadline := fields["Trade review deadline"]
	if deadline.Before != before.ExpiresAt.UTC().Format(time.RFC3339) ||
		deadline.After != before.ExpiresAt.Add(24*time.Hour).UTC().Format(time.RFC3339) {
		t.Fatalf("hold preview deadline = %+v; trade before = %s", deadline, before.ExpiresAt)
	}
	if got := fields["Trade review hold reason"]; got.Before != "No hold reason recorded" || got.After != reason {
		t.Fatalf("hold preview reason = %+v", got)
	}

	receipt := commissionerClient.run(hold)
	if receipt.Entry == "" {
		t.Fatal("trade hold returned no receipt")
	}
	for _, got := range []market.NativeTradeOffer{
		readTrade(owner, "franchise.02"), readTrade(partner, "franchise.03"),
	} {
		if got.ExpiresAt.Sub(before.ExpiresAt) != 24*time.Hour || len(got.Holds) != 1 || got.Holds[0].Reason != reason {
			t.Fatalf("team read missed the held deadline or reason: %+v", got)
		}
	}
	if err := json.Unmarshal(commissionerClient.invoke("commish.market.read", "read", choices, 200).Data, &eligible); err != nil || len(eligible.Trades) != 0 {
		t.Fatalf("held trade remained eligible for a second hold: %+v, %v", eligible.Trades, err)
	}

	beforeRefusals, err := commissionerClient.f.history.SharedCommissionerRecords(t.Context(), "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	secondHold := commissionerClient.command(commissioner.TradeHold,
		commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: offer.Entry}}, "hold.second")
	ownerHold := hold
	ownerHold.Request, ownerHold.Expected = "hold.owner-refused", secondHold.Expected
	commissionerClient.f.post(t, serverhttp.ExecutePath, request(ownerHold), owner.bearer, "", 409)
	commissionerClient.f.post(t, serverhttp.ExecutePath, request(secondHold), commissionerClient.bearer, commissionerClient.proof, 409)

	afterRefusals, err := commissionerClient.f.history.SharedCommissionerRecords(t.Context(), "native.hosted")
	if err != nil || len(afterRefusals) != len(beforeRefusals) {
		t.Fatalf("refused holds appended history: %d -> %d, %v", len(beforeRefusals), len(afterRefusals), err)
	}

	var holdRecord *commissioner.Record
	for i := range afterRefusals {
		if afterRefusals[i].Receipt.Entry == receipt.Entry {
			holdRecord = &afterRefusals[i]
			break
		}
	}
	if holdRecord == nil || holdRecord.Command.Command != commissioner.TradeHold ||
		holdRecord.Command.Reason != reason || holdRecord.Command.Visibility != "league" {
		t.Fatalf("trade hold was not recorded in public league history: %+v", holdRecord)
	}

	revert := commissionerClient.command(commissioner.Revert,
		commissioner.RawArgs{Entry: receipt.Entry}, "hold.revert")
	commissionerClient.f.post(t, serverhttp.ExecutePath, request(revert),
		commissionerClient.bearer, commissionerClient.proof, 200)
	reverted := readTrade(owner, "franchise.02")
	if !reverted.ExpiresAt.Equal(before.ExpiresAt) || len(reverted.Holds) != 0 {
		t.Fatalf("normal commissioner correction did not undo the hold: %+v", reverted)
	}
}
