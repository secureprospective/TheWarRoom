package composition

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type scheduleRefreshTransport struct {
	fail  bool
	reads int
}

func (f *scheduleRefreshTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	switch r.URL.Query().Get("TYPE") {
	case "league":
		return seasonResponse(`{"league":{"baseURL":"https://www47.myfantasyleague.com/2026/home/14432"}}`), nil
	case "schedule":
		f.reads++
		if f.fail {
			return nil, errors.New("offline")
		}
		return seasonResponse(`{"schedule":{"weeklySchedule":[{"week":"1","matchup":{"franchise":[{"id":"0001","isHome":"0"},{"id":"0002","isHome":"1"}]}},{"week":"2"}]}}`), nil
	default:
		return nil, errors.New("unexpected export")
	}
}

func TestLaunchAutomaticallyBackfillsSchedule(t *testing.T) {
	a := commissionerTestApp(t)
	a.core.ClockNow = func() time.Time { return migrationtest.ReadAt.Add(time.Hour) }
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	ledger := leagueservice.MigrationLedger{Store: a.core.History}
	svc := leaguemigration.Service{Ledger: ledger, Now: a.timeNow,
		Source: func(string) (leaguemigration.Source, error) { return &migrationtest.Source{Export: export}, nil }}
	if _, err := svc.Import(t.Context(), export.LeagueID, "Import", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	feed := &scheduleRefreshTransport{}
	a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(feed))
	if err != nil {
		t.Fatal(err)
	}
	before, err := a.core.History.NativeAssets(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	noop := func(context.Context) {}
	launch := controlapp.BackgroundRefresh{Ready: a.ready, Start: func(f func()) { f() },
		Due: func() bool { return false }, Clock: noop, Watch: noop, Launch: func(context.Context) { t.Fatal("ordinary launch refresh was not due") },
		Sync: a.syncLeagueSchedulesAtLaunch, Weeks: noop}
	for pass := range 3 {
		feed.fail = pass == 2
		launch.StartWorkers(t.Context())
		records, err := ledger.Records(t.Context(), league)
		if err != nil || len(records) != 4 {
			t.Fatalf("pass %d records %d: %v", pass, len(records), err)
		}
		schedule, err := commissioner.ScheduleReading(league, records)
		if err != nil || len(schedule.Weeks) != 2 || schedule.Weeks[0].Games[0].Away != "franchise.01" {
			t.Fatalf("published: %+v %v", schedule, err)
		}
	}
	after, err := a.core.History.NativeAssets(t.Context(), league)
	if err != nil || len(before.Snapshot.Players) != len(after.Snapshot.Players) {
		t.Fatal("starting roster changed", err)
	}
	if feed.reads < 3 {
		t.Fatal("refresh did not retry")
	}
}
