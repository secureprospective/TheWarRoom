package commissioner

import (
	"fmt"
	"testing"
	"time"
)

func TestRolloverIndexMatchesEveryPrefix(t *testing.T) {
	t.Parallel()
	b := newLedger(t, PresetLegacy)
	for i := range 300 {
		at := seasonMoment(2).Add(time.Duration(i) * time.Hour)
		if i%4 == 0 {
			b.set(rolloverField, fmt.Sprintf("%q", at.Add(48*time.Hour).Format(time.RFC3339)), "now", at)
		} else {
			b.revert(b.records[(i*i+7*i+11)%len(b.records)].Receipt.Entry, at)
		}
	}
	b.set(rolloverField, rolloverAt, "now", seasonMoment(20))
	b.records[len(b.records)-1].Receipt.Revision = 0
	index := newRolloverIndex()
	for i, r := range b.records {
		got, err := index.add(r)
		if err != nil {
			t.Fatal(err)
		}
		active := Replay(b.records[:i+1]).Active
		var want time.Time
		for j := i; j >= 0; j-- {
			candidate := b.records[j]
			if candidate.Command.Command == RulesSet && candidate.Command.Args.Field == rolloverField && candidate.Command.Effective == "now" && active[candidate.Receipt.Entry] {
				want, err = rolloverDate(candidate)
				if err != nil {
					t.Fatal(err)
				}
				break
			}
		}
		if !got.Equal(want) {
			t.Fatalf("prefix %d: got %v want %v", i+1, got, want)
		}
	}
}

func TestReversalChainLedgerReadingBudget(t *testing.T) {
	b := newLedger(t, PresetEmpty)
	for range 1000 {
		b.revert(b.records[len(b.records)-1].Receipt.Entry, seasonMoment(2))
	}
	started := time.Now()
	reading, err := Reading("native.test", "creator", b.records, CreatorAuthorizer{Creator: "creator"})
	if err != nil || len(reading.Entries) != len(b.records) {
		t.Fatalf("ledger: %d entries, %v", len(reading.Entries), err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("1,001-event ledger exceeded two seconds: %v", elapsed)
	}
	for i, entry := range reading.Entries[:1000] {
		if entry.What != "Reversed a recorded league change" {
			t.Fatalf("entry %d changed its label: %q", i, entry.What)
		}
	}
}
