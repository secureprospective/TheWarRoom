package commissioner

import (
	"slices"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/deepclone"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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
		thresholds, public, ok = memo.epochs.load(records)
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

// load works out the held revision's thresholds once; records must be the revision the memo belongs to.
func (e *readingEpochs) load(records []Record) (all, public []time.Time, ok bool) {
	e.once.Do(func() { e.thresholds, e.public, e.timed = epochThresholds(records) })
	return e.thresholds, e.public, e.timed
}

// rulesCache holds the rules in force for one held revision, by season and reading epoch. Every instant
// RulesInForce compares the reading time with (receipts, effective times, season starts) is an epoch
// threshold, so two instants with the same counts read the same rules (TestRulesCacheMatchesTheFold).
// Both counts are kept, at-or-before and strictly-before, so a comparison of either strictness agrees.
type rulesCache struct {
	mu    sync.Mutex
	views map[rulesCacheKey]rulesCacheEntry
}

type rulesCacheKey struct {
	principals      string
	books           *closedBooks
	season          int
	reached, before int
}

type rulesCacheEntry struct {
	rules leaguerules.RuleSet
	err   error
}

// maxRulesViews bounds the rule sets held per revision: a few seasons, each read near the clock.
const maxRulesViews = 16

// RulesInForce is the rules a season held at an instant (rulesInForce), from the held revision's cache when
// the records carry one. Each caller gets its own deep copy, so no caller can write into the cache.
func RulesInForce(records []Record, season int, at time.Time) (leaguerules.RuleSet, error) {
	memo := memoFor(records)
	if memo == nil {
		return rulesInForce(records, season, at)
	}
	thresholds, _, ok := memo.epochs.load(records)
	if !ok {
		return rulesInForce(records, season, at)
	}
	reachedAt, _ := reached(thresholds, at)
	before, _ := slices.BinarySearchFunc(thresholds, at, time.Time.Compare)
	key := rulesCacheKey{principals: records[0].principals.Key(), books: records[0].books, season: season,
		reached: reachedAt, before: before}
	memo.rules.mu.Lock()
	hit, seen := memo.rules.views[key]
	memo.rules.mu.Unlock()
	if seen {
		return deepclone.Of(hit.rules), hit.err
	}
	// Resolved outside the lock: the resolver may read other caches of this revision. Two callers racing
	// on one key resolve the same answer twice, which is harmless.
	rules, err := rulesInForce(records, season, at)
	memo.rules.mu.Lock()
	if memo.rules.views == nil || len(memo.rules.views) >= maxRulesViews {
		memo.rules.views = map[rulesCacheKey]rulesCacheEntry{}
	}
	memo.rules.views[key] = rulesCacheEntry{rules: rules, err: err}
	memo.rules.mu.Unlock()
	// A copy even when fresh: a recorded rollover's rules are the held ledger entry's own maps.
	return deepclone.Of(rules), err
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
