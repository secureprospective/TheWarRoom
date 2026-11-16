package main

import (
	"strings"
	"testing"
)

// The official statistics job re-pins a league only while it is still on the pin the job read: a pin
// someone chose meanwhile wins and nothing changes.
func TestAdvancePinYieldsToAPinChosenMeanwhile(t *testing.T) {
	ctx := t.Context()
	h := newTwoFileHost(t)
	first, _ := importFootball(t, h.files, "First")
	second, _ := importFootball(t, h.files, "Second")
	signIns := signInStore(h.files)
	if err := recordDirectory(ctx, hostedLeague(t, h.files), signIns); err != nil {
		t.Fatal(err)
	}
	pins := newLeaguePins(h.files.leagues)
	if err := pins.selectPins(ctx, "", nil); err != nil {
		t.Fatal(err)
	}
	if err := pins.setPin(ctx, signIns, "native.hosted", first); err != nil {
		t.Fatal(err)
	}
	if err := pins.advancePin(ctx, signIns, "native.hosted", "", second); err == nil ||
		!strings.Contains(err.Error(), "meanwhile") {
		t.Fatalf("advanced a pin that had moved: %v", err)
	}
	if pin := pins.pin(ctx, "native.hosted"); pin.Digest != first {
		t.Fatalf("the chosen pin was replaced: %s", pin.Digest)
	}
	if err := pins.advancePin(ctx, signIns, "native.hosted", first, second); err != nil {
		t.Fatal(err)
	}
	if pin := pins.pin(ctx, "native.hosted"); pin.Digest != second {
		t.Fatalf("not advanced: %s", pin.Digest)
	}
}
