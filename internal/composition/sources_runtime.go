package composition

import (
	"context"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	draftapp "github.com/secureprospective/TheWarRoom/internal/app/draft"
	offersapp "github.com/secureprospective/TheWarRoom/internal/app/offers"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
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

type appSettings = sourcesapp.Settings

func (a *App) settingsPath() string { return filepath.Join(a.core.Paths.DataDir, "settings.json") }
func (a *App) readSettings() (appSettings, error) {
	return sourcesapp.ReadSettings(a.settingsPath()) //nolint:wrapcheck // Keep settings refusal text.
}
func (a *App) writeSettings(s appSettings) error {
	return sourcesapp.WriteSettings(a.settingsPath(), s) //nolint:wrapcheck // Keep settings refusal text.
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

func (a *App) researchCrosswalkVersion() string { return a.sourcesHeld.SnapshotCrosswalk().LoadedAt }

const dynastyProcessSource = sourcesapp.DynastyProcessSource

func (a *App) cacheScopes() []sourcesapp.CacheScope {
	return sourcesapp.CacheScopes(sourcesapp.CacheRefreshers{
		Pool:     offersapp.New(offersHost{State: a.core, app: a}).TargetPoolRefresh,
		Research: researchapp.New(researchHost{app: a}, a.researchHeld).TargetResearchRefresh,
		Race:     raceapp.New(raceHost{a.core}, a.raceHeld).TargetRaceRefresh,
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

func (h sourcesHost) StoreSourcesRecords(ctx context.Context) { h.app.writeRecordAfterSignals(ctx) }
