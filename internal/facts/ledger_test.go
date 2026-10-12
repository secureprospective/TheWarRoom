package facts

import (
	"strconv"
	"testing"
)

func TestDecodeAsset(t *testing.T) {
	for _, c := range []struct{ code, want string }{
		{"\tFP_0007_2027_2\n", "pick 2027 r2 slot - orig 0007"}, {"DP_3_20", "pick 2026 r4 slot 21 orig -"},
		{"15214", "player"}, {"BB_10", "other"}, {"", "other"}, {"FP_7_2027_2", "other"},
	} {
		code, want := c.code, c.want
		a := decodeAsset(code, 2026)
		got := a.kind
		if a.kind == "pick" {
			got = "pick " + itoa(a.pickYear) + " r" + itoa(a.pickRound) + " slot " + itoa(a.pickSlot) + " orig " + orDash(a.pickOrig)
		}
		if got != want {
			t.Errorf("%q: %s, want %s", code, got, want)
		}
	}
}

func itoa(n *int64) string {
	if n == nil {
		return "-"
	}
	return strconv.FormatInt(*n, 10)
}

func orDash(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}

func TestTransactionsNumberAcrossSeasonsAndDecodeRoles(t *testing.T) {
	facts := compileFor(t, []ResolvedInput{
		inputFor("2025/transactions.json", `{"transactions":{"transaction":[
			{"type":"TRADE","franchise":"0001","franchise2":"0002","franchise1_gave_up":"100,FP_0001_2026_1,",
			 "franchise2_gave_up":"DP_0_4","timestamp":"1735707600","comments":"  "},
			{"type":"FREE_AGENT","franchise":"0003","transaction":"200,|300,","by_commish":"1"}]}}`),
		inputFor("2026/transactions.json", `{"transactions":{"transaction":
			{"type":"IR","franchise":"0004","deactivated":"400","activated":"","timestamp":"0"}}}`),
	})
	x := facts.Txn
	if len(x.TxnID) != 3 || x.TxnID[2] != 3 || x.Year[2] != 2026 || x.SrcIdx[1] != 1 {
		t.Fatalf("numbering %v", x.TxnID)
	}
	if value(x.Day[0]) != "2025-01-01" || x.Day[2] != nil || x.Comments[0] != nil || x.ByCommish[1] != 1 {
		t.Fatal("eastern day, zero time, blank comment or commissioner flag")
	}
	a := facts.TxnAsset
	roles := map[string]string{}
	for i, code := range a.Code {
		roles[code] = a.Role[i] + " " + orDash(a.FromFid[i]) + ">" + orDash(a.ToFid[i])
	}
	want := map[string]string{"100": "sent 0001>0002", "FP_0001_2026_1": "sent 0001>0002", "DP_0_4": "sent 0002>0001",
		"200": "add ->0003", "300": "drop 0003>-", "400": "ir_on 0004>0004"}
	if len(roles) != len(want) {
		t.Fatalf("assets %v", roles)
	}
	for code, role := range want {
		if roles[code] != role {
			t.Errorf("%s: %s, want %s", code, roles[code], role)
		}
	}
	v := facts.Views()
	if len(v.TradeAsset.Code) != 3 || len(v.RosterMove.Role) != 3 || len(v.Offer.TxnID) != 0 {
		t.Fatalf("views: %d trade assets, %d moves", len(v.TradeAsset.Code), len(v.RosterMove.Role))
	}
}

func TestDraftForfeitAndAssetsPicks(t *testing.T) {
	facts := compileFor(t, []ResolvedInput{
		inputFor("2026/draftResults.json", `{"draftResults":{"draftUnit":{"draftPick":[
			{"round":"01","pick":"01","franchise":"0001","player":"----","timestamp":"1"},
			{"round":"01","pick":"02","franchise":"0002","player":"15214","comments":" [Pick traded] "}]}}}`),
		inputFor("2026/assets.json", `{"assets":{"franchise":{"id":"0001",
			"currentYearDraftPicks":{"draftPick":{"pick":"DP_1_0","description":"Round 2 Pick 1"}},
			"futureYearDraftPicks":""}}}`),
	})
	d := facts.DraftPick
	if d.Pid[0] != nil || value(d.Pid[1]) != "15214" || value(d.Comments[1]) != "[Pick traded]" || d.Round[1] != 1 {
		t.Fatalf("draft %+v", d)
	}
	p := facts.PickOwned
	if len(p.Fid) != 1 || *p.Round[0] != 2 || *p.Slot[0] != 1 || value(p.Code[0]) != "DP_1_0" || p.Source[0] != "assets" {
		t.Fatalf("assets picks %+v", p)
	}
}
