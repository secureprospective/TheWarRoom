package main

import (
	"errors"
	"testing"
	"time"
)

func TestBuildLabel(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	a := NewApp()
	got := make(chan DataActResult, 1)
	go func() { got <- a.TargetDataRefresh("league") }()
	select {
	case r := <-got:
		t.Fatalf("TargetDataRefresh answered before startup finished: %+v", r)
	case <-time.After(50 * time.Millisecond):
	}
	a.startupErr = errors.New("startup: lock held")
	close(a.started)
	if r := <-got; r.Error != "startup: lock held" {
		t.Errorf("TargetDataRefresh after startup = %+v, want the startup failure", r)
	}
}
