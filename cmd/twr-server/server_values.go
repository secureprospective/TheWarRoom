package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/sources"
	"github.com/secureprospective/TheWarRoom/internal/appvalue"
	"github.com/secureprospective/TheWarRoom/internal/archive"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	recordstore "github.com/secureprospective/TheWarRoom/internal/store/playerrecord"
)

// Player values on the server: the hosted server collects the inputs the desktop collects (nflverse
// weekly statistics, snaps, injuries, players, draft picks, combine and contracts, into the reference
// file) once a day, writes the player record from those facts after each load, and prepares
// every league's values in the background the way the desktop does. A request only ever reads a
// completed pass. College measures need a CollegeFootballData key the server does not hold, so the
// model reads without them (R11). Operator switch: -signals (launcher: /etc/twr/signals).
const (
	signalsRefresh = 24 * time.Hour
	signalsFirst   = 5 * time.Minute // after start, once live scores and official statistics are back
)

// serverValues prepares and reads player values for every league on the server.
type serverValues struct {
	pins      *leaguePins
	reference *history.Store
	pools     *db.Pools
	dir       string
	prep      appvalue.Preparation
	// recorded closes once player records exist: the first input load writes them, and until then
	// there is nothing to prepare and every read says values are being prepared.
	recorded chan struct{}
	once     sync.Once
}

func newServerValues(pins *leaguePins, dataDir string) *serverValues {
	return &serverValues{pins: pins, reference: pins.files.family.Over(pins.files.reference),
		pools: pins.files.reference, dir: filepath.Join(dataDir, "prepared-values"),
		recorded: make(chan struct{})}
}

// NativeValues reads a completed pass, or says one is being prepared; it never computes one.
func (v *serverValues) NativeValues(ctx context.Context, league, franchise string, season int) (appvalue.Reading, error) {
	select {
	case <-v.recorded:
	default:
		return appvalue.Preparing(league, season, franchise), nil
	}
	return v.prep.Read(ctx, v.source(), league, franchise, season) //nolint:wrapcheck // the service names it
}

func (v *serverValues) records() *recordstore.Store { return recordstore.New(v.pools, v.reference) }

func (v *serverValues) source() appvalue.Source {
	return appvalue.Source{Facts: v.records(), Held: v.reference, Records: v.leagueRecords,
		Params: params.DefaultSet, Now: time.Now, LeagueRevision: v.revision,
		NextFeatureAt: v.reference.NextFeatureAt, Evidence: v.reference.ValueEvidence,
		Saved: appvalue.SavedFiles{Dir: v.dir}}
}

func (v *serverValues) leagueRecords(ctx context.Context, league string) ([]commissioner.Record, error) {
	store, err := v.pins.League(ctx, league)
	if err != nil {
		return nil, err
	}
	return store.SharedCommissionerRecords(ctx, league) //nolint:wrapcheck // the value read names it
}

// revision changes when the reference file or the league's own file commits: that is all one league's
// pass reads, so another league's commit (a live Sunday is full of them) leaves its pass in force.
// League "" (discovery) spans every league.
func (v *serverValues) revision(ctx context.Context, league string) (int64, error) {
	sum := fnv.New64a()
	all := []*db.Pools{v.pools}
	names := []string{league}
	if league == "" {
		names = v.pins.files.leagues()
	}
	for _, name := range names {
		if pools, ok := v.pins.files.pools(name); ok {
			all = append(all, pools)
		}
	}
	for _, pools := range all {
		revision, err := pools.Revision(ctx)
		if err != nil {
			return 0, fmt.Errorf("value revision: %w", err)
		}
		_, _ = fmt.Fprintf(sum, "%d;", revision)
	}
	return int64(sum.Sum64() >> 1), nil
}

func (v *serverValues) markRecorded() { v.once.Do(func() { close(v.recorded) }) }

// waitForRecords reports whether player records exist, waiting for the first input load to write
// them; false when ctx ends first.
func (v *serverValues) waitForRecords(ctx context.Context) bool {
	if held, err := v.records().Records(ctx); err == nil && len(held) > 0 {
		v.markRecorded()
	}
	select {
	case <-ctx.Done():
		return false
	case <-v.recorded:
		return true
	}
}

