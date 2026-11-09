package composition

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/mflsource"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type scheduleRefreshTransport struct {
	fail   bool
	reads  int
	weekly string
}

func (f *scheduleRefreshTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	switch r.URL.Query().Get("TYPE") {
	case "league":
		return seasonResponse(`{"league":{"baseURL":"https://www47.myfantasyleague.com/2026/home/14432"}}`), nil
	case "transactions":
		return seasonResponse(`{"transactions":{}}`), nil
	case "weeklyResults":
		if f.weekly == "" {
			return seasonResponse(`{"allWeeklyResults":{"weeklyResults":{"week":"1"}}}`), nil
		}
		return seasonResponse(f.weekly), nil
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

type scoringLaunchTransport struct{ reads int }

func (f *scoringLaunchTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	name := "weeks.csv"
	plays := r.URL.String() == source.PlaysURL(2026)
	if plays {
		name = "plays.csv"
	}
	body, err := os.ReadFile("internal/fantasyscore/source/testdata/" + name)
	if err != nil {
		return nil, err
	}
	body = bytes.ReplaceAll(body, []byte("2025"), []byte("2026"))
	if plays {
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		if _, err := writer.Write(body); err != nil {
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		body = compressed.Bytes()
	} else if !strings.Contains(r.URL.Path, "stats_player_week_2026.csv") {
		return nil, errors.New("unexpected scoring request")
	}
	f.reads++
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(body))}, nil
}

type nflDatesTransport struct{ weeks []int }

func (f *nflDatesTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	week, err := strconv.Atoi(r.URL.Query().Get("W"))
	if err != nil || r.URL.Query().Get("TYPE") != "nflSchedule" {
		return nil, errors.New("unexpected NFL date request")
	}
	f.weeks = append(f.weeks, week)
	at := time.Date(2026, 9, 11, 0, 15, 0, 0, time.UTC).AddDate(0, 0, (week-1)*7)
	final := "0"
	if week >= 8 {
		final = "3600"
	}
	return seasonResponse(fmt.Sprintf(`{"nflSchedule":{"week":"%d","matchup":{"kickoff":"%d","gameSecondsRemaining":"%s","team":[{"id":"A"},{"id":"B"}]}}}`, week, at.Unix(), final)), nil
}

func TestScheduleRefreshFillsOnlyMissingNFLDates(t *testing.T) {
	at := time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)
	a, _ := calendarTestApp(t, at)
	raw := seedScheduleTest(t, a)
	raw.Args.Schedule.Weeks = 13
	raw.Args.Schedule.Pattern = ""
	executeCommissionerTest(t, a, raw)
	records, err := a.leagueRecords(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, int64(len(records)), "publish-dates"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, a, raw)
	for _, week := range []int{7, 8, 9, 13} {
		heldNativeWeek(t, a, week, time.Date(2026, 9, 11, 0, 15, 0, 0, time.UTC).AddDate(0, 0, (week-1)*7), week < 8)
	}
	// Feed freshness uses wall time; the league reading keeps its pinned instant.
	a.core.History.SetClock(time.Now)
	feed := &nflDatesTransport{}
	stats := &scoringLaunchTransport{}
	a.core.Fetches = &archive.Transport{Base: stats, Sink: a.core.History}
	a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(&archive.Transport{Base: feed, Sink: a.core.History}))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		a.syncLeagueSchedulesAtLaunch(t.Context())
	}
	if stats.reads != 2 {
		t.Fatalf("ordinary launch did not acquire/cache scoring evidence: %d reads", stats.reads)
	}
	held, err := a.core.History.PlayerWeekStats(t.Context(), source.RawStatScope{GSIS: "00-0023459", Season: 2026, Week: 1})
	if err != nil || len(held) != 2 || len(held[0].Missing) != 0 || held[0].Week == nil {
		t.Fatalf("launch held evidence: %+v %v", held, err)
	}
	if !reflect.DeepEqual(feed.weeks, []int{1, 2, 3, 4, 5, 6, 10, 11, 12}) {
		t.Fatalf("missing dates requested: %v", feed.weeks)
	}
	week, _ := nativeOwnerRead(t, a, at)
	if week.Current == nil || week.Current.Week != 8 {
		t.Fatalf("current week: %+v", week)
	}
}

