//go:build unix

package leaguearchive

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestDiskRefusesFIFO(t *testing.T) {
	root := diskFixture(t)
	name := filepath.Join(root, "2026", "league.json")
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(name, 0600); err != nil {
		t.Fatal(err)
	}
	refuseDisk(t, NewDiskFS(root), "2026/league.json")
}

func TestDiskFIFOReplacementDoesNotBlock(t *testing.T) {
	root := diskFixture(t)
	g := NewDiskFS(root)
	rootInfo, fileInfo, err := g.pathInfo("2026/rules.json", []string{"2026", "rules.json"})
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(root, "2026", "rules.json")
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(name, 0600); err != nil {
		t.Fatal(err)
	}
	f, err := g.openChecked("2026/rules.json", rootInfo, fileInfo)
	if f != nil {
		_ = f.Close()
		t.Fatal("FIFO opened")
	}
	if err == nil {
		t.Fatal("FIFO replacement accepted")
	}
}
