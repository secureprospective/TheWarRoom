package leagueservice

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/espnnews"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

const (
	leagueNewsTTL      = 15 * time.Minute
	playerNewsTTL      = 15 * time.Minute
	playerBioTTL       = 24 * time.Hour
	espnFetchTimeout   = 15 * time.Second
	espnRetryBaseDelay = 30 * time.Second

	PlayerNewsRead = "player.news.read"
	NewsFeedRead   = "news.feed.read"
)

// ESPNNewsArchive persists fetched ESPN payloads in server reference storage.
type ESPNNewsArchive interface {
	LoadESPNBody(ctx context.Context, key string) (body []byte, fetchedAt time.Time, ok bool, err error)
	SaveESPNBody(ctx context.Context, key string, body []byte, fetchedAt time.Time) error
}

// PlayerDirectory maps between league player IDs (MFL IDs) and outside provider IDs.
type PlayerDirectory interface {
	SourceID(ctx context.Context, playerID, idType string) (string, error)
	SourceIDs(ctx context.Context, idType string) (map[string]string, error)
}

type PlayerNewsItem struct {
	ID          string `json:"id"`
	Headline    string `json:"headline"`
	Description string `json:"description,omitempty"`
	Published   string `json:"published"`
	URL         string `json:"url,omitempty"`
	Source      string `json:"source"`
	Type        string `json:"type,omitempty"`
	Injury      bool   `json:"injury,omitempty"`
}

type PlayerNewsReading struct {
	League string           `json:"league"`
	Player string           `json:"player"`
	ESPNID string           `json:"espnId"`
	Bio    espnnews.Bio     `json:"bio"`
	Items  []PlayerNewsItem `json:"items"`
	AsOf   string           `json:"asOf"`
	Stale  bool             `json:"stale"`
	Reason string           `json:"reason,omitempty"`
}

type NewsFeedItem struct {
	PlayerNewsItem
	Players []string `json:"players"`
}

type NewsFeedReading struct {
	League string         `json:"league"`
	Items  []NewsFeedItem `json:"items"`
	AsOf   string         `json:"asOf"`
	Stale  bool           `json:"stale"`
	Reason string         `json:"reason,omitempty"`
}

type cachedItems struct {
	items     []espnnews.Item
	at        time.Time
	failures  int
	nextRetry time.Time
	loaded    bool
}

type cachedBio struct {
	bio       espnnews.Bio
	at        time.Time
	failures  int
	nextRetry time.Time
	loaded    bool
}

type archivedPlayerNews struct {
	Kind string          `json:"kind"`
	Raw  json.RawMessage `json:"raw"`
}

// NewsService caches ESPN league news, player news, and player bios without blocking reads on ESPN.
type NewsService struct {
	transport livegame.Transport
	endpoints espnnews.Endpoints
	archive   ESPNNewsArchive
	directory PlayerDirectory
	now       func() time.Time

	mu         sync.Mutex
	closed     bool
	stopCh     chan struct{}
	wg         sync.WaitGroup
	league     cachedItems
	playerNews map[string]cachedItems
	playerBio  map[string]cachedBio
	inFlight   map[string]bool
}

func NewNewsService(
	transport livegame.Transport,
	endpoints espnnews.Endpoints,
	archive ESPNNewsArchive,
	directory PlayerDirectory,
) *NewsService {
	return &NewsService{
		transport:  transport,
		endpoints:  endpoints,
		archive:    archive,
		directory:  directory,
		now:        time.Now,
		stopCh:     make(chan struct{}),
		playerNews: map[string]cachedItems{},
		playerBio:  map[string]cachedBio{},
		inFlight:   map[string]bool{},
	}
}

func (s *NewsService) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.stopCh)
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// StartLeagueRefresh fetches league news at startup and every 15 minutes until ctx ends.
func (s *NewsService) StartLeagueRefresh(ctx context.Context) func() {
	if s == nil {
		return func() {}
	}
	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.refreshLoop(workerCtx)
	}()
	return func() { cancel(); <-done }
}

