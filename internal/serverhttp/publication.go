package serverhttp

import (
	"crypto/rand"
	"slices"
	"strconv"
	"sync"
	"time"
)

// TopicVersion is a freshness hint, never a ledger cursor or an authorization grant.
type TopicVersion struct {
	Kind         string `json:"kind"`
	Version      string `json:"version"`
	Franchise    string `json:"franchise,omitempty"`
	Season       int    `json:"season,omitempty"`
	Week         int    `json:"week,omitempty"`
	Run          string `json:"run,omitempty"`
	NextBoundary string `json:"nextBoundary,omitempty"`
}

type Invalidation struct {
	Protocol    int            `json:"protocol"`
	ServerEpoch string         `json:"serverEpoch"`
	League      string         `json:"league"`
	Topics      []TopicVersion `json:"topics"`
}

type publicationScope struct {
	account, league, kind, franchise, run string
	season, week                          int
}
type recipientScope struct{ account, league string }
type publicationVersion struct {
	counter      uint64
	nextBoundary string
}

// Publisher owns in-memory versions. Public snapshots must be readable before publishing.
// Versions requires an already-authorized account; it does not perform admission itself.
type Publisher struct {
	mu          sync.Mutex
	epoch       string
	versions    map[publicationScope]publicationVersion
	revisions   map[recipientScope]int64
	boundaries  map[string]time.Time
	subscribers map[string]map[chan struct{}]struct{}
}

func NewPublisher() *Publisher {
	return &Publisher{epoch: rand.Text(), versions: make(map[publicationScope]publicationVersion),
		revisions: make(map[recipientScope]int64), boundaries: make(map[string]time.Time),
		subscribers: make(map[string]map[chan struct{}]struct{})}
}

func (p *Publisher) bump(scope publicationScope, boundary string) {
	version := p.versions[scope]
	version.counter++
	version.nextBoundary = boundary
	p.versions[scope] = version
}

func (p *Publisher) publish(scope publicationScope, boundary string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bump(scope, boundary)
	for news := range p.subscribers[scope.league] {
		select {
		case news <- struct{}{}:
		default:
		}
	}
}

func (p *Publisher) PublishScores(league string, season, week int) {
	p.publish(publicationScope{league: league, kind: "scores", season: season, week: week}, "")
}

func (p *Publisher) PublishValues(league string, season int, run string) {
	p.publish(publicationScope{league: league, kind: "values", season: season, run: run}, "")
}

func (p *Publisher) PublishClock(league string, season, week int, nextBoundary time.Time) {
	boundary := ""
	if !nextBoundary.IsZero() {
		boundary = nextBoundary.UTC().Format(time.RFC3339)
	}
	p.publish(publicationScope{league: league, kind: "clock", season: season, week: week}, boundary)
}

// ObserveView records an already-authorized recipient revision, including before a versions read.
func (p *Publisher) ObserveView(account, league string, revision int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := recipientScope{account, league}
	old, known := p.revisions[key]
	if known && old == revision {
		return
	}
	p.revisions[key] = revision
	p.bump(publicationScope{account: account, league: league, kind: "league"}, "")
}

func (p *Publisher) clockBoundary(league string, passed, next time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !passed.After(p.boundaries[league]) {
		return
	}
	p.boundaries[league] = passed
	boundary := ""
	if !next.IsZero() {
		boundary = next.UTC().Format(time.RFC3339)
	}
	p.bump(publicationScope{league: league, kind: "clock"}, boundary)
}

func (p *Publisher) Versions(account, league string) Invalidation {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := Invalidation{Protocol: 2, ServerEpoch: p.epoch, League: league, Topics: []TopicVersion{}}
	for scope, version := range p.versions {
		if scope.league == league && (scope.account == "" || scope.account == account) {
			result.Topics = append(result.Topics, TopicVersion{Kind: scope.kind, Version: strconv.FormatUint(version.counter, 10),
				Franchise: scope.franchise, Season: scope.season, Week: scope.week, Run: scope.run, NextBoundary: version.nextBoundary})
		}
	}
	slices.SortFunc(result.Topics, compareTopics)
	return result
}

func compareTopics(a, b TopicVersion) int {
	if a.Kind != b.Kind {
		if a.Kind < b.Kind {
			return -1
		}
		return 1
	}
	if a.Season != b.Season {
		return a.Season - b.Season
	}
	if a.Week != b.Week {
		return a.Week - b.Week
	}
	if a.Run < b.Run {
		return -1
	}
	if a.Run > b.Run {
		return 1
	}
	return 0
}

func (p *Publisher) subscribe(league string) (<-chan struct{}, func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	news := make(chan struct{}, 1)
	if p.subscribers[league] == nil {
		p.subscribers[league] = make(map[chan struct{}]struct{})
	}
	p.subscribers[league][news] = struct{}{}
	return news, func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		delete(p.subscribers[league], news)
		if len(p.subscribers[league]) == 0 {
			delete(p.subscribers, league)
		}
	}
}
