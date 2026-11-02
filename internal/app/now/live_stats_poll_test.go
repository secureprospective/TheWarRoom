package now

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

type livePollHistory struct {
	lines       map[string]source.PlayerWeek
	writes      int
	fail        bool
	captureFail bool
}

func (h *livePollHistory) SourceIDs(context.Context, string) (map[string]string, error) {
	if h.fail {
		return nil, fmt.Errorf("identity unavailable")
	}
	return map[string]string{}, nil
}
func (h *livePollHistory) SetLiveStats(_ context.Context, _, _ int, _ time.Time, _ string, _ []string, lines map[string]source.PlayerWeek) error {
	if h.captureFail {
		return fmt.Errorf("capture unavailable")
	}
	h.lines = lines
	h.writes++
	return nil
}

type livePollTransport struct {
	body string
	fail bool
}

func (p livePollTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if p.fail {
		return nil, fmt.Errorf("offline")
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(p.body))}, nil
}
func TestLivePollClearsOnlyAfterSuccessfulAcquisition(t *testing.T) {
	for _, tc := range []struct {
		name, body                       string
		networkFail, identityFail, clear bool
	}{
		{name: "empty board", body: `{"events":[]}`, clear: true},
		{name: "missing board", body: `{}`},
		{name: "bad board", body: `not json`},
		{name: "offline", networkFail: true},
		{name: "identity failure", body: `{"events":[]}`, identityFail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &livePollHistory{lines: map[string]source.PlayerWeek{"0099": {Season: 2026, Week: 8}}, fail: tc.identityFail}
			calls, changed := 0, 0
			var poll LiveStatsPoller
			poll.poll(t.Context(), LiveStatsRun{Network: livePollTransport{body: tc.body, fail: tc.networkFail}, Season: 2026,
				Week: func() int { calls++; return 8 }, Changed: func(context.Context) { changed++ }}, h)
			if calls != 1 {
				t.Fatalf("scope captured %d times", calls)
			}
			if tc.clear {
				if len(h.lines) != 0 || h.writes != 1 || changed != 1 {
					t.Fatalf("not cleared: %+v changes %d", h, changed)
				}
			} else if len(h.lines) != 1 || h.writes != 0 || changed != 0 {
				t.Fatalf("failure lost held evidence: %+v", h)
			}
		})
	}
}
func TestLiveGameAcquisitionFailureIsNotAnEmptySuccess(t *testing.T) {
	var finals liveFinals
	lines, ok := liveStatLines(t.Context(), livegameFailureAdapter(), []livegame.Game{{ID: "1", Status: "in", Season: 2026, Week: 8, SeasonType: 2}}, nil, &finals, time.Now())
	if ok || len(lines) != 0 {
		t.Fatalf("failed acquisition marked successful: %t %+v", ok, lines)
	}
}
func livegameFailureAdapter() livegame.Adapter {
	return livegame.Adapter{Fetch: func(context.Context, string, string) ([]byte, error) { return nil, fmt.Errorf("offline") }}
}

func TestLivePollCaptureFailureDoesNotPublishChange(t *testing.T) {
	h := &livePollHistory{captureFail: true, lines: map[string]source.PlayerWeek{"0099": {Season: 2026, Week: 8}}}
	changed := 0
	var poll LiveStatsPoller
	poll.poll(t.Context(), LiveStatsRun{Network: livePollTransport{body: `{"events":[]}`}, Season: 2026, Week: func() int { return 8 }, Changed: func(context.Context) { changed++ }}, h)
	if changed != 0 || h.writes != 0 || len(h.lines) != 1 {
		t.Fatal("capture failure published or lost evidence")
	}
}
