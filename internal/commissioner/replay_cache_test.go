package commissioner

import (
	"reflect"
	"sync"
	"testing"
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
