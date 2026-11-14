package serverhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

// cachedReadingOperation names the league-state readings a server may answer from its reply cache, with
// the admission operation each reading itself makes. Each reads only the ledger, the pinned football
// evidence and the clock through commissioner.ReadingEpoch (TestReplyCacheMatchesFreshReadings). Live
// readings (scores, standings, odds, values, lineups, draft, bids, waivers, trades, market) are never here:
// they read the clock or live evidence directly.
func cachedReadingOperation(reading string) (string, bool) {
	switch reading {
	case "commish.rules.read", "commish.corrections.read", "native.assets.read", "native.reserve.check", "native.roster.check":
		return reading, true
	case LedgerPath:
		return "commish.ledger.read", true
	case PeoplePath:
		return "commish.people.read", true
	}
	return "", false
}

// ReplyCache holds encoded league-state readings for each league's committed head and reading epoch,
// per signed-in principal: one owner's reply is never served to another. A new head or epoch drops the
// league's replies. Every request is still admitted before a cached reply is served.
type ReplyCache struct {
	mu      sync.Mutex
	limit   int64
	size    int64
	leagues map[string]*leagueReplies
}

type leagueReplies struct {
	head    int64
	epoch   int
	replies map[replyKey]*cachedReply
}

type replyKey struct {
	principal, reading, args, football string
}

type cachedReply struct {
	once sync.Once
	body []byte
	ok   bool
	kept bool // counted in size; guarded by ReplyCache.mu
}

// NewReplyCache keeps at most limit bytes of replies; past it, a league's replies are dropped before more
// are kept.
func NewReplyCache(limit int64) *ReplyCache {
	return &ReplyCache{limit: limit, leagues: map[string]*leagueReplies{}}
}

type replyPrincipalKey struct{}

// withReplyPrincipal marks a request as a signed-in person's, the only requests the reply cache serves;
// agent tokens carry their own audit and always read afresh.
func withReplyPrincipal(ctx context.Context, principal string) context.Context {
	return context.WithValue(ctx, replyPrincipalKey{}, principal)
}

// replyScope is where one reading's reply is cached; the zero scope never caches.
type replyScope struct {
	cache  *ReplyCache
	league string
	head   int64
	epoch  int
	key    replyKey
}

// scope admits the reading and names its cache place. It returns the zero scope, and no error, when the
// reading is not cacheable here; an admission refusal is returned as is.
func (h Hosted) replyScope(ctx context.Context, service *leagueservice.Service, reading, league string,
	args []byte, football string) (replyScope, error) {
	operation, cacheable := cachedReadingOperation(reading)
	principal, signedIn := ctx.Value(replyPrincipalKey{}).(string)
	if h.Replies == nil || !cacheable || !signedIn || principal == "" {
		return replyScope{}, nil
	}
	// Read the head before the epoch and the reading, so a reply is never kept under a head older than the
	// records it was read from would allow to be served (heads only grow).
	head, err := h.leagueHead(ctx, league)
	if err != nil {
		return replyScope{}, nil //nolint:nilerr // an unreadable head only skips the cache
	}
	//nolint:contextcheck // The request-scoped StoredHost supplies the request context.
	if err := service.AdmitRead(league, operation, ""); err != nil {
		return replyScope{}, fmt.Errorf("admit cached reading: %w", err)
	}
	//nolint:contextcheck // The request-scoped StoredHost supplies the request context.
	epoch, _, ok, err := service.ReadingEpoch(league, h.pinFor(ctx, league).Football.Kickoffs())
	if err != nil || !ok {
		return replyScope{}, nil //nolint:nilerr // an untimed league only skips the cache
	}
	return replyScope{cache: h.Replies, league: league, head: head, epoch: epoch,
		key: replyKey{principal: principal, reading: reading, args: string(args), football: football}}, nil
}

// reply answers from the cache, or runs read once for every concurrent caller of the same scope and keeps
// a successful answer. A failed read is never kept: the caller that ran it gets its error, and anyone who
// waited on it reads for themselves.
func (s replyScope) reply(read func() ([]byte, error)) ([]byte, error) {
	if s.cache == nil {
		return read()
	}
	entry := s.cache.entry(s)
	var err error
	ran := false
	entry.once.Do(func() {
		ran = true
		var body []byte
		if body, err = read(); err == nil {
			entry.body, entry.ok = body, true
			s.cache.keep(s, entry)
		}
	})
	if entry.ok {
		return entry.body, nil
	}
	s.cache.forget(s, entry)
	if ran {
		return nil, err
	}
	return read()
}

func (c *ReplyCache) entry(s replyScope) *cachedReply {
	c.mu.Lock()
	defer c.mu.Unlock()
	l := c.leagues[s.league]
	if l == nil || l.head != s.head || l.epoch != s.epoch {
		if l != nil && (s.head < l.head || (s.head == l.head && s.epoch < l.epoch)) {
			// A request that read an older head or epoch than the cache holds reads alone.
			return &cachedReply{}
		}
		c.drop(s.league)
		l = &leagueReplies{head: s.head, epoch: s.epoch, replies: map[replyKey]*cachedReply{}}
		c.leagues[s.league] = l
	}
	e := l.replies[s.key]
	if e == nil {
		e = &cachedReply{}
		l.replies[s.key] = e
	}
	return e
}

// keep counts a read reply against the limit. Past the limit every other reply is dropped first; a reply
// larger than the limit, or one whose head or epoch has already moved on, is answered but not kept.
func (c *ReplyCache) keep(s replyScope, e *cachedReply) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := int64(len(e.body))
	l := c.leagues[s.league]
	if l == nil || l.replies[s.key] != e {
		return
	}
	if n > c.limit {
		delete(l.replies, s.key)
		return
	}
	if c.size+n > c.limit {
		for league := range c.leagues {
			c.drop(league)
		}
		l = &leagueReplies{head: s.head, epoch: s.epoch, replies: map[replyKey]*cachedReply{s.key: e}}
		c.leagues[s.league] = l
	}
	c.size += n
	e.kept = true
}

func (c *ReplyCache) forget(s replyScope, e *cachedReply) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if l := c.leagues[s.league]; l != nil && l.replies[s.key] == e {
		delete(l.replies, s.key)
	}
}

// drop removes a league's replies; the caller holds mu.
func (c *ReplyCache) drop(league string) {
	l := c.leagues[league]
	if l == nil {
		return
	}
	for _, e := range l.replies {
		if e.kept {
			c.size -= int64(len(e.body))
		}
	}
	delete(c.leagues, league)
}

// encodeReading is json.Marshal for a cached reading's reply.
func encodeReading[T any](value T, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode reading: %w", err)
	}
	return body, nil
}
