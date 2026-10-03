package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var errNoneCached = errors.New("nothing cached")

// fakeFeed is an in-memory feedCache: fetchErr fails the fetch, putErr the cache write.
type fakeFeed struct {
	live     []string
	fetchErr error
	putErr   error
	payload  string
	at       time.Time
}

func (f *fakeFeed) cache() feedCache[string] {
	return feedCache[string]{
		name: "standings",
		fetch: func(context.Context) ([]string, error) {
			return f.live, f.fetchErr
		},
		put: func(_ context.Context, p string, at time.Time) error {
			if f.putErr != nil {
				return f.putErr
			}
			f.payload, f.at = p, at
			return nil
		},
		cached: func(context.Context) (string, time.Time, error) {
			if f.payload == "" {
				return "", time.Time{}, errNoneCached
			}
			return f.payload, f.at, nil
		},
		noCached: errNoneCached,
	}
}

func TestLiveOrCache(t *testing.T) {
	ctx := context.Background()
	down := errors.New("mfl down")

	t.Run("live fetch is served and cached", func(t *testing.T) {
		f := &fakeFeed{live: []string{"a", "b"}}
		rows, fresh, err := liveOrCache(ctx, ctx, f.cache())
		if err != nil || len(rows) != 2 || fresh.State != FreshLive {
			t.Fatalf("got %v %+v %v, want 2 live rows", rows, fresh, err)
		}
		if f.payload != `["a","b"]` {
			t.Fatalf("cached %q, want the live rows", f.payload)
		}
	})

	t.Run("failed fetch serves the cached copy as stale", func(t *testing.T) {
		f := &fakeFeed{payload: `["old"]`, at: time.Unix(1_700_000_000, 0), fetchErr: down}
		rows, fresh, err := liveOrCache(ctx, ctx, f.cache())
		if err != nil || len(rows) != 1 || rows[0] != "old" || fresh.State != FreshStale {
			t.Fatalf("got %v %+v %v, want the stale cached row", rows, fresh, err)
		}
		if !strings.Contains(fresh.Note, "mfl down") {
			t.Fatalf("stale note %q does not name the fetch failure", fresh.Note)
		}
	})

	t.Run("failed fetch with nothing cached is an error naming the fetch", func(t *testing.T) {
		f := &fakeFeed{fetchErr: down}
		if _, _, err := liveOrCache(ctx, ctx, f.cache()); !errors.Is(err, down) ||
			!strings.Contains(err.Error(), "no cached fallback") {
			t.Fatalf("err = %v, want the fetch error with no fallback", err)
		}
	})

	t.Run("an empty cached copy is an error, not an empty board", func(t *testing.T) {
		f := &fakeFeed{payload: `[]`, fetchErr: down}
		if _, _, err := liveOrCache(ctx, ctx, f.cache()); err == nil {
			t.Fatal("empty cached copy was served")
		}
	})

	t.Run("a failed cache write still serves live data, with a note", func(t *testing.T) {
		f := &fakeFeed{live: []string{"a"}, putErr: errors.New("disk full")}
		rows, fresh, err := liveOrCache(ctx, ctx, f.cache())
		if err != nil || len(rows) != 1 || !strings.Contains(fresh.Note, "disk full") {
			t.Fatalf("got %v %+v %v, want live rows and a note", rows, fresh, err)
		}
	})
}
