//go:build leaguebench

package history

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Hash all fixture/generator sources, including the volume table, in stable filename order.
func bigLeagueKey(t testing.TB) string {
	t.Helper()
	files, err := filepath.Glob("bigleague*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(h, "%s\x00%d\x00%s", file, len(body), body); err != nil {
			t.Fatal(err)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func copyLeagueFile(t testing.TB, from, to string) {
	t.Helper()
	src, err := os.Open(from)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := src.Close(); err != nil {
			t.Error(err)
		}
	}()
	dst, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

func cachedBigLeague(t testing.TB, tier string) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, "logs", "TheWarRoom", "bigleague-fixtures")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, tier+"-"+bigLeagueKey(t)+".db")
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("CACHE tier=%s path=%s\n", tier, path)
		return path
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	// Exclusive lock refuses concurrent builders; incomplete files never become cache hits.
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("cache build lock (remove only after checking its builder is dead): %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(path + ".lock"); err != nil {
			t.Error(err)
		}
	}()
	tmp := filepath.Join(t.TempDir(), "build.db")
	s, pools := openBigLeague(t, tmp)
	volumes := leagueVolumes(tier == "double")
	start := time.Now()
	fmt.Printf("BUILD tier=%s path=%s\n", tier, path)
	generateBigLeague(t, s, volumes)
	setupBigLeagueScoring(t, s, volumes[len(volumes)-1].year, tier == "double")
	assets, err := s.NativeAssets(context.Background(), bigLeagueID)
	if err != nil {
		t.Fatal(err)
	}
	expected := int64(37606)
	if tier == "double" {
		expected = 75341
	}
	if assets.Revision != expected {
		t.Fatalf("fixture revision=%d, want %d", assets.Revision, expected)
	}
	// Closing both pools checkpoints WAL before copying the self-contained database.
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	copyLeagueFile(t, tmp, path+".partial")
	if err := os.Rename(path+".partial", path); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("BUILD complete tier=%s seconds=%.3f path=%s\n", tier, time.Since(start).Seconds(), path)
	return path
}

func TestBigLeagueBuild(t *testing.T) {
	for _, tier := range []string{"legacy", "double"} {
		t.Run(tier, func(t *testing.T) { cachedBigLeague(t, tier) })
	}
}

func TestBigLeagueCopyIsolation(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.db")
	destination := filepath.Join(t.TempDir(), "copy.db")
	if err := os.WriteFile(source, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	copyLeagueFile(t, source, destination)
	if err := os.WriteFile(destination, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "original" {
		t.Fatal("copy modified cache")
	}
	if len(bigLeagueKey(t)) != sha256.Size*2 {
		t.Fatal("invalid generator key")
	}
}