func (s *NewsService) refreshLoop(ctx context.Context) {
	refresh := func() {
		attempt, cancel := context.WithTimeout(ctx, espnFetchTimeout)
		defer cancel()
		_ = s.RefreshLeagueNow(attempt)
	}
	refresh()
	ticker := time.NewTicker(leagueNewsTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			refresh()
		}
	}
}

func (s *NewsService) RefreshLeagueNow(ctx context.Context) error {
	now := s.now().UTC()
	body, items, err := espnnews.FetchLeagueNews(ctx, s.transport, s.endpoints)
	if err != nil {
		s.recordLeagueFailure(now)
		return fmt.Errorf("refresh league news: %w", err)
	}
	s.saveArchive(ctx, s.leagueKey(), body, now)
	s.mu.Lock()
	s.league = cachedItems{items: cloneNewsItems(items), at: now, loaded: true}
	s.mu.Unlock()
	return nil
}

func (s *NewsService) RefreshPlayerNow(
	ctx context.Context, espnID string,
) (espnnews.Bio, []espnnews.Item, error) {
	bioErr := s.refreshBioNow(ctx, espnID)
	newsErr := s.refreshPlayerNewsNow(ctx, espnID)
	s.mu.Lock()
	bio := s.playerBio[espnID].bio
	items := cloneNewsItems(s.playerNews[espnID].items)
	s.mu.Unlock()
	if bioErr != nil && newsErr != nil {
		return bio, items, fmt.Errorf("refresh player %s: bio: %w; news: %v", espnID, bioErr, newsErr)
	}
	return bio, items, nil
}

func (s *NewsService) refreshPlayerNewsNow(ctx context.Context, espnID string) error {
	now := s.now().UTC()
	body, items, kind, err := espnnews.FetchPlayerNews(ctx, s.transport, s.endpoints, espnID)
	if err != nil {
		s.recordPlayerNewsFailure(espnID, now)
		return fmt.Errorf("refresh player news %s: %w", espnID, err)
	}
	if env, mErr := json.Marshal(archivedPlayerNews{Kind: kind, Raw: body}); mErr == nil {
		s.saveArchive(ctx, s.playerNewsKey(espnID), env, now)
	}
	s.mu.Lock()
	s.playerNews[espnID] = cachedItems{items: cloneNewsItems(items), at: now, loaded: true}
	s.mu.Unlock()
	return nil
}

func (s *NewsService) refreshBioNow(ctx context.Context, espnID string) error {
	now := s.now().UTC()
	body, bio, err := espnnews.FetchBio(ctx, s.transport, s.endpoints, espnID)
	if err != nil {
		s.recordBioFailure(espnID, now)
		return fmt.Errorf("refresh player bio %s: %w", espnID, err)
	}
	s.saveArchive(ctx, s.bioKey(espnID), body, now)
	s.mu.Lock()
	s.playerBio[espnID] = cachedBio{bio: bio, at: now, loaded: true}
	s.mu.Unlock()
	return nil
}

// ReadLeagueNews serves the cached league news immediately and schedules a background refresh if needed.
func (s *NewsService) ReadLeagueNews(ctx context.Context) ([]espnnews.Item, time.Time, bool) {
	s.hydrateLeague(ctx)
	now := s.now().UTC()
	s.mu.Lock()
	entry := s.league
	expired := entry.at.IsZero() || now.Sub(entry.at) >= leagueNewsTTL
	stale := expired || entry.failures > 0
	trigger := expired && !now.Before(entry.nextRetry)
	items := cloneNewsItems(entry.items)
	asOf := entry.at
	s.mu.Unlock()
	if trigger {
		s.spawnRefresh(ctx, "league", func(bg context.Context) {
			_ = s.RefreshLeagueNow(bg)
		})
	}
	return items, asOf, stale
}

