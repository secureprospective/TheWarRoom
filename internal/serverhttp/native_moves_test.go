package serverhttp_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
)

func TestHostedNativeMovesKeepsBlindBidsPrivate(t *testing.T) {
	c := workflowClient(t)
	other := c
	other.bearer, other.proof = c.f.login(t, "owner", "private-password"), ""
	const mine = `{"league":"native.hosted","franchise":"franchise.01"}`
	const theirs = `{"league":"native.hosted","franchise":"franchise.02"}`
	before := c.invoke("moves.read", "read", []byte(mine), 200)
	other.run(other.command(commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{
		Player: "6001", Franchise: "franchise.02", Salary: 500000000, Years: 1}}, "private.move"))
	if after := c.invoke("moves.read", "read", []byte(mine), 200); !reflect.DeepEqual(before, after) {
		t.Fatal("another owner's sealed bid changed the HQ reading")
	}
	c.invoke("moves.read", "read", []byte(theirs), 409)
	var own market.NativeMoves
	if err := json.Unmarshal(other.invoke("moves.read", "read", []byte(theirs), 200).Data, &own); err != nil {
		t.Fatal(err)
	}
	if len(own.Moves) != 1 || own.Moves[0].State != "handed_off" || own.Moves[0].Spec.Deadline != nil {
		t.Fatalf("owner's blind move: %+v", own)
	}
}
