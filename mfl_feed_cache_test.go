package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// archiveOf serves bodies newest first, as history.ArchivedBodies does.
type archived struct {
	url, body string
	at        time.Time
}

func archiveOf(bodies ...archived) func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
	return func(_ context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		for _, b := range bodies {
			if strings.Contains(b.url, part) && accept(b.url, []byte(b.body), b.at) {
				return true, nil
			}
		}
		return false, nil
	}
}

func TestLiveOrArchive(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	down := errors.New("mfl down")
	parse := func(b []byte) ([]string, error) {
		if string(b) == "bad" {
			return nil, errors.New("glitch")
		}
		return strings.Split(string(b), ","), nil
	}
	const here = "https://www47.myfantasyleague.com/2026/export?JSON=1&L=14432&TYPE=leagueStandings"
	feed := func(fetchErr error, bodies ...archived) archivedFeed[string] {
		return archivedFeed[string]{
			export: "leagueStandings",
			fetch:  func(context.Context) ([]string, error) { return []string{"live"}, fetchErr },
			parse:  parse,
			bodies: archiveOf(bodies...),
		}
	}
	old := time.Unix(1_700_000_000, 0)
	// lines captures the fallbackLog seam; a live fetch writes nothing, every degraded one
	// writes exactly one line naming the feed and the error.
	var lines []string
	logLine := func(line string) { lines = append(lines, line) }

	rows, fresh, err := liveOrArchive(ctx, ctx, 2026, feed(nil), logLine)
	if err != nil || rows[0] != "live" || fresh.State != FreshLive {
		t.Fatalf("live: %v %+v %v", rows, fresh, err)
	}
	if len(lines) != 0 {
		t.Fatalf("live fetch logged %v", lines)
	}
	rows, fresh, err = liveOrArchive(ctx, ctx, 2026, feed(down,
		archived{here, "bad", old.Add(time.Hour)},
		archived{strings.Replace(here, "/2026/", "/2025/", 1), "last,season", old.Add(time.Minute)},
		archived{strings.Replace(here, "14432", "99999", 1), "other,league", old.Add(time.Minute)},
		archived{here, "a,b", old}), logLine)
	if err != nil || strings.Join(rows, ",") != "a,b" || fresh.State != FreshStale || !strings.Contains(fresh.Note, "mfl down") {
		t.Fatalf("fallback skipping a bad body, another season and another league: %v %+v %v", rows, fresh, err)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "leagueStandings") ||
		!strings.Contains(lines[0], "mfl down") || !strings.Contains(lines[0], "serving the archived copy") {
		t.Fatalf("fallback log = %v, want one line naming the feed and the error", lines)
	}
	lines = nil
	if _, _, err := liveOrArchive(ctx, ctx, 2026, feed(down, archived{here, "bad", old}), logLine); !errors.Is(err, down) ||
		!strings.Contains(err.Error(), "no archived copy") {
		t.Fatalf("nothing usable archived: %v", err)
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "leagueStandings") ||
		!strings.Contains(lines[0], "mfl down") || !strings.Contains(lines[0], "no archived copy") {
		t.Fatalf("empty-fallback log = %v, want one line naming the feed and the error", lines)
	}
}

// TestLiveOrArchiveLeaguelessFallback proves the league-less archived matcher the injuries
// feed needs: a copy fetched without L (the api-host request) matches, and another league's
// league-scoped copy still does not.
func TestLiveOrArchiveLeaguelessFallback(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	down := errors.New("mfl down")
	const bare = "https://api.myfantasyleague.com/2026/export?TYPE=injuries&JSON=1"
	const other = "https://api.myfantasyleague.com/2026/export?TYPE=injuries&L=99999&JSON=1"
	at := time.Unix(1_700_000_000, 0)
	feed := archivedFeed[string]{
		export:     "injuries",
		leagueless: true,
		fetch:      func(context.Context) ([]string, error) { return nil, down },
		parse:      func(b []byte) ([]string, error) { return strings.Split(string(b), ","), nil },
		bodies: archiveOf(
			archived{other, "other,league", at.Add(time.Minute)},
			archived{bare, "api,copy", at}),
	}
	rows, fresh, err := liveOrArchive(ctx, ctx, 2026, feed, func(string) {})
	if err != nil || strings.Join(rows, ",") != "api,copy" || fresh.State != FreshStale {
		t.Fatalf("leagueless fallback: %v %+v %v", rows, fresh, err)
	}
}
