package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSeedReplaysSubsetAndOrder(t *testing.T) {
	wantAlways := []string{"1", "2", "3", "4", "8", "13", "20"}
	if !slices.Equal(alwaysJourneys(), wantAlways) {
		t.Fatalf("alwaysJourneys = %v, want %v", alwaysJourneys(), wantAlways)
	}
	seed, nonce, rng, err := makeSeed("202648-9f3a", 2026, 48)
	if err != nil || seed != 202648 || nonce != 0x9f3a {
		t.Fatalf("seed=%d nonce=%x err=%v", seed, nonce, err)
	}
	first := selectJourneys(rng)
	_, _, rng, _ = makeSeed("202648-9f3a", 2026, 48)
	second := selectJourneys(rng)
	if !slices.Equal(first, second) {
		t.Fatalf("subset or order changed: %v / %v", first, second)
	}
	if len(first) != 12 {
		t.Fatalf("selected %d journeys, want 12: %v", len(first), first)
	}
	if !slices.Equal(first[:len(wantAlways)], wantAlways) {
		t.Fatalf("prefix = %v, want %v", first[:len(wantAlways)], wantAlways)
	}
	_, _, otherRng, err := makeSeed("202648-0001", 2026, 48)
	if err != nil {
		t.Fatal(err)
	}
	if other := selectJourneys(otherRng); slices.Equal(first, other) {
		t.Fatalf("different seeds produced identical order: %v", other)
	}
}

func TestRefusalsHappenBeforeContact(t *testing.T) {
	dir := t.TempDir()
	allow := filepath.Join(dir, "allow")
	if err := os.WriteFile(allow, []byte("native.other\nmfl.14432\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"season target", []string{"-target", "season"}},
		{"not allowlisted", []string{"-target", "lab", "-server", "https://127.0.0.1:1", "-token-file", filepath.Join(dir, "missing"), "-league", "native.absent", "-season", "2026", "-week", "1", "-allow-file", allow}},
		{"MFL league", []string{"-target", "lab", "-server", "https://127.0.0.1:1", "-token-file", filepath.Join(dir, "missing"), "-league", "mfl.14432", "-season", "2026", "-week", "1", "-allow-file", allow}},
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
