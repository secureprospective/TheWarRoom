package archive

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// History is the part of the history store Read uses: the compile attempts of a manifest name
// and the manifests they pinned.
type History interface {
	Snapshots(ctx context.Context, name string) ([]history.SnapshotRecord, error)
	LatestManifest(ctx context.Context, name string) (history.ManifestRecord, bool, error)
	ManifestBySHA(ctx context.Context, sha string) (history.ManifestRecord, error)
}

// Source is everything Read reads, passed in by the composition root.
type Source struct {
	History History
	// Bodies resolves a manifest's pinned inputs: curated ones from the app, the rest from the
	// fetch archive.
	Bodies manifest.Bodies
	// Manifest names the published input manifest; Compiler is the facts compiler whose
	// snapshots count.
	Manifest, Compiler string
}

// Read returns the newest verified snapshot and whether it is stale: an older attempt than the
// newest, or one compiled from a manifest or check rules since replaced. held is the snapshot the
// caller already holds; it is returned as is when it is still the verified one, and otherwise
// rebuilt from the pinned inputs and checked against the recorded fingerprint. With no verified
// compile Read returns the zero Snapshot and no error.
func Read(ctx context.Context, src Source, held Snapshot) (Snapshot, bool, error) {
	attempts, err := src.History.Snapshots(ctx, src.Manifest)
	if err != nil {
		return Snapshot{}, false, fmt.Errorf("archive: snapshots: %w", err)
	}
	for i, r := range attempts {
		if r.Error != "" || r.ResultSHA == "" || r.Compiler != src.Compiler {
			continue
		}
		stale := i != 0
		latest, ok, err := src.History.LatestManifest(ctx, src.Manifest)
		if err != nil {
			return Snapshot{}, false, fmt.Errorf("archive: latest manifest: %w", err)
		}
		stale = stale || !ok || latest.SHA256 != r.ManifestSHA || r.RulesHash != lab.CheckExceptions().SHA256
		if held.SHA == r.ResultSHA {
			return held, stale, nil
		}
		rec, err := src.History.ManifestBySHA(ctx, r.ManifestSHA)
		if err != nil {
			return Snapshot{}, false, fmt.Errorf("archive: snapshot manifest: %w", err)
		}
		m, err := manifest.Decode(rec.Name, rec.SHA256, rec.Entries)
		if err != nil {
			return Snapshot{}, false, fmt.Errorf("archive: decode manifest: %w", err)
		}
		cache, err := resolve(ctx, src.Bodies, m)
		if err != nil {
			return Snapshot{}, false, err
		}
		if cache.Facts.Fingerprint() != r.ResultSHA {
			return Snapshot{}, false, errors.New("archive: rebuilt facts differ from the verified snapshot")
		}
		cache.SHA = r.ResultSHA
		return cache, stale, nil
	}
	return Snapshot{}, false, nil
}

func resolve(ctx context.Context, bodies manifest.Bodies, m manifest.Manifest) (Snapshot, error) {
	gaps, err := lab.FrozenGaps()
	if err != nil {
		return Snapshot{}, fmt.Errorf("archive: gaps: %w", err)
	}
	resolved, _, err := m.ResolveWithGaps(ctx, bodies, gaps)
	if err != nil {
		return Snapshot{}, fmt.Errorf("archive: pinned inputs: %w", err)
	}
	inputs := []facts.ResolvedInput{}
	var notes []facts.RawNoteReading
	var cutoff time.Time
	for _, r := range resolved {
		if r.Entry.Curated() {
			notes, err = facts.ParseCurated(r.Body)
			if err != nil {
				return Snapshot{}, fmt.Errorf("archive: curated notes: %w", err)
			}
			continue
		}
		inputs = append(inputs, facts.ResolvedInput{File: r.Entry.File, SHA256: r.Entry.SHA256, Bytes: r.Bytes, Body: r.Body})
		at, err := time.Parse(time.RFC3339Nano, r.Entry.CapturedAt)
		if err != nil {
			return Snapshot{}, fmt.Errorf("archive: capture time: %w", err)
		}
		if at.After(cutoff) {
			cutoff = at
		}
	}
	compiled, err := facts.Compile(ctx, inputs, notes)
	if err != nil {
		return Snapshot{}, fmt.Errorf("archive: compile pinned snapshot: %w", err)
	}
	return Snapshot{Facts: compiled, Cutoff: cutoff.Local().Format("Jan 2, 2006"), Through: cutoff}, nil
}
