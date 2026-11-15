package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/serverstatus"
)

func TestRequestsSocketAndCommand(t *testing.T) {
	tally := serverhttp.NewRequestTally()
	tally.Unchanged("native.assets.read", "read")
	socket := filepath.Join(t.TempDir(), "status.sock")
	listener, err := serverstatus.Listen(socket)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- serverstatus.Serve(ctx, listener, serverstatus.Source{}, requestRoutes(tally)...) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	var out bytes.Buffer
	if err := runRequests(ctx, []string{"-status-socket", socket}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"COMMAND MODE CALLS", "native.assets.read read 0 0 0 0 1", "Memory:"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in %s", want, out.String())
		}
	}
	t.Logf("table:\n%s", out.String())
	out.Reset()
	if err := runRequests(ctx, []string{"-status-socket", socket, "-reset"}, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "native.assets.read") {
		t.Fatal(out.String())
	}
}
