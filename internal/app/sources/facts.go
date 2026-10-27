package sources

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

func (a Service) CompileFactsWith(ctx context.Context, readings lab.Curated) (string, error) {
	rec, ok, err := a.Facts.LatestManifest(ctx, LabFrozenInputs)
	if err != nil || !ok {
		return "No manifest to compile yet.", err //nolint:wrapcheck // the store names the manifest
	}
	m, err := manifest.Decode(rec.Name, rec.SHA256, rec.Entries)
	if err != nil {
		return "", err //nolint:wrapcheck // names the manifest
	}
	rules := lab.CheckExceptions()
	attempts, err := a.Facts.Snapshots(ctx, LabFrozenInputs)
	if err != nil {
		return "", fmt.Errorf("app: checked facts: %w", err)
	}
	if len(attempts) > 0 && attempts[0].ManifestSHA == m.SHA256 && attempts[0].Compiler == FactsCompiler &&
		attempts[0].RulesHash == rules.SHA256 {
		return "The checked facts are already compiled from these inputs.", nil
	}
	gaps, err := lab.FrozenGaps()
	if err != nil {
		return "", fmt.Errorf("app: frozen input gaps: %w", err)
	}
	if waiting := unapprovedMissing(m, gaps); waiting > 0 {
		return fmt.Sprintf("The checked facts wait for %s pinned files the archive does not hold yet.", Thousands(waiting)), nil
	}
	began := time.Now()
	attempt := history.SnapshotRecord{ManifestSHA: m.SHA256, Compiler: FactsCompiler, RulesHash: rules.SHA256}
	result, checks, cerr := compileManifest(ctx, CuratedBodies{Archive: a.Facts,
		Curated: map[string][]byte{readings.SHA256: readings.Body}}, m, gaps, rules, readings.File)
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

func (a Service) recordFacts(ctx context.Context, attempt history.SnapshotRecord, checks []facts.CheckResult, cerr error) error {
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
	if err := a.Facts.RecordSnapshot(ctx, attempt); err != nil {
		return fmt.Errorf("app: checked facts: %w", err)
	}
	return nil
}

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

func factsOutcome(cerr error) string {
	if cerr != nil {
		return "The checked facts did not compile, so the last good copy is kept: " + cerr.Error()
	}
	return "The checked facts compiled and every check passed."
}

// CuratedBodies resolves curated inputs from the bytes the app ships with, and every other pin
// from the archive.
type CuratedBodies struct {
	Archive manifest.Bodies
	Curated map[string][]byte
}

func (b CuratedBodies) Body(ctx context.Context, sha string) ([]byte, error) {
	if body, ok := b.Curated[sha]; ok {
		return body, nil
	}
	return b.Archive.Body(ctx, sha) //nolint:wrapcheck // the archive names the body
}

const FactsCompiler = "facts 1 (ring 3: 37 relations, nine checks)"

// StartupFacts resumes only an already-imported archive; no manifest means no work.
func StartupFacts(ctx context.Context, exists func(context.Context) (bool, error),
	publish, compile func(context.Context) error, warm, fit func(context.Context), logFailure func(string)) {
	if ok, err := exists(ctx); err != nil || !ok {
		return
	}
	if err := publish(ctx); err != nil {
		logFailure(fmt.Sprintf("the war room: checked facts: %v", err))
		return
	}
	if err := compile(ctx); err != nil {
		logFailure(fmt.Sprintf("the war room: checked facts: %v", err))
		return
	}
	warm(ctx)
	fit(ctx)
}

func WarmArchive(ctx context.Context, began time.Time, read func(context.Context) error, logLine func(string)) {
	if err := read(ctx); err != nil {
		logLine(fmt.Sprintf("the war room: league archive: %v", err))
		return
	}
	logLine(fmt.Sprintf("the war room: league archive read in %s", time.Since(began).Round(time.Millisecond)))
}

func CompileAvailable(lock func() bool, unlock func(), compile func() (string, error)) (string, error) {
	if !lock() {
		return "The checked facts are already compiling.", nil
	}
	defer unlock()
	return compile()
}
