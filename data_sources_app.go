package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"

	"github.com/secureprospective/TheWarRoom/internal/manifest"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// DataActResult is the outcome of one Control Room data Act, in plain words.
type DataActResult struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Summary string `json:"summary"`
}

// DataScope is one part of the app's data a refresh Act can renew.
type DataScope struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Note     string `json:"note"`
	Action   string `json:"action"`
	Progress string `json:"progress"`
}

// DataSources is Control Room › Data and sources: source health and the signal files (C-07), the
// checked facts and the checks that will prove them (L-34), and the refresh scopes (S-09).
type DataSources struct {
	DatabaseChecks *Report           `json:"databaseChecks"`
	Health         Report            `json:"health"`
	Signals        Report            `json:"signals"`
	Facts          Report            `json:"facts"`
	Checks         Report            `json:"checks"`
	ArchiveDir     string            `json:"archiveDir"`
	PhotosOn       bool              `json:"photosOn"`
	Scopes         []DataScope       `json:"scopes"`
	Words          map[string]string `json:"words"`
}

// TargetDataRefresh renews one scope of the app's data and says what it did.
func (a *App) TargetDataRefresh(scope string) DataActResult {
	if err := a.ready(); err != nil {
		return DataActResult{Error: err.Error()}
	}
	summary, err := a.refreshScope(scope)
	if err != nil {
		return DataActResult{Error: err.Error()}
	}
	return DataActResult{OK: true, Summary: summary}
}

func (a *App) refreshScope(scope string) (string, error) {
	switch scope {
	case "league":
		ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
		defer cancel()
		r, err := a.refreshLeague(ctx, a.rulebook, a.league)
		if err != nil {
			return "", fmt.Errorf("league refresh failed: %w", err)
		}
		return leagueRefreshSummary(r), nil
	case "signals":
		ctx, cancel := context.WithTimeout(a.ctx, signalsTimeout)
		defer cancel()
		r := a.loadSignals(ctx)
		if r.Error != "" {
			return "", fmt.Errorf("signals refresh: %s; %s", r.Error, signalsSummary(r))
		}
		for _, l := range r.Loads {
			if l.Status == loadFailed {
				return "", fmt.Errorf("signals refresh: %s; %s", l.Error, signalsSummary(r))
			}
		}
		return signalsSummary(r), nil
	case "crosswalk":
		r := a.LoadCrosswalk()
		if !r.OK {
			return "", fmt.Errorf("crosswalk load failed: %s", r.Error)
		}
		return fmt.Sprintf("%s links; rostered players matched %s of %s, free agents %s of %s.", thousands(r.Links),
			thousands(r.Total.RosteredMatched), thousands(r.Total.Rostered), thousands(r.Total.FreeAgentsMatched),
			thousands(r.Total.FreeAgents)), nil
	case "archive":
		ctx, cancel := context.WithTimeout(a.ctx, archiveImportTimeout)
		defer cancel()
		return a.importLeagueArchive(ctx)
	}
	return "", fmt.Errorf("unknown data scope %q", scope)
}

func leagueRefreshSummary(r RefreshResult) string {
	changed, rules := "nothing new on MFL", "rules unchanged"
	if r.Changed {
		changed = "rosters and salary adjustments updated"
	}
	if r.RulesChanged {
		rules = "rules changed"
	}
	return fmt.Sprintf("Season %d: %s; %s.", r.Season, changed, rules)
}

