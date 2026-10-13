package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/manifest"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// factsCompiler names the facts compiler's output. Change it whenever the compiled facts or the
// checks change for the same inputs, so the snapshot built by the old compiler reads as stale.
const factsCompiler = "facts 1 (ring 3: 37 relations, nine checks)"

// factsTimeout bounds one compile; the full archive compiles in about fifteen seconds.
const factsTimeout = 5 * time.Minute

// compileFactsInBackground compiles the checked facts after launch when the archive has been
// imported and the manifest has no snapshot for this compiler and these rules yet.
func (a *App) compileFactsInBackground(parent context.Context) {
	go func() {
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
	}()
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
	rec, ok, err := a.history.LatestManifest(ctx, labFrozenInputs)
	if err != nil || !ok {
		return "No manifest to compile yet.", err //nolint:wrapcheck // the store names the manifest
	}
	m, err := manifest.Decode(rec.Name, rec.SHA256, rec.Entries)
	if err != nil {
		return "", err //nolint:wrapcheck // names the manifest
	}
	rules := lab.CheckExceptions()
	attempts, err := a.history.Snapshots(ctx, labFrozenInputs)
	if err != nil {
		return "", fmt.Errorf("app: checked facts: %w", err)
	}
	if len(attempts) > 0 && attempts[0].ManifestSHA == m.SHA256 && attempts[0].Compiler == factsCompiler &&
		attempts[0].RulesHash == rules.SHA256 {
		return "The checked facts are already compiled from these inputs.", nil
	}
	gaps, err := lab.FrozenGaps()
	if err != nil {
		return "", fmt.Errorf("app: frozen input gaps: %w", err)
	}
	if waiting := unapprovedMissing(m, gaps); waiting > 0 {
		return fmt.Sprintf("The checked facts wait for %s pinned files the archive does not hold yet.", thousands(waiting)), nil
	}
	began := time.Now()
	attempt := history.SnapshotRecord{ManifestSHA: m.SHA256, Compiler: factsCompiler, RulesHash: rules.SHA256}
	result, checks, cerr := compileManifest(ctx, curatedBodies{archive: a.history,
		curated: map[string][]byte{readings.SHA256: readings.Body}}, m, gaps, rules, readings.File)
	if errors.Is(cerr, context.Canceled) || errors.Is(cerr, context.DeadlineExceeded) {
		return "", cerr // an interrupted compile is no attempt
	}
	debug.FreeOSMemory() // the compiled columns are dropped; hand their memory back
	attempt.ResultSHA, attempt.BuiltAt = result, time.Now()
	if err := a.recordFacts(ctx, attempt, checks, cerr); err != nil {
		return "", err
	}
	log.Printf("the war room: checked facts compiled in %s: %s", time.Since(began).Round(time.Millisecond), factsOutcome(cerr))
	return factsOutcome(cerr), nil
}

// recordFacts stores the attempt with its error and check results.
func (a *App) recordFacts(ctx context.Context, attempt history.SnapshotRecord, checks []facts.CheckResult, cerr error) error {
	if cerr != nil {
		attempt.Error = cerr.Error()
	}
	if len(checks) > 0 {
		b, err := json.Marshal(checks)
		if err != nil {
			return fmt.Errorf("app: checked facts: %w", err)
		}
		attempt.Checks = string(b)
	}
	if err := a.history.RecordSnapshot(ctx, attempt); err != nil {
		return fmt.Errorf("app: checked facts: %w", err)
	}
	return nil
}

// compileManifest resolves the manifest, compiles it and checks the result. The returned sha
// names a good snapshot; it is empty when the compile or a check failed.
func compileManifest(ctx context.Context, bodies manifest.Bodies, m manifest.Manifest, gaps []manifest.Gap,
	rules lab.Curated, readingsFile string) (string, []facts.CheckResult, error) {
	resolved, _, err := m.ResolveWithGaps(ctx, bodies, gaps)
	if err != nil {
		return "", nil, err //nolint:wrapcheck // names every unresolved pin
	}
	var inputs []facts.ResolvedInput
	var notes []facts.RawNoteReading
	for _, r := range resolved {
		if !r.Entry.Curated() {
			inputs = append(inputs, facts.ResolvedInput{File: r.Entry.File, SHA256: r.Entry.SHA256, Bytes: r.Bytes, Body: r.Body})
			continue
		}
		if r.Entry.File != readingsFile {
			return "", nil, fmt.Errorf("manifest %s: unknown curated input %s", m.Name, r.Entry.File)
		}
		if notes, err = facts.ParseCurated(r.Body); err != nil {
			return "", nil, err //nolint:wrapcheck // names the curated row
		}
	}
	compiled, err := facts.Compile(ctx, inputs, notes)
	if err != nil {
		return "", nil, err //nolint:wrapcheck // names the input
	}
	exceptions, err := facts.ParseExceptions(rules.Body)
	if err != nil {
		return "", nil, err //nolint:wrapcheck // names the row
	}
	checks, err := compiled.Check(exceptions)
	if err != nil {
		return "", checks, err //nolint:wrapcheck // names every unpinned difference
	}
	return compiled.Fingerprint(), checks, nil
}

// unapprovedMissing counts the pinned archive files the archive does not hold and no approved gap
// names: until they arrive there is nothing to compile, which is not a failed compile.
func unapprovedMissing(m manifest.Manifest, gaps []manifest.Gap) int {
	approved := map[[2]string]bool{}
	for _, g := range gaps {
		approved[[2]string{g.File, g.SHA256}] = true
	}
	n := 0
	for _, e := range m.Missing() {
		if !approved[[2]string{e.File, e.SHA256}] {
			n++
		}
	}
	return n
}

// curatedBodies resolves curated inputs from the bytes the app ships with, and every other pin
// from the archive.
type curatedBodies struct {
	archive manifest.Bodies
	curated map[string][]byte
}

func (b curatedBodies) Body(ctx context.Context, sha string) ([]byte, error) {
	if body, ok := b.curated[sha]; ok {
		return body, nil
	}
	return b.archive.Body(ctx, sha) //nolint:wrapcheck // the archive names the body
}

func factsOutcome(cerr error) string {
	if cerr != nil {
		return "The checked facts did not compile, so the last good copy is kept: " + cerr.Error()
	}
	return "The checked facts compiled and every check passed."
}
