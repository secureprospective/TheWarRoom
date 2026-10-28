package composition

import (
	"context"
	"log"
	"path/filepath"
	"time"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	commsapp "github.com/secureprospective/TheWarRoom/internal/app/comms"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	draftapp "github.com/secureprospective/TheWarRoom/internal/app/draft"
	offersapp "github.com/secureprospective/TheWarRoom/internal/app/offers"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
)

func (a *App) wireSources() {
	a.core.ReadSourcesSettings = func() (string, bool, error) {
		s, err := a.readSettings()
		return s.LeagueArchiveDir, s.PhotosOff, err
	}
	a.core.WriteSourcesSettings = func(dir string, off bool) error {
		return a.writeSettings(appSettings{LeagueArchiveDir: dir, PhotosOff: off})
	}
	a.core.ReadSourcesDatabaseChecks = func() *report.Report { return a.databaseChecksReading() }
	a.core.ReadSourcesDirectory = a.directory
	a.core.RunSourcesLeagueRefresh = func(ctx context.Context) (int, bool, bool, error) {
		r, err := a.refreshLeague(ctx, a.core.Rulebook, a.core.League)
		return r.Season, r.Changed, r.RulesChanged, err
	}
}

// factsCompiler names the facts compiler's output. Change it whenever the compiled facts or the
// checks change for the same inputs, so the snapshot built by the old compiler reads as stale.
const factsCompiler = sourcesapp.FactsCompiler

// factsTimeout bounds one compile; the full archive compiles in about fifteen seconds.
const factsTimeout = 5 * time.Minute

// compileFactsInBackground compiles the checked facts after launch when the archive has been
// imported and the manifest has no snapshot for this compiler and these rules yet.
func (a *App) compileFactsInBackground() {
	parent := a.core.WorkCtx
	if parent == nil {
		parent = a.fallbackParent()
	}
	a.startWorker(func() {
		if a.ready() != nil {
			return
		}
		ctx, cancel := context.WithTimeout(parent, factsTimeout)
		defer cancel()
		sourcesapp.StartupFacts(ctx, func(ctx context.Context) (bool, error) {
			_, ok, err := a.core.History.LatestManifest(ctx, labFrozenInputs)
			return ok, err //nolint:wrapcheck // Startup compilation silently skips failed manifest reads as before.
		}, func(ctx context.Context) error { _, _, err := a.publishLabManifest(ctx); return err },
			func(ctx context.Context) error { _, err := a.compileFacts(ctx); return err }, a.warmArchive, func(text string) { log.Print(text) })
	})
}

// warmArchive reads the verified snapshot into the archive cache once the checked facts are in,
// so the first archive reader — Comms, for its roster weeks and NFL games — does not pay the
// first-read rebuild (about twenty seconds). The cache is what that first read would have held anyway.
func (a *App) warmArchive(ctx context.Context) {
	began := time.Now()
	a.core.FactsMu.Lock()
	defer a.core.FactsMu.Unlock()
	sourcesapp.WarmArchive(ctx, began, func(ctx context.Context) error { _, _, err := a.readArchiveFacts(ctx); return err }, func(text string) { log.Print(text) })
}

func (a *App) compileFacts(ctx context.Context) (string, error) {
	return sourcesapp.CompileFacts(ctx, sourcesHost{State: a.core, app: a}, lab.NoteReadings()) //nolint:wrapcheck // Preserve the existing compile error.
}

type appSettings = sourcesapp.Settings

func (a *App) settingsPath() string { return filepath.Join(a.core.Paths.DataDir, "settings.json") }
func (a *App) readSettings() (appSettings, error) {
	return sourcesapp.ReadSettings(a.settingsPath()) //nolint:wrapcheck // Keep settings refusal text.
}
func (a *App) writeSettings(s appSettings) error {
	return sourcesapp.WriteSettings(a.settingsPath(), s) //nolint:wrapcheck // Keep settings refusal text.
}

func (a *App) publishLabManifest(ctx context.Context) (manifest.Manifest, bool, error) {
	return sourcesapp.ServiceFor(a.core).PublishLabManifest(ctx) //nolint:wrapcheck // Preserve the existing manifest error.
}

