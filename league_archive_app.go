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
	"slices"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguearchive"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
)

// leagueArchiveSource is the league archive's id in sources.csv.
const leagueArchiveSource = "league_archive"

// labFrozenInputs names the manifest of the archive files the frozen Lab data was compiled from.
const labFrozenInputs = "Legacy NFL 2013–2026 as frozen for the Lab"

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

// checkArchiveDir accepts an absolute folder holding at least one season's league.json.
func checkArchiveDir(dir string) error {
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("the archive folder must be a full path, not %q", dir)
	}
	seasons, err := filepath.Glob(filepath.Join(dir, "[0-9][0-9][0-9][0-9]", "league.json"))
	if err != nil || len(seasons) == 0 {
		return fmt.Errorf("%s holds no season folder with a league.json", dir)
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
	if err := checkArchiveDir(dir); err != nil {
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
	rep, err := a.importArchiveFrom(ctx, s.LeagueArchiveDir)
	if err != nil {
		if lerr := a.history.LoadFailed(ctx, leagueArchiveSource, err); lerr != nil {
			log.Printf("the war room: %v", lerr)
		}
		return "", err
	}
	m, _, err := a.publishLabManifest(ctx)
	if err != nil {
		return "", err
	}
	var errorBodies int
	for _, e := range rep.Errors {
		errorBodies += e.Files
	}
	compiled, err := a.compileFacts(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Looked at %s files: %s new, %s already held. MFL error bodies, kept as errors: %s. "+
		"The Lab's inputs: %s files pinned, %s not in the archive. %s", thousands(rep.Files), thousands(rep.Recorded),
		thousands(rep.Held), thousands(errorBodies), thousands(len(m.Entries)-curatedCount(m)), thousands(len(m.Missing())), compiled), nil
}

func (a *App) importArchiveFrom(ctx context.Context, dir string) (leaguearchive.Report, error) {
	if err := checkArchiveDir(dir); err != nil {
		return leaguearchive.Report{}, err
	}
	fetches, err := a.history.SourceFetches(ctx, leagueArchiveSource)
	if err != nil {
		return leaguearchive.Report{}, fmt.Errorf("app: import league archive: %w", err)
	}
	held := make(map[string]bool, len(fetches))
	for _, f := range fetches {
		held[leaguearchive.HeldKey(f.URL, f.SHA256)] = true
	}
	began := time.Now()
	rep, err := leaguearchive.Import(ctx, os.DirFS(dir), leagueArchiveSource, held, a.history)
	if err != nil {
		return rep, fmt.Errorf("app: import league archive: %w", err)
	}
	if err := a.history.RecordImport(ctx, leagueArchiveSource, rep.Files, rep.Recorded); err != nil {
		return rep, fmt.Errorf("app: import league archive: %w", err)
	}
	log.Printf("the war room: league archive: %d files, %d recorded, %d held, in %s",
		rep.Files, rep.Recorded, rep.Held, time.Since(began).Round(time.Millisecond))
	return rep, nil
}

// labManifest builds the manifest of the frozen Lab's inputs over the archived league archive:
// each file the Lab read, pinned to the archived capture with the same bytes, captured before
// the Lab data was built.
func (a *App) labManifest(ctx context.Context) (manifest.Manifest, error) {
	inputs, err := lab.Inputs()
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("app: lab inputs: %w", err)
	}
	data, err := a.labData()
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("app: lab data: %w", err)
	}
	built, err := time.Parse(time.RFC3339, data.Built)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("app: lab built time %q: %w", data.Built, err)
	}
	fetches, err := a.history.SourceFetches(ctx, leagueArchiveSource)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("app: lab manifest: %w", err)
	}
	pins := make([]manifest.Pin, len(inputs))
	for i, in := range inputs {
		pins[i] = manifest.Pin{File: in.File, SHA256: in.SHA256}
	}
	held := make([]manifest.Fetch, len(fetches))
	for i, f := range fetches {
		held[i] = manifest.Fetch{ID: f.ID, URL: f.URL, SHA256: f.SHA256, Error: f.Error, FetchedAt: f.FetchedAt}
	}
	m, err := manifest.Build(labFrozenInputs, pins, held, slices.Max(data.Years), built)
	if err != nil {
		return manifest.Manifest{}, err //nolint:wrapcheck // names the manifest
	}
	readings := lab.NoteReadings()
	return m.WithCurated(manifest.Pin{File: readings.File, SHA256: readings.SHA256}) //nolint:wrapcheck // names the manifest
}

// publishLabManifest stores the Lab's input manifest; published is false when it is held already.
func (a *App) publishLabManifest(ctx context.Context) (manifest.Manifest, bool, error) {
	m, err := a.labManifest(ctx)
	if err != nil {
		return manifest.Manifest{}, false, err
	}
	enc, err := m.Encode()
	if err != nil {
		return manifest.Manifest{}, false, err //nolint:wrapcheck // names the manifest
	}
	published, err := a.history.PublishManifest(ctx, historyManifest(m, enc))
	if err != nil {
		return manifest.Manifest{}, false, fmt.Errorf("app: publish lab manifest: %w", err)
	}
	return m, published, nil
}

func curatedCount(m manifest.Manifest) int {
	n := 0
	for _, e := range m.Entries {
		if e.Curated() {
			n++
		}
	}
	return n
}

// thousands writes n with comma separators, as the app shows counts.
func thousands(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0 && s[i-1] != '-'; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
