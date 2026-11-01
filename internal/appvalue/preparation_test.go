package appvalue

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
)

func checkPreparation(t *testing.T, src Source, league string, want Reading) {
	t.Helper()
	src.Revision = func(context.Context) (int64, error) { return 1, nil }
	src.NextFeatureAt = func(context.Context, time.Time) (time.Time, error) { return time.Time{}, nil }
	prep := &Preparation{}
	read := func() Reading {
		t.Helper()
		out, err := prep.Read(t.Context(), src, league, "", season)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	if !read().Preparing {
		t.Fatal("cold screen ran valuation")
	}
	changes := 0
	prepare := func() {
		t.Helper()
		prep.sweep(t.Context(), src, func() { changes++ }, func(err error) { t.Error(err) })
	}
	prepare()
	if got := read(); !reflect.DeepEqual(got, want) {
		t.Fatal("prepared values differ from direct values")
	}
	prepare()
	if changes != 1 {
		t.Fatal("warm sweep repeated valuation")
	}
	original := src.Records
	for _, kind := range []string{"roster", "contract", "setting"} {
		t.Run(kind, func(t *testing.T) {
			src.Records = func(ctx context.Context, l string) ([]commissioner.Record, error) {
				records, err := original(ctx, l)
				if err != nil {
					return nil, err
				}
				body, err := json.Marshal(records)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(body, &records); err != nil {
					return nil, err
				}
				for i := range records {
					args := &records[i].Command.Args
					if args.Native != nil {
						if kind == "roster" {
							args.Native.Players[0].Franchise = args.Native.Franchises[1].ID
						}
						if kind == "contract" {
							args.Native.Players[0].Salary += 100000000
							args.Native.Players[0].ExpirationYear++
						}
					}
					if kind == "setting" {
						for j := range args.Seed {
							if args.Seed[j].Field == "scoring.rules" {
								var rules leaguerules.Value[map[leaguerules.StatID][]leaguerules.RangeRule]
								if err := json.Unmarshal(args.Seed[j].Value, &rules); err != nil {
									return nil, err
								}
								rows := rules.Value["receiving_yards"]
								rows[0].Points *= 2
								rules.Value["receiving_yards"] = rows
								body, err := json.Marshal(rules)
								if err != nil {
									return nil, err
								}
								args.Seed[j].Value = body
							}
						}
					}
				}
				return records, nil
			}
			if !read().Preparing {
				t.Fatal("changed inputs returned stale values")
			}
			prepare()
			expected, err := src.Read(t.Context(), league, "", season)
			if err != nil {
				t.Fatal(err)
			}
			if got := read(); got.Preparing || !reflect.DeepEqual(got, expected) {
				t.Fatal("prepared values are not fresh")
			}
			if kind != "contract" && reflect.DeepEqual(expected, want) {
				t.Fatal("fixture change had no effect")
			}
		})
	}
	t.Run("inflight", func(t *testing.T) { checkInflightPreparation(t, src, league) })
	t.Run("startup", func(t *testing.T) { checkStartupPreparation(t, src, league) })
}

type gatedFacts struct {
	Facts
	entered chan struct{}
	release chan struct{}
	calls   *atomic.Int64
	failure error
}

func (f gatedFacts) Book(ctx context.Context, q playerrecord.Query) (playerrecord.Book, error) {
	f.calls.Add(1)
	close(f.entered)
	select {
	case <-f.release:
	case <-ctx.Done():
		return playerrecord.Book{}, ctx.Err()
	}
	if f.failure != nil {
		return playerrecord.Book{}, f.failure
	}
	return f.Facts.Book(ctx, q)
}

func checkInflightPreparation(t *testing.T, src Source, league string) {
	t.Helper()
	var revision, calls atomic.Int64
	src.Revision = func(context.Context) (int64, error) { return revision.Load(), nil }
	entered, release := make(chan struct{}), make(chan struct{})
	src.Facts = gatedFacts{Facts: src.Facts, entered: entered, release: release, calls: &calls}
	prep := &Preparation{}
	prep.cache(Scope{league, season})
	done := make(chan struct{})
	go func() { defer close(done); prep.sweep(t.Context(), src, func() {}, func(err error) { t.Error(err) }) }()
	<-entered
	for range 3 {
		out, err := prep.Read(t.Context(), src, league, "", season)
		if err != nil || !out.Preparing {
			t.Fatalf("inflight read: %+v %v", out, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("screen duplicated background work")
	}
	revision.Add(1)
	close(release)
	<-done
	out, err := prep.Read(t.Context(), src, league, "", season)
	if err != nil || !out.Preparing {
		t.Fatal("published a pass across a commit")
	}
}

func checkStartupPreparation(t *testing.T, src Source, league string) {
	t.Helper()
	prep := &Preparation{}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ready, done := make(chan struct{}, 1), make(chan struct{})
	go func() {
		defer close(done)
		prep.Run(ctx, src, func(context.Context) ([]Scope, error) {
			return src.Scopes(ctx, func(context.Context) ([]commissioner.NativeLeague, error) {
				return []commissioner.NativeLeague{{ID: league}}, nil
			})
		}, func() { ready <- struct{}{} }, func(err error) { t.Error(err) })
	}()
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		t.Fatal("startup did not prepare before first screen")
	}
	out, err := prep.Read(t.Context(), src, league, "", season)
	if err != nil || out.Preparing || len(out.Players) == 0 {
		t.Fatal("first screen did not use startup pass")
	}
	cancel()
	<-done
	failure := errors.New("facts unavailable")
	var calls atomic.Int64
	release := make(chan struct{})
	close(release)
	src.Facts = gatedFacts{Facts: src.Facts, entered: make(chan struct{}), release: release, calls: &calls, failure: failure}
	prep = &Preparation{}
	prep.cache(Scope{league, season})
	prep.sweep(t.Context(), src, func() {}, func(error) {})
	if _, err := prep.Read(t.Context(), src, league, "", season); !errors.Is(err, failure) {
		t.Fatalf("real preparation failure hidden: %v", err)
	}
}
