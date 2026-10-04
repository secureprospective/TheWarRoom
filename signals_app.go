package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/college"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/feeds"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

// signalsTimeout bounds one signals load. The first load on a new history backfills every
// season, about 100 MB of files.
const signalsTimeout = 30 * time.Minute

// A file is loaded again once it is older than its window: the current season and single files
// change weekly, closed seasons only by correction. Loads of unchanged data write nothing.
const (
	openFileWindow   = 20 * time.Hour
	closedFileWindow = 30 * 24 * time.Hour
)

// collegeFirstSeason is the first college season loaded: the final college year of a player
// drafted in 2014, the oldest class still common on NFL rosters in 2021.
const collegeFirstSeason = 2013

// Load outcomes.
const (
	loadDone    = "loaded"
	loadCurrent = "current"
	loadFailed  = "failed"
	loadSkipped = "skipped"
)

// SignalLoad is one file's outcome in a signals load.
type SignalLoad struct {
	Feed       string   `json:"feed"`
	Season     int      `json:"season"`
	Status     string   `json:"status"`
	Rows       int      `json:"rows"`
	Facts      int      `json:"facts"`
	Added      int      `json:"added"`
	Unresolved int      `json:"unresolved"`
	Missing    []string `json:"missing"`
	Error      string   `json:"error"`
}

// SignalsReport is one signals load: every file it looked at.
type SignalsReport struct {
	OK         bool         `json:"ok"`
	Error      string       `json:"error"`
	StartedAt  string       `json:"startedAt"`
	FinishedAt string       `json:"finishedAt"`
	Loads      []SignalLoad `json:"loads"`
}

