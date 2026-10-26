package composition

import (
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	draftapp "github.com/secureprospective/TheWarRoom/internal/app/draft"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"

	commsapp "github.com/secureprospective/TheWarRoom/internal/app/comms"
	offersapp "github.com/secureprospective/TheWarRoom/internal/app/offers"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
)

func (a *App) cacheScopes() []sourcesapp.CacheScope {
	return sourcesapp.CacheScopes(sourcesapp.CacheRefreshers{
		Pool:     offersapp.New(offersHost{State: a.core, app: a}).TargetPoolRefresh,
		Research: researchapp.New(researchHost{app: a}, a.researchHeld).TargetResearchRefresh,
		Race:     raceapp.New(a.core, a.raceHeld).TargetRaceRefresh,
		Comms:    commsapp.New(a.core).TargetCommsRefresh,
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
