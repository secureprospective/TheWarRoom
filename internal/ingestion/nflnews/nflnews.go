// Package nflnews holds public NFL player news for the hosted server: Sleeper's player statuses and
// ESPN's fantasy notes. It reads only public endpoints, keeps the last good copy of each, and never
// carries a league's private facts. Leagues reach it through the news read, which names the rostered
// players it wants; nothing here knows about a league.
package nflnews

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// SleeperURL is the full NFL player table, about 15 MB, keyed by Sleeper player id.
	SleeperURL = "https://api.sleeper.app/v1/players/nfl"
	espnURL    = "https://site.api.espn.com/apis/fantasy/v2/games/ffl/news/players?limit=5&playerId="
	// UserAgent identifies the hosted server to the public sources.
	UserAgent = "TheWarRoom-server/1.0"

	SleeperEvery = 10 * time.Minute
	ESPNEvery    = 30 * time.Minute
	// MaxInFlight caps ESPN requests running at once.
	MaxInFlight = 4
	// Window is how far back items are kept.
	Window = 7 * 24 * time.Hour
	// MaxItems caps one reading.
	MaxItems = 60
	maxText  = 240
	// wantedFor is how long a player stays on the fetch list after a read asked for them.
	wantedFor = time.Hour
	maxBody   = 32 << 20
)

// Player is one rostered player as the read names him. SleeperID and ESPNID come from the league's
// id crosswalk when it holds them; empty means the feed matches on name and position.
type Player struct {
	ID, Name, Position, SleeperID, ESPNID string
}

// SleeperPlayer is the part of a Sleeper player record the news read uses.
type SleeperPlayer struct {
	ID, Name, Position, Team, Status, BodyPart, ESPNID string
	NewsAt                                             time.Time
}

// Note is one ESPN fantasy note.
type Note struct {
	At   time.Time
	Text string
}

// Item is one line of the news read, before the server labels it.
type Item struct {
	At                                   time.Time
	Player, Name, Position, NFLTeam, Tag string
	Text, Source                         string
}

// Reading is what the feed answers: the sleeper snapshot's time, whether any source failed, and the items.
type Reading struct {
	AsOf  time.Time
	Stale bool
	Items []Item
}

type sleeperRecord struct {
	PlayerID    string          `json:"player_id"`
	FullName    string          `json:"full_name"`
	Position    string          `json:"position"`
	Team        string          `json:"team"`
	Injury      string          `json:"injury_status"`
	BodyPart    string          `json:"injury_body_part"`
	NewsUpdated json.RawMessage `json:"news_updated"`
	ESPNID      json.RawMessage `json:"espn_id"`
}

// ParseSleeper reads the Sleeper player table, keeping the fields the news read uses.
func ParseSleeper(r io.Reader) (map[string]SleeperPlayer, error) {
	var raw map[string]sleeperRecord
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse Sleeper players: %w", err)
	}
	out := make(map[string]SleeperPlayer, len(raw))
	for key, rec := range raw {
		id := rec.PlayerID
		if id == "" {
			id = key
		}
		player := SleeperPlayer{ID: id, Name: strings.TrimSpace(rec.FullName), Position: rec.Position,
			Team: rec.Team, Status: strings.TrimSpace(rec.Injury), BodyPart: strings.TrimSpace(rec.BodyPart),
			ESPNID: jsonScalar(rec.ESPNID)}
		if ms, err := strconv.ParseInt(jsonScalar(rec.NewsUpdated), 10, 64); err == nil && ms > 0 {
			player.NewsAt = time.UnixMilli(ms).UTC()
		}
		out[id] = player
	}
	return out, nil
}

// jsonScalar reads a JSON string or number as text; null or absent is empty.
func jsonScalar(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return strings.TrimSpace(text)
	}
	return s
}

type espnEnvelope struct {
	Feed []struct {
		Headline     string `json:"headline"`
		Description  string `json:"description"`
		Published    string `json:"published"`
		LastModified string `json:"lastModified"`
	} `json:"feed"`
}

var htmlTag = regexp.MustCompile(`<[^>]*>`)

