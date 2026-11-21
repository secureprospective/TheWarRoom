package main

import (
	"testing"
	"time"
)

// The results warm-up reads every week of each pinned league without failing the league, and its
// loop stops promptly when the server stops.
func TestResultsWarmReadsPinnedLeaguesAndStops(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	digest, _ := importFootball(t, h.files, "Warm")
	signIns := signInStore(h.files)
	if err := recordDirectory(ctx, hostedLeague(t, h.files), signIns); err != nil {
		t.Fatal(err)
	}
	pins := newLeaguePins(h.files.leagues)
	if err := pins.selectPins(ctx, "", nil); err != nil {
		t.Fatal(err)
	}
	if err := pins.setPin(ctx, signIns, "native.hosted", digest); err != nil {
		t.Fatal(err)
	}
	warmResults(ctx, pins)
	pin := pins.pin(ctx, "native.hosted")
	if _, err := pin.Store.NativeResults(ctx, "native.hosted", pin.Football.Season(), 1); err != nil {
		t.Fatalf("results after warm-up: %v", err)
	}
	stop := startResultsWarm(ctx, pins)
	stopped := make(chan struct{})
	go func() { stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("warm-up loop did not stop")
	}
}
