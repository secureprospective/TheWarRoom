package leaguearchive

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/archive"
)

type diskSink struct{ calls int }

func (s *diskSink) Record(context.Context, archive.Fetch) error { s.calls++; return nil }

const diskLeague = `{"league":{"id":"1","baseURL":"https://example.test"}}`

func diskFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "2026"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"league.json": diskLeague, "rules.json": "{}"} {
		if err := os.WriteFile(filepath.Join(root, "2026", name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func refuseDisk(t *testing.T, g fs.FS, want string) {
	t.Helper()
	var sink diskSink
	_, err := Import(context.Background(), g, "test", nil, &sink)
	if err == nil || !strings.Contains(diagnosticChain(err), want) {
		t.Fatalf("error = %v; want %q", err, want)
	}
	if strings.Contains(err.Error(), "archive:") || !strings.Contains(err.Error(), "Choose") {
		t.Fatalf("owner message = %q", err.Error())
	}
	if sink.calls != 0 {
		t.Fatalf("Record called %d times", sink.calls)
	}
}

func TestDiskRefusesSymlinks(t *testing.T) {
	for _, name := range []string{"rules.json", "league.json"} {
		t.Run(name, func(t *testing.T) {
			root := diskFixture(t)
			outside := filepath.Join(t.TempDir(), "outside")
			body := "{}"
			if name == "league.json" {
				body = diskLeague
			}
			if err := os.WriteFile(outside, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(root, "2026", name)
			if err := os.Remove(target); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, target); err != nil {
				t.Skip(err)
			}
			g := NewDiskFS(root)
			refuseDisk(t, g, "2026/"+name)
			want := int64(512<<20) - 2*int64(len(diskLeague))
			if name == "league.json" {
				want = 512 << 20
			}
			if g.remaining != want {
				t.Fatalf("outside bytes consumed: remaining %d, want %d", g.remaining, want)
			}
		})
	}
	t.Run("root", func(t *testing.T) {
		root := diskFixture(t)
		link := filepath.Join(t.TempDir(), "root")
		if err := os.Symlink(root, link); err != nil {
			t.Skip(err)
		}
		refuseDisk(t, NewDiskFS(link), link)
	})
}

func TestDiskBudgets(t *testing.T) {
	// Discovery and walking both read league.json and enumerate the root.
	bytesRead := int64(2*len(diskLeague) + 2)
	for _, budget := range []string{"file bytes", "total bytes", "files", "entries"} {
		for _, excess := range []bool{false, true} {
			t.Run(budget+"/"+map[bool]string{false: "exact", true: "exceeded"}[excess], func(t *testing.T) {
				g := NewDiskFS(diskFixture(t))
				delta := 0
				if excess {
					delta = 1
				}
				switch budget {
				case "file bytes":
					g.fileLimit = int64(len(diskLeague) - delta)
				case "total bytes":
					g.remaining = bytesRead - int64(delta)
				case "files":
					g.files = 3 - delta
				case "entries":
					g.entries = 4 - delta
				}
				if excess {
					refuseDisk(t, g, "exceeded")
					return
				}
				var sink diskSink
				rep, err := Import(context.Background(), g, "test", nil, &sink)
				if err != nil || rep.Recorded != 2 || sink.calls != 2 {
					t.Fatalf("report %+v, calls %d, error %v", rep, sink.calls, err)
				}
			})
		}
	}
}

func TestDiskDepth(t *testing.T) {
	root := diskFixture(t)
	if err := os.MkdirAll(filepath.Join(root, "2026", "weekly", "deep", "deeper"), 0700); err != nil {
		t.Fatal(err)
	}
	refuseDisk(t, NewDiskFS(root), "depth exceeded")
}

func diagnosticChain(err error) string {
	var text strings.Builder
	for err != nil {
		text.WriteString(err.Error())
		err = errors.Unwrap(err)
	}
	return text.String()
}