// ParseESPN reads one player's ESPN fantasy notes, headline first, HTML stripped and capped.
func ParseESPN(body []byte) ([]Note, error) {
	var env espnEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse ESPN news: %w", err)
	}
	out := make([]Note, 0, len(env.Feed))
	for _, entry := range env.Feed {
		text := clean(entry.Headline)
		if text == "" {
			text = clean(entry.Description)
		}
		if text == "" {
			continue
		}
		at := parseTime(entry.Published)
		if at.IsZero() {
			at = parseTime(entry.LastModified)
		}
		out = append(out, Note{At: at, Text: text})
	}
	return out, nil
}

func clean(s string) string {
	s = strings.Join(strings.Fields(html.UnescapeString(htmlTag.ReplaceAllString(s, " "))), " ")
	if r := []rune(s); len(r) > maxText {
		s = string(r[:maxText])
	}
	return s
}

func parseTime(s string) time.Time {
	at, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return at.UTC()
}

// Tag maps a Sleeper injury status to the news tag. A status the table does not name is untagged.
func Tag(status string) string {
	switch status {
	case "Questionable", "Doubtful":
		return "INJ"
	case "Out":
		return "OUT"
	case "IR":
		return "IR"
	case "Active":
		return "ACT"
	}
	return ""
}

// tagOr returns the status tag, or NOTE when the status names none.
func tagOr(status string) string {
	if tag := Tag(status); tag != "" {
		return tag
	}
	return "NOTE"
}

// Normalize folds a name for matching: lower case, letters and digits only.
func Normalize(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type held struct {
	notes  []Note
	at     time.Time // last good read
	tried  time.Time // last attempt
	failed bool
}

// Feed holds the latest Sleeper table and each wanted player's ESPN notes. Reads never touch the
// network; Run fetches on the cadences.
type Feed struct {
	client *http.Client
	now    func() time.Time
	logf   func(string, ...any)
	mu     sync.Mutex

	requested bool
	sleeper   map[string]SleeperPlayer
	byName    map[string]string // normalized name and position -> Sleeper id
	sleeperAt time.Time
	sleeperTr time.Time
	sleeperFl bool
	notes     map[string]*held
	wanted    map[string]time.Time
}

// NewFeed starts an empty feed; transport nil is http.DefaultTransport.
func NewFeed(transport http.RoundTripper, now func() time.Time) *Feed {
	if now == nil {
		now = time.Now
	}
	return &Feed{client: &http.Client{Timeout: 30 * time.Second, Transport: transport}, now: now, logf: log.Printf,
		notes: map[string]*held{}, wanted: map[string]time.Time{}, byName: map[string]string{}}
}

// Read answers the held items for the named players, newest first, from the last seven days.
func (f *Feed) Read(players []Player) Reading {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.now()
	f.requested = true
	reading := Reading{AsOf: f.sleeperAt, Stale: f.sleeperFl || f.sleeperAt.IsZero(), Items: []Item{}}
	for _, p := range players {
		rec, hasRec := f.lookup(p)
		espnID := p.ESPNID
		if espnID == "" && hasRec {
			espnID = rec.ESPNID
		}
		var notes []Note
		if espnID != "" {
			f.wanted[espnID] = now
			h := f.notes[espnID]
			if h == nil || h.at.IsZero() || h.failed {
				reading.Stale = true
			}
			if h != nil {
				notes = h.notes
			}
		}
		reading.Items = append(reading.Items, playerItems(p, rec, hasRec, notes, now)...)
	}
	slices.SortStableFunc(reading.Items, func(a, b Item) int {
		if c := b.At.Compare(a.At); c != 0 {
			return c
		}
		return strings.Compare(a.Player+a.Text, b.Player+b.Text)
	})
	if len(reading.Items) > MaxItems {
		reading.Items = reading.Items[:MaxItems]
	}
	return reading
}

// playerItems returns one player's ESPN notes inside the window; with none, his Sleeper status change.
func playerItems(p Player, rec SleeperPlayer, hasRec bool, notes []Note, now time.Time) []Item {
	status, team := "", ""
	if hasRec {
		status, team = rec.Status, rec.Team
	}
	var out []Item
	for _, n := range notes {
		if n.At.IsZero() || now.Sub(n.At) > Window {
			continue
		}
		out = append(out, Item{At: n.At, Player: p.ID, Name: p.Name, Position: p.Position, NFLTeam: team,
			Tag: tagOr(status), Text: n.Text, Source: "espn"})
	}
	if len(out) == 0 && hasRec && rec.Status != "" && !rec.NewsAt.IsZero() && now.Sub(rec.NewsAt) <= Window {
		text := rec.Status
		if rec.BodyPart != "" {
			text += " (" + rec.BodyPart + ")"
		}
		out = append(out, Item{At: rec.NewsAt, Player: p.ID, Name: p.Name, Position: p.Position, NFLTeam: team,
			Tag: tagOr(rec.Status), Text: text, Source: "sleeper"})
	}
	return out
}

// lookup finds a player's Sleeper record by his crosswalk id, else by normalized name and position.
// Caller holds f.mu.
func (f *Feed) lookup(p Player) (SleeperPlayer, bool) {
	if p.SleeperID != "" {
		if rec, ok := f.sleeper[p.SleeperID]; ok {
			return rec, true
		}
	}
	if id, ok := f.byName[Normalize(p.Name)+"|"+strings.ToUpper(p.Position)]; ok {
		return f.sleeper[id], true
	}
	return SleeperPlayer{}, false
}

// Refresh runs one pass: the Sleeper table when due and asked for, then the ESPN notes of wanted
// players whose copy is due, at most MaxInFlight at a time. A failed fetch keeps the last good copy.
func (f *Feed) Refresh(ctx context.Context) {
	f.mu.Lock()
	now := f.now()
	doSleeper := f.requested && now.Sub(f.sleeperTr) >= SleeperEvery
	f.mu.Unlock()
	if doSleeper {
		f.refreshSleeper(ctx, now)
	}
	f.mu.Lock()
	var due []string
	for id, asked := range f.wanted {
		if now.Sub(asked) > wantedFor {
			delete(f.wanted, id)
			continue
		}
		if h := f.notes[id]; h == nil || now.Sub(h.tried) >= ESPNEvery {
			due = append(due, id)
		}
	}
	f.mu.Unlock()
	slices.Sort(due)
	sem := make(chan struct{}, MaxInFlight)
	var wg sync.WaitGroup
	for _, id := range due {
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }()
			f.refreshESPN(ctx, id)
		}(id)
	}
	wg.Wait()
}

