package history

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestReplayCacheCorrectionsRolloverRestart(t *testing.T) {
	r := newRolloverDB(t)
	check := func() {
		t.Helper()
		before := fingerprint(t, r.s)
		records, err := r.s.nativeRecords(t.Context(), "native.test")
		if err != nil {
			t.Fatal(err)
		}
		warm, err := r.s.nativeRecords(t.Context(), "native.test")
		if err != nil {
			t.Fatal(err)
		}
		events, err := r.s.Events(t.Context(), "native.test", commissioner.Stream, 0)
		if err != nil {
			t.Fatal(err)
		}
		full, err := commissionerRecords(events)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(commissioner.Replay(records), commissioner.Replay(full)) ||
			!reflect.DeepEqual(commissioner.Replay(warm), commissioner.Replay(full)) {
			t.Fatal("cold/warm projection differs from authoritative replay")
		}
		if fingerprint(t, r.s) != before {
			t.Fatal("read persisted a snapshot")
		}
	}
	check()
	rule := r.rule(t, "cap.amount", "25000000000000", "now")
	check()
	r.do(t, reverseCommand(rule.Entry, r.version(t), "undo-rule"))
	check()
	r.do(t, r.rollover(t, 2027, false, "roll"))
	check()
	r.reopenFile(t)
	check()
}

func TestReplayCacheRetentionBound(t *testing.T) {
	s, _, _ := newStore(t)
	for i := range maxCachedLeagues + 1 {
		league := fmt.Sprintf("native.%d", i)
		s.cacheRecords(league, 1, []commissioner.Record{{Receipt: commissioner.Receipt{Entry: league}}})
	}
	if len(s.records) > maxCachedLeagues {
		t.Fatal("league bound exceeded")
	}
	records := make([]commissioner.Record, maxCachedRecords)
	s.cacheRecords("native.large", int64(len(records)), records)
	if len(s.records) != 1 {
		t.Fatal("record budget did not evict older projections")
	}
	s.cacheRecords("native.oversize", maxCachedRecords+1, make([]commissioner.Record, maxCachedRecords+1))
	if _, ok := s.records["native.oversize"]; ok {
		t.Fatal("oversized revision retained")
	}
}
