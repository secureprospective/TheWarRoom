package platform

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// MaxLogFiles is how many per-launch logs are kept.
const MaxLogFiles = 10

// SetupLogging tees the log to a per-launch file under <dir>/logs as well as stderr, because a
// launcher start has no terminal. A failure is returned, not fatal: logging must never block
// startup.
func SetupLogging(dir string) error {
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return fmt.Errorf("create log dir %q: %w", logDir, err)
	}
	name := fmt.Sprintf("thewarroom-%s.log", time.Now().UTC().Format("20060102T150405Z"))
	// Never closed: it is the log sink until the process exits.
	f, err := os.OpenFile(filepath.Join(logDir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	PruneLogs(logDir)
	return nil
}

// PruneLogs keeps the newest MaxLogFiles. UTC timestamps in the names sort chronologically.
// Best-effort.
func PruneLogs(logDir string) {
	matches, err := filepath.Glob(filepath.Join(logDir, "thewarroom-*.log"))
	if err != nil || len(matches) <= MaxLogFiles {
		return
	}
	sort.Strings(matches)
	for _, old := range matches[:len(matches)-MaxLogFiles] {
		_ = os.Remove(old)
	}
}
