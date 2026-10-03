package main

import (
	"errors"
	"testing"
	"time"
)

func TestBuildLabel(t *testing.T) {
	saved := commit
	t.Cleanup(func() { commit = saved })

	commit = ""
	if got := buildLabel(); got != version {
		t.Errorf("unstamped: buildLabel() = %q, want %q", got, version)
	}
	commit = "abc1234"
	if got, want := buildLabel(), version+" (abc1234)"; got != want {
		t.Errorf("stamped: buildLabel() = %q, want %q", got, want)
	}
}

// Wails on Linux loads the page while startup runs, so an IPC call can arrive first. It must
// wait and report how startup ended, not answer from half-built state.
func TestIPCWaitsForStartup(t *testing.T) {
	a := NewApp()
	got := make(chan AppInfo, 1)
	go func() { got <- a.AppInfo() }()
	select {
	case info := <-got:
		t.Fatalf("AppInfo answered before startup finished: %+v", info)
	case <-time.After(50 * time.Millisecond):
	}
	a.startupErr = errors.New("startup: lock held")
	close(a.started)
	if info := <-got; info.StartupError != "startup: lock held" {
		t.Errorf("AppInfo after startup = %+v, want the startup failure", info)
	}
}
