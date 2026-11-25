package serverhttp

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestRequestTallyBounded(t *testing.T) {
	tally := NewRequestTally()
	for i := range 300 {
		tally.add(fmt.Sprintf("command.%d", i), "read", 0, 0)
	}
	_, counts := tally.Snapshot()
	if len(counts) != 256 || counts["other"].Calls != 45 {
		t.Fatalf("counts not bounded: %d keys, other %+v", len(counts), counts["other"])
	}
}

// timedRequest serves one request through withErrorCodes on a fake clock that advances by took
// inside the handler, and returns the response and the log text.
func timedRequest(t *testing.T, status int, took time.Duration) (*httptest.ResponseRecorder, string) {
	t.Helper()
	var logs bytes.Buffer
	clock := time.Date(2026, 11, 24, 20, 0, 0, 0, time.UTC)
	h := Hosted{ErrorLog: slog.New(slog.NewTextHandler(&logs, nil)), now: func() time.Time { return clock }}
	handler := h.withErrorCodes(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setFailureLeague(r.Context(), "native.timing")
		setFailureCommand(r.Context(), "results.read")
		clock = clock.Add(took)
		w.WriteHeader(status)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/native", nil))
	return rec, logs.String()
}

func TestServerTimingOnEveryResponse(t *testing.T) {
	shape := regexp.MustCompile(`^app;dur=[0-9]+\.[0-9]$`)
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		rec, _ := timedRequest(t, status, 12345*time.Microsecond)
		got := rec.Header().Get("Server-Timing")
		if !shape.MatchString(got) || got != "app;dur=12.3" {
			t.Fatalf("status %d: Server-Timing = %q, want app;dur=12.3", status, got)
		}
	}
}

func TestSlowRequestLoggedWithoutAccount(t *testing.T) {
	_, logs := timedRequest(t, http.StatusOK, 300*time.Millisecond)
	if !strings.Contains(logs, `msg="slow request"`) || !strings.Contains(logs, "command=results.read") ||
		!strings.Contains(logs, "league=native.timing") || !strings.Contains(logs, "ms=300") {
		t.Fatalf("slow request log missing fields: %q", logs)
	}
	if strings.Contains(logs, "account") {
		t.Fatalf("slow request log names an account: %q", logs)
	}
	if _, fast := timedRequest(t, http.StatusOK, 100*time.Millisecond); fast != "" {
		t.Fatalf("100 ms request logged: %q", fast)
	}
}
