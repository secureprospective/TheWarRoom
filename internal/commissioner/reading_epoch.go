package commissioner

import (
	"slices"
	"sync"
	"time"
)

// readingEpochs lists, once per held revision, every instant at which a league-state reading can change
// with no new ledger entry: each record's receipt and effective time, each correction or reserve
// exception's expiry, and each season start. Extra instants only split epochs finer.
type readingEpochs struct {
	once       sync.Once
	thresholds []time.Time
	public     []time.Time // the future-facing instants of entries every member may see
	timed      bool
}

// ReadingEpoch names the stretch of time now falls in: two instants with the same epoch read the same
// league state from the same records and evidence (TestReadingEpochHoldsEveryReading). evidence is the
// sorted instants at which the reading's other inputs change on their own (football kickoffs). next is the
// instant the next public deadline passes, zero when none is later: it is built only from entries every
// member may see, so waking on it never reveals a private entry. ok is false when a recorded time does not parse; such a
// league is never shared.
func ReadingEpoch(records []Record, now time.Time, evidence []time.Time) (epoch int, next time.Time, ok bool) {
	if len(records) == 0 {
		return 0, time.Time{}, false
	}
	var thresholds, public []time.Time
	if memo := memoFor(records); memo != nil {
		memo.epochs.once.Do(func() {
			memo.epochs.thresholds, memo.epochs.public, memo.epochs.timed = epochThresholds(records)
		})
		thresholds, public, ok = memo.epochs.thresholds, memo.epochs.public, memo.epochs.timed
	} else {
		thresholds, public, ok = epochThresholds(records)
	}
	if !ok {
		return 0, time.Time{}, false
	}
	// The raw clock, not ReadingInstant: some readings compare the clock itself with recorded times.
	// Each count only grows with the clock, so equal sums mean equal counts.
	recorded, _ := reached(thresholds, now)
	_, publicNext := reached(public, now)
	seen, evidenceNext := reached(evidence, now)
	next = publicNext
	if next.IsZero() || (!evidenceNext.IsZero() && evidenceNext.Before(next)) {
		next = evidenceNext
	}
	return recorded + seen, next, true
}

// reached counts the sorted instants at or before now and names the first one after it.
func reached(instants []time.Time, now time.Time) (int, time.Time) {
	n, _ := slices.BinarySearchFunc(instants, now, func(t, at time.Time) int {
		if t.After(at) {
			return 1
		}
		return -1
	})
	if n < len(instants) {
		return n, instants[n]
	}
	return n, time.Time{}
}

func epochThresholds(records []Record) (all, public []time.Time, ok bool) {
	add := func(open bool, value string) bool {
		if value == "" {
			return true
		}
		at, err := time.Parse(time.RFC3339Nano, value)
		all = append(all, at)
		if open {
			public = append(public, at)
		}
		return err == nil
	}
	for _, r := range records {
		effective, err := r.Command.EffectiveTime(r.Receipt.RecordedAt)
		if err != nil {
			return nil, nil, false
		}
		// The ledger's own privacy rules (ledgerEntryVisible): a private entry, bid or waiver claim is never public.
		open := r.Command.Visibility != "private" && !privateBidRecord(r, records) && !privateWaiverRecord(r, records)
		all = append(all, r.Receipt.RecordedAt, effective)
		if open {
			public = append(public, effective)
		}
		if c := r.Command.Args.Correction; c != nil && !add(open, c.Until) {
			return nil, nil, false
		}
		if o := r.Command.Args.Override; o != nil && !add(open, o.Until) {
			return nil, nil, false
		}
	}
	starts, err := RolloverInstants(records)
	if err != nil {
		return nil, nil, false
	}
	all, public = append(all, starts...), append(public, starts...)
	for _, list := range []*[]time.Time{&all, &public} {
		slices.SortFunc(*list, time.Time.Compare)
		*list = slices.CompactFunc(*list, time.Time.Equal)
	}
	return all, public, true
}