// LoadSignals loads every signal file that is due, then reports what it did.
func (a *App) LoadSignals() SignalsReport {
	if err := a.ready(); err != nil {
		return SignalsReport{Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, signalsTimeout)
	defer cancel()
	return a.loadSignals(ctx)
}

// loadSignals links the crosswalk first, so the facts resolve as they land, then loads each due
// file of each feed and each due college season. A file that fails is recorded against its
// source and the rest still load (R11).
func (a *App) loadSignals(ctx context.Context) SignalsReport {
	a.loadingSignals.Lock()
	defer a.loadingSignals.Unlock()
	rep := SignalsReport{OK: true, StartedAt: time.Now().Format(time.RFC3339)}
	if err := a.linkDirectory(ctx); err != nil {
		log.Printf("the war room: signals: directory not refreshed, facts may wait for a match: %v", err)
	}
	client := &http.Client{Timeout: rasFetchTimeout, Transport: a.fetches}
	reg := a.history.Registry()
	for _, f := range reg.Feeds {
		for _, season := range a.feedSeasons(f) {
			rep.Loads = append(rep.Loads, a.loadFeedFile(ctx, client, reg, f, season))
		}
	}
	rep.Loads = append(rep.Loads, a.loadCollege(ctx, client, reg)...)
	rep.FinishedAt = time.Now().Format(time.RFC3339)
	var added, failed int
	for _, l := range rep.Loads {
		added += l.Added
		if l.Status == loadFailed {
			failed++
		}
	}
	log.Printf("the war room: signals: %d files looked at, %d failed, %d new values", len(rep.Loads), failed, added)
	a.signalsMu.Lock()
	a.signals = rep
	a.signalsMu.Unlock()
	return rep
}

// linkDirectory refreshes the player directory from DynastyProcess.
func (a *App) linkDirectory(ctx context.Context) error {
	lk, err := a.directory(ctx)
	if err != nil {
		return err
	}
	cw, err := a.fetchCrosswalk(ctx)
	if err != nil {
		return err
	}
	_, err = a.linkCrosswalk(ctx, cw, lk)
	return err
}

// feedSeasons lists the seasons a feed has files for: one per season from its first through the
// league's, or a single 0 for a single file.
func (a *App) feedSeasons(f measures.Feed) []int {
	if f.FirstSeason == 0 {
		return []int{0}
	}
	return seasonRange(f.FirstSeason, a.season)
}

func seasonRange(first, last int) []int {
	var out []int
	for s := first; s <= last; s++ {
		out = append(out, s)
	}
	return out
}

// due reports whether a file needs loading: never loaded, or older than its window. A file is
// open when it can still change week to week.
func (a *App) due(ctx context.Context, url string, open bool) (bool, error) {
	at, ok, err := a.history.LastLoadOf(ctx, url)
	if err != nil {
		return true, fmt.Errorf("app: is %s due: %w", url, err)
	}
	if !ok {
		return true, nil
	}
	window := closedFileWindow
	if open {
		window = openFileWindow
	}
	return time.Since(at) > window, nil
}

func (a *App) loadFeedFile(ctx context.Context, client *http.Client, reg *measures.Registry, f measures.Feed, season int) SignalLoad {
	out := SignalLoad{Feed: f.Name, Season: season}
	url := f.URLFor(season)
	if due, err := a.due(ctx, url, season == 0 || season >= a.season); err != nil || !due {
		out.Status = loadCurrent
		return withError(out, err)
	}
	res, err := feeds.Read(ctx, client, reg, f, season)
	if err != nil {
		return a.failedLoad(ctx, out, f.Source, err)
	}
	out.Rows, out.Missing = res.Rows, res.Missing
	return a.ingest(ctx, out, res.Batch)
}

// loadCollege loads each due college season. Without a CFBD key the college signal is skipped,
// which is a configuration gap, not an outage, so nothing is recorded against the source.
func (a *App) loadCollege(ctx context.Context, client *http.Client, reg *measures.Registry) []SignalLoad {
	key := strings.TrimSpace(os.Getenv(cfbdEnvVar))
	if key == "" {
		return []SignalLoad{{Feed: college.Source, Status: loadSkipped, Error: cfbdEnvVar + " is not set"}}
	}
	known, err := a.history.KnownIDs(ctx, college.IDType)
	if err != nil {
		return []SignalLoad{{Feed: college.Source, Status: loadFailed, Error: err.Error()}}
	}
	var out []SignalLoad
	for _, season := range seasonRange(collegeFirstSeason, a.season) {
		l := SignalLoad{Feed: college.Source, Season: season}
		url := college.SeasonStatsURL + "?year=" + strconv.Itoa(season)
		if due, err := a.due(ctx, url, season >= a.season); err != nil || !due {
			l.Status = loadCurrent
			out = append(out, withError(l, err))
			continue
		}
		b, err := college.Fetch(ctx, client, college.SeasonStatsURL, reg, key, season, func(id string) bool { return known[id] })
		if err != nil {
			out = append(out, a.failedLoad(ctx, l, college.Source, err))
			continue
		}
		out = append(out, a.ingest(ctx, l, b))
	}
	return out
}

func (a *App) ingest(ctx context.Context, l SignalLoad, b measures.Batch) SignalLoad {
	r, err := a.history.Ingest(ctx, b)
	if err != nil {
		l.Status = loadFailed
		return withError(l, err)
	}
	l.Status, l.Facts, l.Added, l.Unresolved = loadDone, r.Facts, r.Added, r.Unresolved
	return l
}

func (a *App) failedLoad(ctx context.Context, l SignalLoad, source string, err error) SignalLoad {
	l.Status = loadFailed
	if lerr := a.history.LoadFailed(ctx, source, err); lerr != nil {
		err = errors.Join(err, lerr)
	}
	log.Printf("the war room: signals: %s %d: %v", l.Feed, l.Season, err)
	return withError(l, err)
}

func withError(l SignalLoad, err error) SignalLoad {
	if err != nil {
		l.Error = fmt.Sprint(err)
	}
	return l
}

// signalsInBackground runs the launch signals load after startup, off the message loop.
func (a *App) signalsInBackground(parent context.Context) {
	go func() {
		if a.ready() != nil {
			return
		}
		ctx, cancel := context.WithTimeout(parent, signalsTimeout)
		defer cancel()
		a.loadSignals(ctx)
	}()
}

// SourceView is one source's health as the Sources screen shows it.
type SourceView struct {
	Source      string `json:"source"`
	Name        string `json:"name"`
	State       string `json:"state"`
	LastSuccess string `json:"lastSuccess"`
	LastError   string `json:"lastError"`
}

// PositionShare is how many rostered players at a position a signal has data for.
type PositionShare struct {
	Position string `json:"position"`
	Rostered int    `json:"rostered"`
	WithData int    `json:"withData"`
}

// SeasonCount is a signal's reach in one season: players with data, and source ids still waiting
// for a directory match.
type SeasonCount struct {
	Season  int `json:"season"`
	Players int `json:"players"`
	Waiting int `json:"waiting"`
}

// FeedView is one signal: its freshness, its reach by season, and its coverage of the rosters.
// CoverageSeason is the season the coverage is measured in: the latest with data, 0 for facts
// that belong to no season, or -1 across every college season.
type FeedView struct {
	Feed           string          `json:"feed"`
	Source         string          `json:"source"`
	LastLoaded     string          `json:"lastLoaded"`
	Fresh          bool            `json:"fresh"`
	Seasons        []SeasonCount   `json:"seasons"`
	CoverageSeason int             `json:"coverageSeason"`
	Coverage       []PositionShare `json:"coverage"`
}

// SignalsView is the Sources screen: every source's health, every signal's coverage, and the
// latest load since launch.
type SignalsView struct {
	OK       bool          `json:"ok"`
	Error    string        `json:"error"`
	Season   int           `json:"season"`
	Sources  []SourceView  `json:"sources"`
	Feeds    []FeedView    `json:"feeds"`
	LastLoad SignalsReport `json:"lastLoad"`
}

// allCollegeSeasons marks coverage measured across every college season.
const allCollegeSeasons = -1

// GetSignals reports source health, freshness and coverage from what history holds.
func (a *App) GetSignals() SignalsView {
	if err := a.ready(); err != nil {
		return SignalsView{Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()
	view, err := a.signalsView(ctx)
	if err != nil {
		return SignalsView{Error: err.Error()}
	}
	a.signalsMu.Lock()
	view.LastLoad = a.signals
	a.signalsMu.Unlock()
	return view
}

func (a *App) signalsView(ctx context.Context) (SignalsView, error) {
	view := SignalsView{OK: true, Season: a.season}
	health, err := a.history.SourceHealth(ctx)
	if err != nil {
		return SignalsView{}, fmt.Errorf("app: signals view: %w", err)
	}
	reg := a.history.Registry()
	maxAge := map[string]time.Duration{}
	for _, s := range reg.Sources {
		maxAge[s.ID] = s.MaxAge
	}
	for _, h := range health {
		v := SourceView{Source: h.Source, State: string(h.State), LastError: h.LastError}
		if src, ok := reg.Source(h.Source); ok {
			v.Name = src.Name
		}
		if !h.LastSuccess.IsZero() {
			v.LastSuccess = h.LastSuccess.Format(time.RFC3339)
		}
		view.Sources = append(view.Sources, v)
	}
	positions := a.rosteredPositions(ctx)
	for _, f := range reg.Feeds {
		var fields []string
		for _, sf := range reg.FeedFields(f) {
			fields = append(fields, sf.Measure)
		}
		fv, err := a.feedView(ctx, f.Name, f.Source, f.URLFor(a.season), a.feedSeasons(f), fields, positions, maxAge[f.Source])
		if err != nil {
			return SignalsView{}, err
		}
		view.Feeds = append(view.Feeds, fv)
	}
	var collegeFields []string
	for _, sf := range reg.Fields {
		if sf.Source == college.Source {
			collegeFields = append(collegeFields, sf.Measure)
		}
	}
	fv, err := a.feedView(ctx, college.Source, college.Source, college.SeasonStatsURL+"?year="+strconv.Itoa(a.season),
		seasonRange(collegeFirstSeason, a.season), collegeFields, positions, maxAge[college.Source])
	if err != nil {
		return SignalsView{}, err
	}
	view.Feeds = append(view.Feeds, fv)
	return view, nil
}

// feedView measures one signal. Coverage is taken in the latest season with data, or across
// every season for college, where a player's seasons are all behind him.
func (a *App) feedView(ctx context.Context, name, source, currentURL string, seasons []int, fields []string,
	positions map[string]string, maxAge time.Duration) (FeedView, error) {
	fv := FeedView{Feed: name, Source: source}
	at, ok, err := a.history.LastLoadOf(ctx, currentURL)
	if err != nil {
		return FeedView{}, fmt.Errorf("app: %s freshness: %w", name, err)
	}
	if ok {
		fv.LastLoaded, fv.Fresh = at.Format(time.RFC3339), time.Since(at) <= maxAge
	}
	union := map[string]bool{}
	var latest map[string]bool
	for _, season := range seasons {
		players, waiting, err := a.history.PlayersWithData(ctx, source, season, fields)
		if err != nil {
			return FeedView{}, fmt.Errorf("app: %s coverage: %w", name, err)
		}
		fv.Seasons = append(fv.Seasons, SeasonCount{Season: season, Players: len(players), Waiting: waiting})
		for id := range players {
			union[id] = true
		}
		if len(players) > 0 {
			latest, fv.CoverageSeason = players, season
		}
	}
	if source == college.Source {
		latest, fv.CoverageSeason = union, allCollegeSeasons
	}
	fv.Coverage = coverage(positions, latest)
	return fv, nil
}

// rosteredPositions maps every rostered player to his league position. Without the players
// directory (MFL down) the coverage table is empty rather than wrong.
func (a *App) rosteredPositions(ctx context.Context) map[string]string {
	lk, err := a.directory(ctx)
	if err != nil {
		log.Printf("the war room: signals coverage: %v", err)
		return map[string]string{}
	}
	out := map[string]string{}
	for id := range rosteredBy(a.league.Reader()) {
		if f, ok := lk.Facts(id); ok {
			out[id] = string(f.Position)
		}
	}
	return out
}

// coverage counts, per league position, the rostered players and those with data.
func coverage(positions map[string]string, withData map[string]bool) []PositionShare {
	shares := map[string]*PositionShare{}
	for id, pos := range positions {
		if shares[pos] == nil {
			shares[pos] = &PositionShare{Position: pos}
		}
		shares[pos].Rostered++
		if withData[id] {
			shares[pos].WithData++
		}
	}
	var out []PositionShare
	for _, p := range reportPositions() {
		if s, ok := shares[string(p)]; ok {
			out = append(out, *s)
		}
	}
	return out
}
