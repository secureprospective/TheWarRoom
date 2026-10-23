package sources_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
)

func TestGuardedRealArchive(t *testing.T) {
	root := os.Getenv("WARROOM_TEST_ARCHIVE")
	if root == "" {
		t.Skip("set WARROOM_TEST_ARCHIVE for read-only real archive test")
	}
	a := testHost(t)
	service := sourcesapp.ServiceFor(a)
	ctx := context.Background()
	first, err := service.ImportArchiveFrom(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if first.Files != 1720 || first.Recorded != 1720 {
		t.Fatalf("first import: %+v", first)
	}
	again, err := service.ImportArchiveFrom(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if again.Files != 1720 || again.Held != 1720 || again.Recorded != 0 {
		t.Fatalf("repeat: %+v", again)
	}
}

func TestCheckArchiveRefusesLinkedRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "2026"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "2026", "league.json"), []byte(`{"league":{"id":"1","baseURL":"https://example.test"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := sourcesapp.CheckArchiveDir(root); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Skip(err)
	}
	if err := sourcesapp.CheckArchiveDir(link); err == nil {
		t.Fatal("linked root accepted")
	}
}
