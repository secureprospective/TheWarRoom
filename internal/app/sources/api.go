package sources

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/headshot"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Host owns shared state, scheduling and operation serializers, not private reports.
type Host interface {
	ServiceHost
	CrosswalkHost
	SettingsHost
	ReadingHost
	CacheHost
	Ready() error
	SourcesContext() context.Context
	RefreshSourcesLeague(context.Context) (int, bool, bool, error)
	SerializeSourcesLoad(context.Context, func())
	SerializeSourcesCompile(func() (string, error)) (string, error)
}

// ServiceHost supplies startup-owned dependencies and the presentation zone.
type ServiceHost interface {
	SourcesInputs() (*history.Store, int, http.RoundTripper, func() (*lab.Data, error))
	SourcesZone() *time.Location
}

// CrosswalkHost keeps directory acquisition before writes and coverage reads after them.
type CrosswalkHost interface {
	SourcesDirectory(context.Context) (normalize.Lookup, error)
	SourcesRostered() map[string]string
	SourcesNames() map[string]string
}

// SettingsHost preserves the shared settings read-modify-write sequence.
type SettingsHost interface {
	SourcesSettings() (string, bool, error)
	SaveSourcesSettings(string, bool) error
	// SetPhotosOff publishes the photo switch to the photo route and the player cards.
	SetPhotosOff(bool)
}

// ReadingHost supplies detached shared readings at their original capture points.
type ReadingHost interface {
	SourcesDatabaseChecks() *report.Report
	SourcesPhotos() (headshot.Stats, int, int64, bool, bool)
	SourcesPositions() map[string]string
}

// CacheHost lists the held copies a person can ask to read afresh and runs that copy's own forced
// refresh, which belongs to the feature that holds it.
type CacheHost interface {
	CacheScopes() []DataScope
	// ClearCache runs scope's forced refresh. found is false for an unknown scope; label names it.
	ClearCache(scope string) (label string, found bool, err error)
}

type API struct {
	host Host
	held *Held
}

func New(host Host, held *Held) *API { return &API{host: host, held: held} }

type DataActResult struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Summary string `json:"summary"`
}

type DataSources struct {
	DatabaseChecks *report.Report    `json:"databaseChecks"`
	Health         report.Report     `json:"health"`
	Signals        report.Report     `json:"signals"`
	Facts          report.Report     `json:"facts"`
	Checks         report.Report     `json:"checks"`
	ArchiveDir     string            `json:"archiveDir"`
	PhotosOn       bool              `json:"photosOn"`
	Scopes         []DataScope       `json:"scopes"`
	Words          map[string]string `json:"words"`
}

func (a *API) LoadCrosswalk() CrosswalkReport {
	if err := a.host.Ready(); err != nil {
		return CrosswalkReport{Error: report.FailureText(err)}
	}
	ctx, cancel := context.WithTimeout(a.host.SourcesContext(), RASFetchTimeout)
	defer cancel()
	rep, err := linkDirectory(ctx, a.host, a.held)
	if err != nil {
		return CrosswalkReport{Error: report.FailureText(err)}
	}
	return rep
}

func (a *API) GetCrosswalkReport() CrosswalkReport {
	if err := a.host.Ready(); err != nil {
		return CrosswalkReport{Error: report.FailureText(err)}
	}
	return a.held.SnapshotCrosswalk()
}

func (a *API) TargetCrosswalk() report.Report {
	return CrosswalkReading(a.GetCrosswalkReport(), a.host.SourcesZone())
}

