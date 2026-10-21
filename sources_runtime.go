package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/manifest"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"

	"github.com/secureprospective/TheWarRoom/internal/lab"
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
		// A manifest published by an older build may lack an input this build pins (the curated
		// readings); re-publishing is idempotent and only ever follows an archive import.
		if _, ok, err := a.core.History.LatestManifest(ctx, labFrozenInputs); err != nil || !ok {
			return
		}
		if _, _, err := a.publishLabManifest(ctx); err != nil {
			log.Printf("the war room: checked facts: %v", err)
			return
		}
		if _, err := a.compileFacts(ctx); err != nil {
			log.Printf("the war room: checked facts: %v", err)
			return
		}
		a.warmArchive(ctx)
		a.fitAssetModel(ctx)
	})
}

// warmArchive reads the verified snapshot into the archive cache once the checked facts are in,
// so League Pulse › Archive opens without its first-read rebuild (about twenty seconds).
// The cache is what the first open would have held anyway.
func (a *App) warmArchive(ctx context.Context) {
	began := time.Now()
	a.core.FactsMu.Lock()
	defer a.core.FactsMu.Unlock()
	if _, _, err := a.readArchiveFacts(ctx); err != nil {
		log.Printf("the war room: league archive: %v", err)
		return
	}
	log.Printf("the war room: league archive read in %s", time.Since(began).Round(time.Millisecond))
}

func (a *App) compileFacts(ctx context.Context) (string, error) {
	return sourcesapp.CompileFacts(ctx, a.core, lab.NoteReadings()) //nolint:wrapcheck // Preserve the existing compile error.
}

// appSettings is the app's own configuration, kept as settings.json in its data directory.
type appSettings struct {
	LeagueArchiveDir string `json:"leagueArchiveDir"`
	PhotosOff        bool   `json:"photosOff,omitempty"`
}

func (a *App) settingsPath() string { return filepath.Join(a.core.Paths.DataDir, "settings.json") }

func (a *App) readSettings() (appSettings, error) {
	var s appSettings
	b, err := os.ReadFile(a.settingsPath())
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, fmt.Errorf("app: read settings: %w", err)
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("app: settings.json: %w", err)
	}
	return s, nil
}

func (a *App) writeSettings(s appSettings) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("app: encode settings: %w", err)
	}
	tmp := a.settingsPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("app: write settings: %w", err)
	}
	if err := os.Rename(tmp, a.settingsPath()); err != nil {
		return fmt.Errorf("app: write settings: %w", err)
	}
	return nil
}

func (a *App) publishLabManifest(ctx context.Context) (manifest.Manifest, bool, error) {
	return sourcesapp.ServiceFor(a.core).PublishLabManifest(ctx) //nolint:wrapcheck // Preserve the existing manifest error.
}

func (a *App) loadSignals(ctx context.Context) sourcesapp.SignalsReport {
	return sourcesapp.LoadSignals(ctx, a.core, a.sourcesHeld)
}

func (a *App) healthInput(ctx context.Context) (sourcesHealthInput, error) {
	in, err := sourcesapp.HealthFor(ctx, a.core)
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
