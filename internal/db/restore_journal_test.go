package db

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRestoreRefusesNewerJournalWithoutTouchingFiles(t *testing.T) {
	for _, extra := range []string{"", `,"future_action":"remove"`} {
		t.Run(extra, func(t *testing.T) {
			b, pools := backupFixture(t)
			for _, p := range pools {
				if err := p.Close(); err != nil {
					t.Fatal(err)
				}
			}
			body := `{"version":2,"set":"saved","stage":".restore-new","step":"complete","fresh":true` + extra + `}`
			if err := os.Mkdir(filepath.Join(b.dir, ".restore-new"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(b.dir, ".restore-new", "held"), []byte("keep staged files"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(b.dir, "restore.journal"), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			before := restoreFiles(t, b.dir)
			for _, operation := range []func() error{
				func() error { return b.Recover(t.Context()) },
				func() error {
					_, err := b.Restore(t.Context(), filepath.Join(b.dir, "backups", "saved"), backupAt())
					return err
				},
			} {
				err := operation()
				if !errors.Is(err, ErrRestoreJournalNewer) || !strings.Contains(err.Error(), "update the app") {
					t.Fatalf("refusal: %v", err)
				}
				if after := restoreFiles(t, b.dir); !reflect.DeepEqual(before, after) {
					t.Fatalf("files changed: before=%v after=%v", before, after)
				}
			}
		})
	}
}

func restoreFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	tree := root.FS()
	files := map[string]string{}
	err = fs.WalkDir(tree, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files[path] = info.Mode().String()
		if !entry.IsDir() {
			body, err := fs.ReadFile(tree, path)
			if err != nil {
				return err
			}
			files[path] += string(body)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestRestoreJournalLegacyAndUnknownFields(t *testing.T) {
	for _, version := range []int{0, 1} {
		j := RawRestoreJournal{Version: version, Set: "saved", Stage: ".restore-test", Step: "prepared", Fresh: true}
		path := filepath.Join(t.TempDir(), "restore.journal")
		if err := durableJSON(path, j); err != nil {
			t.Fatal(err)
		}
		got, err := readRestoreJournal(path)
		if err != nil || got != j {
			t.Fatalf("journal=%+v err=%v", got, err)
		}
	}
	path := filepath.Join(t.TempDir(), "restore.journal")
	if err := os.WriteFile(path, []byte(`{"set":"saved","stage":".restore-test","step":"complete","fresh":true,"future":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRestoreJournal(path); err == nil {
		t.Fatal("unknown legacy field accepted")
	}
}
