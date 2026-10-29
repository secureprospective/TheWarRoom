package market

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"

	"github.com/secureprospective/TheWarRoom/internal/pickvalue"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
)

func TestPickSlotsContinueAfter2026(t *testing.T) {
	t.Parallel()
	for _, season := range []int{2026, 2027, 2028} {
		t.Run(fmt.Sprint(season), func(t *testing.T) {
			records := []playerrecord.Record{
				{MFL: "0001", Draft: playerrecord.Draft{Year: 2027, Status: playerrecord.Drafted, Overall: 5}},
				{MFL: "0002", Draft: playerrecord.Draft{Year: 2027, Status: playerrecord.Undrafted}},
				{MFL: "0003", Draft: playerrecord.Draft{Year: 2027, Status: playerrecord.DraftUnknown}},
				{MFL: "0004", Draft: playerrecord.Draft{Year: 2028, Status: playerrecord.Drafted, Overall: 10}},
			}
			url := func(year int, league string) string {
				return fmt.Sprintf("https://example.test/%d/export?TYPE=draftResults&L=%s", year, league)
			}
			body := func(player string) []byte {
				return []byte(`{"draftResults":{"draftUnit":{"unit":"LEAGUE","draftPick":[` +
					`{"round":"2","pick":"1","player":"` + player + `","franchise":"0001"},` +
					`{"round":"2","pick":"2","player":"0002","franchise":"0002"},` +
					`{"round":"2","pick":"3","player":"0003","franchise":"0003"},` +
					`{"round":"2","pick":"4","player":"","franchise":"0004"}]}}}`)
			}
			bodies := func(_ context.Context, _ string, take func(string, []byte, time.Time) bool) (bool, error) {
				take(url(2027, "other"), body("0001"), time.Now())
				take(url(2027, "14432"), []byte(`bad`), time.Now())
				take(url(2028, "14432"), body("0004"), time.Now())
				take(url(2027, "14432"), body("0001"), time.Now())
				take(url(2027, "14432"), body("0003"), time.Now()) // older correction is not counted again
				return false, nil
			}
			h, err := HeldPickHistory(t.Context(), bodies, "14432", season, records)
			if err != nil {
				t.Fatal(err)
			}
			want := map[int]int{2026: 0, 2027: 2, 2028: 3}[season]
			if len(h) != want {
				t.Fatalf("slots: %+v, want %d", h, want)
			}
			if season >= 2027 && (h[0].Selection != 5 || h[0].NFLOverall != 5 || h[1].NFLOverall != 0) {
				t.Fatalf("wrong slots or undrafted handling: %+v", h)
			}
			base, err := pickvalue.History()
			if err != nil {
				t.Fatal(err)
			}
			if len(base) != 1689 {
				t.Fatal("frozen approved sample changed")
			}
			n := NewNative(NativeInput{History: h, Records: records, Season: season, Discount: 0.75, Teams: 4})
			if season >= 2027 {
				v := n.Pick(season+1, 1, "", time.Now())
				if n.picks == nil || v.Valued || v.Reason == "" {
					t.Fatal("history fitted without player values must explain its unavailable value", n.pickWhy, v)
				}
			}
		})
	}
}

func TestPickHistoryFailureDoesNotRemovePlayerValues(t *testing.T) {
	failure := errors.New("archive unavailable")
	_, err := HeldPickHistory(t.Context(), func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
		return false, failure
	}, "14432", 2027, nil)
	if !errors.Is(err, failure) {
		t.Fatal("lost archive failure", err)
	}
	n := NewNative(NativeInput{Values: map[string]float64{"0042": 6}, PickWhy: err.Error()})
	if !n.Player("0042", time.Now()).Valued || n.Pick(2027, 1, "", time.Now()).Valued {
		t.Fatal("archive failure removed player values or fabricated pick history")
	}
}

