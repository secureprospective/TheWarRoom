package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
)

type archiveCache struct {
	sha       string
	facts     facts.Facts
	pools     map[int64]facts.PoolSeason
	cutoff    string
	through   time.Time
	poolError string
}

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
	a.factsMu.Lock()
	defer a.factsMu.Unlock()
	cache, stale, err := a.readArchiveFacts(ctx)
	if err != nil {
		return ArchiveBoard{}, err
	}
	return buildArchive(cache, stale, season, team, other)
}

func (a *App) readArchiveFacts(ctx context.Context) (archiveCache, bool, error) {
	attempts, err := a.history.Snapshots(ctx, labFrozenInputs)
	if err != nil {
		return archiveCache{}, false, fmt.Errorf("archive: snapshots: %w", err)
	}
	for i, r := range attempts {
		if r.Error != "" || r.ResultSHA == "" || r.Compiler != factsCompiler {
			continue
		}
		stale := i != 0
		latest, ok, err := a.history.LatestManifest(ctx, labFrozenInputs)
		if err != nil {
			return archiveCache{}, false, fmt.Errorf("archive: latest manifest: %w", err)
		}
		stale = stale || !ok || latest.SHA256 != r.ManifestSHA || r.RulesHash != lab.CheckExceptions().SHA256
		if a.archiveCache.sha == r.ResultSHA {
			return a.archiveCache, stale, nil
		}
		rec, err := a.history.ManifestBySHA(ctx, r.ManifestSHA)
		if err != nil {
			return archiveCache{}, false, fmt.Errorf("archive: snapshot manifest: %w", err)
		}
		m, err := manifest.Decode(rec.Name, rec.SHA256, rec.Entries)
		if err != nil {
			return archiveCache{}, false, fmt.Errorf("archive: decode manifest: %w", err)
		}
		cache, err := resolveArchive(ctx, curatedBodies{archive: a.history, curated: map[string][]byte{lab.NoteReadings().SHA256: lab.NoteReadings().Body}}, m)
		if err != nil {
			return archiveCache{}, false, err
		}
		if cache.facts.Fingerprint() != r.ResultSHA {
			return archiveCache{}, false, fmt.Errorf("archive: rebuilt facts differ from the verified snapshot")
		}
		cache.pools, cache.poolError = a.readPinnedPools(ctx)
		cache.sha = r.ResultSHA
		a.archiveCache = cache
		return cache, stale, nil
	}
	return archiveCache{}, false, nil
}

func resolveArchive(ctx context.Context, bodies manifest.Bodies, m manifest.Manifest) (archiveCache, error) {
	gaps, err := lab.FrozenGaps()
	if err != nil {
		return archiveCache{}, fmt.Errorf("archive: gaps: %w", err)
	}
	resolved, _, err := m.ResolveWithGaps(ctx, bodies, gaps)
	if err != nil {
		return archiveCache{}, fmt.Errorf("archive: pinned inputs: %w", err)
	}
	inputs := []facts.ResolvedInput{}
	var notes []facts.RawNoteReading
	var cutoff time.Time
	for _, r := range resolved {
		if r.Entry.Curated() {
			notes, err = facts.ParseCurated(r.Body)
			if err != nil {
				return archiveCache{}, fmt.Errorf("archive: curated notes: %w", err)
			}
			continue
		}
		inputs = append(inputs, facts.ResolvedInput{File: r.Entry.File, SHA256: r.Entry.SHA256, Bytes: r.Bytes, Body: r.Body})
		at, err := time.Parse(time.RFC3339Nano, r.Entry.CapturedAt)
		if err != nil {
			return archiveCache{}, fmt.Errorf("archive: capture time: %w", err)
		}
		if at.After(cutoff) {
			cutoff = at
		}
	}
	compiled, err := facts.Compile(ctx, inputs, notes)
	if err != nil {
		return archiveCache{}, fmt.Errorf("archive: compile pinned snapshot: %w", err)
	}
	pools, err := facts.CompilePools(inputs)
	if err != nil {
		return archiveCache{}, fmt.Errorf("archive: pools: %w", err)
	}
	return archiveCache{facts: compiled, pools: pools, cutoff: cutoff.Local().Format("Jan 2, 2006"), through: cutoff}, nil
}

func (a *App) readPinnedPools(ctx context.Context) (map[int64]facts.PoolSeason, string) {
	pins, err := poolPins()
	if err != nil {
		return nil, err.Error()
	}
	data, err := a.labData()
	if err != nil {
		return nil, err.Error()
	}
	cutoff, err := time.Parse(time.RFC3339, data.Built)
	if err != nil {
		return nil, err.Error()
	}
	fetches, err := a.history.SourceFetches(ctx, leagueArchiveSource)
	if err != nil {
		return nil, err.Error()
	}
	held := make([]manifest.Fetch, len(fetches))
	for i, f := range fetches {
		held[i] = manifest.Fetch{ID: f.ID, URL: f.URL, SHA256: f.SHA256, Error: f.Error, FetchedAt: f.FetchedAt}
	}
	m, err := manifest.Build("archive pools", pins, held, 2026, cutoff)
	if err != nil {
		return nil, err.Error()
	}
	resolved, err := m.Resolve(ctx, a.history)
	if err != nil {
		return nil, err.Error()
	}
	inputs := []facts.ResolvedInput{}
	for _, r := range resolved {
		body, err := a.history.Body(ctx, r.Entry.SHA256)
		if err != nil {
			return nil, err.Error()
		}
		inputs = append(inputs, facts.ResolvedInput{File: r.Entry.File, SHA256: r.Entry.SHA256, Bytes: r.Bytes, Body: body})
	}
	pools, err := facts.CompilePools(inputs)
	if err != nil {
		return nil, err.Error()
	}
	return pools, ""
}

//go:embed pool_inputs.csv
var poolInputsCSV []byte

// poolPins are the archived pool files the Pools report reads, each pinned by its sha256.
func poolPins() ([]manifest.Pin, error) {
	rows, err := csv.NewReader(bytes.NewReader(poolInputsCSV)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("pool pins: %w", err)
	}
	pins := []manifest.Pin{}
	for _, row := range rows[1:] {
		if len(row) != 2 {
			return nil, fmt.Errorf("pool pins: invalid row")
		}
		if hash, err := hex.DecodeString(row[1]); err != nil || len(hash) != 32 {
			return nil, fmt.Errorf("pool pins: invalid hash")
		}
		pins = append(pins, manifest.Pin{File: row[0], SHA256: row[1]})
	}
	return pins, nil
}
