package sources

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguearchive"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
)

func CheckArchiveDir(dir string) error {
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("the archive folder must be a full path, not %q", dir)
	}
	if err := leaguearchive.NewDiskFS(dir).Check(); err != nil {
		return fmt.Errorf("archive folder: %w", err)
	}
	return nil
}

func (a Service) ImportArchiveFrom(ctx context.Context, dir string) (leaguearchive.Report, error) {
	if !filepath.IsAbs(dir) {
		return leaguearchive.Report{}, fmt.Errorf("the archive folder must be a full path, not %q", dir)
	}
	fetches, err := a.Archive.SourceFetches(ctx, LeagueArchiveSource)
	if err != nil {
		return leaguearchive.Report{}, fmt.Errorf("app: import league archive: %w", err)
	}
	held := make(map[string]bool, len(fetches))
	for _, f := range fetches {
		held[leaguearchive.HeldKey(f.URL, f.SHA256)] = true
	}
	began := time.Now()
	rep, err := leaguearchive.Import(ctx, leaguearchive.NewDiskFS(dir), LeagueArchiveSource, held, a.Archive)
	if err != nil {
		return rep, fmt.Errorf("app: import league archive: %w", err)
	}
	if err := a.Archive.RecordImport(ctx, LeagueArchiveSource, rep.Files, rep.Recorded); err != nil {
		return rep, fmt.Errorf("app: import league archive: %w", err)
	}
	log.Printf("the war room: league archive: %d files, %d recorded, %d held, in %s",
		rep.Files, rep.Recorded, rep.Held, time.Since(began).Round(time.Millisecond))
	return rep, nil
}

func (a Service) LabManifest(ctx context.Context) (manifest.Manifest, error) {
	inputs, err := lab.Inputs()
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("app: lab inputs: %w", err)
	}
	data, err := a.LabData()
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("app: lab data: %w", err)
	}
	built, err := time.Parse(time.RFC3339, data.Built)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("app: lab built time %q: %w", data.Built, err)
	}
	fetches, err := a.Archive.SourceFetches(ctx, LeagueArchiveSource)
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
	m, err := manifest.Build(LabFrozenInputs, pins, held, slices.Max(data.Years), built)
	if err != nil {
		return manifest.Manifest{}, err //nolint:wrapcheck // names the manifest
	}
	readings := lab.NoteReadings()
	return m.WithCurated(manifest.Pin{File: readings.File, SHA256: readings.SHA256}) //nolint:wrapcheck // names the manifest
}

func (a Service) PublishLabManifest(ctx context.Context) (manifest.Manifest, bool, error) {
	m, err := a.LabManifest(ctx)
	if err != nil {
		return manifest.Manifest{}, false, err
	}
	enc, err := m.Encode()
	if err != nil {
		return manifest.Manifest{}, false, err //nolint:wrapcheck // names the manifest
	}
	published, err := a.Archive.PublishManifest(ctx, HistoryManifest(m, enc))
	if err != nil {
		return manifest.Manifest{}, false, fmt.Errorf("app: publish lab manifest: %w", err)
	}
	return m, published, nil
}

func CuratedCount(m manifest.Manifest) int {
	n := 0
	for _, e := range m.Entries {
		if e.Curated() {
			n++
		}
	}
	return n
}

const LabFrozenInputs = "Legacy NFL 2013–2026 as frozen for the Lab"
const LeagueArchiveSource = "league_archive"

func (a Service) ImportLeagueArchive(ctx context.Context, dir string, compile func(context.Context) (string, error)) (string, error) {
	rep, err := a.ImportArchiveFrom(ctx, dir)
	if err != nil {
		if lerr := a.Archive.LoadFailed(ctx, LeagueArchiveSource, err); lerr != nil {
			log.Printf("the war room: %v", lerr)
		}
		return "", err
	}
	m, _, err := a.PublishLabManifest(ctx)
	if err != nil {
		return "", err
	}
	var errorBodies int
	for _, e := range rep.Errors {
		errorBodies += e.Files
	}
	compiled, err := compile(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Looked at %s files: %s new, %s already held. MFL error bodies, kept as errors: %s. "+
		"The Lab's inputs: %s files pinned, %s not in the archive. %s", Thousands(rep.Files), Thousands(rep.Recorded),
		Thousands(rep.Held), Thousands(errorBodies), Thousands(len(m.Entries)-CuratedCount(m)), Thousands(len(m.Missing())), compiled), nil
}