// ReadPlayer serves cached player bio and news immediately and triggers background refresh when stale.
func (s *NewsService) ReadPlayer(
	ctx context.Context, espnID string,
) (espnnews.Bio, []espnnews.Item, time.Time, bool) {
	s.hydratePlayer(ctx, espnID)
	now := s.now().UTC()
	s.mu.Lock()
	newsEntry := s.playerNews[espnID]
	bioEntry := s.playerBio[espnID]
	newsExpired := newsEntry.at.IsZero() || now.Sub(newsEntry.at) >= playerNewsTTL
	bioExpired := bioEntry.at.IsZero() || now.Sub(bioEntry.at) >= playerBioTTL
	triggerNews := newsExpired && !now.Before(newsEntry.nextRetry)
	triggerBio := bioExpired && !now.Before(bioEntry.nextRetry)
	stale := newsExpired || newsEntry.failures > 0 || bioExpired || bioEntry.failures > 0
	bio := bioEntry.bio
	items := cloneNewsItems(newsEntry.items)
	asOf := newsEntry.at
	if asOf.IsZero() {
		asOf = bioEntry.at
	}
	s.mu.Unlock()
	if triggerNews {
		s.spawnRefresh(ctx, "news:"+espnID, func(bg context.Context) {
			_ = s.refreshPlayerNewsNow(bg, espnID)
		})
	}
	if triggerBio {
		s.spawnRefresh(ctx, "bio:"+espnID, func(bg context.Context) {
			_ = s.refreshBioNow(bg, espnID)
		})
	}
	return bio, items, asOf, stale
}

func (s *NewsService) hydrateLeague(ctx context.Context) {
	s.mu.Lock()
	loaded := s.league.loaded
	s.mu.Unlock()
	if loaded || s.archive == nil {
		return
	}
	body, at, ok, err := s.archive.LoadESPNBody(ctx, s.leagueKey())
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.league.loaded {
		return
	}
	if err == nil && ok {
		if items, pErr := espnnews.ParseLeagueNews(body); pErr == nil {
			s.league = cachedItems{items: items, at: at.UTC(), loaded: true}
			return
		}
	}
	s.league.loaded = true
}

func (s *NewsService) hydratePlayer(ctx context.Context, espnID string) {
	s.mu.Lock()
	newsLoaded := s.playerNews[espnID].loaded
	bioLoaded := s.playerBio[espnID].loaded
	s.mu.Unlock()
	if (newsLoaded && bioLoaded) || s.archive == nil {
		return
	}
	if !newsLoaded {
		body, at, ok, err := s.archive.LoadESPNBody(ctx, s.playerNewsKey(espnID))
		s.mu.Lock()
		if !s.playerNews[espnID].loaded {
			if err == nil && ok {
				if items, pErr := decodeArchivedPlayerNews(body); pErr == nil {
					s.playerNews[espnID] = cachedItems{items: items, at: at.UTC(), loaded: true}
				} else {
					s.playerNews[espnID] = cachedItems{loaded: true}
				}
			} else {
				s.playerNews[espnID] = cachedItems{loaded: true}
			}
		}
		s.mu.Unlock()
	}
	if !bioLoaded {
		body, at, ok, err := s.archive.LoadESPNBody(ctx, s.bioKey(espnID))
		s.mu.Lock()
		if !s.playerBio[espnID].loaded {
			if err == nil && ok {
				if bio, pErr := espnnews.ParseBio(body); pErr == nil {
					s.playerBio[espnID] = cachedBio{bio: bio, at: at.UTC(), loaded: true}
				} else {
					s.playerBio[espnID] = cachedBio{loaded: true}
				}
			} else {
				s.playerBio[espnID] = cachedBio{loaded: true}
			}
		}
		s.mu.Unlock()
	}
}

func decodeArchivedPlayerNews(body []byte) ([]espnnews.Item, error) {
	var env archivedPlayerNews
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("decode archived player news: %w", err)
	}
	if env.Kind == "site" {
		items, err := espnnews.ParseLeagueNews(env.Raw)
		if err != nil {
			return nil, fmt.Errorf("parse archived site player news: %w", err)
		}
		return items, nil
	}
	items, err := espnnews.ParseFantasyNews(env.Raw)
	if err != nil {
		return nil, fmt.Errorf("parse archived fantasy player news: %w", err)
	}
	return items, nil
}

