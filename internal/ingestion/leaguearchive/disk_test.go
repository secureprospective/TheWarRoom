package leaguearchive

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/archive"
)

type diskSink struct {
	calls   int
	fetches []archive.Fetch
}

func (s *diskSink) Record(_ context.Context, fetch archive.Fetch) error {
	s.calls++
	s.fetches = append(s.fetches, fetch)
	return nil
}

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

func TestDiskRefusesReplacementAfterPathCheck(t *testing.T) {
	for _, replacement := range []string{"file", "season", "root", "internal link"} {
		t.Run(replacement, func(t *testing.T) {
			root := diskFixture(t)
			g := NewDiskFS(root)
			rootInfo, fileInfo, err := g.pathInfo("2026/rules.json", []string{"2026", "rules.json"})
			if err != nil {
				t.Fatal(err)
			}
			outside := diskFixture(t)
			if err := os.WriteFile(filepath.Join(outside, "2026", "rules.json"), []byte("sentinel-outside-secret"), 0600); err != nil {
				t.Fatal(err)
			}
			target, link := filepath.Join(root, "2026", "rules.json"), filepath.Join(outside, "2026", "rules.json")
			switch replacement {
			case "season":
				target, link = filepath.Join(root, "2026"), filepath.Join(outside, "2026")
			case "root":
				target, link = root, outside
			case "internal link":
				link = "private.json"
				if err := os.WriteFile(filepath.Join(root, "2026", link), []byte("sentinel-outside-secret"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Rename(target, target+".held"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(link, target); err != nil {
				t.Skip(err)
			}
			f, err := g.openChecked("2026/rules.json", rootInfo, fileInfo)
			if f != nil {
				_ = f.Close()
				t.Fatal("replacement opened")
			}
			if err == nil || strings.Contains(err.Error(), "sentinel-outside-secret") {
				t.Fatalf("unsafe refusal: %v", err)
			}
		})
	}
}

type swappingDisk struct {
	*DiskFS
	outside string
}

func (g swappingDisk) Open(name string) (fs.File, error) {
	if name != "2026/rules.json" {
		return g.DiskFS.Open(name)
	}
	rootInfo, fileInfo, err := g.pathInfo(name, strings.Split(name, "/"))
	if err != nil {
		return nil, err
	}
	if err := os.Rename(filepath.Join(g.root, "2026"), filepath.Join(g.root, "held")); err != nil {
		return nil, err
	}
	if err := os.Symlink(g.outside, filepath.Join(g.root, "2026")); err != nil {
		return nil, err
	}
	return g.openChecked(name, rootInfo, fileInfo)
}

func (g swappingDisk) ReadFile(name string) ([]byte, error) {
	// Hide DiskFS.ReadFile so fs.ReadFile exercises the checkpoint swap in Open.
	return fs.ReadFile(struct{ fs.FS }{g}, name)
}

func TestDiskSwapPreservesPriorImport(t *testing.T) {
	root := diskFixture(t)
	var sink diskSink
	if _, err := Import(t.Context(), NewDiskFS(root), "test", nil, &sink); err != nil {
		t.Fatal(err)
	}
	prior := append([]archive.Fetch(nil), sink.fetches...)
	outside := filepath.Join(diskFixture(t), "2026")
	if err := os.WriteFile(filepath.Join(outside, "rules.json"), []byte(`{"secret":"sentinel-outside-secret"}`), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Import(t.Context(), swappingDisk{NewDiskFS(root), outside}, "test", nil, &sink)
	if err == nil || !reflect.DeepEqual(sink.fetches, prior) || strings.Contains(err.Error(), "sentinel-outside-secret") {
		t.Fatalf("failed import changed prior records: before=%d after=%d err=%v", len(prior), len(sink.fetches), err)
	}
}
