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
