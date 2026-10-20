package main

import (
	"context"
	"log"
	"time"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

// factsCompiler names the facts compiler's output. Change it whenever the compiled facts or the
// checks change for the same inputs, so the snapshot built by the old compiler reads as stale.
const factsCompiler = sourcesapp.FactsCompiler

// factsTimeout bounds one compile; the full archive compiles in about fifteen seconds.
const factsTimeout = 5 * time.Minute

// compileFactsInBackground compiles the checked facts after launch when the archive has been
// imported and the manifest has no snapshot for this compiler and these rules yet.
func (a *App) compileFactsInBackground() {
	parent := a.workCtx
	if parent == nil {
		parent = a.fallbackParent()
	}
	a.startWorker(func() {
		if a.ready() != nil {
			return
		}
		ctx, cancel := context.WithTimeout(parent, factsTimeout)
		defer cancel()
		// A manifest published by an older build may lack an input this build pins (the curated
		// readings); re-publishing is idempotent and only ever follows an archive import.
		if _, ok, err := a.history.LatestManifest(ctx, labFrozenInputs); err != nil || !ok {
			return
		}
		if _, _, err := a.publishLabManifest(ctx); err != nil {
			log.Printf("the war room: checked facts: %v", err)
			return
		}
		if _, err := a.compileFacts(ctx); err != nil {
			log.Printf("the war room: checked facts: %v", err)
			return
		}
		a.warmArchive(ctx)
		a.fitAssetModel(ctx)
	})
}

// warmArchive reads the verified snapshot into the archive cache once the checked facts are in,
// so League Pulse › Archive opens without its first-read rebuild (about twenty seconds).
// The cache is what the first open would have held anyway.
func (a *App) warmArchive(ctx context.Context) {
	began := time.Now()
	a.factsMu.Lock()
	defer a.factsMu.Unlock()
	if _, _, err := a.readArchiveFacts(ctx); err != nil {
		log.Printf("the war room: league archive: %v", err)
		return
	}
	log.Printf("the war room: league archive read in %s", time.Since(began).Round(time.Millisecond))
}

// compileFacts compiles the published manifest's pinned inputs into the league facts, runs the
// nine checks, and records the attempt with its check results. It does nothing when the latest
// attempt already used these inputs, this compiler and these rules: the facts are a pure function
// of them, so the result, good or failed, would be the same. A failed compile or check keeps the
// last good snapshot, which then reads as stale.
func (a *App) compileFacts(ctx context.Context) (string, error) {
	return a.compileFactsWith(ctx, lab.NoteReadings())
}

// compileFactsWith compiles with the given curated readings, which the manifest must pin.
func (a *App) compileFactsWith(ctx context.Context, readings lab.Curated) (string, error) {
	if !a.factsMu.TryLock() {
		return "The checked facts are already compiling.", nil
	}
	defer a.factsMu.Unlock()
	return a.sourcesService().CompileFactsWith(ctx, readings) //nolint:wrapcheck // preserve the existing contextual error shown by the binding
}

// unapprovedMissing counts the pinned archive files the archive does not hold and no approved gap
// names: until they arrive there is nothing to compile, which is not a failed compile.
