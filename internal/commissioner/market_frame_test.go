package commissioner

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

func TestMarketFrame(t *testing.T) {
	fixture := newMarketFixture(t)
	records := fixture.b.records
	f, err := NewFrame(rolloverLeague, records, fixture.now, fixture.facts, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	assets, book, err := nativeAssetsState(rolloverLeague, records)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.Assets, assets) || !reflect.DeepEqual(f.Book, book) {
		t.Fatal("frame differs from native fold")
	}
	if err := f.ReadRules(); err != nil {
		t.Fatal(err)
	}
	if err := f.ReadGovernance(); err != nil {
		t.Fatal(err)
	}
	rules, err := RulesNow(records, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	govern, err := GovernanceState(records)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.Rules, rules) || !reflect.DeepEqual(f.Govern, govern) {
		t.Fatal("frame differs from rules or governance")
	}
	if err := f.Seeded("missing"); err != nil {
		t.Fatal(err)
	}
	f.Assets.Seeded = false
	if err := f.Seeded("missing"); err == nil || err.Error() != "missing" {
		t.Fatalf("seed refusal: %v", err)
	}
	scope := ExceptionScope{Franchise: "franchise.01"}
	if !reflect.DeepEqual(f.Exceptions(scope), scopedDecisionExceptions(records, scope, fixture.now)) {
		t.Fatal("exceptions differ")
	}
	raw := RawCommand{League: rolloverLeague, Command: "unrelated", Request: "frame-test"}
	before := slices.Clone(records)
	got, gotErr := f.Trial(raw, "principal")
	trial := append(slices.Clone(records), Record{Command: raw, Principal: "principal", Receipt: Receipt{Entry: "proposed.frame-test", RecordedAt: fixture.now}})
	want, wantErr := NativeAssetsReading(rolloverLeague, trial)
	if !reflect.DeepEqual(got, want) || (gotErr == nil) != (wantErr == nil) {
		t.Fatalf("trial differs: %v / %v", gotErr, wantErr)
	}
	if !reflect.DeepEqual(records, before) {
		t.Fatal("trial changed records")
	}
}

func TestMarketDueWithoutSeed(t *testing.T) {
	records := legacyLeague(t)
	if got, err := DueBids("native.test", records, noChangeTime(), nil); err != nil || len(got) != 0 {
		t.Fatalf("bids: %v %v", got, err)
	}
	if got, err := DueWaiverWindows("native.test", records, noChangeTime()); err != nil || len(got) != 0 {
		t.Fatalf("waivers: %v %v", got, err)
	}
	if got, err := NextDueDeadline("native.test", records, noChangeTime()); err != nil || !got.IsZero() {
		t.Fatalf("deadline: %v %v", got, err)
	}
}

func TestFrameDefersRulesAndPreservesEarlyReturns(t *testing.T) {
	records := legacyLeague(t)
	records = append(records, Record{Command: RawCommand{League: "native.test", Command: RulesSet,
		Args: RawArgs{Field: "cap.amount", Value: []byte(`"invalid"`)}, Effective: "now"},
		Receipt: Receipt{Entry: "bad-rule", RecordedAt: noChangeTime()}})
	f, err := NewFrame("native.test", records, noChangeTime(), nil, time.UTC)
	if err != nil {
		t.Fatalf("fold loaded rules: %v", err)
	}
	if err := f.ReadRules(); err == nil {
		t.Fatal("invalid rules accepted")
	}
	if got, err := dueBidsFrame(f); err != nil || len(got) != 0 {
		t.Fatalf("unseeded bids loaded rules: %v %v", got, err)
	}
	if got, err := dueWaiverWindowsFrame(f); err != nil || len(got) != 0 {
		t.Fatalf("unseeded waivers loaded rules: %v %v", got, err)
	}
	if got, err := nextDueDeadlineFrame(f); err != nil || !got.IsZero() {
		t.Fatalf("unseeded deadline loaded rules: %v %v", got, err)
	}
}

func TestFrameCorrectionTime(t *testing.T) {
	fixture := newMarketFixture(t)
	records := fixture.b.records
	at := fixture.now.Add(24 * time.Hour)
	f, err := NewFrame(rolloverLeague, records, at, fixture.facts, time.UTC, at)
	if err != nil {
		t.Fatal(err)
	}
	assets, book, err := nativeAssetsState(rolloverLeague, records, at)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.Assets, assets) || !reflect.DeepEqual(f.Book, book) {
		t.Fatal("requested correction time differs from native fold")
	}
}
