package serverclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverversion"
)

func TestUpdateScreenBlocksNativeActionsAndRetriesSafely(t *testing.T) {
	d, err := NewDesktop("https://league.example.org", "native.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	d.PracticeUpdate()
	for _, action := range []string{"status", "enter"} {
		view, err := d.Act(context.Background(), RawDesktop{Action: action})
		if err != nil || view.Update != "app" || view.Name != "" {
			t.Fatalf("screen not blocking: %+v %v", view, err)
		}
	}
	for _, attempt := range []int{1, 2} {
		d.update = func() error { return fmt.Errorf("corrupt update") }
		view, err := d.Act(context.Background(), RawDesktop{Action: "update"})
		if err == nil || view.Update != "app" {
			t.Fatalf("failed attempt %d lost update screen", attempt)
		}
	}
	d.required = serverversion.Server
	d.update = func() error { t.Fatal("server update tried to install desktop"); return nil }
	if _, err := d.Act(context.Background(), RawDesktop{Action: "update"}); err == nil {
		t.Fatal("server update installed desktop")
	}
	view, err := d.Act(context.Background(), RawDesktop{Action: "retry"})
	if err != nil || view.Update != "" {
		t.Fatal("server retry did not return to sign in")
	}
}

func TestLaunchKeepsOldWindowOnFailureAndPreservesLeagueOnSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX process fixture")
	}
	d, err := NewDesktop("https://league.example.org", "native.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	closed := false
	quit := func() { closed = true }
	if err := d.launchUpdate(filepath.Join(t.TempDir(), "missing"), quit); err == nil || closed {
		t.Fatal("failed launch closed the old window")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "desktop")
	// Fixed fixture uses its executable folder, never interpolates test paths into a shell.
	script := []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$(dirname \"$0\")/arguments\"\n")
	if err := os.WriteFile(path, script, 0700); err != nil { //nolint:gosec // Executable test fixture in a private temporary folder.
		t.Fatal(err)
	}
	if err := d.launchUpdate(path, quit); err != nil || !closed {
		t.Fatalf("launch failed: %v", err)
	}
	for range 100 {
		data, err := os.ReadFile(filepath.Join(dir, "arguments"))
		if err == nil && string(data) == "-native\nhttps://league.example.org\nnative.test\n" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("updated desktop did not receive the original origin and league")
}
