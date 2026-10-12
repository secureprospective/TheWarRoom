// Package headshot fetches and caches player photos: ESPN first, then Sleeper, keyed through the
// player directory's espn and sleeper ids. The photos are copyrighted and for personal use only
// (spec §7): they live in the app's data directory, never in the repo, the binary or history.db
// (images are not facts).
package headshot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// espnURL asks ESPN's resizer for a 96×70 crop (about 7 KB) instead of the 230 KB original.
	espnURL    = "https://a.espncdn.com/combiner/i?img=/i/headshots/nfl/players/full/%s.png&w=96&h=70&scale=crop"
	sleeperURL = "https://sleepercdn.com/content/nfl/players/%s.jpg"

	minBytes = 200     // smaller is no photo
	maxBytes = 2 << 20 // larger is not a headshot

	// DefaultCap bounds the cache; the least recently served photos go first.
	DefaultCap = 200 << 20
	// RetryNone is how long a player with no photo waits before both hosts are asked again.
	RetryNone = 7 * 24 * time.Hour
	// interval spaces requests to the photo hosts: four a second.
	interval = 250 * time.Millisecond
)

// IDs returns the player's ESPN and Sleeper ids from the player directory; either may be empty.
type IDs func(ctx context.Context, mflID string) (espn, sleeper string, err error)

// ErrNoPhoto means neither host has a photo of the player; the card shows its fallback.
var ErrNoPhoto = errors.New("headshot: no photo")

var validID = regexp.MustCompile(`^[0-9]{1,8}$`)

// Service is the photo cache. Its zero value is not usable; build it with New.
type Service struct {
	dir    string
	ids    IDs
	client *http.Client
	now    func() time.Time
	cap    int64
	// silhouettes are the sha256s of generic "no photo" images a host serves with a 200. ESPN's
	// resizer answered a missing id with a 404 when probed (2026-10-11), so none is known yet.
	silhouettes map[string]bool

	mu       sync.Mutex // serialises fetches, so one photo is fetched once and the hosts are paced
	lastCall time.Time
	smu      sync.Mutex // guards stats, which cache hits update without waiting on a fetch
	stats    Stats
}

// Stats counts this launch's photo traffic for source health.
type Stats struct {
	Served, Fetched, Missing int
	LastFailure              time.Time
	LastError                string
}

// New builds the cache in dir with the given id lookup and HTTP client.
func New(dir string, ids IDs, client *http.Client) *Service {
	return &Service{dir: dir, ids: ids, client: client, now: time.Now, cap: DefaultCap, silhouettes: map[string]bool{}}
}

