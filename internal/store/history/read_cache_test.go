package history

import (
	"bytes"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestNativeRecordsCacheTracksCommittedRevision(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	receipt, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.nativeRecords(t.Context(), receipt.League)
	if err != nil {
		t.Fatal(err)
	}
	if s.records[receipt.League].revision != receipt.Revision {
		t.Fatal("decoded ledger was not cached by revision")
	}
	// Cached nested values are immutable internally; callers own the outer slice.
	first[0] = commissioner.Record{}
	s.records[receipt.League].records[0].Receipt.Message = "cached"
	second, err := s.nativeRecords(t.Context(), receipt.League)
	if err != nil || second[0].Receipt.Entry != receipt.Entry || second[0].Receipt.Message != "cached" {
		t.Fatalf("caller changed cached records: %+v, %v", second, err)
	}
	other := New(s.pools, s.reg)
	other.SetClock(s.now)
	reverse, err := other.ExecuteCommissioner(t.Context(), reverseCommand(receipt.Entry, receipt.Revision, "again"), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := s.nativeRecords(t.Context(), receipt.League)
	if err != nil || len(latest) != 2 || latest[1].Receipt.Entry != reverse.Entry {
		t.Fatalf("append left a stale decoded ledger: %+v, %v", latest, err)
	}
}

func TestNativeLeaguesReuseUntilAnEventIsAppended(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	receipt, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.NativeLeagues(t.Context())
	if err != nil || len(first) != 1 {
		t.Fatalf("leagues: %+v, %v", first, err)
	}
	first[0].Name = "caller changed this"
	if s.leagues.names[0].Name == first[0].Name {
		t.Fatal("caller changed the cached league list")
	}
	// A sentinel in the private cache proves an idle clock does not query the list again.
	s.leagues.names[0].Name = "cached"
	second, err := s.NativeLeagues(t.Context())
	if err != nil || second[0].Name != "cached" {
		t.Fatalf("idle read rescanned the event table: %+v, %v", second, err)
	}
	if _, err := s.ExecuteCommissioner(t.Context(), reverseCommand(receipt.Entry, receipt.Revision, "again"), "creator", nil); err != nil {
		t.Fatal(err)
	}
	latest, err := s.NativeLeagues(t.Context())
	if err != nil || latest[0].Name == "cached" || latest[0].Name == first[0].Name {
		t.Fatalf("event did not invalidate league names: %+v, %v", latest, err)
	}
}

func TestRecordCacheDoesNotEscapeTransactionSnapshot(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	receipt, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := s.pools.Read().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	old, err := s.transactionRecords(t.Context(), tx, receipt.League)
	if err != nil || len(old) != 1 {
		t.Fatalf("old snapshot: %+v, %v", old, err)
	}
	if _, err := s.ExecuteCommissioner(t.Context(), reverseCommand(receipt.Entry, receipt.Revision, "again"), "creator", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.nativeRecords(t.Context(), receipt.League); err != nil {
		t.Fatal(err)
	}
	old, err = s.transactionRecords(t.Context(), tx, receipt.League)
	if err != nil || len(old) != 1 {
		t.Fatalf("cache escaped the transaction's snapshot: %+v, %v", old, err)
	}
}

// Callers own every byte they read: writing through a returned record's nested data must not
// change the cached ledger the next read returns.
func TestRecordCacheReturnsDeepCopies(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	create := createCommand()
	create.Args.Preset = commissioner.PresetLegacy
	receipt, err := s.ExecuteCommissioner(t.Context(), create, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	read := func() []commissioner.Record {
		t.Helper()
		tx, err := s.pools.Read().BeginTx(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		records, err := s.transactionRecords(t.Context(), tx, receipt.League)
		if err != nil || len(records) != 1 || len(records[0].Command.Args.Seed) == 0 {
			t.Fatalf("read: %+v, %v", records, err)
		}
		return records
	}
	for _, first := range [][]commissioner.Record{read(), read()} { // a miss, then a hit
		want := first[0].Command.Args.Seed[0]
		want.Value = slices.Clone(want.Value)
		first[0].Command.Args.Seed[0].Field = "changed"
		first[0].Command.Args.Seed[0].Value[0] = '!'
		if got := read()[0].Command.Args.Seed[0]; got.Field != want.Field || !bytes.Equal(got.Value, want.Value) {
			t.Fatalf("a caller's write reached the cache: %s %s, want %s %s", got.Field, got.Value, want.Field, want.Value)
		}
	}
}

func TestDeepCloneSharesNothing(t *testing.T) {
	t.Parallel()
	type inner struct{ Tags []string }
	type value struct {
		At    time.Time
		Ptr   *inner
		Map   map[string][]int
		Any   any
		Fixed [2]*inner
	}
	src := value{At: time.Unix(5, 0), Ptr: &inner{Tags: []string{"a"}}, Map: map[string][]int{"k": {1}},
		Any: &inner{Tags: []string{"b"}}, Fixed: [2]*inner{{Tags: []string{"c"}}}}
	out := deepClone(src)
	out.Ptr.Tags[0], out.Map["k"][0], out.Fixed[0].Tags[0] = "x", 9, "x"
	out.Any.(*inner).Tags[0] = "x"                                                          //nolint:forcetypeassert // the test built it
	if src.Ptr.Tags[0] != "a" || src.Map["k"][0] != 1 || src.Any.(*inner).Tags[0] != "b" || //nolint:forcetypeassert // the test built it
		src.Fixed[0].Tags[0] != "c" || !out.At.Equal(src.At) || out.Fixed[1] != nil {
		t.Fatalf("clone shared state with its source: %+v", src)
	}
}
