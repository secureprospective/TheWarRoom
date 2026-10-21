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

// cacheScope is one copy the app holds that can be read afresh from its source, with the app's own
// forced refresh that does it.
type cacheScope struct {
	sourcesapp.DataScope
	run func(*App, bool) error
}

// cacheScopes lists, in the order offered, every held copy the app can read again on request. Each runs the
// refresh its own screen already has, asked to ignore the copy held.
func cacheScopes() []cacheScope {
	return []cacheScope{
		{sourcesapp.DataScope{ID: "pool", Label: "Free agents and market reports", Note: "The free-agent list and the market reports built on it.",
			Action: "Read free agents again", Progress: "Reading…"}, func(a *App, force bool) error {
			return offersapp.New(offersHost{State: a.core, app: a}).TargetPoolRefresh(force)
		}},
		{sourcesapp.DataScope{ID: "research", Label: "Player research", Note: "Player scores, activity and depth charts.",
			Action: "Read player research again", Progress: "Reading…"}, func(a *App, force bool) error {
			return researchapp.New(researchHost{app: a}, a.researchHeld).TargetResearchRefresh(force)
		}},
		{sourcesapp.DataScope{ID: "race", Label: "League race", Note: "The standings and the race for the playoffs.",
			Action: "Read the league race again", Progress: "Reading…"}, func(a *App, force bool) error { return raceapp.New(a.core, a.raceHeld).TargetRaceRefresh(force) }},
		{sourcesapp.DataScope{ID: "comms", Label: "League board", Note: "The league's message board posts.",
			Action: "Read the league board again", Progress: "Reading…"}, func(a *App, force bool) error { return commsapp.New(a.core).TargetCommsRefresh(force) }},
		{sourcesapp.DataScope{ID: "draft", Label: "Draft reports", Note: "The draft boards and reports.",
			Action: "Read draft reports again", Progress: "Reading…"}, func(a *App, force bool) error {
			return draftapp.New(draftHost{State: a.core, app: a}).TargetDraftReportsRefresh(force)
		}},
	}
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
	out := []sourcesapp.DataScope{}
	for _, c := range cacheScopes() {
		out = append(out, c.DataScope)
	}
	return out
}

func (h sourcesHost) ClearCache(scope string) (string, bool, error) {
	for _, c := range cacheScopes() {
		if c.ID == scope {
			return c.Label, true, c.run(h.app, true)
		}
	}
	return "", false, nil
}
