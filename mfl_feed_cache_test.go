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

	rows, fresh, err := liveOrArchive(ctx, ctx, 2026, feed(nil))
	if err != nil || rows[0] != "live" || fresh.State != FreshLive {
		t.Fatalf("live: %v %+v %v", rows, fresh, err)
	}
	rows, fresh, err = liveOrArchive(ctx, ctx, 2026, feed(down,
		archived{here, "bad", old.Add(time.Hour)},
		archived{strings.Replace(here, "/2026/", "/2025/", 1), "last,season", old.Add(time.Minute)},
		archived{strings.Replace(here, "14432", "99999", 1), "other,league", old.Add(time.Minute)},
		archived{here, "a,b", old}))
	if err != nil || strings.Join(rows, ",") != "a,b" || fresh.State != FreshStale || !strings.Contains(fresh.Note, "mfl down") {
		t.Fatalf("fallback skipping a bad body, another season and another league: %v %+v %v", rows, fresh, err)
	}
	if _, _, err := liveOrArchive(ctx, ctx, 2026, feed(down, archived{here, "bad", old})); !errors.Is(err, down) ||
		!strings.Contains(err.Error(), "no archived copy") {
		t.Fatalf("nothing usable archived: %v", err)
	}
}
