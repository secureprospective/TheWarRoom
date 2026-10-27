package composition

import (
	"errors"
	"testing"
	"time"

	sourcesapp "github.com/secureprospective/TheWarRoom/internal/app/sources"
)

// Wails on Linux loads the page while startup runs, so an IPC call can arrive first. It must
// wait and report how startup ended, not answer from half-built state.
func TestIPCWaitsForStartup(t *testing.T) {
	t.Parallel()
	a := NewApp()
	got := make(chan sourcesapp.DataActResult, 1)
	go func() { got <- a.sourcesAPI().TargetDataRefresh("league") }()
	select {
	case r := <-got:
		t.Fatalf("TargetDataRefresh answered before startup finished: %+v", r)
	case <-time.After(50 * time.Millisecond):
	}
	a.core.StartupErr = errors.New("startup: lock held")
	close(a.core.Started)
	if r := <-got; r.Error != "startup: lock held" {
		t.Errorf("TargetDataRefresh after startup = %+v, want the startup failure", r)
	}
}
