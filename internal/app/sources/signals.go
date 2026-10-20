package sources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/college"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/contracts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/feeds"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

// RASFetchTimeout bounds the static-CDN fetches (crosswalk, research files); a tight budget
// fails on a cold cache.
const RASFetchTimeout = 90 * time.Second

// CFBDEnvVar holds the CollegeFootballData token, read at wire time. Unset skips the CFBD signals.
const CFBDEnvVar = "CFBD_API_KEY"

// SignalsTimeout bounds one signals load. The first load on a new history backfills every
// season, about 100 MB of files.
const SignalsTimeout = 30 * time.Minute

// A file is loaded again once it is older than its window: the current season and single files
// change weekly, closed seasons only by correction. Loads of unchanged data write nothing.
const (
	OpenFileWindow   = 20 * time.Hour
	ClosedFileWindow = 30 * 24 * time.Hour
)

// CollegeFirstSeason is the first college season loaded: the final college year of a player
// drafted in 2014, the oldest class still common on NFL rosters in 2021.
const CollegeFirstSeason = 2013

// Load outcomes.
const (
	LoadDone    = "loaded"
	LoadCurrent = "current"
	LoadFailed  = "failed"
	LoadSkipped = "skipped"
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
	OK            bool         `json:"ok"`
	Error         string       `json:"error"`
	StartedAt     string       `json:"startedAt"`
	FinishedAt    string       `json:"finishedAt"`
	Loads         []SignalLoad `json:"loads"`
	CrosswalkNote string       `json:"crosswalkNote"`
}

// LoadSignals links the crosswalk first, so the facts resolve as they land, then loads each due
// file of each feed and each due college season. A file that fails is recorded against its
// source and the rest still load (R11).
func (a Service) LoadSignals(ctx context.Context, link func(context.Context) error) SignalsReport {
	rep := SignalsReport{OK: true, StartedAt: time.Now().Format(time.RFC3339),
		CrosswalkNote: "Player crosswalk reloaded and relinked before signals; its Loaded time records that relink."}
	if err := link(ctx); err != nil {
		rep.CrosswalkNote = "Player crosswalk relink failed; its last Loaded time is kept: " + err.Error()
		log.Printf("the war room: signals: directory not refreshed, facts may wait for a match: %v", err)
	}
	client := &http.Client{Timeout: RASFetchTimeout, Transport: a.Transport}
	reg := a.Signals.Registry()
	for _, f := range reg.Feeds {
		for _, season := range a.feedSeasons(f) {
			rep.Loads = append(rep.Loads, a.loadFeedFile(ctx, client, reg, f, season))
		}
	}
	rep.Loads = append(rep.Loads, a.loadCollege(ctx, client, reg)...)
	rep.Loads = append(rep.Loads, a.loadContracts(ctx, client))
	rep.FinishedAt = time.Now().Format(time.RFC3339)
	var added, failed int
	for _, l := range rep.Loads {
		added += l.Added
		if l.Status == LoadFailed {
			failed++
		}
	}
	log.Printf("the war room: signals: %d files looked at, %d failed, %d new values", len(rep.Loads), failed, added)
	return rep
}

