package platform

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHeapProfilesOffWhenUnset(t *testing.T) {
	t.Setenv("TWR_HEAPPROFILE_DIR", "")
	directory := t.TempDir()
	t.Chdir(directory)
	StartHeapProfiles()()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("disabled diagnostic wrote %v", entries)
	}
}

func TestHeapProfilesSamplesAndStops(t *testing.T) {
	directory := t.TempDir()
	if err := recordHeapProfiles(t.Context(), directory, time.Millisecond, 2); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(directory, "heap-002.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		t.Fatal("not a gzip heap profile")
	}
	file, err := os.Open(filepath.Join(directory, "memstats.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || len(rows[0]) != 7 {
		t.Fatalf("unexpected samples: %v", rows)
	}
	if _, err := os.Stat(filepath.Join(directory, "heap-003.pprof")); !os.IsNotExist(err) {
		t.Fatalf("extra profile: %v", err)
	}
}

func TestHeapProfilesCancellationAndFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	directory := t.TempDir()
	if err := recordHeapProfiles(ctx, directory, time.Hour, 20); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "heap-001.pprof")); !os.IsNotExist(err) {
		t.Fatalf("cancelled profile: %v", err)
	}
	if err := recordHeapProfiles(ctx, directory, time.Millisecond, 1); err == nil {
		t.Fatal("must not overwrite existing evidence")
	}
	if err := writeHeapProfile(directory); err == nil {
		t.Fatal("directory is not a writable profile")
	}
}
