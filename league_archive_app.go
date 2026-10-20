package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguearchive"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
)

// leagueArchiveSource is the league archive's id in sources.csv.
const leagueArchiveSource = sourcesapp.LeagueArchiveSource

// labFrozenInputs names the manifest of the archive files the frozen Lab data was compiled from.
const labFrozenInputs = sourcesapp.LabFrozenInputs

// archiveImportTimeout bounds one import of the whole archive.
const archiveImportTimeout = 10 * time.Minute

// appSettings is the app's own configuration, kept as settings.json in its data directory.
type appSettings struct {
	LeagueArchiveDir string `json:"leagueArchiveDir"`
	PhotosOff        bool   `json:"photosOff,omitempty"`
}

func (a *App) settingsPath() string { return filepath.Join(a.dataDir, "settings.json") }

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

// SetLeagueArchiveDir sets the folder the league archive is imported from: the archive's raw/
// folder, holding one folder per season.
func (a *App) SetLeagueArchiveDir(dir string) DataActResult {
	if err := a.ready(); err != nil {
		return DataActResult{Error: err.Error()}
	}
	dir = filepath.Clean(dir)
	if err := sourcesapp.CheckArchiveDir(dir); err != nil {
		return DataActResult{Error: err.Error()}
	}
	s, err := a.readSettings()
	if err != nil {
		return DataActResult{Error: err.Error()}
	}
	s.LeagueArchiveDir = dir
	if err := a.writeSettings(s); err != nil {
		return DataActResult{Error: err.Error()}
	}
	return DataActResult{OK: true, Summary: "League archive folder set to " + dir + "."}
}

// importLeagueArchive records every archive file not held yet, publishes the Lab's input manifest
// over what the archive holds, and compiles the checked facts from it. A failed import is recorded
// against the source.
func (a *App) importLeagueArchive(ctx context.Context) (string, error) {
	s, err := a.readSettings()
	if err != nil {
		return "", err
	}
	if s.LeagueArchiveDir == "" {
		return "", errors.New("no league archive folder is set; set it under Data refresh first")
	}
	return a.sourcesService().ImportLeagueArchive(ctx, s.LeagueArchiveDir, a.compileFacts) //nolint:wrapcheck // preserve the existing contextual error shown by the binding
}

// thousands writes n with comma separators, as the app shows counts.
func thousands(n int) string { return sourcesapp.Thousands(n) }

func (a *App) importArchiveFrom(ctx context.Context, dir string) (leaguearchive.Report, error) {
	return a.sourcesService().ImportArchiveFrom(ctx, dir) //nolint:wrapcheck // preserve the existing contextual error shown by the binding
}

func (a *App) publishLabManifest(ctx context.Context) (manifest.Manifest, bool, error) {
	return a.sourcesService().PublishLabManifest(ctx) //nolint:wrapcheck // preserve the existing contextual error shown by the binding
}
