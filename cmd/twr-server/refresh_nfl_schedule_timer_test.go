package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func TestHostedNFLSchedulePassFetchesEachWeekOnceAndRePinsChangedLeagues(t *testing.T) {
	host, _, _ := scheduleRefreshFixture(t)
	ctx := t.Context()
	if err := host.files.Close(); err != nil {
		t.Fatal(err)
	}
	body, digest := footballEvidenceWithWeeks(t, "Schedule refresh", futureScheduleWeek(22))
	leagues := []string{"native.second", "native.third"}
	for _, league := range leagues {
		args := []string{"-db", filepath.Join(host.dir, "server.db"), "-league", league, "-name", league, "-preset", "default_32", "-season", "2026", "-commissioner"}
		credentials := strings.NewReader(`{"username":"commissioner","password":"private-password"}`)
		if err := runCreateLeague(ctx, args, credentials, discardOutput{}); err != nil {
			t.Fatalf("create %s: %v", league, err)
		}
	}
	host.files = openTestServer(t, host.dir)
	firstStore, err := leagueStore(ctx, host.files, "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	if err := firstStore.ImportFootball(ctx, body, digest); err != nil {
		t.Fatal(err)
	}
	if err := signInStore(host.files).PinFootball(ctx, "native.hosted", digest); err != nil {
		t.Fatal(err)
	}
	account, err := accounts.New(signInStore(host.files)).ConsoleAccount(ctx, accounts.RawCredentials{Username: "commissioner", Password: "private-password"})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := account.Principal()
	if err != nil {
		t.Fatal(err)
	}
	for _, league := range leagues {
		store, err := leagueStore(ctx, host.files, league)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.ImportFootball(ctx, body, digest); err != nil {
			t.Fatalf("import evidence for %s: %v", league, err)
		}
		if err := signInStore(host.files).PinFootball(ctx, league, digest); err != nil {
			t.Fatalf("pin evidence for %s: %v", league, err)
		}
		if league != "native.third" {
			publishTestSchedule(t, store, league, string(principal))
		}
	}
	// Publish the same schedule in the original league; the third league remains unpublished.
	publishTestSchedule(t, firstStore, "native.hosted", string(principal))
	pins := newLeaguePins(host.files.leagues)
	directory, err := signInStore(host.files).Directory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := pins.selectPins(ctx, "", directory); err != nil {
		t.Fatal(err)
	}
	fetchCount := map[int]int{}
	fetch := func(_ context.Context, season, week int) (leagueweek.Week, error) {
		if season != 2026 {
			t.Fatalf("fetch season = %d", season)
		}
		fetchCount[week]++
		return futureScheduleWeek(week), nil
	}
	pass := refreshHostedNFLSchedulePass(ctx, pins, signInStore(host.files), 2026, fetch)
	if pass.leaguesRevised != 2 || pass.leaguesUnchanged != 1 || pass.errors != 0 {
		t.Fatalf("pass = %+v", pass)
	}
	if pass.weeksFetched != len(fetchCount) || pass.weeksFetched == 0 {
		t.Fatalf("unique fetched weeks = %d; fetches = %v", pass.weeksFetched, fetchCount)
	}
	for week, count := range fetchCount {
		if count != 1 {
			t.Fatalf("week %d fetched %d times; all counts: %v", week, count, fetchCount)
		}
	}
	for _, league := range []string{"native.hosted", "native.second"} {
		if got := pins.digest(league); got == digest || got == "" {
			t.Fatalf("%s was not re-pinned: %q", league, got)
		}
	}
	if got := pins.digest("native.third"); got != digest {
		t.Fatalf("unpublished league pin = %q, want %q", got, digest)
	}
	firstPins := map[string]string{"native.hosted": pins.digest("native.hosted"), "native.second": pins.digest("native.second"), "native.third": pins.digest("native.third")}
	second := refreshHostedNFLSchedulePass(ctx, pins, signInStore(host.files), 2026, fetch)
	if second.leaguesRevised != 0 || second.leaguesUnchanged != 3 || second.errors != 0 {
		t.Fatalf("second pass = %+v", second)
	}
	for league, want := range firstPins {
		if got := pins.digest(league); got != want {
			t.Fatalf("second pass changed %s digest from %s to %s", league, want, got)
		}
	}
	// A commissioner re-pins native.hosted while a pass is fetching: the pass must not overwrite that choice.
	// Put the league back on the version that lacks its weeks, so the pass fetches; mid-fetch the commissioner
	// chooses the revised version instead.
	chosen := firstPins["native.hosted"]
	if err := pins.setPin(ctx, signInStore(host.files), "native.hosted", digest); err != nil {
		t.Fatal(err)
	}
	repinned := false
	racing := func(ctx context.Context, _, week int) (leagueweek.Week, error) {
		if !repinned {
			repinned = true
			if err := pins.setPin(ctx, signInStore(host.files), "native.hosted", chosen); err != nil {
				t.Fatalf("commissioner re-pin: %v", err)
			}
		}
		moved := futureScheduleWeek(week)
		for i := range moved.Games {
			moved.Games[i].Kickoff = moved.Games[i].Kickoff.Add(time.Hour)
		}
		return moved, nil
	}
	if _, err := refreshHostedNFLScheduleLeague(ctx, pins, signInStore(host.files), "native.hosted", 2026, racing); err == nil || !repinned {
		t.Fatalf("the timer re-pinned over a version the commissioner chose meanwhile (err=%v, fetched=%t)", err, repinned)
	}
	if got := pins.digest("native.hosted"); got != chosen {
		t.Fatalf("native.hosted pin = %q, want the commissioner's %q", got, chosen)
	}
}

