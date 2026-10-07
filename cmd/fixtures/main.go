package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) (err error) {
	flags := flag.NewFlagSet("fixtures", flag.ContinueOnError)
	path := flags.String("db", "", "league snapshot database")
	historyPath := flags.String("history", "", "history snapshot database")
	whatifPath := flags.String("whatif", "", "what-if snapshot database (phase log, calendar)")
	out := flags.String("out", "", "fixture output directory")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("fixtures: arguments: %w", err)
	}
	if *path == "" || *historyPath == "" || *whatifPath == "" || *out == "" || flags.NArg() != 0 {
		return fmt.Errorf("fixtures: -db, -history, -whatif and -out required; no positional arguments")
	}
	temp, err := os.MkdirTemp("", "warroom-fixtures-")
	if err != nil {
		return fmt.Errorf("fixtures: temp directory: %w", err)
	}
	defer func() { err = errors.Join(err, os.RemoveAll(temp)) }()
	pools, err := copyPools(ctx, *path, temp)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, pools.Close()) }()
	hp, err := copyPools(ctx, *historyPath, temp)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, hp.Close()) }()
	wp, err := copyPools(ctx, *whatifPath, temp)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, wp.Close()) }()
	snap, clock, err := build(ctx, pools, hp, wp)
	if err != nil {
		return err
	}
	if err := writeSnapshot(*out, snap); err != nil {
		return err
	}
	if err := writeEnvelopeDemo(*out, snap); err != nil {
		return err
	}
	return writeClock(*out, clock)
}

func copyPools(ctx context.Context, src, temp string) (*db.Pools, error) {
	body, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("fixtures: read snapshot: %w", err)
	}
	file, err := os.CreateTemp(temp, "snapshot-*.db")
	if err != nil {
		return nil, fmt.Errorf("fixtures: create copy: %w", err)
	}
	_, writeErr := file.Write(body)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return nil, fmt.Errorf("fixtures: copy snapshot: %w", err)
	}
	pools, err := db.Open(ctx, file.Name())
	if err != nil {
		return nil, fmt.Errorf("fixtures: open copy: %w", err)
	}
	return pools, nil
}

type offline struct{}

func (offline) Fetch(context.Context) (league.RawConfig, error) {
	return league.RawConfig{}, fmt.Errorf("fixtures run offline")
}

type clockReading = snapshot.Sourced[leagueclock.Reading]

// build runs the app's own stores over the snapshot copies and the same builders as the bindings.
func build(ctx context.Context, pools, hp, wp *db.Pools) (snapshot.Snapshot, clockReading, error) {
	rb := rulebook.New(pools)
	if err := rb.Initialize(ctx, offline{}); err != nil {
		return snapshot.Snapshot{}, clockReading{}, fmt.Errorf("fixtures: rulebook: %w", err)
	}
	mirror := state.NewMirror(pools, rb)
	if err := mirror.Initialize(ctx); err != nil {
		return snapshot.Snapshot{}, clockReading{}, fmt.Errorf("fixtures: mirror: %w", err)
	}
	reg, err := measures.Embedded()
	if err != nil {
		return snapshot.Snapshot{}, clockReading{}, fmt.Errorf("fixtures: registry: %w", err)
	}
	hs := history.New(hp, reg)
	if err := hs.Initialize(ctx); err != nil {
		return snapshot.Snapshot{}, clockReading{}, fmt.Errorf("fixtures: history: %w", err)
	}
	dir, err := archivedDirectory(ctx, hs, mirror.Season())
	if err != nil {
		return snapshot.Snapshot{}, clockReading{}, err
	}
	snap, err := snapshot.Build(ctx, snapshot.NewSource(mirror, rb), dir)
	if err != nil {
		return snapshot.Snapshot{}, clockReading{}, fmt.Errorf("fixtures: build: %w", err)
	}
	snap.League.Provenance.Kind, snap.Franchises.Provenance.Kind = "fixture", "fixture"
	snap.Rosters.Provenance.Kind, snap.Players.Provenance.Kind = "fixture", "fixture"
	clock, err := buildClock(ctx, wp, rb, mirror, snap)
	return snap, clock, err
}

// buildClock reads the clock at the roster snapshot's instant, so the fixture is reproducible.
func buildClock(ctx context.Context, wp *db.Pools, rb *rulebook.Store, mirror *state.Mirror,
	snap snapshot.Snapshot) (clockReading, error) {
	whatif := state.New(wp, ingestion.LeagueID, mirror.Season(), rb)
	if err := whatif.Initialize(ctx, mirror); err != nil {
		return clockReading{}, fmt.Errorf("fixtures: what-if league: %w", err)
	}
	at, err := time.Parse(time.RFC3339, snap.Rosters.Provenance.Freshness.FetchedAt)
	if err != nil {
		return clockReading{}, fmt.Errorf("fixtures: clock time: %w", err)
	}
	fresh := snap.Rosters.Provenance.Freshness
	fresh.Note = "phase log and commissioner calendar from a what-if snapshot; league windows not yet captured"
	clock, err := snapshot.BuildClock(ctx, at, mirror.Season(), whatif, nil, snapshot.Provenance{
		Source: "phase-log+commissioner-calendar", Kind: "fixture", Freshness: fresh,
	})
	if err != nil {
		return clockReading{}, fmt.Errorf("fixtures: clock: %w", err)
	}
	return clock, nil
}

func archivedDirectory(ctx context.Context, hs *history.Store, season int) (snapshot.Directory, error) {
	dir := snapshot.Directory{Provenance: snapshot.Provenance{Source: "mfl-players-archive", Kind: "fixture",
		Freshness: domain.Freshness{State: domain.FreshFail, Note: "no archived players export"}}}
	_, err := hs.ArchivedBodies(ctx, "TYPE=players", func(src string, body []byte, at time.Time) bool {
		if !ingestion.SameExport(src, "players", season) {
			return false
		}
		raw, err := players.Parse(ctx, body)
		if err != nil {
			return false
		}
		lk, err := normalize.NewLookup(raw)
		if err != nil {
			return false
		}
		dir.Lookup = lk
		dir.Provenance.Freshness = domain.Freshness{State: domain.FreshStale, FetchedAt: at.UTC().Format(time.RFC3339), Note: "archived players export; no live fetch attempted"}
		return true
	})
	if err != nil {
		return dir, fmt.Errorf("fixtures: archived directory: %w", err)
	}
	return dir, nil
}

func writeSnapshot(out string, snap snapshot.Snapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("fixtures: encode: %w", err)
	}
	if err := os.MkdirAll(out, 0o750); err != nil {
		return fmt.Errorf("fixtures: output directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(out, "snapshot.json"), append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("fixtures: write: %w", err)
	}
	return nil
}
