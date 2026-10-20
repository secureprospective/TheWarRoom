package main

import (
	"context"
	"fmt"
	"time"

	archiveapp "github.com/secureprospective/TheWarRoom/internal/app/archive"

	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
)

type ArchiveBoard struct {
	Board   ReportBoard  `json:"board"`
	Season  int          `json:"season"`
	Seasons []RaceChoice `json:"seasons"`
	Team    string       `json:"team"`
	Other   string       `json:"other"`
	Teams   []RaceChoice `json:"teams"`
}

func (a *App) TargetArchive(season int, team, other string) (ArchiveBoard, error) {
	if err := a.ready(); err != nil {
		return ArchiveBoard{}, fmt.Errorf("archive: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, factsTimeout)
	defer cancel()
	if other == "" {
		other = a.archiveOpponent(team)
	}
	a.factsMu.Lock()
	defer a.factsMu.Unlock()
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
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	for _, m := range a.seasonLineups.value.Matchups {
		if a.seasonLineups.value.Week != week {
			break
		}
		if m.Home == team {
			return m.Away
		}
		if m.Away == team {
			return m.Home
		}
	}
	for _, w := range a.race.weeks.value {
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

func (a *App) readArchiveFacts(ctx context.Context) (archiveapp.Snapshot, bool, error) {
	attempts, err := a.history.Snapshots(ctx, labFrozenInputs)
	if err != nil {
		return archiveapp.Snapshot{}, false, fmt.Errorf("archive: snapshots: %w", err)
	}
	for i, r := range attempts {
		if r.Error != "" || r.ResultSHA == "" || r.Compiler != factsCompiler {
			continue
		}
		stale := i != 0
		latest, ok, err := a.history.LatestManifest(ctx, labFrozenInputs)
		if err != nil {
			return archiveapp.Snapshot{}, false, fmt.Errorf("archive: latest manifest: %w", err)
		}
		stale = stale || !ok || latest.SHA256 != r.ManifestSHA || r.RulesHash != lab.CheckExceptions().SHA256
		if a.archiveCache.SHA == r.ResultSHA {
			return a.archiveCache, stale, nil
		}
		rec, err := a.history.ManifestBySHA(ctx, r.ManifestSHA)
		if err != nil {
			return archiveapp.Snapshot{}, false, fmt.Errorf("archive: snapshot manifest: %w", err)
		}
		m, err := manifest.Decode(rec.Name, rec.SHA256, rec.Entries)
		if err != nil {
			return archiveapp.Snapshot{}, false, fmt.Errorf("archive: decode manifest: %w", err)
		}
		cache, err := resolveArchive(ctx, curatedBodies{archive: a.history, curated: map[string][]byte{lab.NoteReadings().SHA256: lab.NoteReadings().Body}}, m)
		if err != nil {
			return archiveapp.Snapshot{}, false, err
		}
		if cache.Facts.Fingerprint() != r.ResultSHA {
			return archiveapp.Snapshot{}, false, fmt.Errorf("archive: rebuilt facts differ from the verified snapshot")
		}
		cache.SHA = r.ResultSHA
		a.archiveCache = cache
		return cache, stale, nil
	}
	return archiveapp.Snapshot{}, false, nil
}

func resolveArchive(ctx context.Context, bodies manifest.Bodies, m manifest.Manifest) (archiveapp.Snapshot, error) {
	gaps, err := lab.FrozenGaps()
	if err != nil {
		return archiveapp.Snapshot{}, fmt.Errorf("archive: gaps: %w", err)
	}
	resolved, _, err := m.ResolveWithGaps(ctx, bodies, gaps)
	if err != nil {
		return archiveapp.Snapshot{}, fmt.Errorf("archive: pinned inputs: %w", err)
	}
	inputs := []facts.ResolvedInput{}
	var notes []facts.RawNoteReading
	var cutoff time.Time
	for _, r := range resolved {
		if r.Entry.Curated() {
			notes, err = facts.ParseCurated(r.Body)
			if err != nil {
				return archiveapp.Snapshot{}, fmt.Errorf("archive: curated notes: %w", err)
			}
			continue
		}
		inputs = append(inputs, facts.ResolvedInput{File: r.Entry.File, SHA256: r.Entry.SHA256, Bytes: r.Bytes, Body: r.Body})
		at, err := time.Parse(time.RFC3339Nano, r.Entry.CapturedAt)
		if err != nil {
			return archiveapp.Snapshot{}, fmt.Errorf("archive: capture time: %w", err)
		}
		if at.After(cutoff) {
			cutoff = at
		}
	}
	compiled, err := facts.Compile(ctx, inputs, notes)
	if err != nil {
		return archiveapp.Snapshot{}, fmt.Errorf("archive: compile pinned snapshot: %w", err)
	}
	return archiveapp.Snapshot{Facts: compiled, Cutoff: cutoff.Local().Format("Jan 2, 2006"), Through: cutoff}, nil
}
