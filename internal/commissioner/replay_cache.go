package commissioner

import (
	"maps"
	"slices"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// replayKey includes every input to compensation replay, including draft restores.
// It deliberately excludes payloads read by the other domain folds.
type replayKey struct {
	entry, command, principal, name, target, correction, restore string
	revision                                                     int64
	season                                                       int
	startup                                                      bool
}

func replayKeyFor(r Record) replayKey {
	k := replayKey{entry: r.Receipt.Entry, revision: r.Receipt.Revision, command: r.Command.Command,
		principal: r.Principal, name: r.Command.Args.Name, target: r.Command.Args.Entry}
	if c := r.Command.Args.Correction; c != nil {
		k.correction = c.Entry
	}
	if d := r.Command.Args.Draft; d != nil {
		k.season, k.startup = d.Season, d.Startup
		if d.Act != nil {
			k.restore = d.Act.To
		}
	}
	return k
}

type replayMemo struct {
	keys       []replayKey
	projection Projection
}

// governanceCache holds people-state folds for one held revision. It is separate from replayMemo so it
// survives WithoutClosedBooks (verified identities strip the projection memo); keys and canonical
// requests revalidate it on every use.
type governanceCache struct {
	keys      []replayKey
	canonical []string

	mu    sync.Mutex
	views map[governanceMemoKey]governanceMemo

	assets assetsCache
	epochs readingEpochs
}

// governanceMemoKey separates policy views: verified identities and closed books change the fold.
type governanceMemoKey struct {
	principals string
	books      *closedBooks
}

// maxGovernanceViews bounds the policy views held per revision; a held revision rarely sees more than two.
const maxGovernanceViews = 8

type governanceMemo struct {
	state governance.State
	err   error
}

// WithReplayCache attaches a memory-only projection to an independently owned outer slice.
// The store bounds retained revisions; edited, truncated or extended copies replay normally.
func WithReplayCache(records []Record) []Record {
	out := slices.Clone(records)
	if len(out) == 0 {
		return out
	}
	out[0].replay = nil
	out[0].govern = nil
	memo := &replayMemo{keys: make([]replayKey, len(out)), projection: Replay(out)}
	gov := &governanceCache{keys: memo.keys, canonical: make([]string, len(out))}
	for i, r := range out {
		memo.keys[i] = replayKeyFor(r)
		gov.canonical[i] = r.Canonical
	}
	out[0].replay, out[0].govern = memo, gov
	return out
}

func cachedReplay(records []Record) (Projection, bool) {
	if len(records) == 0 {
		return Projection{}, false
	}
	memo := records[0].replay
	if memo == nil || len(records) != len(memo.keys) {
		return Projection{}, false
	}
	for i, r := range records {
		if replayKeyFor(r) != memo.keys[i] {
			return Projection{}, false
		}
	}
	p := memo.projection
	p.Active = maps.Clone(p.Active)
	return p, true
}

// cachedTrialReplay answers a priced proposal: a held, cached ledger with one proposed statistics decision
// appended. That record is active, reverses nothing and restores no draft, so the replay is the held
// projection plus that entry (TestTrialReplayMatchesFullReplay).
func cachedTrialReplay(records []Record) (Projection, bool) {
	n := len(records)
	if n < 2 || !proposedStatDecision(records) || records[0].replay == nil || len(records[0].replay.keys) != n-1 {
		return Projection{}, false
	}
	p, ok := cachedReplay(records[:n-1])
	if !ok {
		return Projection{}, false
	}
	p.Active[records[n-1].Receipt.Entry] = true
	return p, true
}

// memoFor returns the memo only when records are exactly the revision it was built from, payloads included
// (Canonical is the recorded request, so it covers every argument a fold reads).
func memoFor(records []Record) *governanceCache {
	if len(records) == 0 {
		return nil
	}
	memo := records[0].govern
	if memo == nil || len(records) != len(memo.keys) {
		return nil
	}
	for i, r := range records {
		if replayKeyFor(r) != memo.keys[i] || r.Canonical != memo.canonical[i] {
			return nil
		}
	}
	return memo
}

// cachedGovernance folds people state once per held ledger revision and policy view. State is
// value-immutable to callers (Apply clones), so the same result is shared.
func cachedGovernance(records []Record, fold func([]Record) (governance.State, error)) (governance.State, error) {
	memo := memoFor(records)
	if memo == nil {
		return fold(records)
	}
	key := governanceMemoKey{principals: records[0].principals.Key(), books: records[0].books}
	memo.mu.Lock()
	defer memo.mu.Unlock()
	if hit, ok := memo.views[key]; ok {
		return hit.state, hit.err
	}
	state, err := fold(records)
	if memo.views == nil {
		memo.views = map[governanceMemoKey]governanceMemo{}
	}
	if len(memo.views) >= maxGovernanceViews {
		clear(memo.views)
	}
	memo.views[key] = governanceMemo{state: state, err: err}
	return state, err
}
