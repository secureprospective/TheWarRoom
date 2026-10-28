package commissioner

import (
	"maps"
	"slices"
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

// WithReplayCache attaches a memory-only projection to an independently owned outer slice.
// The store bounds retained revisions; edited, truncated or extended copies replay normally.
func WithReplayCache(records []Record) []Record {
	out := slices.Clone(records)
	if len(out) == 0 {
		return out
	}
	out[0].replay = nil
	memo := &replayMemo{keys: make([]replayKey, len(out)), projection: Replay(out)}
	for i, r := range out {
		memo.keys[i] = replayKeyFor(r)
	}
	out[0].replay = memo
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