func TestHostedNFLSchedulePinSwapServesOldOrNewStore(t *testing.T) {
	host, store, digest := scheduleRefreshFixture(t)
	ctx := t.Context()
	pins := newLeaguePins(host.files.leagues)
	directory, err := signInStore(host.files).Directory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := pins.selectPins(ctx, "", directory); err != nil {
		t.Fatal(err)
	}
	next, err := store.ReviseFootballSchedule(ctx, digest, nil, []leagueweek.Week{scheduleWeek(2, [2]string{"BUF", "MIA"})}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	readErr := make(chan error, 1)
	var readers sync.WaitGroup
	readers.Add(1)
	go func() {
		defer readers.Done()
		for {
			select {
			case <-stop:
				return
			default:
				pin := pins.pin(ctx, "native.hosted")
				if pin.Err != nil || (pin.Digest != digest && pin.Digest != next) || pin.Store == nil {
					select {
					case readErr <- fmt.Errorf("read during pin swap: %+v", pin):
					default:
					}
					return
				}
			}
		}
	}()
	if err := pins.setPin(ctx, signInStore(host.files), "native.hosted", next); err != nil {
		t.Fatal(err)
	}
	close(stop)
	readers.Wait()
	select {
	case err := <-readErr:
		t.Fatal(err)
	default:
	}
	pin := pins.pin(ctx, "native.hosted")
	if pin.Err != nil || pin.Digest != next {
		t.Fatalf("final pin = %+v; want %s", pin, next)
	}
}

func futureScheduleWeek(week int) leagueweek.Week {
	return leagueweek.Week{Number: week, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: time.Now().UTC().Add(30 * 24 * time.Hour)}}}
}

func publishTestSchedule(t *testing.T, store interface {
	SharedCommissionerRecords(context.Context, string) ([]commissioner.Record, error)
	ExecuteCommissioner(context.Context, commissioner.RawCommand, string, commissioner.Authorizer) (commissioner.Receipt, error)
}, league, principal string) {
	t.Helper()
	for i, command := range []string{commissioner.ScheduleGenerate, commissioner.SchedulePublish} {
		records, err := store.SharedCommissionerRecords(t.Context(), league)
		if err != nil {
			t.Fatal(err)
		}
		args := &commissioner.RawScheduleArgs{}
		if command == commissioner.ScheduleGenerate {
			teams := commissioner.LeagueFranchises(records)
			games := make([]commissioner.ScheduleGame, len(teams)/2)
			for game := range games {
				games[game] = commissioner.ScheduleGame{ID: fmt.Sprintf("week-1-game-%02d", game), Home: teams[game], Away: teams[len(teams)-1-game]}
			}
			args = &commissioner.RawScheduleArgs{Weeks: 1, Fixtures: []commissioner.ScheduleWeek{{Week: 1, Games: games}}}
		}
		raw := commissioner.RawCommand{League: league, Command: command, Expected: int64(len(records)),
			Args:    commissioner.RawArgs{Schedule: args},
			Request: fmt.Sprintf("timer schedule %s %d", league, i), Reason: "schedule refresh test",
			Effective: "now", Visibility: "league"}
		if command == commissioner.SchedulePublish {
			raw.Args.Schedule = &commissioner.RawScheduleArgs{}
		}
		if _, err := store.ExecuteCommissioner(t.Context(), raw, principal, nil); err != nil {
			t.Fatalf("%s for %s: %v", command, league, err)
		}
	}
}

type discardOutput struct{}

func (discardOutput) Write(p []byte) (int, error) { return len(p), nil }
