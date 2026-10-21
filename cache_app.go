package main

import (
	"fmt"

	draftapp "github.com/secureprospective/TheWarRoom/internal/app/draft"

	commsapp "github.com/secureprospective/TheWarRoom/internal/app/comms"
	offersapp "github.com/secureprospective/TheWarRoom/internal/app/offers"
	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
)

type DataScope struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Note     string `json:"note"`
	Action   string `json:"action"`
	Progress string `json:"progress"`
}

// cacheScope is one copy the app holds that can be read afresh from its source, with the app's own
// forced refresh that does it.
type cacheScope struct {
	DataScope
	run func(*App, bool) error
}

// cacheScopes lists, in the order offered, every held copy the app can read again on request. Each runs the
// refresh its own screen already has, asked to ignore the copy held.
func cacheScopes() []cacheScope {
	return []cacheScope{
		{DataScope{"pool", "Free agents and market reports", "The free-agent list and the market reports built on it.",
			"Read free agents again", "Reading…"}, func(a *App, force bool) error {
			return offersapp.New(offersHost{State: a.core, app: a}).TargetPoolRefresh(force)
		}},
		{DataScope{"research", "Player research", "Player scores, activity and depth charts.",
			"Read player research again", "Reading…"}, func(a *App, force bool) error {
			return researchapp.New(researchHost{app: a}, a.researchHeld).TargetResearchRefresh(force)
		}},
		{DataScope{"race", "League race", "The standings and the race for the playoffs.",
			"Read the league race again", "Reading…"}, func(a *App, force bool) error { return raceapp.New(a.core, a.raceHeld).TargetRaceRefresh(force) }},
		{DataScope{"comms", "League board", "The league's message board posts.",
			"Read the league board again", "Reading…"}, func(a *App, force bool) error { return commsapp.New(a.core).TargetCommsRefresh(force) }},
		{DataScope{"draft", "Draft reports", "The draft boards and reports.",
			"Read draft reports again", "Reading…"}, func(a *App, force bool) error {
			return draftapp.New(draftHost{State: a.core, app: a}).TargetDraftReportsRefresh(force)
		}},
	}
}

// TargetCacheScopes lists the held copies a person can ask the app to read afresh.
func (a *App) TargetCacheScopes() []DataScope {
	out := []DataScope{}
	for _, c := range cacheScopes() {
		out = append(out, c.DataScope)
	}
	return out
}

// TargetCacheClear reads one held copy afresh, ignoring the copy held. If the source does not answer, the
// copy held stays and the reason is returned.
func (a *App) TargetCacheClear(scope string) DataActResult {
	if err := a.ready(); err != nil {
		return DataActResult{Error: err.Error()}
	}
	for _, c := range cacheScopes() {
		if c.ID != scope {
			continue
		}
		if err := c.run(a, true); err != nil {
			return DataActResult{Error: fmt.Sprintf("%s could not be read again: %v. The copy held stays.", c.Label, err)}
		}
		return DataActResult{OK: true, Summary: "Asked for a fresh copy of " + c.Label +
			". Each screen shows how fresh what it holds is."}
	}
	return DataActResult{Error: fmt.Sprintf("unknown held copy %q", scope)}
}