func (s *NewsService) spawnRefresh(parent context.Context, key string, work func(context.Context)) {
	if s.transport == nil {
		return
	}
	s.mu.Lock()
	if s.closed || s.inFlight[key] {
		s.mu.Unlock()
		return
	}
	s.inFlight[key] = true
	s.wg.Add(1)
	s.mu.Unlock()
	base := context.WithoutCancel(parent)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			delete(s.inFlight, key)
			s.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(base, espnFetchTimeout)
		defer cancel()
		done := make(chan struct{})
		go func() {
			select {
			case <-s.stopCh:
				cancel()
			case <-done:
			}
		}()
		work(ctx)
		close(done)
	}()
}

func (s *NewsService) recordLeagueFailure(now time.Time) {
	s.mu.Lock()
	s.league.failures++
	s.league.nextRetry = now.Add(livegame.Backoff(espnRetryBaseDelay, s.league.failures-1))
	s.league.loaded = true
	s.mu.Unlock()
}

func (s *NewsService) recordPlayerNewsFailure(espnID string, now time.Time) {
	s.mu.Lock()
	entry := s.playerNews[espnID]
	entry.failures++
	entry.nextRetry = now.Add(livegame.Backoff(espnRetryBaseDelay, entry.failures-1))
	entry.loaded = true
	s.playerNews[espnID] = entry
	s.mu.Unlock()
}

func (s *NewsService) recordBioFailure(espnID string, now time.Time) {
	s.mu.Lock()
	entry := s.playerBio[espnID]
	entry.failures++
	entry.nextRetry = now.Add(livegame.Backoff(espnRetryBaseDelay, entry.failures-1))
	entry.loaded = true
	s.playerBio[espnID] = entry
	s.mu.Unlock()
}

func (s *NewsService) saveArchive(ctx context.Context, key string, body []byte, at time.Time) {
	if s.archive == nil || len(body) == 0 {
		return
	}
	_ = s.archive.SaveESPNBody(ctx, key, body, at)
}

func (s *NewsService) leagueKey() string { return s.endpoints.LeagueNews }
func (s *NewsService) playerNewsKey(espnID string) string {
	return "https://espn-cache.local/player-news/" + espnID
}
func (s *NewsService) bioKey(espnID string) string {
	return strings.ReplaceAll(s.endpoints.Bio, "{id}", espnID)
}

func (a *Service) WithNews(news *NewsService) *Service {
	a.news = news
	return a
}

