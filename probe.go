package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/output"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

const probeStepBudget = 20 * time.Second

// probeStep runs one named init step with a timeout, so a HANG surfaces as a
// "TIMED OUT" line (exit 2) and an ERROR as a "FAILED" line (exit 1) instead of a
// frozen black window. Returns only on success.
func probeStep(ctx context.Context, name string, fn func(context.Context) error) {
	c, cancel := context.WithTimeout(ctx, probeStepBudget)
	start := time.Now()
	log.Printf("PROBE  %-30s → starting", name)
	done := make(chan error, 1)
	go func() { done <- fn(c) }()
	select {
	case err := <-done:
		cancel()
		if err != nil {
			log.Printf("PROBE  %-30s FAILED in %s: %v", name, time.Since(start), err)
			os.Exit(1)
		}
		log.Printf("PROBE  %-30s ok in %s", name, time.Since(start))
	case <-c.Done():
		cancel()
		log.Printf("PROBE  %-30s TIMED OUT after %s (this step is the hang)", name, probeStepBudget)
		os.Exit(2)
	}
}

// runProbe is the headless startup diagnostic (`thewarroom -probe`). It runs the
// EXACT store-floor init chain app.startup runs, but WITHOUT Wails — each step
// timed and timeout-bounded. Run it over SSH against the real DB to see precisely
// where startup stalls or errors, with no UI thread and no compositor to hide it.
func runProbe() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	ctx := context.Background()

	path, err := databasePath()
	if err != nil {
		log.Fatalf("PROBE resolve db path: %v", err)
	}
	season, err := strconv.Atoi(ingestion.SeasonYear)
	if err != nil {
		log.Fatalf("PROBE season parse: %v", err)
	}
	client, err := mfl.New("api", 2)
	if err != nil {
		log.Fatalf("PROBE mfl client: %v", err)
	}
	log.Printf("PROBE  db path = %s", path)

	var pools *db.Pools
	probeStep(ctx, "db.Open (pools + WAL)", func(c context.Context) error {
		p, e := db.Open(c, path)
		pools = p
		return e //nolint:wrapcheck // diagnostic: surface the raw error verbatim
	})
	defer func() { _ = pools.Close() }()

	// Same order as initStoreFloor. On an already-seeded DB rulebook/state make no
	// network call; the Ship-4 suspect is state.Initialize (schema + migrateMoneyCents
	// + dropLegacyMoneyColumns + load).
	rb := rulebook.New(pools)
	st := state.New(pools, ingestion.LeagueID, season, rb)

	probeStep(ctx, "params.Initialize", params.New(pools).Initialize)
	probeStep(ctx, "rulebook.Initialize", func(c context.Context) error {
		return rb.Initialize(c, league.APISource{Client: client, Year: ingestion.SeasonYear, LeagueID: ingestion.LeagueID})
	})
	probeStep(ctx, "state.Initialize (Ship-4)", func(c context.Context) error {
		return st.Initialize(c, probeSeed{})
	})
	probeStep(ctx, "transactions.New", func(_ context.Context) error {
		_, e := transactions.New(st.Writer())
		return e //nolint:wrapcheck // diagnostic
	})
	probeStep(ctx, "output.Initialize", output.New(pools).Initialize)

	log.Printf("PROBE  ALL STEPS PASSED — startup chain is healthy on this DB.")
}

// probeSeed is a no-network seed source. state.Initialize only calls Rosters on an
// UNSEEDED DB; the probe targets an existing seeded DB, so a call here is itself the
// finding (the DB is empty and would need a network re-seed at startup).
type probeSeed struct{}

func (probeSeed) Rosters(_ context.Context) ([]domain.Roster, error) {
	return nil, errProbeUnseeded
}

var errProbeUnseeded = errProbe("PROBE: DB is unseeded — state wanted a network roster fetch; this DB has no state")

type errProbe string

func (e errProbe) Error() string { return string(e) }