func TestDraftSlotSelectionUsesThatDraftsWidth(t *testing.T) {
	for _, teams := range []int{2, 4, 16, 32} {
		t.Run(fmt.Sprint(teams), func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{"draftResults":{"draftUnit":{"draftPick":[
 {"round":"1","pick":"%d","player":"","franchise":"0002"},
 {"round":"2","pick":"1","player":"0001","franchise":"0001"}]}}}`, teams))
			records := []playerrecord.Record{{MFL: "0001", Draft: playerrecord.Draft{Year: 2027,
				Status: playerrecord.Drafted, Overall: 10}}}
			bodies := func(_ context.Context, _ string, take func(string, []byte, time.Time) bool) (bool, error) {
				take("https://example.test/2027/export?TYPE=draftResults&L=14432", body, time.Now())
				return false, nil
			}
			h, err := HeldPickHistory(t.Context(), bodies, "14432", 2028, records)
			if err != nil || len(h) != 1 || h[0].Selection != teams+1 {
				t.Fatalf("historical width replaced by today's league: %+v %v", h, err)
			}
		})
	}
}

func TestPickTotalsMissingAndOwnerHorizon(t *testing.T) {
	at := time.Time{}
	owners := []string{"zero", "buyer", "missing"}
	inventory := []draftreports.FuturePick{{Year: 2027, Round: 1, Original: "original", Owner: "buyer"}, {Year: 2028, Round: 1, Owner: "buyer"}, {Year: 2029, Round: 1, Owner: "buyer"}, {Year: 2026, Round: 1, Owner: "buyer"}, {Year: 2027, Round: 2, Owner: "missing"}}
	model := pickTotalModel{}
	got := PickTotals(model, inventory, 2026, owners, at)
	if got["buyer"] != 4 || got["zero"] != 0 {
		t.Fatal(got)
	}
	if _, ok := got["missing"]; ok {
		t.Fatal("unreadable pick became zero", got)
	}
	if PickTotals(nil, inventory, 2026, owners, at) != nil || PickTotals(model, nil, 2026, owners, at) != nil {
		t.Fatal("missing inputs became zero")
	}
	if got := PickTotals(model, []draftreports.FuturePick{}, 2026, owners, at); len(got) != 3 {
		t.Fatal("known empty inventory not zero", got)
	}
}

type pickTotalModel struct {
	reduced bool
	reason  string
}

func (pickTotalModel) Player(string, time.Time) Dynasty { return Dynasty{} }
func (m pickTotalModel) Pick(_ int, round int, _ string, _ time.Time) Dynasty {
	return Dynasty{Valued: round == 1, Discounted: 2, ReducedInputs: m.reduced, Reason: m.reason}
}

func TestNativeHistoryDoesNotIncludeStartupSkipsUnknownOrFuture(t *testing.T) {
	records := []playerrecord.Record{{MFL: "0042", Draft: playerrecord.Draft{Year: 2027, Status: playerrecord.Drafted, Overall: 5}}, {MFL: "0043", Draft: playerrecord.Draft{Year: 2027, Status: playerrecord.Undrafted}}}
	order := []rookiedraft.RoundOrder{{Round: 1, Picks: []rookiedraft.Pick{{OwnedPick: rookiedraft.OwnedPick{Round: 1, Original: "seller", Owner: "buyer"}, Slot: 1}, {OwnedPick: rookiedraft.OwnedPick{Round: 1, Original: "other", Owner: "other"}, Slot: 2}}}}
	drafts := []commissioner.NativeDraft{{Season: 2027, Order: order, Draft: rookiedraft.Draft{Picks: []rookiedraft.Result{
		{Pick: rookiedraft.Pick{OwnedPick: rookiedraft.OwnedPick{Round: 1}, Slot: 1}, Player: "0042"},
		{Pick: rookiedraft.Pick{OwnedPick: rookiedraft.OwnedPick{Round: 2}, Slot: 2}, Player: "0043"},
		{Pick: rookiedraft.Pick{OwnedPick: rookiedraft.OwnedPick{Round: 1}, Slot: 2}, Player: "0042", Skipped: true},
		{Pick: rookiedraft.Pick{OwnedPick: rookiedraft.OwnedPick{Round: 1}, Slot: 2}, Player: "missing"},
		{Pick: rookiedraft.Pick{OwnedPick: rookiedraft.OwnedPick{Round: 1}, Slot: 2}, Player: "0042", Extra: 1},
	}}}, {Season: 2027, Startup: true, Order: order, Draft: rookiedraft.Draft{Picks: []rookiedraft.Result{{Player: "0042"}}}}}
	h, slots := NativePickHistory(drafts, 2027, records)
	if len(h) != 2 || h[0].NFLOverall != 5 || h[1].NFLOverall != 0 || h[1].Selection != 4 || slots[PickKey{Year: 2027, Round: 1, Original: "seller"}] != 1 {
		t.Fatal(h, slots)
	}
	h, _ = NativePickHistory(drafts, 2026, records)
	if len(h) != 0 {
		t.Fatal("future draft trained curve", h)
	}
}

func TestPickCapitalRetainsReducedAndMissingInputReasons(t *testing.T) {
	why := "league points for 2024 are not ready"
	inventory := []draftreports.FuturePick{{Year: 2027, Round: 1, Owner: "buyer"}, {Year: 2027, Round: 2, Owner: "missing"}}
	capital := pickCapital(pickTotalModel{reduced: true, reason: why}, inventory, 2026, []string{"buyer", "missing"}, time.Time{})
	if capital.Totals["buyer"] != 2 || !capital.Status["buyer"].ReducedInputs || capital.Status["buyer"].Reason != why {
		t.Fatal(capital)
	}
	if _, ok := capital.Totals["missing"]; ok || !capital.Status["missing"].Unavailable || capital.Status["missing"].Reason != why {
		t.Fatal(capital)
	}
}