// run prepares every league's values until ctx ends, starting once player records exist. The
// producer retries a failing pass every second; logf hears each distinct failure once.
func (v *serverValues) run(ctx context.Context, logf func(string, ...any)) {
	if !v.waitForRecords(ctx) {
		return
	}
	src := v.source()
	last := ""
	v.prep.Run(ctx, src, func(ctx context.Context) ([]appvalue.Scope, error) {
		return src.Scopes(ctx, v.pins.Leagues)
	}, func() {}, func(err error) {
		if ctx.Err() == nil && err.Error() != last {
			last = err.Error()
			logf("server values: %v", err)
		}
	})
}

// signalsKeeper loads the value inputs into the reference file once a day.
type signalsKeeper struct {
	values *serverValues
	client *http.Client
	now    func() time.Time
	logf   func(string, ...any)
	mu     sync.Mutex // one load at a time
}

// startServerValues, with -signals, starts the daily input load and the background preparation and
// hands the handler the values they produce.
func startServerValues(ctx context.Context, pins *leaguePins, config hostedFlags, handler *serverhttp.Hosted) func() {
	if !*config.signals {
		return func() {}
	}
	values := newServerValues(pins, filepath.Dir(*config.path))
	handler.Values = values
	k := &signalsKeeper{values: values, client: &http.Client{Timeout: time.Minute},
		now: time.Now, logf: log.Printf}
	scope, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); values.run(scope, k.logf) }()
	go func() { defer wg.Done(); k.loop(scope) }()
	return func() { cancel(); wg.Wait() }
}

func (k *signalsKeeper) loop(ctx context.Context) {
	delay := signalsFirst
	for {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		k.pass(ctx)
		delay = signalsRefresh
	}
}

// pass loads each due feed file, then writes the player record from the facts it holds.
func (k *signalsKeeper) pass(ctx context.Context) {
	k.mu.Lock()
	defer k.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, sources.SignalsTimeout)
	defer cancel()
	season := k.leagueSeason(ctx)
	if season == 0 {
		k.logf("server values: no league is pinned to an NFL season yet; nothing to load")
		return
	}
	ref := k.values.reference
	service := sources.Service{Zone: time.UTC, Health: ref, Archive: ref, Crosswalk: ref, Signals: ref,
		Season: season, Transport: k.signalTransport()}
	report := service.LoadSignals(ctx, func(ctx context.Context) error {
		_, err := linkLiveIdentities(ctx, k.client, crosswalk.SourceURL, ref)
		return err
	})
	k.logf("server values: inputs for %d: %s", season, signalsSummary(report))
	k.writeRecords(ctx)
}

func (k *signalsKeeper) signalTransport() http.RoundTripper {
	return &archive.Transport{Base: k.client.Transport, Sink: k.values.reference}
}

// writeRecords writes the player record from the facts the load holds. The server holds no MFL players
// listing, so each record takes its position, age and draft from nflverse (playerrecord.Assemble's
// unlisted path); names come from the league's own roster.
func (k *signalsKeeper) writeRecords(ctx context.Context) {
	n, err := k.values.records().Write(ctx, noListing{}, k.now())
	if err != nil {
		k.logf("server values: player record: %v", err)
		return
	}
	k.logf("server values: player record written for %d players", n)
	if n > 0 {
		k.values.markRecorded()
	}
}

// noListing lists nobody: every record resolves from the held facts alone.
type noListing struct{}

func (noListing) Facts(string) (domain.PlayerFacts, bool) { return domain.PlayerFacts{}, false }
func (noListing) IDs() []string                           { return nil }

// leagueSeason is the newest NFL season any league is pinned to.
func (k *signalsKeeper) leagueSeason(ctx context.Context) int {
	season := 0
	for _, name := range k.values.pins.files.leagues() {
		if pin := k.values.pins.pin(ctx, name); pin.Err == nil && pin.Football != nil {
			season = max(season, pin.Football.Season())
		}
	}
	return season
}

func signalsSummary(r sources.SignalsReport) string {
	count := map[string]int{}
	facts := 0
	var failed []string
	for _, l := range r.Loads {
		count[l.Status]++
		facts += l.Added
		if l.Status == sources.LoadFailed {
			failed = append(failed, fmt.Sprintf("%s %d: %s", l.Feed, l.Season, l.Error))
		}
	}
	out := fmt.Sprintf("%d files loaded, %d current, %d skipped, %d failed; %d facts added",
		count[sources.LoadDone], count[sources.LoadCurrent], count[sources.LoadSkipped], count[sources.LoadFailed], facts)
	if r.Error != "" {
		out += "; " + r.Error
	}
	for _, f := range failed {
		out += "; " + f
	}
	return out
}
