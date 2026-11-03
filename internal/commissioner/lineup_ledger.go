package commissioner

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// LineupLedger owns derived inputs for one immutable ledger revision. Its source's
// nested payloads must remain read-only, just like the store's shared records.
// No base, rule map or asset slice is exposed through a reading.
type LineupLedger struct {
	records    []Record
	latest     time.Time
	boundaries []time.Time
	err        error
	mu         sync.Mutex
	bases      map[lineupBaseKey]NativeBase
}

type lineupBaseKey struct {
	interval   int
	principals string
}

// NewLineupLedger binds the cache lifetime to the store's bounded held revision.
func NewLineupLedger(records []Record) *LineupLedger {
	l := &LineupLedger{records: records, latest: ReadingInstant(records, time.Time{})}
	l.boundaries, l.err = RolloverInstants(records)
	for _, r := range records {
		at, err := nativeEntryTime(r)
		if err != nil {
			l.err = err
			break
		}
		l.boundaries = append(l.boundaries, at)
		if c := r.Command.Args.Correction; c != nil && c.Until != "" {
			until, err := time.Parse(time.RFC3339Nano, c.Until)
			if err == nil {
				l.boundaries = append(l.boundaries, until)
			}
		}
	}
	sort.Slice(l.boundaries, func(i, j int) bool { return l.boundaries[i].Before(l.boundaries[j]) })
	return l
}

// Read shares ledger folds, not authorization or time-sensitive football facts.
func (l *LineupLedger) Read(ctx context.Context, league, franchise, principal string, week int,
	at time.Time, env LineupFacts) (NativeLineup, error) {
	if week < 1 || week > 22 {
		return NativeLineup{}, fmt.Errorf("choose a week")
	}
	if l.err != nil {
		return NativeLineup{}, l.err
	}
	records, err := ResolvePrincipals(ctx, league, l.records)
	if err != nil {
		return NativeLineup{}, err
	}
	instant := at
	if instant.Before(l.latest) {
		instant = l.latest
	}
	key := lineupBaseKey{interval: sort.Search(len(l.boundaries), func(i int) bool {
		return l.boundaries[i].After(instant)
	})}
	if len(records) > 0 {
		key.principals = records[0].principals.Key()
	}
	base, err := l.base(league, records, key, instant)
	if err != nil {
		return NativeLineup{}, err
	}
	return nativeLineupFromBase(league, franchise, principal, week, base, instant, env)
}

func (l *LineupLedger) base(league string, records []Record, key lineupBaseKey, at time.Time) (NativeBase, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if hit, ok := l.bases[key]; ok {
		return hit, nil
	}
	base, err := ReadNativeBase(league, records, at)
	if err != nil {
		return NativeBase{}, err
	}
	if l.bases == nil {
		l.bases = map[lineupBaseKey]NativeBase{}
	}
	if len(l.bases) >= maxGovernanceViews {
		clear(l.bases)
	}
	l.bases[key] = base
	return base, nil
}
