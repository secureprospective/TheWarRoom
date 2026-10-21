package main

import (
	"context"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// dynastyProcessSource is the crosswalk's id in sources.csv; its loads are the directory's.
const dynastyProcessSource = sourcesapp.DynastyProcessSource

// CrosswalkRate is one position's match rate, rostered and in the free-agent pool.
type CrosswalkRate struct {
	Position          string `json:"position"`
	Rostered          int    `json:"rostered"`
	RosteredMatched   int    `json:"rosteredMatched"`
	FreeAgents        int    `json:"freeAgents"`
	FreeAgentsMatched int    `json:"freeAgentsMatched"`
}

// CrosswalkMiss is one rostered player with no NFL id link, and why.
type CrosswalkMiss struct {
	MFLID     string `json:"mflId"`
	Name      string `json:"name"`
	Team      string `json:"team"`
	Position  string `json:"position"`
	Franchise string `json:"franchise"`
	Reason    string `json:"reason"`
}

// CrosswalkReport is the latest crosswalk load: links written and the match rates they give.
type CrosswalkReport struct {
	OK        bool            `json:"ok"`
	Error     string          `json:"error"`
	LoadedAt  string          `json:"loadedAt"`
	Links     int             `json:"links"`
	Promoted  int             `json:"promoted"`
	Rates     []CrosswalkRate `json:"rates"`
	Total     CrosswalkRate   `json:"total"`
	Unmatched []CrosswalkMiss `json:"unmatched"`
}

// LoadCrosswalk fetches DynastyProcess, links it into the player directory and reports the match
// rates. ScoreLeague does the same with the crosswalk it already fetches.
func (a *App) LoadCrosswalk() CrosswalkReport {
	if err := a.ready(); err != nil {
		return CrosswalkReport{Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, rasFetchTimeout)
	defer cancel()
	lk, err := a.directory(ctx)
	if err != nil {
		return CrosswalkReport{Error: err.Error()}
	}
	cw, err := a.fetchCrosswalk(ctx)
	if err != nil {
		return CrosswalkReport{Error: err.Error()}
	}
	rep, err := a.linkCrosswalk(ctx, cw, lk)
	if err != nil {
		return CrosswalkReport{Error: err.Error()}
	}
	return rep
}

// GetCrosswalkReport returns the latest load's report, held since this launch.
func (a *App) GetCrosswalkReport() CrosswalkReport {
	if err := a.ready(); err != nil {
		return CrosswalkReport{Error: err.Error()}
	}
	a.core.CrosswalkMu.Lock()
	defer a.core.CrosswalkMu.Unlock()
	return a.crosswalk
}

// fetchCrosswalk reads DynastyProcess through the archive. A failure is recorded against the
// source, so its health shows it.
func (a *App) fetchCrosswalk(ctx context.Context) (crosswalk.Map, error) {
	return a.sourcesService().FetchCrosswalk(ctx) //nolint:wrapcheck // preserve the existing contextual error shown by the binding
}

// linkCrosswalk writes the crosswalk's links to the player directory and holds the report.
func (a *App) linkCrosswalk(ctx context.Context, cw crosswalk.Map, lk normalize.Lookup) (CrosswalkReport, error) {
	rep, err := a.sourcesService().LinkCrosswalk(ctx, cw, lk, func() map[string]string { return rosteredBy(a.core.League.Reader()) }, a.core.Rulebook.FranchiseNames)
	if err != nil {
		return CrosswalkReport{}, err //nolint:wrapcheck // preserve the existing contextual error shown by the binding
	}
	out := wireCrosswalk(rep)
	a.core.CrosswalkMu.Lock()
	a.crosswalk = out
	a.core.CrosswalkMu.Unlock()
	return out, nil
}

// rosteredBy maps every rostered MFL id to its franchise.
func rosteredBy(r state.Reader) map[string]string {
	out := map[string]string{}
	for _, fid := range r.Franchises() {
		roster, _ := r.Roster(fid)
		for _, p := range roster {
			out[p.MFLID] = fid
		}
	}
	return out
}

// missReason says why a rostered player has no gsis link.
