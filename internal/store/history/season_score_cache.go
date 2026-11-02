package history

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// keepScoreRevisions bounds the season score cache: scores are kept for this many distinct revisions (a
// league's ledger, held archive, rules and player facts together), newest used first.
const keepScoreRevisions = 8

// seasonScoreCache scores each franchise-week once per revision and shares it between every reading of that
// revision: standings, odds, final standings, draft order, results and the bracket. A revision is everything
// a week's score is made of except live lines: the exact ledger entries, the held archive, the rules in force
// at the reading's instant and the player facts. Live lines move one NFL week at a time, so each week carries
// its own live generation and a live poll re-scores only the week it touches.
//
// Scoring does not depend on who reads: lineups and scores fold the whole ledger, never a viewer's slice of it,
// so no viewer key is part of the revision.
type seasonScoreCache struct {
	mu        sync.Mutex
	off       bool // tests: score every time, as before the cache
	flights   map[scoreFlightKey]chan struct{}
	revisions map[string]*scoreRevision
	order     []string // least recently used first
	canon     map[string]canonEntries
	hits      int
	misses    int
}

type scoreFlightKey struct {
	revision string
	week     weekScoreKey
}

type scoreRevision struct {
	scores map[weekScoreKey]commissioner.FranchiseWeekScore
}

type weekScoreKey struct {
	seat, lineup string
	season, week int
	live         liveWeekGen
}

// canonEntries is a league's durable ledger as last read: every entry it holds. A ledger whose entries are
// not all durable (a proposed change, a waiting rule counted early) is a trial and is never cached.
type canonEntries struct {
	mark    string
	entries map[string]bool
}

// liveWeekGen names the live lines of one NFL week as held: epoch moves when the book is restored, week when
// a line or revision of that week changes.
type liveWeekGen struct {
	epoch, week uint64
}

// scoreRevisionKey is one reading's revision, worked out on its first scored week. ok is false when the
// reading cannot share: a trial ledger, test-supplied evidence or lineups, or the cache switched off.
type scoreRevisionKey struct {
	ledger    []commissioner.Record
	ledgerLen int
	key       string
	ok        bool
}

// MemoWeekScore serves a franchise's week from the season score cache when this reading's revision has
// scored it, and scores it otherwise. Every caller gets its own deep copy; the cached value is never handed out.
func (n *nativeSources) MemoWeekScore(ctx context.Context, records []commissioner.Record, rules leaguerules.LineupRules,
	seat commissioner.NativeDirectorySeat, season, week int,
	score func() (commissioner.FranchiseWeekScore, error)) (commissioner.FranchiseWeekScore, error) {
	if err := ctx.Err(); err != nil {
		return commissioner.FranchiseWeekScore{}, fmt.Errorf("score week: %w", err)
	}
	revision, ok := n.scoreRevision(ctx, records)
	if !ok {
		return score()
	}
	seatKey, err := json.Marshal(seat)
	if err != nil {
		return score()
	}
	lineupKey, err := json.Marshal(rules)
	if err != nil {
		return score()
	}
	key := weekScoreKey{seat: string(seatKey), lineup: string(lineupKey), season: season, week: week,
		live: n.store.liveWeekGeneration(season, week)}
	c := &n.store.scores
	for {
		if hit, ok := c.get(revision, key); ok {
			return hit, nil
		}
		done, owner := c.start(revision, key)
		if owner {
			defer c.finish(revision, key, done)
			// Another scorer may have filled the cache between get and start.
			if hit, ok := c.get(revision, key); ok {
				return hit, nil
			}
			break
		}
		select {
		case <-done:
		case <-ctx.Done():
			return commissioner.FranchiseWeekScore{}, fmt.Errorf("wait for week score: %w", ctx.Err())
		}
	}
	out, err := score()
	if err != nil {
		return out, err
	}
	// A live line that landed while this week was scored names a newer week; the score read for the older
	// one is not kept under either.
	if n.store.liveWeekGeneration(season, week) == key.live {
		c.put(revision, key, deepClone(out))
	}
	return out, nil
}

func (c *seasonScoreCache) start(revision string, key weekScoreKey) (chan struct{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.flights == nil {
		c.flights = map[scoreFlightKey]chan struct{}{}
	}
	k := scoreFlightKey{revision, key}
	if done := c.flights[k]; done != nil {
		return done, false
	}
	done := make(chan struct{})
	c.flights[k] = done
	return done, true
}

func (c *seasonScoreCache) finish(revision string, key weekScoreKey, done chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.flights, scoreFlightKey{revision, key})
	close(done)
}

func (c *seasonScoreCache) get(revision string, key weekScoreKey) (commissioner.FranchiseWeekScore, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := c.revisions[revision]
	if r == nil {
		c.misses++
		return commissioner.FranchiseWeekScore{}, false
	}
	hit, ok := r.scores[key]
	if !ok {
		c.misses++
		return commissioner.FranchiseWeekScore{}, false
	}
	c.hits++
	c.touch(revision)
	return deepClone(hit), true
}

func (c *seasonScoreCache) put(revision string, key weekScoreKey, score commissioner.FranchiseWeekScore) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.revisions == nil {
		c.revisions = map[string]*scoreRevision{}
	}
	r := c.revisions[revision]
	if r == nil {
		r = &scoreRevision{scores: map[weekScoreKey]commissioner.FranchiseWeekScore{}}
		c.revisions[revision] = r
	}
	// Older live generations of this week can never be read again.
	for old := range r.scores {
		if old.week == key.week && old.season == key.season && old.seat == key.seat && old.lineup == key.lineup &&
			old.live != key.live {
			delete(r.scores, old)
		}
	}
	r.scores[key] = score
	c.touch(revision)
	for len(c.order) > keepScoreRevisions {
		delete(c.revisions, c.order[0])
		c.order = c.order[1:]
	}
}

