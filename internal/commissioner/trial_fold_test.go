package commissioner

import (
	"encoding/json"

	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// A proposal over the held ledger extends the held fold; it must equal folding the whole trial ledger,
// leave the shared held fold unchanged, and fold the held ledger once however many proposals are priced.
func TestTrialFoldMatchesFullFold(t *testing.T) {
	b := pinnedStatLedger(t)
	first, second, third := tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6), tackleVersion(2, "LB", 30000, 7)
	rejection := b.statCommand(StatReject, RawStatChange{Item: itemOf(second)}, 8)
	b.statCommand(StatApprove, RawStatChange{Item: itemOf(third)}, 8)
	b.revert(rejection, seasonMoment(9))
	held := WithReplayCache(b.records)
	memo := NewRulesMemo()
	before := foldLedger(held)
	if !reflect.DeepEqual(memo.fold(held), before) {
		t.Fatal("held fold differs")
	}
	items := []string{itemOf(first), itemOf(second), itemOf(third), "2026.1.0100.1.ffffffffffff", "bad"}
	for _, command := range []string{StatApprove, StatReject} {
		for _, item := range items {
			trial := scoreTrial(RawCommand{League: "native.test", Command: command, Request: "queue." + item,
				Args: RawArgs{Stat: &RawStatChange{Item: item}}}, held)
			want := foldLedger(trial)
			for range 2 {
				if got := memo.fold(trial); !reflect.DeepEqual(got, want) {
					t.Fatalf("%s %s: trial fold differs from the full fold\n got %+v\nwant %+v", command, item, got.effects, want.effects)
				}
			}
		}
	}
	if !reflect.DeepEqual(memo.fold(held), before) {
		t.Fatal("a proposal wrote into the held fold")
	}
	if memo.FullFolds() != 1 {
		t.Fatalf("held ledger folded %d times, want once", memo.FullFolds())
	}
}

// Every priced item must read exactly as pricing it alone, the pre-memo way: a full trial fold for the
// points and a standalone result-flip check.
func TestStatQueuePricingMatchesStandalonePricing(t *testing.T) {
	b := pinnedStatLedger(t)
	first, second, third := tackleVersion(0, "LB", 10000, 5), tackleVersion(1, "LB", 20000, 6), tackleVersion(1, "DT", 30000, 7)
	other := statWeek(first, third)
	other.Player = "0100"
	out := queueOf(t, b, statWeek(first, second), other)
	if len(out.Open) != 2 {
		t.Fatalf("two changed weeks, two items: %+v", out)
	}
	byPlayer := map[string]StatWeek{"0099": statWeek(first, second), "0100": other}
	at := seasonMoment(20)
	for _, item := range out.Open {
		week := byPlayer[item.Player]
		raw := RawCommand{League: "native.test", Command: StatApprove, Request: "queue." + item.ID,
			Args: RawArgs{Stat: &RawStatChange{Item: item.ID}}}
		scope := source.RawNativeScope{Player: week.Player, Season: 2026, Week: week.Week}
		before, err := NativePlayerScoreWith("native.test", scope, b.records, week.Evidence, at, RulesInForce)
		if err != nil {
			t.Fatal(err)
		}
		after, err := NativePlayerScoreWith("native.test", scope, scoreTrial(raw, b.records), week.Evidence, at, RulesInForce)
		if err != nil {
			t.Fatal(err)
		}
		flip, err := ResultFlipCheck(t.Context(), "native.test", b.records, noGames{}, raw, 2026, week.Week, at)
		if err != nil {
			t.Fatal(err)
		}
		want, _ := json.Marshal([]any{pointsChange(before, after), flip.Status, flip.Words})
		got, _ := json.Marshal([]any{item.Points, item.Flip, item.FlipWords})
		if string(got) != string(want) {
			t.Fatalf("%s priced %s, standalone %s", item.ID, got, want)
		}
	}
}

func TestTrialReplayMatchesFullReplay(t *testing.T) {
	b := pinnedStatLedger(t)
	second := tackleVersion(1, "LB", 20000, 6)
	rejection := b.statCommand(StatReject, RawStatChange{Item: itemOf(second)}, 8)
	b.revert(rejection, seasonMoment(9))
	held := WithReplayCache(b.records)
	for _, command := range []string{StatApprove, StatReject} {
		trial := scoreTrial(RawCommand{League: "native.test", Command: command, Request: "queue.x",
			Args: RawArgs{Stat: &RawStatChange{Item: itemOf(second)}}}, held)
		got, ok := cachedTrialReplay(trial)
		if !ok || !reflect.DeepEqual(got, replayFull(trial)) {
			t.Fatalf("%s: trial replay differs from the full replay", command)
		}
	}
	other := scoreTrial(RawCommand{League: "native.test", Command: LineupCorrect, Request: "x"}, held)
	if _, ok := cachedTrialReplay(other); ok {
		t.Fatal("only a proposed statistics decision may extend the held replay")
	}
}