func (a *API) TargetDataSources() (DataSources, error) {
	if err := a.host.Ready(); err != nil {
		return DataSources{}, fmt.Errorf("data and sources: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.SourcesContext(), 120*time.Second)
	defer cancel()
	dir, off, err := a.host.SourcesSettings()
	if err != nil {
		return DataSources{}, err //nolint:wrapcheck // Preserve the binding's existing error text.
	}
	view, err := ServiceFor(a.host).Read(ctx, photoHealth(a.host), a.host.SourcesPositions, a.held.SnapshotSignals)
	if err != nil {
		return DataSources{}, err
	}
	return DataSources{DatabaseChecks: a.host.SourcesDatabaseChecks(), Health: view.Health, Signals: view.Signals,
		Facts: view.Facts, Checks: view.Checks, ArchiveDir: dir, PhotosOn: !off, Scopes: view.Scopes, Words: view.Words}, nil
}

func (a *API) TargetDataRefresh(scope string) DataActResult {
	if err := a.host.Ready(); err != nil {
		return DataActResult{Error: report.FailureText(err)}
	}
	summary, err := a.refreshScope(scope)
	if err != nil {
		return DataActResult{Error: report.FailureText(err)}
	}
	return DataActResult{OK: true, Summary: summary}
}

func (a *API) SetLeagueArchiveDir(dir string) DataActResult {
	if err := a.host.Ready(); err != nil {
		return DataActResult{Error: report.FailureText(err)}
	}
	dir = filepath.Clean(dir)
	if err := CheckArchiveDir(dir); err != nil {
		return DataActResult{Error: report.FailureText(err)}
	}
	_, off, err := a.host.SourcesSettings()
	if err != nil {
		return DataActResult{Error: report.FailureText(err)}
	}
	if err := a.host.SaveSourcesSettings(dir, off); err != nil {
		return DataActResult{Error: report.FailureText(err)}
	}
	return DataActResult{OK: true, Summary: "League archive folder set to " + dir + "."}
}

// SetPhotos turns player photos on or off; off means no request reaches a photo host.
func (a *API) SetPhotos(on bool) DataActResult {
	if err := a.host.Ready(); err != nil {
		return DataActResult{Error: report.FailureText(err)}
	}
	dir, _, err := a.host.SourcesSettings()
	if err != nil {
		return DataActResult{Error: report.FailureText(err)}
	}
	if err := a.host.SaveSourcesSettings(dir, !on); err != nil {
		return DataActResult{Error: report.FailureText(err)}
	}
	a.host.SetPhotosOff(!on)
	if on {
		return DataActResult{OK: true, Summary: "Player photos are on; each shows the first time its card is drawn."}
	}
	return DataActResult{OK: true, Summary: "Player photos are off; cards show the NFL team instead."}
}

func (a *API) refreshScope(scope string) (string, error) {
	switch scope {
	case "league":
		ctx, cancel := context.WithTimeout(a.host.SourcesContext(), 2*time.Minute)
		defer cancel()
		season, changed, rules, err := a.host.RefreshSourcesLeague(ctx)
		if err != nil {
			return "", fmt.Errorf("league refresh failed: %w", err)
		}
		return LeagueRefreshSummary(season, changed, rules), nil
	case "signals":
		ctx, cancel := context.WithTimeout(a.host.SourcesContext(), SignalsTimeout)
		defer cancel()
		r := LoadSignals(ctx, a.host, a.held)
		if r.Error != "" {
			return "", fmt.Errorf("signals refresh: %s; %s", r.Error, SignalsSummary(r))
		}
		for _, l := range r.Loads {
			if l.Status == LoadFailed {
				return "", fmt.Errorf("signals refresh: %s; %s", l.Error, SignalsSummary(r))
			}
		}
		return SignalsSummary(r), nil
	case "crosswalk":
		r := a.LoadCrosswalk()
		if !r.OK {
			return "", fmt.Errorf("crosswalk load failed: %s", r.Error)
		}
		return fmt.Sprintf("%s links; rostered players matched %s of %s, free agents %s of %s.", Thousands(r.Links),
			Thousands(r.Total.RosteredMatched), Thousands(r.Total.Rostered), Thousands(r.Total.FreeAgentsMatched), Thousands(r.Total.FreeAgents)), nil
	case "archive":
		ctx, cancel := context.WithTimeout(a.host.SourcesContext(), 10*time.Minute)
		defer cancel()
		dir, _, err := a.host.SourcesSettings()
		if err != nil {
			return "", err //nolint:wrapcheck // Preserve the settings error shown by the binding.
		}
		if dir == "" {
			return "", fmt.Errorf("no league archive folder is set; set it under Data refresh first")
		}
		return ServiceFor(a.host).ImportLeagueArchive(ctx, dir, func(ctx context.Context) (string, error) {
			return CompileFacts(ctx, a.host, lab.NoteReadings())
		})
	}
	return "", fmt.Errorf("unknown data scope %q", scope)
}

func LeagueRefreshSummary(season int, changed, rulesChanged bool) string {
	changedText, rules := "nothing new on MFL", "rules unchanged"
	if changed {
		changedText = "rosters and salary adjustments updated"
	}
	if rulesChanged {
		rules = "rules changed"
	}
	return fmt.Sprintf("Season %d: %s; %s.", season, changedText, rules)
}

func ServiceFor(host ServiceHost) Service {
	hist, season, transport, data := host.SourcesInputs()
	return Service{Health: hist, Facts: hist, Archive: hist, Crosswalk: hist, Signals: hist,
		Season: season, Zone: host.SourcesZone(), Transport: transport, LabData: data, CollegeKey: strings.TrimSpace(os.Getenv(CFBDEnvVar))}
}

func photoHealth(host ReadingHost) *PhotoHealth {
	st, held, size, off, ok := host.SourcesPhotos()
	if !ok {
		return nil
	}
	return &PhotoHealth{Off: off, Held: held, Bytes: size, Fetched: st.Fetched, Missing: st.Missing,
		LastFail: st.LastFailure, LastError: st.LastError}
}

func HealthFor(ctx context.Context, host Host) (HealthInput, error) {
	return ServiceFor(host).HealthInput(ctx, photoHealth(host))
}

func linkDirectory(ctx context.Context, host Host, held *Held) (CrosswalkReport, error) {
	lk, err := host.SourcesDirectory(ctx)
	if err != nil {
		return CrosswalkReport{}, err //nolint:wrapcheck // Preserve the directory error shown by the binding.
	}
	service := ServiceFor(host)
	cw, err := service.FetchCrosswalk(ctx)
	if err != nil {
		return CrosswalkReport{}, err
	}
	rep, err := service.LinkCrosswalk(ctx, cw, lk, host.SourcesRostered, host.SourcesNames)
	if err != nil {
		return CrosswalkReport{}, err
	}
	held.PublishCrosswalk(rep)
	return rep, nil
}

func LoadSignals(ctx context.Context, host Host, held *Held) SignalsReport {
	var rep SignalsReport
	host.SerializeSourcesLoad(ctx, func() {
		rep = ServiceFor(host).LoadSignals(ctx, func(ctx context.Context) error {
			_, err := linkDirectory(ctx, host, held)
			return err
		})
		held.PublishSignals(rep)
	})
	return rep
}

func CompileFacts(ctx context.Context, host Host, readings lab.Curated) (string, error) {
	return host.SerializeSourcesCompile(func() (string, error) { return ServiceFor(host).CompileFactsWith(ctx, readings) }) //nolint:wrapcheck // Preserve the serializer's existing error.
}

func Commands() []catalog.Command { return catalog.DesktopCommands("sources", nil) }

// TargetCacheScopes lists the held copies a person can ask the app to read afresh.
func (a *API) TargetCacheScopes() []DataScope { return a.host.CacheScopes() }

// TargetCacheClear reads one held copy afresh, ignoring the copy held. If the source does not answer, the
// copy held stays and the reason is returned.
func (a *API) TargetCacheClear(scope string) DataActResult {
	if err := a.host.Ready(); err != nil {
		return DataActResult{Error: report.FailureText(err)}
	}
	label, found, err := a.host.ClearCache(scope)
	if !found {
		return DataActResult{Error: fmt.Sprintf("unknown held copy %q", scope)}
	}
	if err != nil {
		return DataActResult{Error: fmt.Sprintf("%s could not be read again: %v. The copy held stays.", label, report.FailureText(err))}
	}
	return DataActResult{OK: true, Summary: "Asked for a fresh copy of " + label +
		". Each screen shows how fresh what it holds is."}
}