// Photo returns the player's photo bytes and content type, from the cache or fetched on first
// use. ErrNoPhoto is ordinary; any other error is a host or disk failure.
func (s *Service) Photo(ctx context.Context, mflID string) ([]byte, string, error) {
	if !validID.MatchString(mflID) {
		return nil, "", fmt.Errorf("headshot: invalid player id %q", mflID)
	}
	if b, kind, ok := s.cached(mflID); ok {
		return b, kind, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if b, kind, ok := s.cached(mflID); ok { // fetched while this call waited
		return b, kind, nil
	}
	if s.recentlyMissing(mflID) {
		return nil, "", ErrNoPhoto
	}
	espn, sleeper, err := s.ids(ctx, mflID)
	if err != nil {
		return nil, "", fmt.Errorf("headshot: ids for %s: %w", mflID, err)
	}
	var failures []error
	for _, u := range sources(espn, sleeper) {
		b, err := s.fetch(ctx, u)
		if err == nil {
			kind := http.DetectContentType(b)
			if err := s.store(mflID, b, kind); err != nil {
				return nil, "", err
			}
			s.count(func(st *Stats) { st.Fetched++ })
			return b, kind, nil
		}
		if !errors.Is(err, ErrNoPhoto) {
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 { // a host failed: do not record "no photo", so the next view retries
		err := errors.Join(failures...)
		s.count(func(st *Stats) { st.LastFailure, st.LastError = s.now(), err.Error() })
		return nil, "", err
	}
	s.count(func(st *Stats) { st.Missing++ })
	return nil, "", s.markMissing(mflID)
}

// Stats returns this launch's counts and the number and size of cached photos.
func (s *Service) Stats() (Stats, int, int64) {
	s.smu.Lock()
	st := s.stats
	s.smu.Unlock()
	files, size := s.photos()
	return st, len(files), size
}

func (s *Service) count(f func(*Stats)) {
	s.smu.Lock()
	f(&s.stats)
	s.smu.Unlock()
}

func sources(espn, sleeper string) []string {
	var out []string
	if validID.MatchString(espn) {
		out = append(out, fmt.Sprintf(espnURL, espn))
	}
	if validID.MatchString(sleeper) {
		out = append(out, fmt.Sprintf(sleeperURL, sleeper))
	}
	return out
}

// fetch reads one photo URL, pacing requests. A 403/404, a non-image body or a known silhouette
// is ErrNoPhoto; anything else that fails is the host's failure.
func (s *Service) fetch(ctx context.Context, url string) ([]byte, error) {
	if wait := interval - s.now().Sub(s.lastCall); wait > 0 {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("headshot: %w", ctx.Err())
		case <-time.After(wait):
		}
	}
	s.lastCall = s.now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("headshot: %w", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("headshot: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden:
		return nil, ErrNoPhoto
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("headshot: %s answered HTTP %d", hostOf(url), resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("headshot: read %s: %w", hostOf(url), err)
	}
	return b, s.check(b)
}

// check accepts only a real PNG or JPEG of sane size that is not a generic silhouette.
func (s *Service) check(b []byte) error {
	if len(b) < minBytes || len(b) > maxBytes {
		return ErrNoPhoto
	}
	if kind := http.DetectContentType(b); kind != "image/png" && kind != "image/jpeg" {
		return ErrNoPhoto
	}
	sum := sha256.Sum256(b)
	if s.silhouettes[hex.EncodeToString(sum[:])] {
		return ErrNoPhoto
	}
	return nil
}

func hostOf(url string) string {
	host, _, _ := strings.Cut(strings.TrimPrefix(url, "https://"), "/")
	return host
}

func (s *Service) path(mflID, ext string) string { return filepath.Join(s.dir, mflID+ext) }

// cached serves a held photo and marks it recently used.
func (s *Service) cached(mflID string) ([]byte, string, bool) {
	for _, ext := range []string{".png", ".jpg"} {
		p := s.path(mflID, ext)
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		now := s.now()
		_ = os.Chtimes(p, now, now)
		s.count(func(st *Stats) { st.Served++ })
		return b, http.DetectContentType(b), true
	}
	return nil, "", false
}

func (s *Service) recentlyMissing(mflID string) bool {
	info, err := os.Stat(s.path(mflID, ".none"))
	return err == nil && s.now().Sub(info.ModTime()) < RetryNone
}

func (s *Service) markMissing(mflID string) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("headshot: cache: %w", err)
	}
	p := s.path(mflID, ".none")
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		return fmt.Errorf("headshot: cache: %w", err)
	}
	now := s.now()
	_ = os.Chtimes(p, now, now)
	return ErrNoPhoto
}

// store writes the photo atomically, clears any "no photo" mark and evicts down to the cap.
func (s *Service) store(mflID string, b []byte, kind string) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("headshot: cache: %w", err)
	}
	ext := ".png"
	if kind == "image/jpeg" {
		ext = ".jpg"
	}
	p := s.path(mflID, ext)
	if err := os.WriteFile(p+".tmp", b, 0o600); err != nil {
		return fmt.Errorf("headshot: cache: %w", err)
	}
	if err := os.Rename(p+".tmp", p); err != nil {
		return fmt.Errorf("headshot: cache: %w", err)
	}
	now := s.now()
	_ = os.Chtimes(p, now, now)
	_ = os.Remove(s.path(mflID, ".none"))
	return s.evict()
}

type photo struct {
	path string
	size int64
	used time.Time
}

func (s *Service) photos() ([]photo, int64) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, 0
	}
	var out []photo
	var total int64
	for _, e := range entries {
		if ext := filepath.Ext(e.Name()); ext != ".png" && ext != ".jpg" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, photo{filepath.Join(s.dir, e.Name()), info.Size(), info.ModTime()})
		total += info.Size()
	}
	return out, total
}

// evict removes the least recently served photos until the cache fits its cap.
func (s *Service) evict() error {
	files, total := s.photos()
	slices.SortFunc(files, func(a, b photo) int { return a.used.Compare(b.used) })
	for _, f := range files {
		if total <= s.cap {
			return nil
		}
		if err := os.Remove(f.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("headshot: evict: %w", err)
		}
		total -= f.size
	}
	return nil
}

// Handler serves GET /headshot/<mfl id>: 200 with the photo, 404 when there is none (the card
// shows its fallback), 400 for a malformed id, 502 when a host failed, 503 when photos are off.
func (s *Service) Handler(enabled func() bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := strings.CutPrefix(r.URL.Path, "/headshot/")
		switch {
		case !ok || r.Method != http.MethodGet:
			http.NotFound(w, r)
			return
		case !enabled():
			http.Error(w, "photos are off", http.StatusServiceUnavailable)
			return
		case !validID.MatchString(id):
			http.Error(w, "invalid player id", http.StatusBadRequest)
			return
		}
		b, kind, err := s.Photo(r.Context(), id)
		switch {
		case errors.Is(err, ErrNoPhoto):
			http.NotFound(w, r)
		case err != nil:
			http.Error(w, "photo host unavailable", http.StatusBadGateway)
		default:
			w.Header().Set("Content-Type", kind)
			w.Header().Set("Cache-Control", "max-age=86400")
			_, _ = io.Copy(w, bytes.NewReader(b))
		}
	})
}
