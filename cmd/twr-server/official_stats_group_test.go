package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// officialGroupFixture holds the original league and two more, each pinned to football from; built is a second
// football version the fake build returns, other a third one for a league on its own version.
func officialGroupFixture(t *testing.T) (*leaguePins, *twoFileHost, string, string, string) {
	t.Helper()
	host, _, _ := scheduleRefreshFixture(t)
	ctx := t.Context()
	if err := host.files.Close(); err != nil {
		t.Fatal(err)
	}
	for _, league := range []string{"native.second", "native.third"} {
		args := []string{"-db", filepath.Join(host.dir, "server.db"), "-league", league, "-name", league, "-preset", "default_32", "-season", "2026", "-commissioner"}
		if err := runCreateLeague(ctx, args, strings.NewReader(`{"username":"commissioner","password":"private-password"}`), discardOutput{}); err != nil {
			t.Fatalf("create %s: %v", league, err)
		}
	}
	host.files = openTestServer(t, host.dir)
	fromBody, from := footballEvidenceWithWeeks(t, "Official from", futureScheduleWeek(20))
	builtBody, built := footballEvidenceWithWeeks(t, "Official built", futureScheduleWeek(21))
	otherBody, other := footballEvidenceWithWeeks(t, "Commissioner choice", futureScheduleWeek(22))
	for _, league := range []string{"native.hosted", "native.second", "native.third"} {
		store, err := leagueStore(ctx, host.files, league)
		if err != nil {
			t.Fatal(err)
		}
		for body, digest := range map[string]string{string(fromBody): from, string(builtBody): built, string(otherBody): other} {
			if err := store.ImportFootball(ctx, []byte(body), digest); err != nil {
				t.Fatal(err)
			}
		}
		if err := signInStore(host.files).PinFootball(ctx, league, from); err != nil {
			t.Fatal(err)
		}
	}
	pins := newLeaguePins(host.files.leagues)
	directory, err := signInStore(host.files).Directory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := pins.selectPins(ctx, "", directory); err != nil {
		t.Fatal(err)
	}
	return pins, host, from, built, other
}

func emptyGzip(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := gzip.NewWriter(&b).Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func officialGroupKeeper(t *testing.T, pins *leaguePins, host *twoFileHost, build func(context.Context, serverhttp.LeaguePin, history.OfficialSeason) (history.OfficialPin, error)) *officialKeeper {
	empty := emptyGzip(t)
	return &officialKeeper{pins: pins, signIns: signInStore(host.files), directory: pins.files.family.Over(pins.files.reference),
		files: func(context.Context, int) (seasonFiles, error) { return seasonFiles{weekly: empty, plays: empty}, nil },
		now:   func() time.Time { return time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC) }, logf: t.Logf, build: build}
}

// Three leagues on one football version: official statistics are built once, not once per league, and a league
// a commissioner re-pins while the pass runs keeps the commissioner's choice.
func TestOfficialStatsBuildOncePerFootballVersion(t *testing.T) {
	pins, host, from, built, other := officialGroupFixture(t)
	ctx := t.Context()
	builds := 0
	k := officialGroupKeeper(t, pins, host, func(_ context.Context, pin serverhttp.LeaguePin, _ history.OfficialSeason) (history.OfficialPin, error) {
		builds++
		if pin.Digest != from {
			t.Fatalf("built from %s, want %s", pin.Digest, from)
		}
		// A commissioner pins native.third to a version of their own while the pass runs.
		if err := pins.setPin(ctx, signInStore(host.files), "native.third", other); err != nil {
			t.Fatal(err)
		}
		return history.OfficialPin{Digest: built}, nil
	})
	k.pass(ctx)
	if builds != 1 {
		t.Fatalf("official statistics built %d times for 3 leagues on one football version, want 1", builds)
	}
	for league, want := range map[string]string{"native.hosted": built, "native.second": built, "native.third": other} {
		if got := pins.pin(ctx, league).Digest; got != want {
			t.Fatalf("%s pinned %s, want %s", league, got, want)
		}
	}
}

// Leagues on two football versions get one build each, and a league is only ever moved from its own version.
func TestOfficialStatsBuildOncePerVersionNotAcross(t *testing.T) {
	pins, host, from, built, _ := officialGroupFixture(t)
	ctx := t.Context()
	if err := pins.setPin(ctx, signInStore(host.files), "native.third", built); err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	k := officialGroupKeeper(t, pins, host, func(_ context.Context, pin serverhttp.LeaguePin, _ history.OfficialSeason) (history.OfficialPin, error) {
		seen[pin.Digest]++
		return history.OfficialPin{Digest: map[string]string{from: built, built: ""}[pin.Digest]}, nil
	})
	k.pass(ctx)
	if len(seen) != 2 || seen[from] != 1 || seen[built] != 1 {
		t.Fatalf("builds per version = %v, want one each for %s and %s", seen, from, built)
	}
	for _, league := range []string{"native.hosted", "native.second", "native.third"} {
		if got := pins.pin(ctx, league).Digest; got != built {
			t.Fatalf("%s pinned %s, want %s", league, got, built)
		}
	}
}
