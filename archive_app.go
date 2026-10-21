package main

import (
	"context"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

// archiveOpponentFallback reads Race's held weeks under the caller's SeasonMu.
func (a *App) archiveOpponentFallback(week int, team string) string {
	for _, w := range a.raceHeld.Snapshot().Weeks.Value {
		if w.Week != week {
			continue
		}
		if g, ok := season.FindGame(w, team); ok {
			if g.Home.Franchise == team {
				return g.Away.Franchise
			}
			return g.Home.Franchise
		}
	}
	return ""
}

// readArchiveFacts reads the verified archive snapshot through the service and keeps it on App.
// The caller holds factsMu, which guards archiveCache.
func (a *App) readArchiveFacts(ctx context.Context) (archiveapp.Snapshot, bool, error) {
	notes := lab.NoteReadings()
	src := archiveapp.Source{History: a.core.History, Bodies: sourcesapp.CuratedBodies{Archive: a.core.History, Curated: map[string][]byte{notes.SHA256: notes.Body}},
		Manifest: labFrozenInputs, Compiler: factsCompiler}
	cache, stale, err := archiveapp.Read(ctx, src, a.archiveCache)
	if err != nil {
		return archiveapp.Snapshot{}, false, err //nolint:wrapcheck // the service's errors already start "archive:"
	}
	if cache.SHA != "" {
		a.archiveCache = cache
	}
	return cache, stale, nil
}

// readCommsArchive borrows the snapshot under its caller's FactsMu; core detaches the columns.
func (a *App) readCommsArchive(ctx context.Context) (facts.RosterWeekColumns, facts.NflGameColumns, error) {
	cache, _, err := a.readArchiveFacts(ctx)
	return cache.Facts.RosterWeek, cache.Facts.NflGame, err
}

func (a *App) configureArchive() {
	a.core.ArchiveOpponentFallback = a.archiveOpponentFallback
	a.core.ReadArchive = a.readArchiveFacts
	a.core.ReadCommsArchive = a.readCommsArchive
}