func (f *Feed) refreshSleeper(ctx context.Context, now time.Time) {
	players, err := f.fetchSleeper(ctx)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sleeperTr = now
	if err != nil {
		f.sleeperFl = true
		f.logf("nflnews: sleeper: %v", err)
		return
	}
	byName := make(map[string]string, len(players))
	for id, p := range players {
		if p.Name != "" {
			byName[Normalize(p.Name)+"|"+strings.ToUpper(p.Position)] = id
		}
	}
	f.sleeper, f.byName, f.sleeperAt, f.sleeperFl = players, byName, now, false
}

// fetchSleeper reads and parses the table without holding the feed's lock.
func (f *Feed) fetchSleeper(ctx context.Context) (map[string]SleeperPlayer, error) {
	body, err := f.get(ctx, SleeperURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()
	return ParseSleeper(io.LimitReader(body, maxBody))
}

func (f *Feed) refreshESPN(ctx context.Context, id string) {
	body, err := f.getBytes(ctx, espnURL+url.QueryEscape(id))
	now := f.now()
	f.mu.Lock()
	defer f.mu.Unlock()
	h := f.notes[id]
	if h == nil {
		h = &held{}
		f.notes[id] = h
	}
	h.tried = now
	if err == nil {
		var notes []Note
		notes, err = ParseESPN(body)
		if err == nil {
			h.notes, h.at, h.failed = notes, now, false
			return
		}
	}
	h.failed = true
	f.logf("nflnews: ESPN player %s: %v", id, err)
}

func (f *Feed) get(ctx context.Context, address string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("news request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("news fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("news fetch: status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (f *Feed) getBytes(ctx context.Context, address string) ([]byte, error) {
	body, err := f.get(ctx, address)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("news read: %w", err)
	}
	return raw, nil
}

// Run refreshes the feed every minute until ctx ends; the cadences live in Refresh.
func Run(ctx context.Context, f *Feed) {
	for {
		f.Refresh(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
	}
}
