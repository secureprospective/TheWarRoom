package commissioner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestSeedCommandRetryDoesNotRecapture(t *testing.T) {
	t.Parallel()
	input := RawNativeSeedRequest{League: "native.test", Expected: 4, Request: "request", Reason: "starting"}
	for _, tc := range []struct {
		command  string
		expected int64
		reason   string
		ok       bool
	}{
		{commissioner.NativeSeed, 4, "starting", true}, {commissioner.NativeSeed, 5, "starting", false}, {commissioner.NativeSeed, 4, "other", false}, {commissioner.RulesSet, 4, "starting", false},
	} {
		raw := commissioner.RawCommand{Command: tc.command, Expected: tc.expected, Reason: tc.reason}
		got, err := BuildSeedCommand(t.Context(), input, func(context.Context, string, string) (commissioner.RawCommand, bool, error) { return raw, true, nil }, func() (commissioner.RawNativeSnapshot, error) {
			t.Fatal("retry recaptured held state")
			return commissioner.RawNativeSnapshot{}, nil
		})
		if (err == nil) != tc.ok || got.Command != raw.Command || got.Expected != raw.Expected || got.Reason != raw.Reason {
			t.Fatalf("retry altered: %+v %v", got, err)
		}
	}
	failure := errors.New("read failed")
	if _, err := BuildSeedCommand(t.Context(), input, func(context.Context, string, string) (commissioner.RawCommand, bool, error) {
		return commissioner.RawCommand{}, false, failure
	}, nil); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	raw, err := BuildSeedCommand(t.Context(), input, func(context.Context, string, string) (commissioner.RawCommand, bool, error) {
		return commissioner.RawCommand{}, false, nil
	}, func() (commissioner.RawNativeSnapshot, error) {
		return commissioner.RawNativeSnapshot{Season: 2026}, nil
	})
	if err != nil || raw.League != input.League || raw.Args.Native.Season != 2026 || raw.Visibility != "league" || raw.Effective != "now" || raw.Request != input.Request {
		t.Fatalf("seed command: %+v %v", raw, err)
	}
}

func TestLeagueSeedUsesHeldCaptureAndPreservesFailure(t *testing.T) {
	t.Parallel()
	input := RawNativeSeedRequest{League: "native.held", Expected: 1, Request: "start", Reason: "Start"}
	failure := errors.New("held data unavailable")
	for _, captureErr := range []error{nil, failure} {
		calls := 0
		raw, err := BuildLeagueSeedCommand(t.Context(), input,
			func(context.Context, string) ([]commissioner.Record, error) {
				return []commissioner.Record{{Command: commissioner.RawCommand{League: input.League, Command: commissioner.Create, Args: commissioner.RawArgs{Preset: commissioner.PresetLegacy, Season: 2026}}, Receipt: commissioner.Receipt{RecordedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}}}, nil
			},
			func() time.Time { return time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC) },
			func() (commissioner.RawNativeSnapshot, error) {
				calls++
				return commissioner.RawNativeSnapshot{Season: 2026}, captureErr
			})
		if calls != 1 || !errors.Is(err, captureErr) {
			t.Fatalf("capture calls=%d, error=%v", calls, err)
		}
		if err == nil && (raw.Args.Native.Season != 2026 || raw.Request != input.Request) {
			t.Fatalf("held seed changed: %+v", raw)
		}
	}
}

func TestLeagueSeedChoosesCaptureBySetting(t *testing.T) {
	for _, tc := range []struct {
		name, preset, source string
		held                 bool
		lots                 int
	}{
		{"old Legacy", commissioner.PresetLegacy, "", true, 0},
		{"import with empty preset", commissioner.PresetEmpty, commissioner.SourceMFLImport, true, 0},
		{"empty native", commissioner.PresetEmpty, commissioner.SourceNative, false, 0},
		{"native Legacy rules", commissioner.PresetLegacy, commissioner.SourceNative, false, 0},
		{"default native", commissioner.PresetDefault32, commissioner.SourceNative, false, 32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			create, err := (commissioner.RawCommand{League: "native.test", Command: commissioner.Create,
				Args: commissioner.RawArgs{Name: "Ours", Preset: tc.preset, Source: tc.source,
					Season: 2026, FranchiseCount: 32}}).Prepared()
			if err != nil {
				t.Fatal(err)
			}
			create.Args.Source = tc.source
			if tc.name == "old Legacy" { // Records predating the two settings carry neither.
				seed := create.Args.Seed[:0]
				for _, s := range create.Args.Seed {
					if s.Field != "lineup.lock" && s.Field != "league.startingAssets" {
						seed = append(seed, s)
					}
				}
				create.Args.Seed = seed
			}
			at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
			records := []commissioner.Record{{Command: create, Principal: "creator",
				Receipt: commissioner.Receipt{Entry: "create", Revision: 1, RecordedAt: at}}}
			calls := 0
			input := RawNativeSeedRequest{League: "native.test", Expected: 1, Request: "start", Reason: "Start"}
			raw, err := BuildLeagueSeedCommand(t.Context(), input,
				func(context.Context, string) ([]commissioner.Record, error) { return records, nil },
				func() time.Time {

					return at.Add(time.Hour)
				}, func() (commissioner.RawNativeSnapshot, error) {
					calls++
					return commissioner.RawNativeSnapshot{Season: 2026}, nil
				})
			if err != nil || (calls == 1) != tc.held || raw.Args.Native == nil || len(raw.Args.Native.Pool) != tc.lots {
				t.Fatalf("calls=%d seed=%+v error=%v", calls, raw, err)
			}
		})
	}
}
