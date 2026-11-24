//go:build practice

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

// Practice-only test seams for the season box (docs/architecture/season-in-a-box.md, B4). This file is absent
// from release binaries: a release server never reads these variables (backup_hosted.go). Never for production.
const (
	// practiceClockEnv names a file holding one RFC 3339 instant. The server's clock reads that instant plus the
	// wall time since the file was last written, so a test moves time by rewriting the file.
	practiceClockEnv = "TWR_PRACTICE_CLOCK_FILE"
	// practiceOfficialEnv names a directory the official-statistics job reads instead of nflverse:
	// stats_player_week_<season>.csv.gz, play_by_play_<season>.csv.gz and, optionally, players.csv.
	practiceOfficialEnv = "TWR_PRACTICE_OFFICIAL_DIR"
)

// serverClock is the hosted server's one clock: the store, the due worker and the official job read it.
func serverClock() leagueservice.DueClock {
	path := os.Getenv(practiceClockEnv)
	if path == "" {
		return leagueservice.NewHostClock()
	}
	log.Printf("DISPOSABLE PRACTICE BUILD: fake clock from %s; never use for real league state", path)
	return fileClock{path: path}
}

type fileClock struct{ path string }

// Now is the file's instant plus the wall time since it was written. An unreadable file is wall time, logged.
func (c fileClock) Now() time.Time {
	info, err := os.Stat(c.path)
	if err != nil {
		log.Printf("practice clock: %v; using wall time", err)
		return time.Now()
	}
	data, err := os.ReadFile(c.path)
	if err != nil {
		log.Printf("practice clock: %v; using wall time", err)
		return time.Now()
	}
	at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(data)))
	if err != nil {
		log.Printf("practice clock: %v; using wall time", err)
		return time.Now()
	}
	return at.Add(max(time.Since(info.ModTime()), 0))
}

func (fileClock) Trusted() error { return nil }

// officialSource is where the official job reads a season's files, and how soon and how often it reads them.
func officialSource(network officialFiles) (officialFiles, time.Duration, time.Duration) {
	dir := os.Getenv(practiceOfficialEnv)
	if dir == "" {
		return network, officialFirst, officialRefresh
	}
	log.Printf("DISPOSABLE PRACTICE BUILD: official statistics from %s; never use for real league state", dir)
	return directoryFiles(dir), time.Second, time.Second
}

func directoryFiles(dir string) officialFiles {
	return func(_ context.Context, season int) (seasonFiles, error) {
		var out seasonFiles
		var err error
		if out.weekly, err = os.ReadFile(filepath.Join(dir, fmt.Sprintf("stats_player_week_%d.csv.gz", season))); err != nil {
			return out, fmt.Errorf("official weekly statistics: %w", err)
		}
		if out.plays, err = os.ReadFile(filepath.Join(dir, fmt.Sprintf("play_by_play_%d.csv.gz", season))); err != nil {
			return out, fmt.Errorf("official play-by-play: %w", err)
		}
		if out.roster, err = os.ReadFile(filepath.Join(dir, "players.csv")); err != nil && !os.IsNotExist(err) {
			return out, fmt.Errorf("nflverse players: %w", err)
		}
		return out, nil
	}
}
