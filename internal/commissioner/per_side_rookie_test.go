package commissioner

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func sideRookieLeague(t *testing.T) (*ledgerBuilder, nativecheck.Facts, time.Time) {
	t.Helper()
	b := mirrorLeague(t)
	ids := LeagueFranchises(b.records)
	seed := b.records[1].Command.Args.Native
	for _, id := range ids {
		owner := id
		if id == ids[0] {
			owner = ids[8]
		}
		if id == ids[8] {
			owner = ids[0]
		}
		seed.Picks = append(seed.Picks, NativePick{Season: 2027, Round: 1, Original: id, Owner: owner})
	}
	b.set("calendar.windows", `{}`, "now", seasonMoment(2))
	b.set("calendar.boundary", `"start_of_day"`, "now", seasonMoment(2))
	b.set("draft.rounds", `1`, "now", seasonMoment(2))
	b.set("roster.limits", `{"min":0,"max":100}`, "now", seasonMoment(2))
	facts := nativecheck.Facts{}
	for i := range 9 {
		facts[fmt.Sprintf("%05d", 30000+i)] = nativecheck.PlayerFacts{Name: fmt.Sprintf("Rookie %d", i),
			Position: leaguerules.WR, DraftYear: 2027, DraftYearHeld: true}
	}
	at := seasonMoment(3)
	b.record(RawCommand{League: "native.test", Command: DraftStart, Request: "start",
		Args:      RawArgs{Draft: &RawDraftArgs{Season: 2027, At: at.Add(time.Minute)}},
		Effective: "now", Visibility: "league"}, facts, at)
	b.record(RawCommand{League: "native.test", Command: DraftOrderSet, Request: "order",
		Args: RawArgs{Draft: &RawDraftArgs{Season: 2027, Order: ids}}, Effective: "now", Visibility: "league"}, facts, at)
	return b, facts, at.Add(time.Minute)
}

func sideRookieReading(t *testing.T, b *ledgerBuilder, facts nativecheck.Facts,
	at time.Time, franchise string) NativeDraft {
	t.Helper()
	d, err := NativeDraftReading(RawNativeDraftScope{League: "native.test", Season: 2027,
		Franchise: franchise, At: at}, b.records, time.UTC, nil, facts)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestPerSideRookieDraftCopiesTradesCloseAndRestore(t *testing.T) {
	b, facts, at := sideRookieLeague(t)
	for i := range 8 {
		for _, team := range []string{"franchise.01", "franchise.09"} {
			d := sideRookieReading(t, b, facts, at, team)
			wantSide := "AFC"
			if team == "franchise.09" {
				wantSide = "NFC"
			}
			if d.Side != wantSide || len(d.Order) != 1 || len(d.Order[0].Picks) != 8 {
				t.Fatalf("%s draft: side %q order %+v words %q", team, d.Side, d.Order, d.Words)
			}
			player := fmt.Sprintf("%05d", 30000+i)
			if !slices.ContainsFunc(d.Available, func(p DraftPlayer) bool { return p.ID == player }) {
				t.Fatalf("%s lost the other side's rookie %s", d.Side, player)
			}
			raw, err := PlanDraftPick(DraftPickInput{League: "native.test", Season: 2027,
				Franchise: d.Draft.OnClock.Owner, Player: player, Request: fmt.Sprintf("pick.%s.%d", d.Side, i),
				Expected: int64(len(b.records))}, b.records, facts, at, time.UTC, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := raw.Args.Validate(DraftPick); err != nil {
				t.Fatal(err)
			}
			b.add(raw, at)
		}
	}
	due := DueDrafts(b.records, at, nil)
	if len(due) != 2 {
		t.Fatalf("due: %+v", due)
	}
	for _, key := range due {
		b.record(RawCommand{League: "native.test", Command: DraftArchive, Request: "close." + key.Side,
			Args:      RawArgs{Draft: &RawDraftArgs{Season: key.Season, Side: key.Side}},
			Effective: "now", Visibility: "league"}, facts, at)
	}
	assets, err := NativeAssetsReading("native.test", b.records)
	if err != nil {
		t.Fatal(err)
	}
	for _, side := range []string{"AFC", "NFC"} {
		if assets.Snapshot.PoolOn(side, "30008") < 0 {
			t.Fatalf("%s missing undrafted rookie", side)
		}
		if !assets.Snapshot.OnSide(side, "30000") {
			t.Fatalf("%s missing drafted rookie", side)
		}
	}
	// Save and reload the entire ledger, including both drafts and their close outcomes.
	body, err := json.Marshal(b.records)
	if err != nil {
		t.Fatal(err)
	}
	var restored []Record
	if err := json.Unmarshal(body, &restored); err != nil {
		t.Fatal(err)
	}
	if digest(t, b.records) != digest(t, restored) {
		t.Fatal("ledger changed on reload")
	}
	b.records = restored
	for _, team := range []string{"franchise.01", "franchise.09"} {
		d := sideRookieReading(t, b, facts, at, team)
		if d.Closed == nil || len(d.Closed.Pool) != 1 {
			t.Fatalf("reloaded close: %+v", d.Closed)
		}
		checkpoint := ""
		for _, c := range d.Checkpoints {
			for _, r := range b.records {
				if r.Receipt.Entry == c.Entry && r.Command.Command == DraftPick {
					checkpoint = c.Entry
				}
			}
		}
		b.record(RawCommand{League: "native.test", Command: DraftRevert, Request: "restore." + d.Side,
			Args:      RawArgs{Draft: &RawDraftArgs{Season: 2027, Side: d.Side, Act: &RawDraftAct{To: checkpoint}}},
			Effective: "now", Visibility: "league"}, facts, at)
		reopened := sideRookieReading(t, b, facts, at, team)
		if reopened.Closed != nil || reopened.Draft.Exhausted {
			t.Fatal("restore did not reopen side")
		}
	}
}

func TestPerSideRookieDraftOrdersEachRoundsTradedPicks(t *testing.T) {
	b, _, at := sideRookieLeague(t)
	b.set("draft.rounds", `2`, "now", seasonMoment(2))
	b.set("draft.laterPattern", `"linear"`, "now", seasonMoment(2))
	seed := b.records[1].Command.Args.Native
	for _, id := range LeagueFranchises(b.records) {
		seed.Picks = append(seed.Picks, NativePick{Season: 2027, Round: 2, Original: id, Owner: id})
	}
	basis, err := readDraft(b.records, DraftKey{Season: 2027, Side: "AFC"}, at, nil)
	if err != nil || basis.words != "" {
		t.Fatalf("different originals across rounds: %v %q", err, basis.words)
	}
	if len(basis.order) != 2 || basis.order[0].Picks[7].Original != "franchise.09" ||
		basis.order[1].Picks[0].Original != "franchise.01" {
		t.Fatalf("traded round order: %+v", basis.order)
	}
}
