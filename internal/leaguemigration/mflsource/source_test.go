package mflsource_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type fixtureTransport struct {
	t             *testing.T
	reads, writes int
}

func (f *fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	kind := req.URL.Query().Get("TYPE")
	paths := map[string]string{
		"league":           "league/testdata/league-2026.json",
		"rules":            "league/testdata/rules-2026.json",
		"rosters":          "testdata/comms/rosters-2026.json",
		"draftResults":     "draftreports/testdata/results-2026.json",
		"futureDraftPicks": "draftreports/testdata/future-2026.json",
	}
	if req.Method != http.MethodGet || (paths[kind] == "" && kind != "salaryAdjustments" && kind != "schedule" && kind != "transactions" && kind != "weeklyResults") {
		f.writes++
		return nil, fmt.Errorf("unexpected MFL operation: %s %s", req.Method, kind)
	}
	f.reads++
	body := []byte(`{"salaryAdjustments":{}}`)
	if kind == "transactions" {
		body = []byte(`{"transactions":{}}`)
	}
	if kind == "weeklyResults" {
		body = []byte(`{"allWeeklyResults":{"weeklyResults":{"week":"1"}}}`)
	}
	if kind == "schedule" {
		body = []byte(`{"schedule":{"weeklySchedule":{"week":"1"}}}`)
	}
	if path := paths[kind]; path != "" {
		var err error
		body, err = os.ReadFile("../../ingestion/" + path)
		if err != nil {
			f.t.Fatal(err)
		}
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{},
		Body: io.NopCloser(strings.NewReader(string(body))), Request: req}, nil
}

func TestSourceAdapterOnlyReadsMFL(t *testing.T) {
	t.Parallel()
	fixture, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	transport := &fixtureTransport{t: t}
	client, err := mfl.New("api", 100000, mfl.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}
	source := &mflsource.Source{Client: client, ID: fixture.LeagueID,
		Now: func() time.Time { return migrationtest.ReadAt }, Held: func() map[string]string { return fixture.Players }}
	candidate, err := source.Read(t.Context(), migrationtest.ReadAt)
	if err != nil || len(candidate.Problems) != 0 || len(candidate.Plan.Seed.Players) != len(fixture.Rosters) {
		t.Fatalf("source adapter: candidate=%+v err=%v", candidate.Problems, err)
	}
	if len(candidate.Plan.Schedule) != 1 || transport.reads == 0 || transport.writes != 0 {
		t.Fatalf("source traffic: reads=%d writes=%d", transport.reads, transport.writes)
	}
}

func TestSourceThreadsSelectedBase(t *testing.T) {
	t.Parallel()
	fixture, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	for _, preset := range []string{"legacy_nfl", "default32", "mirror16"} {
		t.Run(preset, func(t *testing.T) {
			base, err := leaguemigration.Base(preset)
			if err != nil {
				t.Fatal(err)
			}
			transport := &fixtureTransport{t: t}
			client, err := mfl.New("api", 100000, mfl.WithTransport(transport))
			if err != nil {
				t.Fatal(err)
			}
			source := &mflsource.Source{Client: client, ID: fixture.LeagueID, Base: base,
				Now: func() time.Time { return migrationtest.ReadAt }, Held: func() map[string]string { return fixture.Players }}
			candidate, err := source.Read(t.Context(), migrationtest.ReadAt)
			if err != nil || len(candidate.Problems) > 0 {
				t.Fatalf("%v: %v", err, candidate.Problems)
			}
			if candidate.Plan.Rules.Draft.StartupRounds.Value != base.Draft.StartupRounds.Value {
				t.Fatal("source dropped the selected base")
			}
			if candidate.Plan.Rules.Cap.Amount.Value != 12500000000 || transport.writes != 0 {
				t.Fatal("base replaced MFL's cap or the source attempted a write")
			}
		})
	}
}
