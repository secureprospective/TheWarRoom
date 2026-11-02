package history

import (
	"context"
	"database/sql"
	"errors"
	"slices"
)

// maxCachedEvents bounds the decoded whole-stream events held across all streams.
const maxCachedEvents = 100_000

type eventStreamKey struct{ league, stream string }

// eventRevision is one stream's decoded prefix; callers receive deep copies.
type eventRevision struct {
	revision int64
	lastID   string
	events   []LeagueEvent
}

// wholeStream serves Events(ctx, league, stream, 0). The background value producer and Home's
// league reads replay the whole ledger every pass; live stat approvals grow it by thousands of
// entries a day, so re-decoding it each time starved the read pool and pinned the CPU.
func (s *Store) wholeStream(ctx context.Context, league, stream string) ([]LeagueEvent, error) {
	key := eventStreamKey{league, stream}
	// The lock guards only the map: the database reads run outside it, so one slow reader does not hold up
	// the others. Held events are never changed in place (the tail goes onto a clipped copy).
	s.eventsMu.Lock()
	seen := s.events[key]
	s.eventsMu.Unlock()
	hit := seen
	if hit.revision > 0 && !s.prefixHeld(ctx, key, hit) {
		hit = eventRevision{}
	}
	tail, err := s.eventsAfter(ctx, league, stream, hit.revision)
	if err != nil {
		return nil, err
	}
	if len(tail) == 0 && hit.revision > 0 {
		return deepClone(hit.events), nil
	}
	hit.events = append(slices.Clip(hit.events), tail...)
	if n := len(hit.events); n > 0 {
		hit.revision, hit.lastID = hit.events[n-1].Revision, hit.events[n-1].ID
	}
	s.eventsMu.Lock()
	// Another reader may have held a newer copy meanwhile; keep it rather than going back.
	if now := s.events[key]; now.revision == seen.revision && now.lastID == seen.lastID {
		s.cacheEvents(key, hit)
	}
	s.eventsMu.Unlock()
	return deepClone(hit.events), nil
}

// prefixHeld reports whether the held newest entry is still the stream's entry at that revision,
// so a replaced or restored database drops the held prefix. A failed check also drops it.
func (s *Store) prefixHeld(ctx context.Context, key eventStreamKey, hit eventRevision) bool {
	var id string
	err := s.pools.Read().QueryRowContext(ctx, `SELECT id FROM league_event WHERE league=? AND stream=? AND revision=?`,
		key.league, key.stream, hit.revision).Scan(&id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false
	}
	return err == nil && id == hit.lastID
}

// cacheEvents runs under eventsMu; oversized streams are still read, just not held.
func (s *Store) cacheEvents(key eventStreamKey, hit eventRevision) {
	delete(s.events, key)
	if hit.revision == 0 || len(hit.events) > maxCachedEvents {
		return
	}
	held := len(hit.events)
	for _, other := range s.events {
		held += len(other.events)
	}
	if held > maxCachedEvents {
		clear(s.events)
	}
	if s.events == nil {
		s.events = map[eventStreamKey]eventRevision{}
	}
	s.events[key] = hit
}
