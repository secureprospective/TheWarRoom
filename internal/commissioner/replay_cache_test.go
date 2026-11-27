package commissioner

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestReplayCacheOwnershipAndInvalidation(t *testing.T) {
	records := []Record{
		{Command: RawCommand{Command: Create, Args: RawArgs{Name: "League"}}, Principal: "creator", Receipt: Receipt{Entry: "create", Revision: 1}},
		{Command: RawCommand{Command: Revert, Args: RawArgs{Entry: "create"}}, Receipt: Receipt{Entry: "undo", Revision: 2}},
	}
	cached := WithReplayCache(records)
	want := Replay(records)
	if _, ok := cachedReplay(cached); !ok {
		t.Fatal("cache miss")
	}
	got := Replay(cached)
	got.Active["create"] = true
	if !reflect.DeepEqual(Replay(cached), want) {
		t.Fatal("caller mutated memo")
	}
	for _, edit := range []func([]Record) []Record{
		func(r []Record) []Record { r[0].Principal = "other"; return r },
		func(r []Record) []Record { r[1].Command.Args.Entry = "missing"; return r },
		func(r []Record) []Record { r[1].Receipt.Revision++; return r },
		func(r []Record) []Record { return r[:1] },
		func(r []Record) []Record { return append(r, Record{Receipt: Receipt{Entry: "new"}}) },
	} {
		changed := edit(WithReplayCache(records))
		if _, ok := cachedReplay(changed); ok {
			t.Fatal("edited revision hit cache")
		}
		full := WithoutClosedBooks(changed)
		if !reflect.DeepEqual(Replay(changed), Replay(full)) {
			t.Fatal("edited copy did not replay")
		}
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 10 {
				p := Replay(cached)
				if !reflect.DeepEqual(p, want) {
					t.Error("concurrent replay differs")
				}
				p.Active["create"] = true
			}
		})
	}
	wg.Wait()
}

func TestReplayCacheTracksDraftRestoreSeason(t *testing.T) {
	records := []Record{
		{Command: RawCommand{Command: DraftStart, Args: RawArgs{Draft: &RawDraftArgs{Season: 2026}}}, Receipt: Receipt{Entry: "start"}},
		{Command: RawCommand{Command: DraftPick, Args: RawArgs{Draft: &RawDraftArgs{Season: 2026}}}, Receipt: Receipt{Entry: "pick"}},
		{Command: RawCommand{Command: DraftRevert, Args: RawArgs{Draft: &RawDraftArgs{Season: 2026, Act: &RawDraftAct{To: "start"}}}}, Receipt: Receipt{Entry: "restore"}},
	}
	cached := WithReplayCache(records)
	if Replay(cached).Active["pick"] {
		t.Fatal("restore did not compensate pick")
	}
	cached[1].Command.Args.Draft.Season = 2027
	if _, ok := cachedReplay(cached); ok {
		t.Fatal("changed draft season hit memo")
	}
	if !Replay(cached).Active["pick"] {
		t.Fatal("changed draft season used stale projection")
	}
}

