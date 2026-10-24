package commissioner

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func TestClosedBooksResumeAndCompensationFallback(t *testing.T) {
	b := rolloverLedger(t)
	before := b.add(RawCommand{Command: "commish.role.grant", Args: RawArgs{People: &RawPeople{Principal: "dot", Role: "DOT"}}}, seasonMoment(5))
	b.rollover(2027, rolloverMoment())
	raw, err := CloseBooks(b.records)
	if err != nil {
		t.Fatal(err)
	}
	if err := raw.Validate(); err != nil {
		t.Fatal(err)
	}
	b.set("cap.amount", `123456789000`, "now", rolloverMoment().Add(time.Hour))
	full := slices.Clone(b.records)
	folded, ok := WithClosedBooks(full, raw)
	if !ok || usableClosedBooks(folded) == nil {
		t.Fatal("valid summary was not used")
	}
	assertClosedBooksCore(t, folded, full)
	// Current-season reversals can replay locally.
	b.revert(b.records[len(b.records)-1].Receipt.Entry, rolloverMoment().Add(2*time.Hour))
	folded, ok = WithClosedBooks(b.records, raw)
	if !ok || usableClosedBooks(folded) == nil {
		t.Fatal("current-season reversal unnecessarily discarded checkpoint")
	}
	assertClosedBooksCore(t, folded, b.records)
	// People changes may be reversed across seasons. Never trust a season boundary alone.
	b.revert(before, rolloverMoment().Add(3*time.Hour))
	folded, ok = WithClosedBooks(b.records, raw)
	if !ok || usableClosedBooks(folded) != nil {
		t.Fatal("cross-season compensation did not select full replay")
	}
	assertClosedBooksCore(t, folded, b.records)
	b.revert(b.records[len(b.records)-1].Receipt.Entry, rolloverMoment().Add(4*time.Hour))
	folded, _ = WithClosedBooks(b.records, raw)
	assertClosedBooksCore(t, folded, b.records)
}

func assertClosedBooksCore(t *testing.T, folded, full []Record) {
	t.Helper()
	if !reflect.DeepEqual(Replay(folded), Replay(full)) {
		t.Fatal("projection differs from full replay")
	}
	got, gotErr := GovernanceState(folded)
	want, wantErr := GovernanceState(full)
	if !reflect.DeepEqual(got, want) || (gotErr != nil) != (wantErr != nil) {
		t.Fatalf("people differs: %v / %v", gotErr, wantErr)
	}
}

func TestClosedBooksRejectsChangedPrefixAndHistoricalSlices(t *testing.T) {
	b := rolloverLedger(t)
	b.rollover(2027, rolloverMoment())
	raw, err := CloseBooks(b.records)
	if err != nil {
		t.Fatal(err)
	}
	changed := slices.Clone(b.records)
	changed[0].Command.Args.Name = "different source"
	if _, ok := WithClosedBooks(changed, raw); ok {
		t.Fatal("mismatched source accepted")
	}
	folded, ok := WithClosedBooks(b.records, raw)
	if !ok {
		t.Fatal("valid summary rejected")
	}
	if usableClosedBooks(folded[:len(folded)-1]) != nil {
		t.Fatal("historical prefix used a future checkpoint")
	}
	folded[0].Principal = "different creator"
	if usableClosedBooks(folded) != nil {
		t.Fatal("edited caller copy used stale governance")
	}
	assertClosedBooksCore(t, folded, changedCreator(b.records))
}

func changedCreator(records []Record) []Record {
	out := slices.Clone(records)
	out[0].Principal = "different creator"
	return out
}

func TestClosedBooksResumedErrorsAndOwnership(t *testing.T) {
	b := rolloverLedger(t)
	b.rollover(2027, rolloverMoment())
	raw, err := CloseBooks(b.records)
	if err != nil {
		t.Fatal(err)
	}
	b.add(RawCommand{Command: "commish.seat.remove", Args: RawArgs{People: &RawPeople{Franchise: "absent"}}}, rolloverMoment().Add(time.Hour))
	folded, ok := WithClosedBooks(b.records, raw)
	if !ok {
		t.Fatal("valid summary rejected")
	}
	_, got := GovernanceState(folded)
	_, want := GovernanceState(b.records)
	if got == nil || want == nil || got.Error() != want.Error() {
		t.Fatalf("error differs: %v / %v", got, want)
	}
	raw.Projection.Active[raw.Entry] = false
	raw.People.Roles["creator"]["commissioner"] = false
	if !Replay(folded).Active[raw.Entry] {
		t.Fatal("caller mutated attached checkpoint")
	}
	if !folded[0].books.raw.People.Roles["creator"]["commissioner"] {
		t.Fatal("caller mutated attached people")
	}
	projection := Replay(folded)
	projection.Active[raw.Entry] = false
	if !Replay(folded).Active[raw.Entry] {
		t.Fatal("projection exposed immutable checkpoint map")
	}
}

