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
			func(context.Context, string) ([]commissioner.Record, error) { return nil, nil },
			func() time.Time { t.Fatal("held capture read the default clock"); return time.Time{} },
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
