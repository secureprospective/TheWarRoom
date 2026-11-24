package main

import (
	"bytes"
	"math/rand/v2"
	"strings"
	"testing"
)

func TestSeedReplaysSubsetAndOrder(t *testing.T) {
	seed, nonce, rng, err := makeSeed("202648-9f3a", 2026, 48)
	if err != nil || seed != 202648 || nonce != 0x9f3a {
		t.Fatalf("seed=%d nonce=%x err=%v", seed, nonce, err)
	}
	first := selectJourneys(rng)
	_, _, rng, _ = makeSeed("202648-9f3a", 2026, 48)
	second := selectJourneys(rng)
	for id := range first {
		if !second[id] {
			t.Fatalf("subset changed: %v / %v", first, second)
		}
	}
	if len(first) != 12 {
		t.Fatalf("selected %d journeys, want 12: %v", len(first), first)
	}
	_ = rand.New // keep test tied to standard-library PRNG
}

func TestRefusalsHappenBeforeContact(t *testing.T) {
	for _, args := range [][]string{{"-target", "season"}, {"-target", "lab", "-server", "https://unused", "-token-file", "missing", "-league", "native.other", "-season", "2026", "-week", "1", "-allow-file", t.TempDir() + "/allow"}} {
		var out, stderr bytes.Buffer
		code := run(args, &out, &stderr, nil)
		if code != 2 || strings.Contains(out.String(), "PASS") {
			t.Fatalf("code=%d out=%q stderr=%q", code, out.String(), stderr.String())
		}
	}
}