func (c *seasonScoreCache) touch(revision string) {
	if i := slices.Index(c.order, revision); i >= 0 {
		c.order = slices.Delete(c.order, i, i+1)
	}
	c.order = append(c.order, revision)
}

// scoreRevision works out the reading's revision once per ledger it scores.
func (n *nativeSources) scoreRevision(ctx context.Context, records []commissioner.Record) (string, bool) {
	if len(records) == 0 {
		return "", false
	}
	if r := n.revision; r != nil && r.ledgerLen == len(records) && &r.ledger[0] == &records[0] {
		return r.key, r.ok
	}
	key, ok := n.store.scoreRevisionFor(ctx, n, records)
	n.revision = &scoreRevisionKey{ledger: records, ledgerLen: len(records), key: key, ok: ok}
	return key, ok
}

func (s *Store) scoreRevisionFor(ctx context.Context, n *nativeSources, records []commissioner.Record) (string, bool) {
	s.scores.mu.Lock()
	off := s.scores.off
	s.scores.mu.Unlock()
	if off || s.evidence != nil || s.pools == nil {
		return "", false
	}
	if _, ledger := s.lineups.(revisionLineups); !ledger {
		return "", false
	}
	ledger, ok := s.durableLedgerDigest(ctx, n.league, records)
	if !ok {
		return "", false
	}
	if err := s.RestoreLiveStats(ctx); err != nil {
		return "", false
	}
	held, _, err := s.heldSeasonKey(ctx, n.season)
	if err != nil {
		return "", false
	}
	rules, err := n.rules.InForce(records, n.season, commissioner.ReadingInstant(records, n.now))
	rulesBody, jerr := json.Marshal(rules)
	if jerr != nil {
		return "", false
	}
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x00%d\x00%s\x00%d:%d:%x\x00%s\x00%v\x00", n.league, n.season, ledger,
		held.fetches, held.newest, held.identity, rulesBody, err)
	if s.lineupEnv != nil {
		_, _ = fmt.Fprintf(h, "%T:%s", s.lineupEnv, factsDigest(s.lineupEnv, n.playerFacts()))
	}
	return hex.EncodeToString(h.Sum(nil)), true
}

// factsDigest covers what scoring reads from the player facts: a starter's name and the position it is
// scored at. A pinned environment is fixed data, so its identity stands for it.
func factsDigest(env LineupEnvironment, facts nativecheck.Facts) string {
	if pinned, ok := env.(*PinnedFootball); ok {
		return fmt.Sprintf("pinned:%p", pinned)
	}
	h := sha256.New()
	for _, id := range slices.Sorted(maps.Keys(facts)) {
		_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\n", id, facts[id].Name, facts[id].Position)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// durableLedgerDigest names the exact entries of a ledger whose every entry is durable in the league's own
// ledger. Any other ledger is a trial (a proposed or waiting change) and has no shared revision.
func (s *Store) durableLedgerDigest(ctx context.Context, league string, records []commissioner.Record) (string, bool) {
	canon, err := s.nativeRecords(ctx, league)
	if err != nil {
		return "", false
	}
	mark := recordsMark(canon)
	c := &s.scores
	c.mu.Lock()
	known, ok := c.canon[league]
	c.mu.Unlock()
	if !ok || known.mark != mark {
		known = canonEntries{mark: mark, entries: make(map[string]bool, len(canon))}
		for _, r := range canon {
			known.entries[r.Receipt.Entry] = true
		}
		c.mu.Lock()
		if c.canon == nil {
			c.canon = map[string]canonEntries{}
		}
		c.canon[league] = known
		c.mu.Unlock()
	}
	h := sha256.New()
	seen := make(map[string]bool, len(records))
	for _, r := range records {
		if !known.entries[r.Receipt.Entry] || seen[r.Receipt.Entry] {
			return "", false
		}
		seen[r.Receipt.Entry] = true
		_, _ = fmt.Fprintf(h, "%s\n", r.Receipt.Entry)
	}
	return hex.EncodeToString(h.Sum(nil)), true
}

// revisionLineups is a lineup source whose answer is fixed by the ledger it is given, so a week scored
// from it can stand for the same ledger later. The ledger's own lineups are; a replaced source is not
// unless it says so.
type revisionLineups interface{ lineupsFollowLedger() }

func (ledgerLineups) lineupsFollowLedger() {}

// liveWeekGeneration is the live generation of one NFL week.
func (s *Store) liveWeekGeneration(season, week int) liveWeekGen {
	s.liveMu.Lock()
	defer s.liveMu.Unlock()
	return liveWeekGen{epoch: s.liveEpoch, week: s.liveWeeks[[2]int{season, week}]}
}

// bumpLiveWeekLocked marks one week's live lines changed. The caller holds liveMu.
func (s *Store) bumpLiveWeekLocked(season, week int) {
	if s.liveWeeks == nil {
		s.liveWeeks = map[[2]int]uint64{}
	}
	s.liveWeeks[[2]int{season, week}]++
}

// SetSeasonScoreCache switches the season score cache. Tests use it to prove readings are the same without it.
func (s *Store) SetSeasonScoreCache(on bool) {
	s.scores.mu.Lock()
	defer s.scores.mu.Unlock()
	s.scores.off = !on
	s.scores.revisions, s.scores.order = nil, nil
}

// seasonScoreCounts is the cache's hits and misses so far, for tests.
func (s *Store) seasonScoreCounts() (hits, misses int) {
	s.scores.mu.Lock()
	defer s.scores.mu.Unlock()
	return s.scores.hits, s.scores.misses
}
