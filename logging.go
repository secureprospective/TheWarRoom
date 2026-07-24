package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// maxLogFiles bounds the per-launch log files kept under <configdir>/logs.
const maxLogFiles = 10

// setupLogging tees the standard logger to a per-launch file under
// <configdir>/logs in ADDITION to stderr (Tier 3 self-protection). A GUI launch
// from a .desktop entry has no attached terminal, so stderr-only log.Printf goes
// nowhere — this leaves a durable diagnostic trail on disk while a terminal launch
// still sees everything live. Rotation is by file COUNT: the newest maxLogFiles
// launches are kept and older files pruned. A logging failure is returned, NOT
// fatal — the caller keeps stderr and continues (logging must never brick startup).
func setupLogging(dir string) error {
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return fmt.Errorf("create log dir %q: %w", logDir, err)
	}
	name := fmt.Sprintf("thewarroom-%s.log", time.Now().UTC().Format("20060102T150405Z"))
	// Held for the process lifetime as the log sink; the OS closes it on exit.
	f, err := os.OpenFile(filepath.Join(logDir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	pruneLogs(logDir)
	return nil
}

// pruneLogs keeps only the newest maxLogFiles log files. The fixed-width UTC
// timestamp in the name makes lexical order == chronological. Best-effort: a prune
// failure is swallowed (it must not affect startup).
func pruneLogs(logDir string) {
	matches, err := filepath.Glob(filepath.Join(logDir, "thewarroom-*.log"))
	if err != nil || len(matches) <= maxLogFiles {
		return
	}
	sort.Strings(matches)
	for _, old := range matches[:len(matches)-maxLogFiles] {
		_ = os.Remove(old)
	}
}
