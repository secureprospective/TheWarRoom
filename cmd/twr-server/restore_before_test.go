package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func TestRestoreBeforeDatePicksOlderSetWithoutOpeningNewer(t *testing.T) {
	key, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	dir, config := newHostFiles(t, bytes.Repeat([]byte{7}, 32))
	host := openLeagueHost(t, dir, config, time.Now)
	records, err := leagueservice.FixtureRecords()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := host.ledger.ExecuteCommissioner(t.Context(), records[0].Command, "fixture.creator", nil); err != nil {
		t.Fatal(err)
	}
	sets := &memorySets{objects: map[string][]byte{}}
	host.capture(t, sets, key.Recipient().String())
	host.capture(t, sets, key.Recipient().String())
	names, err := sets.Sets(t.Context())
	if err != nil || len(names) != 2 {
		t.Fatalf("sets: %v %v", names, err)
	}
	older, newer := "sets/20261020T120000.000000000Z.tar.gz.age", "sets/20261025T120000.000000000Z.tar.gz.age"
	sets.objects = map[string][]byte{older: sets.objects[names[1]], newer: sets.objects[names[0]]}
	target := filepath.Join(t.TempDir(), "server.db")
	if err := migrateServer(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	var report bytes.Buffer
	before := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	if err := restoreSpareHost(t.Context(), sets, key, restoreSettings(key), target, filepath.Join(t.TempDir(), "r"), before, &report); err != nil {
		t.Fatalf("%v: %s", err, report.String())
	}
	if !strings.Contains(report.String(), "Unlocked "+older) || sets.opened != 1 {
		t.Fatalf("restored the wrong set (opened %d): %s", sets.opened, report.String())
	}
	early := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	err = restoreSpareHost(t.Context(), sets, key, restoreSettings(key), target, filepath.Join(t.TempDir(), "e"), early, &report)
	if err == nil || !strings.Contains(err.Error(), "on or before that date") {
		t.Fatalf("no set before date: %v", err)
	}
}
