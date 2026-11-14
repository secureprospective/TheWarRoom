package serverclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestInvalidationProtocolRejectsCursorsAndBoundsFrames(t *testing.T) {
	for _, body := range []string{
		"id: 99\nevent: invalidate\ndata: {\"protocol\":1}\n\n",
		"event: invalidate\ndata: {\"protocol\":2}\n\n",
		"event: ledger\ndata: {\"protocol\":1}\n\n",
		"data: " + strings.Repeat("x", 5000) + "\n\n",
	} {
		calls := 0
		response := &http.Response{Body: io.NopCloser(strings.NewReader(body))}
		_, err := consumeInvalidations(t.Context(), response, func(context.Context) error { calls++; return nil })
		if err == nil || calls != 0 {
			t.Fatalf("unvalidated event reached reread: calls=%d err=%v", calls, err)
		}
	}
}

func TestInvalidationsRereadOnlyCompleteValidatedEvents(t *testing.T) {
	response := &http.Response{Body: io.NopCloser(strings.NewReader(": keepalive\n\nevent: invalidate\ndata: {\"protocol\":1}\n\nevent: invalidate\ndata: {\"protocol\":1}"))}
	calls := 0
	retry, err := consumeInvalidations(t.Context(), response, func(context.Context) error { calls++; return nil })
	if !retry || err == nil || calls != 1 {
		t.Fatalf("calls=%d retry=%v err=%v", calls, retry, err)
	}
}

// TestReconnectBackoffGrowsCapsJittersAndResets: ceilings double from 1 s to the 30 s cap, waits are
// spread across each ceiling, and a stream that stayed up a minute starts again from 1 s.
func TestReconnectBackoffGrowsCapsJittersAndResets(t *testing.T) {
	ceilings := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second, 30 * time.Second, 30 * time.Second}
	waits := make([]map[time.Duration]bool, len(ceilings))
	for range 200 {
		var backoff reconnectBackoff
		for i, ceiling := range ceilings {
			wait := backoff.next(time.Second)
			if wait <= 0 || wait > ceiling {
				t.Fatalf("attempt %d waited %v, want (0, %v]", i, wait, ceiling)
			}
			if waits[i] == nil {
				waits[i] = map[time.Duration]bool{}
			}
			waits[i][wait] = true
		}
		if wait := backoff.next(streamHealthyAfter); wait > time.Second {
			t.Fatalf("a healthy stream did not reset the backoff: %v", wait)
		}
	}
	var lateHigh bool
	for i, seen := range waits {
		if len(seen) < 50 {
			t.Fatalf("attempt %d is not jittered: %d distinct waits", i, len(seen))
		}
		for wait := range seen {
			lateHigh = lateHigh || i >= 5 && wait > 16*time.Second
		}
	}
	if !lateHigh {
		t.Fatal("waits never grew past 16 s")
	}
}
