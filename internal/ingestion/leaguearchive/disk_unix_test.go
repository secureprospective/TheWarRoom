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