func TestClosedNativeStartsAtBoundaryAndRejectsEditedBodies(t *testing.T) {
	b := rolloverLedger(t)
	b.records[1].Command.Args.Native.Conferences = map[string]string{"div.0": "Conference"}
	b.rollover(2027, rolloverMoment())
	raw, err := CloseBooks(b.records)
	if err != nil {
		t.Fatal(err)
	}
	folded, ok := WithClosedBooks(b.records, raw)
	if !ok {
		t.Fatal("valid summary rejected")
	}
	var assets NativeAssets
	var waivers waiverState
	since, err := resumeClosedNative(&assets, folded, &waivers)
	if err != nil || since != raw.Revision {
		t.Fatalf("native resume at %d, want %d: %v", since, raw.Revision, err)
	}
	assets.Snapshot.Players[0].Salary++
	assets.Snapshot.Conferences["div.0"] = "Changed"
	since, err = resumeClosedNative(&assets, folded, &waivers)
	if err != nil || since != raw.Revision || assets.Snapshot.Conferences["div.0"] != "Conference" {
		t.Fatal("native reading mutated shared checkpoint")
	}
	// Changing a nested command body does not change its stored canonical text.
	// Native resume must bind the actual payload, not just that canonical text.
	seed := *folded[1].Command.Args.Native
	seed.Players = slices.Clone(seed.Players)
	seed.Players[0].Salary++
	folded[1].Command.Args.Native = &seed
	since, err = resumeClosedNative(&assets, folded, &waivers)
	if err != nil || since != 0 {
		t.Fatal("edited native source used a stale checkpoint")
	}
}

func TestClosedNativeUnhashableSourceFallsBack(t *testing.T) {
	b := rolloverLedger(t)
	b.rollover(2027, rolloverMoment())
	raw, err := CloseBooks(b.records)
	if err != nil {
		t.Fatal(err)
	}
	folded, ok := WithClosedBooks(b.records, raw)
	if !ok {
		t.Fatal("valid summary rejected")
	}
	folded[2].Command.Args.Value = []byte("{")
	full := WithoutClosedBooks(folded)
	got, gotErr := NativeAssetsReading(rolloverLeague, folded)
	want, wantErr := NativeAssetsReading(rolloverLeague, full)
	if !reflect.DeepEqual(got, want) || (gotErr != nil) != (wantErr != nil) {
		t.Fatalf("edited source differs from full replay: %v / %v", gotErr, wantErr)
	}
	var assets NativeAssets
	var waivers waiverState
	since, err := resumeClosedNative(&assets, folded, &waivers)
	if err != nil || since != 0 {
		t.Fatalf("unhashable source did not select full replay: %d %v", since, err)
	}
}

func TestClosedBooksDeputyOrderOwnership(t *testing.T) {
	b := rolloverLedger(t)
	for _, p := range []string{"A", "B", "C"} {
		b.add(RawCommand{Command: roleGrant, Args: RawArgs{People: &RawPeople{Principal: p, Role: "co-commissioner"}}}, seasonMoment(5))
	}
	b.add(RawCommand{Command: DeputyOrder, Args: RawArgs{People: &RawPeople{Order: []governance.Principal{"C", "A", "B"}}}}, seasonMoment(5))
	orderIndex := len(b.records) - 1
	b.rollover(2027, rolloverMoment())
	raw, err := CloseBooks(b.records)
	if err != nil {
		t.Fatal(err)
	}
	folded, ok := WithClosedBooks(b.records, raw)
	if !ok || usableClosedBooks(folded) == nil {
		t.Fatal("order checkpoint refused")
	}
	assertClosedBooksCore(t, folded, b.records)
	folded[orderIndex].Command.Args.People.Order[0] = "B"
	folded[orderIndex].Command.Args.People.Order[2] = "C"
	if usableClosedBooks(folded) != nil {
		t.Fatal("changed order reused a stale checkpoint")
	}
	assertClosedBooksCore(t, folded, WithoutClosedBooks(folded))
}
