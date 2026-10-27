package commissioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type draftBodyFixture struct {
	b     *ledgerBuilder
	at    time.Time
	facts nativecheck.Facts
}

func newDraftBodyFixture(t *testing.T) *draftBodyFixture {
	t.Helper()
	b := rolloverLedger(t)
	seed := b.records[1].Command.Args.Native
	for round := 1; round <= 5; round++ {
		for n := 1; n <= 32; n++ {
			team := fmt.Sprintf("franchise.%02d", n)
			seed.Picks = append(seed.Picks, NativePick{Season: 2027, Round: round, Original: team, Owner: team})
		}
	}
	facts := nativecheck.Facts{}
	for n := 5001; n <= 5006; n++ {
		facts[fmt.Sprint(n)] = nativecheck.PlayerFacts{DraftYear: 2027, DraftYearHeld: true, Name: fmt.Sprintf("Rookie %d", n)}
	}
	return &draftBodyFixture{b: b, at: seasonMoment(13), facts: facts}
}

func (f *draftBodyFixture) golden(t *testing.T, name string, raw RawCommand) {
	t.Helper()
	got, err := json.MarshalIndent(raw.Args, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := "testdata/draft-bodies/" + name + ".json"
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s: appended body differs from golden", name)
	}
}

func (f *draftBodyFixture) act(t *testing.T, name, command string, d RawDraftArgs, save bool) string {
	t.Helper()
	input := RawCommand{League: rolloverLeague, Command: command, Args: RawArgs{Draft: &d}, Request: fmt.Sprintf("draft.%d", len(f.b.records)), Expected: int64(len(f.b.records)), Reason: "Run the draft", Effective: "now", Visibility: "league"}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Decode(string(body))
	if err != nil {
		t.Fatalf("%s decode: %v", name, err)
	}
	if err = CheckDraftAppend(raw, f.b.records, f.at, nil); err != nil {
		t.Fatalf("%s check: %v", name, err)
	}
	if err = PrepareDraftAppend(&raw, f.b.records, f.at, f.facts, nil); err != nil {
		t.Fatalf("%s prepare: %v", name, err)
	}
	if save {
		f.golden(t, name, raw)
	}
	return f.b.add(raw, f.at)
}

func TestDraftBodies(t *testing.T) {
	f := newDraftBodyFixture(t)
	order := make([]string, 32)
	for i := range order {
		order[i] = fmt.Sprintf("franchise.%02d", i+1)
	}
	f.act(t, "draft.order.auto", DraftOrderAuto, RawDraftArgs{Season: 2027, Source: leaguerules.WorstStandingsFirst}, true)
	f.act(t, "draft.order.set", DraftOrderSet, RawDraftArgs{Season: 2027, Order: order}, true)
	f.act(t, "draft.order.pattern", DraftOrderPattern, RawDraftArgs{Pattern: leaguerules.Snake}, true)
	rules, err := RulesNow(f.b.records, f.at)
	if err != nil {
		t.Fatal(err)
	}
	if rules.Draft.LaterPattern.Value != leaguerules.Snake {
		t.Fatalf("pattern did not fold: %+v", rules.Draft.LaterPattern)
	}
	f.act(t, "draft.order.tiebreak", DraftOrderTiebreak, RawDraftArgs{Tie: "Better record"}, true)
	start := time.Date(2027, 7, 14, 16, 0, 0, 0, time.UTC)
	f.act(t, "draft.start", DraftStart, RawDraftArgs{Season: 2027, At: start}, true)
	f.at = start
	in := DraftPickInput{League: rolloverLeague, Season: 2027, Franchise: order[0], Player: "5001", Request: "pick", Expected: int64(len(f.b.records))}
	raw, err := PlanDraftPick(in, f.b.records, f.facts, f.at, time.UTC, nil)
	if err != nil {
		t.Fatalf("draft.pick: %v", err)
	}
	f.golden(t, "draft.pick", raw)
	f.b.add(raw, f.at)
	act := func(name, command string, a RawDraftAct) string {
		return f.act(t, name, command, RawDraftArgs{Season: 2027, Act: &a}, true)
	}
	checkpoint := act("draft.skip", DraftSkip, RawDraftAct{Round: 1, Original: order[1]})
	act("draft.assign", DraftAssign, RawDraftAct{Round: 1, Original: order[1], Player: "5002"})
	act("draft.insert", DraftInsert, RawDraftAct{Round: 1, Original: order[1], Franchise: order[2], Player: "5003"})
	act("draft.change", DraftChange, RawDraftAct{Round: 1, Original: order[1], Player: "5004", Replaces: "5002"})
	assets, err := NativeAssetsReading(rolloverLeague, f.b.records)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, pick := range assets.Snapshot.Picks {
		if pick.Season == 2027 && pick.Round == 1 && pick.Original == order[1] {
			found = true
			if pick.Player != "5004" {
				t.Fatalf("changed pick holds %q, want 5004", pick.Player)
			}
		}
	}
	if !found {
		t.Fatal("changed pick not found")
	}
	act("draft.change-again", DraftChange, RawDraftAct{Round: 1, Original: order[1], Player: "5005", Replaces: "5004"})
	act("draft.timer.pause", DraftTimer, RawDraftAct{Timer: TimerPause})
	act("draft.timer.resume", DraftTimer, RawDraftAct{Timer: TimerResume})
	act("draft.timer.set", DraftTimer, RawDraftAct{Timer: TimerSet, Seconds: 1800})
	act("draft.revert", DraftRevert, RawDraftAct{To: checkpoint})
	f.act(t, "draft.archive", DraftArchive, RawDraftArgs{Season: 2027}, true)
}