func (a *App) loadSignals(ctx context.Context) sourcesapp.SignalsReport {
	return sourcesapp.LoadSignals(ctx, sourcesHost{State: a.core, app: a}, a.sourcesHeld)
}

func (a *App) healthInput(ctx context.Context) (sourcesHealthInput, error) {
	in, err := sourcesapp.HealthFor(ctx, sourcesHost{State: a.core, app: a})
	return sourcesHealthInput{HealthInput: in, now: in.Now}, err //nolint:wrapcheck // Preserve the existing health error.
}

type sourcesHealthInput struct {
	sourcesapp.HealthInput
	now time.Time
}

func sourceStanding(in sourcesHealthInput) (string, string) {
	return sourcesapp.SourceStanding(in.HealthInput)
}
func countText(n int, noun string) string      { return sourcesapp.CountText(n, noun) }
func pluralNoun(n float64, noun string) string { return report.PluralNoun(n, noun) }

const rasFetchTimeout = sourcesapp.RASFetchTimeout
const signalsTimeout = sourcesapp.SignalsTimeout
const labFrozenInputs = sourcesapp.LabFrozenInputs

func (a *App) researchCrosswalkVersion() string { return a.sourcesHeld.SnapshotCrosswalk().LoadedAt }

const dynastyProcessSource = sourcesapp.DynastyProcessSource

// readArchiveFacts reads the verified archive snapshot through the service and keeps it on App.
// The caller holds factsMu, which guards archiveCache.
func (a *App) readArchiveFacts(ctx context.Context) (archiveapp.Snapshot, bool, error) {
	notes := lab.NoteReadings()
	src := archiveapp.Source{History: a.core.History, Bodies: sourcesapp.CuratedBodies{Archive: a.core.History, Curated: map[string][]byte{notes.SHA256: notes.Body}},
		Manifest: labFrozenInputs, Compiler: factsCompiler}
	return archiveapp.ReadCached(ctx, src, &a.archiveCache) //nolint:wrapcheck // Archive's errors already name the step.

}

// readCommsArchive borrows the snapshot under its caller's FactsMu; core detaches the columns.
func (a *App) readCommsArchive(ctx context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error) {
	cache, _, err := a.readArchiveFacts(ctx)
	return cache.Facts.RosterWeek, cache.Facts.NflGame, err
}

func (a *App) configureArchive() { a.core.ReadCommsArchive = a.readCommsArchive }

func (a *App) cacheScopes() []sourcesapp.CacheScope {
	return sourcesapp.CacheScopes(sourcesapp.CacheRefreshers{
		Pool:     offersapp.New(offersHost{State: a.core, app: a}).TargetPoolRefresh,
		Research: researchapp.New(researchHost{app: a}, a.researchHeld).TargetResearchRefresh,
		Race:     raceapp.New(raceHost{a.core}, a.raceHeld).TargetRaceRefresh,
		Comms:    commsapp.New(commsHost{a.core}).TargetCommsRefresh,
		Draft:    draftapp.New(draftHost{State: a.core, app: a}).TargetDraftReportsRefresh,
	})
}

// sourcesHost is the Sources Host: core's shared state plus the held copies other features own. It is
// never bound: only sources.API exposes methods to Wails.
type sourcesHost struct {
	*core.State
	app *App
}

func (a *App) sourcesAPI() *sourcesapp.API {
	return sourcesapp.New(sourcesHost{State: a.core, app: a}, a.sourcesHeld)
}

func (h sourcesHost) CacheScopes() []sourcesapp.DataScope {
	return sourcesapp.CacheScopeReadings(h.app.cacheScopes())

}

func (h sourcesHost) ClearCache(scope string) (string, bool, error) {
	return sourcesapp.ClearCache(h.app.cacheScopes(), scope) //nolint:wrapcheck // Preserve the selected refresh error.
}

func (h sourcesHost) SourcesPositions() map[string]string {
	lookup, held := h.SourcesLookup()
	return sourcesapp.Positions(lookup, held, h.SourcesRostered)
}
func (h sourcesHost) SerializeSourcesCompile(compile func() (string, error)) (string, error) {
	return sourcesapp.CompileAvailable(h.FactsMu.TryLock, h.FactsMu.Unlock, compile) //nolint:wrapcheck // Preserve compile errors and the busy result.
}
