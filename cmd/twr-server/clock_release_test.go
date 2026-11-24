//go:build !practice

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A release build ignores the practice clock, official-directory and live-directory variables.
func TestReleaseServerIgnoresPracticeSeams(t *testing.T) {
	dir := t.TempDir()
	clockFile := filepath.Join(dir, "clock")
	if err := os.WriteFile(clockFile, []byte("2001-01-01T00:00:00Z"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TWR_PRACTICE_CLOCK_FILE", clockFile)
	t.Setenv("TWR_PRACTICE_OFFICIAL_DIR", dir)
	t.Setenv("TWR_PRACTICE_LIVE_DIR", dir)
	if got := serverClock().Now(); time.Since(got).Abs() > time.Minute {
		t.Fatalf("release clock read the practice file: %v", got)
	}
	network := nflverseFiles(nil)
	files, first, refresh := officialSource(network)
	if first != officialFirst || refresh != officialRefresh || files == nil {
		t.Fatalf("release official source changed: %v %v", first, refresh)
	}
	if _, _, ok := startPracticeLive(t.Context(), nil, nil, nil); ok {
		t.Fatal("release live job read the practice live directory")
	}
}
