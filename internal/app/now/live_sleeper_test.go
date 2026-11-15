package now

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type sleeperPollHistory struct{ livePollHistory }

func (*sleeperPollHistory) SourceIDs(context.Context, string) (map[string]string, error) {
	return map[string]string{"0099": "1", "0100": "2"}, nil
}

type sleeperTransport struct {
	t      *testing.T
	status int
	body   string
	calls  int
}

func (s *sleeperTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.calls++
	if r.URL.Path != "/v1/stats/nfl/regular/2026/8" {
		s.t.Fatal(r.URL)
	}
	return &http.Response{StatusCode: s.status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(s.body))}, nil
}

func TestLiveSleeperAcquisitionFillsOnlyGaps(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		fill   bool
	}{
		{"success", 200, `{"1":{"pass_yd":999,"rec_drop":2},"2":{"st_tkl_solo":1},"3":{"rec_drop":9}}`, true},
		{"unavailable", 503, `offline`, false},
		{"bad response", 200, `{"1":{"pass_yd":-999,"rec_drop":-1}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			network := &sleeperTransport{t: t, status: tc.status, body: tc.body}
			lines := map[string]source.PlayerWeek{"0099": {Season: 2026, Week: 8, Line: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"passing_yards": 0}}}}
			fillLiveSleeper(t.Context(), LiveStatsRun{Network: network}, &sleeperPollHistory{}, []livegame.Game{{Season: 2026, Week: 8, SeasonType: 2, Status: "in"}, {Season: 2026, Week: 8, SeasonType: 2, Status: "post"}}, lines)
			if network.calls != 1 {
				t.Fatalf("scope fetched %d times", network.calls)
			}
			if value, ok := lines["0099"].Line.Quantities["passing_yards"]; !ok || value != 0 {
				t.Fatal("ESPN overwritten")
			}
			if tc.fill {
				if lines["0099"].Line.Quantities["receiving_drop"] != 20000 || lines["0099"].Provenance != "ESPN live + Sleeper" || lines["0100"].Provenance != source.Sleeper || len(lines) != 2 {
					t.Fatalf("%+v", lines)
				}
			} else if len(lines) != 1 || len(lines["0099"].Line.Quantities) != 1 {
				t.Fatalf("failure mutated evidence: %+v", lines)
			}
		})
	}
}

type outageSleeperTransport struct{ failSleeper bool }

func (n outageSleeperTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if !strings.Contains(r.URL.Path, "/stats/nfl/regular/") || n.failSleeper {
		return nil, fmt.Errorf("provider unavailable")
	}
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"1":{"pass_yd":245,"pass_td":2},"2":{"rec_drop":1}}`))}, nil
}

func TestLiveESPNOutageSelectsIndependentSleeper(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint("SleeperDown=", fail), func(t *testing.T) {
			held := &sleeperPollHistory{livePollHistory: livePollHistory{lines: map[string]source.PlayerWeek{"held": {Season: 2026, Week: 8}}}}
			var poll LiveStatsPoller
			changes := 0
			poll.poll(t.Context(), LiveStatsRun{Season: 2026, Week: func() int { return 8 }, Network: outageSleeperTransport{failSleeper: fail}, Changed: func(context.Context) { changes++ }}, held)
			if fail {
				if held.writes != 0 || len(held.lines) != 1 || changes != 0 {
					t.Fatal("outage discarded held evidence", held)
				}
			} else if held.writes != 1 || changes != 1 || held.lines["0099"].Line.Quantities["passing_yards"] != 2450000 || held.lines["0099"].Provenance != source.Sleeper {
				t.Fatalf("Sleeper did not pick up: %+v", held)
			}
		})
	}
}
