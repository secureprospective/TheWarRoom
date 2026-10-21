package main

import (
	"context"
	"fmt"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

// ArchiveBoard is the wire type of League Pulse › Archive; the board is built by internal/app/archive.
type ArchiveBoard struct {
	Board   ReportBoard  `json:"board"`
	Season  int          `json:"season"`
	Seasons []RaceChoice `json:"seasons"`
	Team    string       `json:"team"`
	Other   string       `json:"other"`
	Teams   []RaceChoice `json:"teams"`
}

// TargetArchive is the Archive binding. The opponent defaults to this week's; the read and the
// held snapshot are under factsMu.
func (a *App) TargetArchive(season int, team, other string) (ArchiveBoard, error) {
	if err := a.ready(); err != nil {
		return ArchiveBoard{}, fmt.Errorf("archive: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, factsTimeout)
	defer cancel()
	if other == "" {
		other = a.archiveOpponent(team)
	}
	a.core.FactsMu.Lock()
	defer a.core.FactsMu.Unlock()
	cache, stale, err := a.readArchiveFacts(ctx)
	if err != nil {
		return ArchiveBoard{}, err
	}
	return buildArchive(cache, stale, season, team, other)
}

// buildArchive builds the board in the feature service and converts it to its wire type.
func buildArchive(cache archiveapp.Snapshot, stale bool, season int, team, other string) (ArchiveBoard, error) {
	b, err := archiveapp.Build(cache, stale, season, team, other)
	if err != nil {
		return ArchiveBoard{}, err //nolint:wrapcheck // the service's errors already start "archive:"
	}
	return ArchiveBoard{Board: wireBoard(b.Board), Season: b.Season, Seasons: wireChoices(b.Seasons),
		Team: b.Team, Other: b.Other, Teams: wireChoices(b.Teams)}, nil
}

func (a *App) archiveOpponent(team string) string {
	week := a.marketWeek()
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	for _, m := range a.core.SeasonLineups.Value.Matchups {
		if a.core.SeasonLineups.Value.Week != week {
			break
		}
		if m.Home == team {
			return m.Away
		}
		if m.Away == team {
			return m.Home
		}
	}
	for _, w := range a.race.weeks.Value {
		if w.Week != week {
			continue
		}
		if g, ok := findGame(w, team); ok {
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
	src := archiveapp.Source{History: a.core.History, Bodies: curatedBodies{archive: a.core.History, curated: map[string][]byte{notes.SHA256: notes.Body}},
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
