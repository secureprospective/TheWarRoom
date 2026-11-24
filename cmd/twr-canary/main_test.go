package main

import (
	"bytes"
	"os"
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
	if !sameSelection(first, second) {
		t.Fatalf("subset or order changed: %v / %v", first, second)
	}
	if len(first) != 12 {
		t.Fatalf("selected %d journeys, want 12: %v", len(first), first)
	}
}

func sameSelection(a, b map[string]bool) bool {
	for id, selected := range a {
		if b[id] != selected {
			return false
		}
	}
	return len(a) == len(b)
}

func TestRefusalsHappenBeforeContact(t *testing.T) {
	dir := t.TempDir()
	allow := dir + "/allow"
	if err := os.WriteFile(allow, []byte("native.other\\nmfl.14432\\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"season target", []string{"-target", "season"}},
		{"not allowlisted", []string{"-target", "lab", "-server", "https://127.0.0.1:1", "-token-file", dir + "/missing", "-league", "native.absent", "-season", "2026", "-week", "1", "-allow-file", allow}},
		{"MFL league", []string{"-target", "lab", "-server", "https://127.0.0.1:1", "-token-file", dir + "/missing", "-league", "mfl.14432", "-season", "2026", "-week", "1", "-allow-file", allow}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			code := run(tc.args, &out, &stderr, nil)
			if code != 2 || strings.Contains(out.String(), "PASS") {
				t.Fatalf("code=%d out=%q stderr=%q", code, out.String(), stderr.String())
			}
		})
	}
}
