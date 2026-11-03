package platform

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"
)

// StartHeapProfiles records sampled heap allocations without forcing GC. An unset
// TWR_HEAPPROFILE_DIR starts no worker and opens no files. Profiles contain both
// inuse_space and alloc_space; select either with go tool pprof.
func StartHeapProfiles() func() {
	directory := os.Getenv("TWR_HEAPPROFILE_DIR")
	if directory == "" {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := recordHeapProfiles(ctx, directory, 15*time.Second, 20); err != nil {
			log.Printf("heap diagnostic: %v", err)
		}
	}()
	return func() { cancel(); <-done }
}

func recordHeapProfiles(ctx context.Context, directory string, interval time.Duration, samples int) (err error) {
	if err = os.MkdirAll(directory, 0o700); err != nil { //nolint:gosec // The operator explicitly chooses the diagnostic output directory.
		return fmt.Errorf("create heap profile directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(directory, "memstats.csv"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // Operator-selected directory; fixed filename and exclusive creation.
	if err != nil {
		return fmt.Errorf("create memory CSV: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("close memory CSV: %w", closeErr)
		}
	}()
	writer := csv.NewWriter(file)
	if err = writer.Write([]string{"time", "HeapAlloc", "HeapInuse", "HeapIdle", "HeapReleased", "Sys", "NumGC"}); err != nil {
		return fmt.Errorf("write memory header: %w", err)
	}
	writer.Flush()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for sample := 1; sample <= samples; sample++ {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)
			row := []string{now.UTC().Format(time.RFC3339Nano)}
			for _, value := range []uint64{stats.HeapAlloc, stats.HeapInuse, stats.HeapIdle, stats.HeapReleased, stats.Sys, uint64(stats.NumGC)} {
				row = append(row, strconv.FormatUint(value, 10))
			}
			if err = writer.Write(row); err != nil {
				return fmt.Errorf("write memory sample: %w", err)
			}
			writer.Flush()
			if err = writer.Error(); err != nil {
				return fmt.Errorf("flush memory CSV: %w", err)
			}
			if err = writeHeapProfile(filepath.Join(directory, fmt.Sprintf("heap-%03d.pprof", sample))); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeHeapProfile(path string) (err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // Operator-selected directory; generated filename and exclusive creation.
	if err != nil {
		return fmt.Errorf("create heap profile: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("close heap profile: %w", closeErr)
		}
	}()
	if err = pprof.WriteHeapProfile(file); err != nil {
		return fmt.Errorf("write heap profile: %w", err)
	}
	return nil
}