// feedSeasons lists the seasons a feed has files for: one per season from its first through the
// league's, or a single 0 for a single file.
func (a Service) feedSeasons(f measures.Feed) []int {
	if f.FirstSeason == 0 {
		return []int{0}
	}
	return seasonRange(f.FirstSeason, a.Season)
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
func (a Service) due(ctx context.Context, url string, open bool) (bool, error) {
	at, ok, err := a.Signals.LastLoadOf(ctx, url)
	if err != nil {
		return true, fmt.Errorf("app: is %s due: %w", url, err)
	}
	if !ok {
		return true, nil
	}
	window := ClosedFileWindow
	if open {
		window = OpenFileWindow
	}
	return time.Since(at) > window, nil
}

func (a Service) loadFeedFile(ctx context.Context, client *http.Client, reg *measures.Registry, f measures.Feed, season int) SignalLoad {
	out := SignalLoad{Feed: f.Name, Season: season}
	url := f.URLFor(season)
	if due, err := a.due(ctx, url, season == 0 || season >= a.Season); err != nil || !due {
		out.Status = LoadCurrent
		return withError(out, err)
	}
	res, err := feeds.Read(ctx, client, reg, f, season)
	if err != nil {
		return a.failedLoad(ctx, out, f.Source, err)
	}
	out.Rows, out.Missing = res.Rows, res.Missing
	return a.ingest(ctx, out, res.Batch)
}

// ContractsFeed names the contracts file in the Signals console.
const ContractsFeed = "contracts"

// loadContracts loads NFL contracts when the file is due. It is read as a closed file, every 30
// days: nflverse rebuilds the 11 MB file daily and every body fetched is archived, so a daily read
// would add gigabytes a year for terms that change a few times a season.
func (a Service) loadContracts(ctx context.Context, client *http.Client) SignalLoad {
	l := SignalLoad{Feed: ContractsFeed}
	if due, err := a.due(ctx, contracts.SourceURL, false); err != nil || !due {
		l.Status = LoadCurrent
		return withError(l, err)
	}
	known, err := a.Signals.KnownIDs(ctx, contracts.IDType)
	if err != nil {
		l.Status = LoadFailed
		return withError(l, err)
	}
	b, err := contracts.Fetch(ctx, client, contracts.SourceURL, func(id string) bool { return known[id] })
	if err != nil {
		return a.failedLoad(ctx, l, contracts.Source, err)
	}
	return a.ingest(ctx, l, b)
}

// loadCollege loads each due college season. Without a CFBD key the college signal is skipped,
// which is a configuration gap, not an outage, so nothing is recorded against the source.
func (a Service) loadCollege(ctx context.Context, client *http.Client, reg *measures.Registry) []SignalLoad {
	key := a.CollegeKey
	if key == "" {
		return []SignalLoad{{Feed: college.Source, Status: LoadSkipped, Error: CFBDEnvVar + " is not set"}}
	}
	known, err := a.Signals.KnownIDs(ctx, college.IDType)
	if err != nil {
		return []SignalLoad{{Feed: college.Source, Status: LoadFailed, Error: err.Error()}}
	}
	var out []SignalLoad
	for _, season := range seasonRange(CollegeFirstSeason, a.Season) {
		l := SignalLoad{Feed: college.Source, Season: season}
		url := college.SeasonStatsURL + "?year=" + strconv.Itoa(season)
		if due, err := a.due(ctx, url, season >= a.Season); err != nil || !due {
			l.Status = LoadCurrent
			if err == nil {
				l = a.remapCollege(ctx, reg, l, url, known)
			}
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

// remapCollege re-reads a season that is not due from its archived body when that body now maps a
// measure the season holds no value of: a mapping added after the season was loaded (the college
// team's quarterback hurries, 2026-10) reaches every stored season without spending a CFBD call.
func (a Service) remapCollege(ctx context.Context, reg *measures.Registry, l SignalLoad, url string,
	known map[string]bool) SignalLoad {
	held, err := a.Signals.MeasuresHeld(ctx, college.Source, l.Season)
	if err != nil {
		return withError(l, err)
	}
	var b measures.Batch
	var mapErr error
	_, err = a.Signals.ArchivedBodies(ctx, url, func(_ string, body []byte, _ time.Time) bool {
		b, mapErr = college.Map(reg, l.Season, body, func(id string) bool { return known[id] })
		return true
	})
	if err = errors.Join(err, mapErr); err != nil || !mapsNewMeasure(reg, b, held) {
		return withError(l, err)
	}
	return a.ingest(ctx, l, b)
}

// mapsNewMeasure reports whether b carries a value of a measure held does not list.
func mapsNewMeasure(reg *measures.Registry, b measures.Batch, held map[string]bool) bool {
	for _, f := range b.Facts {
		if sf, ok := reg.Field(b.Source, f.Field); ok && !held[sf.Measure] {
			return true
		}
	}
	return false
}

func (a Service) ingest(ctx context.Context, l SignalLoad, b measures.Batch) SignalLoad {
	r, err := a.Signals.Ingest(ctx, b)
	if err != nil {
		l.Status = LoadFailed
		return withError(l, err)
	}
	l.Status, l.Facts, l.Added, l.Unresolved = LoadDone, r.Facts, r.Added, r.Unresolved
	return l
}

func (a Service) failedLoad(ctx context.Context, l SignalLoad, source string, err error) SignalLoad {
	l.Status = LoadFailed
	if lerr := a.Signals.LoadFailed(ctx, source, err); lerr != nil {
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

// SignalsView is every signal's freshness and coverage, as history holds them.
type SignalsView struct {
	Feeds []FeedView
}

// AllSeasons marks coverage measured across every season: college seasons, all behind a player,
// and contract signings, the latest of which is the one in force.
const AllSeasons = -1

// ContractsFirstSeason is the first signing season the Signals console counts: the oldest
// contract a rostered veteran could still be playing on.
const ContractsFirstSeason = 2014

func (a Service) SignalsView(ctx context.Context, positions map[string]string) (SignalsView, error) {
	var view SignalsView
	reg := a.Signals.Registry()
	maxAge := map[string]time.Duration{}
	for _, s := range reg.Sources {
		maxAge[s.ID] = s.MaxAge
	}
	for _, f := range reg.Feeds {
		var fields []string
		for _, sf := range reg.FeedFields(f) {
			fields = append(fields, sf.Measure)
		}
		fv, err := a.feedView(ctx, f.Name, f.Source, f.URLFor(a.Season), a.feedSeasons(f), fields, positions, maxAge[f.Source])
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
	fv, err := a.feedView(ctx, college.Source, college.Source, college.SeasonStatsURL+"?year="+strconv.Itoa(a.Season),
		seasonRange(CollegeFirstSeason, a.Season), collegeFields, positions, maxAge[college.Source])
	if err != nil {
		return SignalsView{}, err
	}
	view.Feeds = append(view.Feeds, fv)
	var contractFields []string
	for _, sf := range reg.Fields {
		if strings.HasPrefix(sf.Field, "contracts.") {
			contractFields = append(contractFields, sf.Measure)
		}
	}
	fv, err = a.feedView(ctx, ContractsFeed, contracts.Source, contracts.SourceURL,
		seasonRange(ContractsFirstSeason, a.Season), contractFields, positions, ClosedFileWindow+24*time.Hour)
	if err != nil {
		return SignalsView{}, err
	}
	view.Feeds = append(view.Feeds, fv)
	return view, nil
}

// feedView measures one signal. Coverage is taken in the latest season with data, or across
// every season for college, where a player's seasons are all behind him, and for contracts.
func (a Service) feedView(ctx context.Context, name, source, currentURL string, seasons []int, fields []string,
	positions map[string]string, maxAge time.Duration) (FeedView, error) {
	fv := FeedView{Feed: name, Source: source}
	at, ok, err := a.Signals.LastLoadOf(ctx, currentURL)
	if err != nil {
		return FeedView{}, fmt.Errorf("app: %s freshness: %w", name, err)
	}
	if ok {
		fv.LastLoaded, fv.Fresh = at.Format(time.RFC3339), time.Since(at) <= maxAge
	}
	union := map[string]bool{}
	var latest map[string]bool
	for _, season := range seasons {
		players, waiting, err := a.Signals.PlayersWithData(ctx, source, season, fields)
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
	if name == college.Source || name == ContractsFeed {
		latest, fv.CoverageSeason = union, AllSeasons
	}
	fv.Coverage = coverage(positions, latest)
	return fv, nil
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