// TargetPlayerNews reads cached ESPN news and bio for one player in an admitted league.
func (a *Service) TargetPlayerNews(scope RawPlayerReadScope) (PlayerNewsReading, error) {
	if err := scope.Validate(); err != nil {
		return PlayerNewsReading{}, fmt.Errorf("player news input: %w", err)
	}
	access := commissioner.Access{
		League: scope.League, Operation: PlayerNewsRead, Mode: "read", Player: scope.Player,
	}
	if err := a.admitReading(access); err != nil {
		return PlayerNewsReading{}, err
	}
	if err := a.host.Ready(); err != nil {
		return PlayerNewsReading{}, fmt.Errorf("player news startup: %w", err)
	}
	reading := PlayerNewsReading{
		League: scope.League, Player: scope.Player, Items: []PlayerNewsItem{},
	}
	if a.news == nil {
		reading.Stale = true
		reading.Reason = "Player news is not available on this server yet."
		return reading, nil
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	espnID, err := a.lookupESPNID(ctx, scope.Player)
	if err != nil {
		return PlayerNewsReading{}, fmt.Errorf("player news directory: %w", err)
	}
	if espnID == "" {
		reading.Reason = "No ESPN player mapping is on file for this player."
		return reading, nil
	}
	bio, items, asOf, stale := a.news.ReadPlayer(ctx, espnID)
	reading.ESPNID = espnID
	reading.Bio = bio
	reading.Items = toPlayerNewsItems(items)
	reading.AsOf = formatAsOf(asOf)
	reading.Stale = stale
	return reading, nil
}

// TargetNewsFeed reads cached ESPN league-wide news tagged with matching league player IDs.
func (a *Service) TargetNewsFeed(league string) (NewsFeedReading, error) {
	if err := (RawLeagueScheduleScope{League: league}).Validate(); err != nil {
		return NewsFeedReading{}, fmt.Errorf("news feed input: %w", err)
	}
	if err := a.admitRead(league, NewsFeedRead, ""); err != nil {
		return NewsFeedReading{}, err
	}
	if err := a.host.Ready(); err != nil {
		return NewsFeedReading{}, fmt.Errorf("news feed startup: %w", err)
	}
	reading := NewsFeedReading{League: league, Items: []NewsFeedItem{}}
	if a.news == nil {
		reading.Stale = true
		reading.Reason = "League news is not available on this server yet."
		return reading, nil
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	items, asOf, stale := a.news.ReadLeagueNews(ctx)
	espnToMFL, err := a.lookupESPNToMFL(ctx)
	if err != nil {
		return NewsFeedReading{}, fmt.Errorf("news feed directory: %w", err)
	}
	reading.Items = toNewsFeedItems(items, espnToMFL)
	reading.AsOf = formatAsOf(asOf)
	reading.Stale = stale
	return reading, nil
}

func (a *Service) lookupESPNID(ctx context.Context, playerID string) (string, error) {
	if a.news.directory == nil {
		return "", nil
	}
	id, err := a.news.directory.SourceID(ctx, playerID, "espn")
	if err != nil {
		return "", fmt.Errorf("lookup espn id for %s: %w", playerID, err)
	}
	return strings.TrimSpace(id), nil
}

func (a *Service) lookupESPNToMFL(ctx context.Context) (map[string]string, error) {
	if a.news.directory == nil {
		return nil, nil
	}
	mflToESPN, err := a.news.directory.SourceIDs(ctx, "espn")
	if err != nil {
		return nil, fmt.Errorf("lookup espn directory: %w", err)
	}
	out := make(map[string]string, len(mflToESPN))
	for mflID, espnID := range mflToESPN {
		if espnID != "" {
			out[espnID] = mflID
		}
	}
	return out, nil
}

func toPlayerNewsItems(items []espnnews.Item) []PlayerNewsItem {
	out := make([]PlayerNewsItem, 0, len(items))
	for _, it := range items {
		out = append(out, toPlayerNewsItem(it))
	}
	return out
}

func toPlayerNewsItem(it espnnews.Item) PlayerNewsItem {
	pub := ""
	if !it.Published.IsZero() {
		pub = it.Published.UTC().Format(time.RFC3339)
	}
	return PlayerNewsItem{
		ID:          it.ID,
		Headline:    it.Headline,
		Description: it.Description,
		Published:   pub,
		URL:         it.URL,
		Source:      it.Source,
		Type:        it.Type,
		Injury:      it.Injury,
	}
}

func toNewsFeedItems(items []espnnews.Item, espnToMFL map[string]string) []NewsFeedItem {
	out := make([]NewsFeedItem, 0, len(items))
	for _, it := range items {
		players := []string{}
		seen := map[string]bool{}
		for _, espnID := range it.AthleteIDs {
			if mflID := espnToMFL[espnID]; mflID != "" && !seen[mflID] {
				seen[mflID] = true
				players = append(players, mflID)
			}
		}
		slices.Sort(players)
		out = append(out, NewsFeedItem{PlayerNewsItem: toPlayerNewsItem(it), Players: players})
	}
	return out
}

func formatAsOf(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.UTC().Format(time.RFC3339)
}

func cloneNewsItems(items []espnnews.Item) []espnnews.Item {
	if len(items) == 0 {
		return nil
	}
	out := make([]espnnews.Item, len(items))
	for i, it := range items {
		it.AthleteIDs = slices.Clone(it.AthleteIDs)
		out[i] = it
	}
	return out
}