// TargetDataSources reads what history holds about every source and the checked facts.
func (a *App) TargetDataSources() (DataSources, error) {
	if err := a.ready(); err != nil {
		return DataSources{}, fmt.Errorf("data and sources: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m1Timeout)
	defer cancel()
	settings, err := a.readSettings()
	if err != nil {
		return DataSources{}, err
	}
	view, err := a.sourcesService().Read(ctx, a.photoHealth(), a.rosteredPositions, a.lastSignals)
	if err != nil {
		return DataSources{}, err //nolint:wrapcheck // preserve existing binding errors
	}
	return DataSources{DatabaseChecks: a.databaseChecksReport(), Health: wireReport(view.Health), Signals: wireReport(view.Signals), Facts: wireReport(view.Facts), Checks: wireReport(view.Checks), ArchiveDir: settings.LeagueArchiveDir, PhotosOn: !settings.PhotosOff, Scopes: wireScopes(view.Scopes), Words: view.Words}, nil
}
func (a *App) lastSignals() sourcesapp.SignalsReport {
	a.signalsMu.Lock()
	defer a.signalsMu.Unlock()
	return serviceSignals(a.signals)
}
func (a *App) photoHealth() *sourcesapp.PhotoHealth {
	if photos := a.headshots.Load(); photos != nil {
		st, held, size := photos.Stats()
		return &sourcesapp.PhotoHealth{Off: a.photosOff.Load(), Held: held, Bytes: size, Fetched: st.Fetched, Missing: st.Missing, LastFail: st.LastFailure, LastError: st.LastError}
	}
	return nil
}

func (a *App) healthInput(ctx context.Context) (sourcesHealthInput, error) {
	in, err := a.sourcesService().HealthInput(ctx, a.photoHealth())
	return sourcesHealthInput{HealthInput: in, now: in.Now}, err //nolint:wrapcheck // preserve the existing contextual error shown by the binding
}
func (a *App) factsInput(ctx context.Context) (sourcesapp.FactsInput, error) {
	return a.sourcesService().FactsInput(ctx) //nolint:wrapcheck // preserve the existing contextual error shown by the binding
}
func historyManifest(m manifest.Manifest, entries []byte) history.ManifestRecord {
	return sourcesapp.HistoryManifest(m, entries)
}

func signalsSummary(r SignalsReport) string { return sourcesapp.SignalsSummary(serviceSignals(r)) }

func (a *App) TargetCrosswalk() Report {
	return wireReport(sourcesapp.CrosswalkReading(serviceCrosswalk(a.GetCrosswalkReport()), time.Local))
}

type sourcesHealthInput struct {
	sourcesapp.HealthInput
	now time.Time
}

func (a *App) sourcesService() sourcesapp.Service {
	return sourcesapp.Service{Health: a.history, Facts: a.history, Archive: a.history, Crosswalk: a.history, Signals: a.history, Season: a.season, CollegeKey: strings.TrimSpace(os.Getenv(cfbdEnvVar)), Transport: a.fetches, LabData: a.labData}
}
func wireScopes(in []sourcesapp.DataScope) []DataScope {
	if in == nil {
		return nil
	}
	out := make([]DataScope, len(in))
	for i, v := range in {
		out[i] = DataScope(v)
	}
	return out
}

func wireCrosswalkRate(in sourcesapp.CrosswalkRate) CrosswalkRate {
	return CrosswalkRate(in)
}

func serviceCrosswalkRate(in CrosswalkRate) sourcesapp.CrosswalkRate {
	return sourcesapp.CrosswalkRate(in)
}

func wireCrosswalkMiss(in sourcesapp.CrosswalkMiss) CrosswalkMiss {
	return CrosswalkMiss(in)
}

func serviceCrosswalkMiss(in CrosswalkMiss) sourcesapp.CrosswalkMiss {
	return sourcesapp.CrosswalkMiss(in)
}

func wireCrosswalk(in sourcesapp.CrosswalkReport) CrosswalkReport {
	out := CrosswalkReport{
		OK:       in.OK,
		Error:    in.Error,
		LoadedAt: in.LoadedAt,
		Links:    in.Links,
		Promoted: in.Promoted,
		Total:    wireCrosswalkRate(in.Total),
	}
	if in.Rates != nil {
		out.Rates = make([]CrosswalkRate, len(in.Rates))
		for i, v := range in.Rates {
			out.Rates[i] = wireCrosswalkRate(v)
		}
	}
	if in.Unmatched != nil {
		out.Unmatched = make([]CrosswalkMiss, len(in.Unmatched))
		for i, v := range in.Unmatched {
			out.Unmatched[i] = wireCrosswalkMiss(v)
		}
	}
	return out
}

func serviceCrosswalk(in CrosswalkReport) sourcesapp.CrosswalkReport {
	out := sourcesapp.CrosswalkReport{
		OK:       in.OK,
		Error:    in.Error,
		LoadedAt: in.LoadedAt,
		Links:    in.Links,
		Promoted: in.Promoted,
		Total:    serviceCrosswalkRate(in.Total),
	}
	if in.Rates != nil {
		out.Rates = make([]sourcesapp.CrosswalkRate, len(in.Rates))
		for i, v := range in.Rates {
			out.Rates[i] = serviceCrosswalkRate(v)
		}
	}
	if in.Unmatched != nil {
		out.Unmatched = make([]sourcesapp.CrosswalkMiss, len(in.Unmatched))
		for i, v := range in.Unmatched {
			out.Unmatched[i] = serviceCrosswalkMiss(v)
		}
	}
	return out
}

func wireSignalLoad(in sourcesapp.SignalLoad) SignalLoad {
	return SignalLoad(in)
}

func serviceSignalLoad(in SignalLoad) sourcesapp.SignalLoad {
	return sourcesapp.SignalLoad(in)
}

func wireSignals(in sourcesapp.SignalsReport) SignalsReport {
	out := SignalsReport{
		OK:            in.OK,
		Error:         in.Error,
		StartedAt:     in.StartedAt,
		FinishedAt:    in.FinishedAt,
		CrosswalkNote: in.CrosswalkNote,
	}
	if in.Loads != nil {
		out.Loads = make([]SignalLoad, len(in.Loads))
		for i, v := range in.Loads {
			out.Loads[i] = wireSignalLoad(v)
		}
	}
	return out
}

func serviceSignals(in SignalsReport) sourcesapp.SignalsReport {
	out := sourcesapp.SignalsReport{
		OK:            in.OK,
		Error:         in.Error,
		StartedAt:     in.StartedAt,
		FinishedAt:    in.FinishedAt,
		CrosswalkNote: in.CrosswalkNote,
	}
	if in.Loads != nil {
		out.Loads = make([]sourcesapp.SignalLoad, len(in.Loads))
		for i, v := range in.Loads {
			out.Loads[i] = serviceSignalLoad(v)
		}
	}
	return out
}

func countText(n int, noun string) string      { return sourcesapp.CountText(n, noun) }
func pluralNoun(n float64, noun string) string { return report.PluralNoun(n, noun) }

type curatedBodies struct {
	archive manifest.Bodies
	curated map[string][]byte
}

func (b curatedBodies) Body(ctx context.Context, sha string) ([]byte, error) {
	return (sourcesapp.CuratedBodies{Archive: b.archive, Curated: b.curated}).Body(ctx, sha) //nolint:wrapcheck // preserve the existing contextual error shown by the binding
}

func sourceStanding(in sourcesHealthInput) (string, string) {
	return sourcesapp.SourceStanding(in.HealthInput)
}
