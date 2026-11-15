package sources

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguearchive"
)

func CheckArchiveDir(dir string) error {
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("the archive folder must be a full path, not %q", dir)
	}
	if err := leaguearchive.NewDiskFS(dir).Check(); err != nil {
		return archive.WithMessage(fmt.Errorf("archive folder: %w", err), "The archive folder could not be opened. Check that it is available and try again.")
	}
	return nil
}

func (a Service) ImportArchiveFrom(ctx context.Context, dir string) (leaguearchive.Report, error) {
	if !filepath.IsAbs(dir) {
		return leaguearchive.Report{}, fmt.Errorf("the archive folder must be a full path, not %q", dir)
	}
	fetches, err := a.Archive.SourceFetches(ctx, LeagueArchiveSource)
	if err != nil {
		return leaguearchive.Report{}, archive.WithMessage(fmt.Errorf("app: import league archive: %w", err), "The app could not save the imported files. Check free disk space and folder permissions, then try the import again.")
	}
	held := make(map[string]bool, len(fetches))
	for _, f := range fetches {
		held[leaguearchive.HeldKey(f.URL, f.SHA256)] = true
	}
	began := time.Now()
	rep, err := leaguearchive.Import(ctx, leaguearchive.NewDiskFS(dir), LeagueArchiveSource, held, a.Archive)
	if err != nil {
		return rep, archive.WithMessage(fmt.Errorf("app: import league archive: %w", err), "The app could not save the imported files. Check free disk space and folder permissions, then try the import again.")
	}
	if err := a.Archive.RecordImport(ctx, LeagueArchiveSource, rep.Files, rep.Recorded); err != nil {
		return rep, archive.WithMessage(fmt.Errorf("app: import league archive: %w", err), "The app could not save the imported files. Check free disk space and folder permissions, then try the import again.")
	}
	log.Printf("the war room: league archive: %d files, %d recorded, %d held, in %s",
		rep.Files, rep.Recorded, rep.Held, time.Since(began).Round(time.Millisecond))
	return rep, nil
}

const LeagueArchiveSource = "league_archive"

func (a Service) ImportLeagueArchive(ctx context.Context, dir string) (string, error) {
	rep, err := a.ImportArchiveFrom(ctx, dir)
	if err != nil {
		if lerr := a.Archive.LoadFailed(ctx, LeagueArchiveSource, err); lerr != nil {
			log.Printf("the war room: %v", lerr)
		}
		return "", err
	}
	var errorBodies int
	for _, e := range rep.Errors {
		errorBodies += e.Files
	}
	return fmt.Sprintf("Looked at %s files: %s new, %s already held. MFL error bodies, kept as errors: %s.",
		Thousands(rep.Files), Thousands(rep.Recorded), Thousands(rep.Held), Thousands(errorBodies)), nil
}
