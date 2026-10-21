package main

import (
	"context"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
)

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

type SignalsReport struct {
	OK            bool         `json:"ok"`
	Error         string       `json:"error"`
	StartedAt     string       `json:"startedAt"`
	FinishedAt    string       `json:"finishedAt"`
	Loads         []SignalLoad `json:"loads"`
	CrosswalkNote string       `json:"crosswalkNote"`
}

type PositionShare struct {
	Position string `json:"position"`
	Rostered int    `json:"rostered"`
	WithData int    `json:"withData"`
}

type SeasonCount struct {
	Season  int `json:"season"`
	Players int `json:"players"`
	Waiting int `json:"waiting"`
}

type FeedView struct {
	Feed           string          `json:"feed"`
	Source         string          `json:"source"`
	LastLoaded     string          `json:"lastLoaded"`
	Fresh          bool            `json:"fresh"`
	Seasons        []SeasonCount   `json:"seasons"`
	CoverageSeason int             `json:"coverageSeason"`
	Coverage       []PositionShare `json:"coverage"`
}

type SignalsView struct {
	Feeds []FeedView
}

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

func (a *App) rosteredPositions() map[string]string {
	a.core.LookupMu.Lock()
	lk, ok := a.core.Lookup, a.core.HasLookup
	a.core.LookupMu.Unlock()
	if !ok {
		return map[string]string{}
	}
	out := map[string]string{}
	for id := range rosteredBy(a.core.League.Reader()) {
		if f, ok := lk.Facts(id); ok {
			out[id] = string(f.Position)
		}
	}
	return out
}

func (a *App) loadSignals(ctx context.Context) SignalsReport {
	defer a.sourceHealthChanged(ctx)
	a.core.LoadingSignals.Lock()
	defer a.core.LoadingSignals.Unlock()
	rep := wireSignals(a.sourcesService().LoadSignals(ctx, a.linkDirectory))
	a.core.SignalsMu.Lock()
	a.signals = rep
	a.core.SignalsMu.Unlock()
	return rep
}

const rasFetchTimeout = sourcesapp.RASFetchTimeout
const cfbdEnvVar = sourcesapp.CFBDEnvVar
const signalsTimeout = sourcesapp.SignalsTimeout
const loadDone = sourcesapp.LoadDone
const loadCurrent = sourcesapp.LoadCurrent
const loadFailed = sourcesapp.LoadFailed
const loadSkipped = sourcesapp.LoadSkipped