func TestDraftRevertAllowedWhileOwedPickRemains(t *testing.T) {
	const league = "native.test"
	create, err := RawCommand{League: league, Command: Create, Args: RawArgs{
		Name: "Ours", Preset: PresetDefault32, Source: SourceNative, Season: 2026, FranchiseCount: 2,
	}, Request: "create", Effective: "now", Visibility: "league"}.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	b := &ledgerBuilder{t: t}
	b.add(create, seasonMoment(1))
	for _, f := range []string{"franchise.01", "franchise.02"} {
		b.add(RawCommand{League: league, Command: "commish.seat.replace",
			Args:    RawArgs{People: &RawPeople{Franchise: f, Principal: "owner." + f}},
			Request: "seat." + f, Effective: "now", Visibility: "league"}, seasonMoment(1))
	}
	b.set("draft.rounds", "1", "now", seasonMoment(2))
	b.set("draft.clock", "10000000000", "now", seasonMoment(2))
	seed, err := NativeStart(league, b.records, seasonMoment(2))
	if err != nil {
		t.Fatal(err)
	}
	seed.Picks = []NativePick{
		{Season: 2027, Round: 1, Original: "franchise.01", Owner: "franchise.01"},
		{Season: 2027, Round: 1, Original: "franchise.02", Owner: "franchise.02"},
	}
	b.add(RawCommand{League: league, Command: NativeSeed, Args: RawArgs{Native: &seed},
		Request: "seed", Effective: "now", Visibility: "league"}, seasonMoment(2))
	b.add(RawCommand{League: league, Command: DraftOrderSet,
		Args:    RawArgs{Draft: &RawDraftArgs{Season: 2027, Order: []string{"franchise.01", "franchise.02"}}},
		Request: "order", Effective: "now", Visibility: "league"}, seasonMoment(2))
	startAt := seasonMoment(3)
	b.add(RawCommand{League: league, Command: DraftStart,
		Args:    RawArgs{Draft: &RawDraftArgs{Season: 2027, At: startAt}},
		Request: "start", Effective: "now", Visibility: "league"}, seasonMoment(2))

	facts := nativecheck.Facts{
		"1001": {Name: "Rookie One", Position: leaguerules.WR, DraftYearHeld: true, DraftYear: 2027},
		"1002": {Name: "Rookie Two", Position: leaguerules.RB, DraftYearHeld: true, DraftYear: 2027},
	}
	t1 := startAt.Add(11 * time.Second)
	pick2, err := PlanDraftPick(DraftPickInput{League: league, Season: 2027, Franchise: "franchise.02",
		Player: "1001", Request: "pick2", Expected: int64(len(b.records))}, b.records, facts, t1, time.UTC, NoDraftStandings{})
	if err != nil {
		t.Fatal(err)
	}
	owedCheckpoint := b.add(pick2, t1)

	t2 := t1.Add(5 * time.Second)
	makeup, err := PlanDraftPick(DraftPickInput{League: league, Season: 2027, Franchise: "franchise.01",
		Player: "1002", Request: "makeup1", Expected: int64(len(b.records))}, b.records, facts, t2, time.UTC, NoDraftStandings{})
	if err != nil {
		t.Fatal(err)
	}
	makeupCheckpoint := b.add(makeup, t2)

	t3 := t2.Add(5 * time.Second)
	archive := RawCommand{League: league, Command: DraftArchive,
		Args: RawArgs{Draft: &RawDraftArgs{Season: 2027}}, Request: "close", Effective: "now", Visibility: "league"}
	if err := PrepareDraftAppend(&archive, b.records, t3, facts, NoDraftStandings{}); err != nil {
		t.Fatal(err)
	}
	b.add(archive, t3)

	t4 := t3.Add(5 * time.Second)
	revertToOwed := RawCommand{League: league, Command: DraftRevert,
		Args:    RawArgs{Draft: &RawDraftArgs{Season: 2027, Act: &RawDraftAct{To: owedCheckpoint}}},
		Request: "revert.owed", Reason: "Reopen owed pick", Effective: "now", Visibility: "league"}
	if err := CheckDraftAppend(revertToOwed, b.records, t4, NoDraftStandings{}); err != nil {
		t.Fatalf("restore to owed-pick checkpoint refused: %v", err)
	}

	revertToMakeup := RawCommand{League: league, Command: DraftRevert,
		Args:    RawArgs{Draft: &RawDraftArgs{Season: 2027, Act: &RawDraftAct{To: makeupCheckpoint}}},
		Request: "revert.makeup", Reason: "Should refuse", Effective: "now", Visibility: "league"}
	if err := CheckDraftAppend(revertToMakeup, b.records, t4, NoDraftStandings{}); err == nil {
		t.Fatal("restore with no remaining owed pick should still be refused")
	}
}

func TestReplayCacheCorrectionAndReversalOfReversal(t *testing.T) {
	records := []Record{
		{Command: RawCommand{Command: Create, Args: RawArgs{Name: "League"}}, Receipt: Receipt{Entry: "create"}},
		{Command: RawCommand{Command: Revert, Args: RawArgs{Entry: "create"}}, Receipt: Receipt{Entry: "undo"}},
		{Command: RawCommand{Command: CorrectTransaction, Args: RawArgs{Correction: &RawCorrection{Entry: "undo"}}}, Receipt: Receipt{Entry: "correct"}},
		{Command: RawCommand{Command: Revert, Args: RawArgs{Entry: "correct"}}, Receipt: Receipt{Entry: "undo-correction"}},
	}
	for n := 1; n <= len(records); n++ {
		cached := WithReplayCache(records[:n])
		if !reflect.DeepEqual(Replay(cached), replayFull(records[:n])) {
			t.Fatal("correction cache differs")
		}
	}
	cached := WithReplayCache(records[:3])
	cached[2].Command.Args.Correction.Entry = "create"
	if _, ok := cachedReplay(cached); ok {
		t.Fatal("edited correction target hit cache")
	}
	if !reflect.DeepEqual(Replay(cached), replayFull(cached)) {
		t.Fatal("edited correction did not replay")
	}
}
