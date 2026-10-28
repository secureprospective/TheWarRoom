package sources

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

func TestStartupFactsStopsFailuresAndOrdersPublication(t *testing.T) {
	failure := errors.New("failed")
	for _, stop := range []string{"missing", "manifest error", "publish", "compile", ""} {
		calls := []string{}
		step := func(name string) func(context.Context) error {
			return func(context.Context) error {
				calls = append(calls, name)
				if stop == name {
					return failure
				}
				return nil
			}
		}
		StartupFacts(context.Background(), func(context.Context) (bool, error) {
			calls = append(calls, "exists")
			if stop == "manifest error" {
				return false, failure
			}
			return stop != "missing", nil
		},
			step("publish"), step("compile"), func(context.Context) { calls = append(calls, "warm") }, func(string) { calls = append(calls, "log") })
		want := []string{"exists", "publish", "compile", "warm"}
		switch stop {
		case "missing", "manifest error":
			want = want[:1]
		case "publish":
			want = []string{"exists", "publish", "log"}
		case "compile":
			want = []string{"exists", "publish", "compile", "log"}
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatal(stop, calls, want)
		}
	}
}

func TestDirectoryCachesOnlyNormalizedSuccess(t *testing.T) {
	failure := errors.New("fetch failed")
	publishes := 0
	publish := func(normalize.Lookup, time.Time) { publishes++ }
	if _, err := Directory(context.Background(), normalize.Lookup{}, false, func(context.Context) ([]players.RawPlayer, error) { return nil, failure }, publish); !errors.Is(err, failure) || publishes != 0 {
		t.Fatal(err, publishes)
	}
	if _, err := Directory(context.Background(), normalize.Lookup{}, false, func(context.Context) ([]players.RawPlayer, error) { return []players.RawPlayer{{ID: "invalid"}}, nil }, publish); err == nil || publishes != 0 {
		t.Fatal(err, publishes)
	}
	lookup, err := Directory(context.Background(), normalize.Lookup{}, false, func(context.Context) ([]players.RawPlayer, error) {
		return []players.RawPlayer{{ID: "0042", Name: "Player, One", Position: "QB", Team: "GB"}}, nil
	}, publish)
	if err != nil || publishes != 1 {
		t.Fatal(err, publishes)
	}
	held, err := Directory(context.Background(), lookup, true, func(context.Context) ([]players.RawPlayer, error) {
		t.Fatal("fetched cached directory")
		return nil, nil
	}, publish)
	if err != nil || !reflect.DeepEqual(held.IDs(), []string{"0042"}) || publishes != 1 {
		t.Fatal(err, held.IDs(), publishes)
	}
}

func TestRecordIngestKeepsFailuresVisible(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                 string
		open                 bool
		listingErr, writeErr error
		writes, logs         int
	}{
		{"before startup", false, nil, nil, 0, 0},
		{"listing failure", true, errors.New("offline"), nil, 0, 1},
		{"write failure", true, nil, errors.New("disk full"), 1, 1},
		{"success", true, nil, nil, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writes, logs := 0, 0
			WriteRecordAfterSignals(context.Background(), func(context.Context) (normalize.Lookup, error) {
				return normalize.Lookup{}, tc.listingErr
			}, RecordIngest{Open: func() bool { return tc.open },
				Write: func(context.Context, normalize.Lookup) (int, error) {
					writes++
					return 2, tc.writeErr
				}, Log: func(string) { logs++ }})
			if writes != tc.writes || logs != tc.logs {
				t.Fatalf("writes/logs = %d/%d, want %d/%d", writes, logs, tc.writes, tc.logs)
			}
		})
	}
}
