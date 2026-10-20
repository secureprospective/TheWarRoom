package history

import (
	"testing"

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
