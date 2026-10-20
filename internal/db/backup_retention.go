package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type backupSet struct {
	path     string
	manifest RawBackupManifest
}

func (b *Backups) sets(ctx context.Context) ([]backupSet, []string, error) {
	root := filepath.Join(b.dir, "backups")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("db: list backup sets: %w", err)
	}
	var sets []backupSet
	var reports []string
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		manifest, err := b.verify(ctx, path)
		if err != nil {
			reports = append(reports, fmt.Sprintf("left %s: %v", entry.Name(), err))
			continue
		}
		sets = append(sets, backupSet{path, manifest})
	}
	sort.Slice(sets, func(i, j int) bool {
		if sets[i].manifest.CapturedAt.Equal(sets[j].manifest.CapturedAt) {
			return sets[i].path > sets[j].path
		}
		return sets[i].manifest.CapturedAt.After(sets[j].manifest.CapturedAt)
	})
	return sets, reports, nil
}

// Prune reports foreign or invalid directories without deleting them. A caller can
// protect a set selected for restore; Restore also holds this coordinator's mutex.
func (b *Backups) Prune(ctx context.Context, protected string) ([]string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sets, reports, err := b.sets(ctx)
	if err != nil {
		return reports, err
	}
	keep := retentionSets(sets)
	for _, set := range sets {
		if keep[set.path] || filepath.Clean(set.path) == filepath.Clean(protected) {
			continue
		}
		if err := os.RemoveAll(set.path); err != nil {
			return reports, fmt.Errorf("db: prune backup: %w", err)
		}
	}
	if len(sets) > 0 {
		return reports, syncDir(filepath.Join(b.dir, "backups"))
	}
	return reports, nil
}

func weekStart(at time.Time) time.Time {
	at = at.UTC()
	day := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
	return day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
}

func retentionSets(sets []backupSet) map[string]bool {
	keep := make(map[string]bool)
	counts := make(map[string]int)
	weeks := make(map[time.Time]bool)
	var cutoff time.Time
	for _, set := range sets {
		kind := set.manifest.Kind
		counts[kind]++
		if kind != "daily" {
			if counts[kind] <= 5 {
				keep[set.path] = true
			}
			continue
		}
		if counts[kind] <= 7 {
			keep[set.path] = true
			cutoff = weekStart(set.manifest.CapturedAt)
			continue
		}
		week := weekStart(set.manifest.CapturedAt)
		if !week.Before(cutoff) || week.Before(cutoff.AddDate(0, 0, -28)) || weeks[week] {
			continue
		}
		keep[set.path], weeks[week] = true, true
	}
	return keep
}