func TestLaunchAutomaticallyImportsPlayedLineups(t *testing.T) {
	a := commissionerTestApp(t)
	a.core.ClockNow = func() time.Time { return migrationtest.ReadAt.Add(time.Hour) }
	a.core.History.SetLineupEnvironment(appLineupEnv{app: a})
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	ledger := leagueservice.MigrationLedger{Store: a.core.History}
	svc := leaguemigration.Service{Ledger: ledger, Now: a.timeNow, Source: func(string) (leaguemigration.Source, error) { return &migrationtest.Source{Export: export}, nil }}
	if _, err := svc.Import(t.Context(), export.LeagueID, "Import", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	assets, err := a.core.History.NativeAssets(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	p := assets.Snapshot.Players[0]
	// Launch has no target snapshot; the previously fetched player directory is sufficient.
	recordSeasonArchiveForm(t, a, "players", "", fmt.Sprintf(`{"players":{"player":{"id":%q,"name":"Held starter","position":"QB","team":"BUF"}}}`, p.ID), migrationtest.ReadAt, true)
	feed := &scheduleRefreshTransport{weekly: fmt.Sprintf(`{"allWeeklyResults":{"weeklyResults":[{"week":"1","franchise":{"id":"0001","score":"10","player":[{"id":%q,"status":"starter","score":"10"},{"id":"0000","status":"starter","score":"0"}]}},{"week":"2","franchise":{"id":"0001","player":{"id":%q,"status":"starter"}}}]}}`, p.ID, p.ID)}
	a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(feed))
	if err != nil {
		t.Fatal(err)
	}
	for pass := range 2 {
		a.syncLeagueSchedulesAtLaunch(t.Context())
		records, err := ledger.Records(t.Context(), league)
		if err != nil || len(records) != 5 {
			t.Fatalf("pass %d entries=%d: %v", pass, len(records), err)
		}
		held, found, err := commissioner.RecordedLineupFor(records, p.Franchise, export.Season, 1)
		if err != nil || !found || held.By != "creator" || len(held.Starters) != 1 || held.Starters[0] != p.ID {
			t.Fatalf("%+v %t %v", held, found, err)
		}
		if _, found, err := commissioner.RecordedLineupFor(records, p.Franchise, export.Season, 2); err != nil || found {
			t.Fatal("future lineup imported", err)
		}
	}
}

func TestScheduleFollowingBackfillAndCutover(t *testing.T) {
	a := commissionerTestApp(t)
	a.core.ClockNow = func() time.Time { return migrationtest.ReadAt.Add(time.Hour) }
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	plan, err := mflsource.Prepare(export)
	if err != nil {
		t.Fatal(err)
	}
	// Reproduce an import recorded before the setting existed.
	for i, seed := range plan.Create.Seed {
		if seed.Field == "schedule.follow" {
			plan.Create.Seed = append(plan.Create.Seed[:i], plan.Create.Seed[i+1:]...)
			break
		}
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	ledger := leagueservice.MigrationLedger{Store: a.core.History}
	raws := []commissioner.RawCommand{
		{League: league, Command: commissioner.Create, Args: plan.Create, Request: "old.create", Reason: "Import", Effective: "now", Visibility: "league"},
		{League: league, Command: commissioner.NativeSeed, Args: commissioner.RawArgs{Native: &plan.Seed}, Request: "old.seed", Reason: "Import", Effective: "now", Visibility: "league"},
	}
	if _, err := ledger.Batch(t.Context(), raws, "creator"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := leaguemigration.BackfillFollowing(t.Context(), ledger, league); err != nil {
			t.Fatal(err)
		}
	}
	records, err := ledger.Records(t.Context(), league)
	if err != nil || len(records) != 3 {
		t.Fatalf("backfill entries=%d: %v", len(records), err)
	}
	if id, _, _, ok := mflsource.FollowingLeague(league, records); !ok || id != export.LeagueID {
		t.Fatal("backfill did not follow", id, ok)
	}
	setFollow := func(value, request string) {
		t.Helper()
		held, err := ledger.Records(t.Context(), league)
		if err != nil {
			t.Fatal(err)
		}
		raw := commissioner.RawCommand{League: league, Command: commissioner.RulesSet, Expected: int64(len(held)),
			Args:    commissioner.RawArgs{Field: "schedule.follow", Value: json.RawMessage(value)},
			Request: request, Reason: "Choose reference following", Effective: "now", Visibility: "league"}
		if _, err := ledger.Execute(t.Context(), raw, "creator"); err != nil {
			t.Fatal(err)
		}
	}
	setFollow(`{"Source":"","League":"","Season":0}`, "clear.follow")
	if err := leaguemigration.BackfillFollowing(t.Context(), ledger, league); err != nil {
		t.Fatal(err)
	}
	cleared, err := ledger.Records(t.Context(), league)
	if err != nil || len(cleared) != 4 {
		t.Fatal("backfill replaced explicit clear", err)
	}
	if _, _, _, ok := mflsource.FollowingLeague(league, cleared); ok {
		t.Fatal("explicit clear still follows")
	}
	setFollow(fmt.Sprintf(`{"Source":"mfl","League":%q,"Season":%d}`, export.LeagueID, export.Season), "set.follow")
	svc := leaguemigration.Service{Ledger: ledger, Now: a.timeNow, Source: func(string) (leaguemigration.Source, error) { return &migrationtest.Source{Export: export}, nil }}
	req := leaguemigration.Request{League: league, MFLLeague: export.LeagueID}
	review, err := svc.Review(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Cutover(t.Context(), req, review.Digest, "Run here", "creator"); err != nil {
		t.Fatal(err)
	}
	records, err = ledger.Records(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, ok := mflsource.FollowingLeague(league, records); ok {
		t.Fatal("cutover still follows")
	}
	rules, err := commissioner.RulesNow(records, a.timeNow())
	if err != nil || rules.Schedule.Follow.Value != (leaguerules.ScheduleFollow{}) {
		t.Fatal("cutover did not clear setting", err)
	}
	if err := leaguemigration.BackfillFollowing(t.Context(), ledger, league); err != nil {
		t.Fatal(err)
	}
	after, err := ledger.Records(t.Context(), league)
	if err != nil || len(after) != len(records) {
		t.Fatal("backfill undid cutover", err)
	}
}

type blockedScoringTransport struct {
	entered chan struct{}
	release chan struct{}
}

func (f *blockedScoringTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	select {
	case f.entered <- struct{}{}:
	default:
	}
	select {
	case <-f.release:
		return (&scoringLaunchTransport{}).RoundTrip(r)
	case <-r.Context().Done():
		return nil, r.Context().Err()
	}
}

func TestScheduleEvidenceDoesNotHoldRefreshReadLock(t *testing.T) {
	at := time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)
	a, _ := calendarTestApp(t, at)
	raw := seedScheduleTest(t, a)
	executeCommissionerTest(t, a, raw)
	records, err := a.leagueRecords(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, int64(len(records)), "publish-unblocked"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	executeCommissionerTest(t, a, raw)
	feed := &blockedScoringTransport{entered: make(chan struct{}, 1), release: make(chan struct{})}
	a.core.Fetches = &archive.Transport{Base: feed, Sink: a.core.History}
	a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(&nflDatesTransport{}))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { defer close(done); a.syncLeagueSchedulesAtLaunch(t.Context()) }()
	defer func() { close(feed.release); <-done }()
	select {
	case <-feed.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("scoring acquisition did not start")
	}
	if !a.core.RefreshMu.TryLock() {
		t.Fatal("background scoring acquisition blocks foreground refresh reads")
	}
	a.core.RefreshMu.Unlock()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if _, err := a.core.History.NativeAssets(ctx, raw.League); err != nil {
		t.Fatalf("held assets blocked behind scoring: %v", err)
	}
}
