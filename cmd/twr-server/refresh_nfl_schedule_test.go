package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestFetchMissingNFLWeeksOnlyRequestsUnheldPublishedWeeks(t *testing.T) {
	held := []leagueweek.Week{{Number: 2, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)}}}}
	var requested []int
	fetch := func(_ context.Context, season, week int) (leagueweek.Week, error) {
		if season != 2026 {
			t.Fatalf("requested season %d", season)
		}
		requested = append(requested, week)
		return scheduleWeek(week, [2]string{"DAL", "NYG"}), nil
	}
	got, err := fetchMissingNFLWeeks(context.Background(), 2026, []int{1, 2, 3}, held, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Number != 1 || got[1].Number != 3 {
		t.Fatalf("missing weeks = %+v", got)
	}
	if len(requested) != 2 || requested[0] != 1 || requested[1] != 3 {
		t.Fatalf("source requests = %v", requested)
	}
}

func TestFetchMissingNFLWeeksDoesNotReturnPartialRefresh(t *testing.T) {
	wantErr := errors.New("source unavailable")
	fetch := func(_ context.Context, _, week int) (leagueweek.Week, error) {
		if week == 2 {
			return leagueweek.Week{}, wantErr
		}
		return scheduleWeek(week, [2]string{"DAL", "NYG"}), nil
	}
	got, err := fetchMissingNFLWeeks(context.Background(), 2026, []int{1, 2, 3}, nil, fetch)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want source error", err)
	}
	if got != nil {
		t.Fatalf("partial additions escaped after failure: %+v", got)
	}
}

func TestFetchMissingNFLWeeksRejectsWrongSourceWeek(t *testing.T) {
	fetch := func(context.Context, int, int) (leagueweek.Week, error) {
		return scheduleWeek(4, [2]string{"DAL", "NYG"}), nil
	}
	if _, err := fetchMissingNFLWeeks(context.Background(), 2026, []int{3}, nil, fetch); err == nil {
		t.Fatal("wrong source week was accepted")
	}
}

func TestNFLWeekFetcherUsesValidatedArchivedMFLResponse(t *testing.T) {
	host, store, _ := scheduleRefreshFixture(t)
	before := referenceArchiveRows(t, host)
	kickoff := time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC).Unix()
	body := fmt.Sprintf(`{"nflSchedule":{"week":"2","matchup":{"kickoff":"%d","team":[{"id":"BUF","isHome":"1"},{"id":"MIA","isHome":"0"}]}}}`, kickoff)
	transport := scheduleResponseTransport(func(request *http.Request) (*http.Response, error) {
		if request.URL.Query().Get("TYPE") != "nflSchedule" || request.URL.Query().Get("W") != "2" {
			return nil, fmt.Errorf("unexpected MFL request %s", request.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	week, err := newNFLWeekFetcher(store, transport)(t.Context(), 2026, 2)
	if err != nil {
		t.Fatal(err)
	}
	if week.Number != 2 || len(week.Games) != 1 || week.Games[0].Teams != [2]string{"BUF", "MIA"} {
		t.Fatalf("NFL week = %+v", week)
	}
	fetches, err := store.SourceFetches(t.Context(), "mfl")
	if err != nil {
		t.Fatal(err)
	}
	if len(fetches) != 1 || fetches[0].SHA256 == "" {
		t.Fatalf("MFL fetches = %+v", fetches)
	}
	fetchURL, err := url.Parse(fetches[0].URL)
	if err != nil || fetchURL.Query().Get("TYPE") != "nflSchedule" || fetchURL.Query().Get("W") != "2" {
		t.Fatalf("archived MFL URL = %q (%v)", fetches[0].URL, err)
	}
	archived, err := store.Body(t.Context(), fetches[0].SHA256)
	if err != nil || string(archived) != body {
		t.Fatalf("archived source body = %q (%v)", archived, err)
	}
	if after := referenceArchiveRows(t, host); after != before+1 {
		t.Fatalf("source archive rows changed from %d to %d", before, after)
	}
}

type scheduleResponseTransport func(*http.Request) (*http.Response, error)

func (f scheduleResponseTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestRefreshNFLWeekEvidenceAddsDatesAndPinsOnlyThatLeague(t *testing.T) {
	host, store, digest := scheduleRefreshFixture(t)
	ctx := t.Context()
	beforeEvents := commissionerEvents(t, store)
	beforeReference := referenceArchiveRows(t, host)
	fetch := func(_ context.Context, season, week int) (leagueweek.Week, error) {
		if season != 2026 {
			t.Fatalf("requested season %d", season)
		}
		return scheduleWeek(week, [2]string{"BUF", "MIA"}), nil
	}
	var output bytes.Buffer
	if err := refreshNFLWeekEvidence(ctx, store, signInStore(host.files), "native.hosted", 2026, digest, []int{1, 2}, fetch, &output); err != nil {
		t.Fatal(err)
	}
	entries, err := signInStore(host.files).Directory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Football == "" || entries[0].Football == digest {
		t.Fatalf("league pin = %+v", entries)
	}
	old, err := store.LoadFootball(ctx, digest)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.LoadFootball(ctx, entries[0].Football)
	if err != nil {
		t.Fatal(err)
	}
	oldWeeks, err := old.ScheduleDates(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	newWeeks, err := updated.ScheduleDates(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(oldWeeks) != 1 || len(newWeeks) != 2 {
		t.Fatalf("old/new week counts = %d/%d", len(oldWeeks), len(newWeeks))
	}
	if after := commissionerEvents(t, store); after != beforeEvents {
		t.Fatalf("league events changed from %d to %d", beforeEvents, after)
	}
	if after := referenceArchiveRows(t, host); after != beforeReference+1 {
		t.Fatalf("reference archive rows changed from %d to %d", beforeReference, after)
	}
	if output.Len() == 0 {
		t.Fatal("refresh produced no operator report")
	}
}

func TestRefreshNFLWeekEvidenceLeavesPinAndArchiveUntouchedOnFetchFailure(t *testing.T) {
	host, store, digest := scheduleRefreshFixture(t)
	ctx := t.Context()
	beforeEvents := commissionerEvents(t, store)
	beforeReference := referenceArchiveRows(t, host)
	wantErr := errors.New("week 3 not published by source")
	fetch := func(_ context.Context, _, week int) (leagueweek.Week, error) {
		if week == 3 {
			return leagueweek.Week{}, wantErr
		}
		return scheduleWeek(week, [2]string{"BUF", "MIA"}), nil
	}
	if err := refreshNFLWeekEvidence(ctx, store, signInStore(host.files), "native.hosted", 2026, digest, []int{1, 2, 3}, fetch, &bytes.Buffer{}); !errors.Is(err, wantErr) {
		t.Fatalf("refresh error = %v", err)
	}
	entries, err := signInStore(host.files).Directory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Football != digest {
		t.Fatalf("failed refresh changed pin: %+v", entries)
	}
	if after := referenceArchiveRows(t, host); after != beforeReference {
		t.Fatalf("failed refresh wrote reference data: before=%d after=%d", beforeReference, after)
	}
	if after := commissionerEvents(t, store); after != beforeEvents {
		t.Fatalf("failed refresh changed league events: before=%d after=%d", beforeEvents, after)
	}
}

func scheduleRefreshFixture(t *testing.T) (*twoFileHost, *history.Store, string) {
	t.Helper()
	host := newTwoFileHost(t)
	ctx := t.Context()
	store, err := leagueStore(ctx, host.files, "native.hosted")
	if err != nil {
		t.Fatal(err)
	}
	if err := recordDirectory(ctx, store, signInStore(host.files)); err != nil {
		t.Fatal(err)
	}
	body, digest := footballEvidence(t, "Schedule refresh")
	if err := store.ImportFootball(ctx, body, digest); err != nil {
		t.Fatal(err)
	}
	if err := signInStore(host.files).PinFootball(ctx, "native.hosted", digest); err != nil {
		t.Fatal(err)
	}
	return host, store, digest
}

// footballEvidenceWithWeeks is footballEvidence holding the given schedule week instead of week 1.
func footballEvidenceWithWeeks(t *testing.T, name string, weeks ...leagueweek.Week) ([]byte, string) {
	t.Helper()
	raw := history.RawFootballEvidence{Version: 1, Season: 2026, Source: "reference move fixture", AsOf: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
		Players: map[string]history.RawFootballPlayer{"0099": {Identity: "reference.99", Facts: nativecheck.PlayerFacts{Name: name, Position: leaguerules.QB, Team: "DAL", DesignationHeld: true}}},
		Weeks:   weeks}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	return body, hex.EncodeToString(sum[:])
}

func scheduleWeek(number int, teams [2]string) leagueweek.Week {
	return leagueweek.Week{Number: number, Games: []leagueweek.Game{{Teams: teams, Kickoff: time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)}}}
}

func commissionerEvents(t *testing.T, store *history.Store) int {
	t.Helper()
	events, err := store.Events(t.Context(), "native.hosted", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	return len(events)
}

func referenceArchiveRows(t *testing.T, host *twoFileHost) int64 {
	t.Helper()
	proof, err := history.ProveReference(t.Context(), host.files.reference)
	if err != nil {
		t.Fatal(err)
	}
	return proof.Archive.Rows
}

func TestRefetchStaleNFLWeeksFetchesOnlyPastWeeksNotFinal(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	past := now.Add(-48 * time.Hour)
	future := now.Add(48 * time.Hour)
	held := []leagueweek.Week{
		{Number: 9, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: past}}},
		{Number: 8, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: past, Final: true}}},
		{Number: 10, Games: []leagueweek.Game{{Teams: [2]string{"KC", "LV"}, Kickoff: future}}},
	}
	var requested []int
	fetch := func(_ context.Context, _, week int) (leagueweek.Week, error) {
		requested = append(requested, week)
		return scheduleWeek(week, [2]string{"BUF", "MIA"}), nil
	}
	stale, _, err := refetchStaleNFLWeeks(t.Context(), 2026, now, held, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if len(requested) != 1 || requested[0] != 9 || len(stale) != 1 || stale[0] != 9 {
		t.Fatalf("requested = %v, stale = %v", requested, stale)
	}
}

func TestRefetchStaleNFLWeeksReturnsOnlyWeeksNowFinal(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	past := now.Add(-48 * time.Hour)
	held := []leagueweek.Week{
		{Number: 9, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: past}}},
		{Number: 8, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: past}}},
	}
	fetch := func(_ context.Context, _, week int) (leagueweek.Week, error) {
		if week == 8 {
			return leagueweek.Week{Number: 8, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: past}}}, nil
		}
		return leagueweek.Week{Number: 9, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: past, Final: true}}}, nil
	}
	stale, final, err := refetchStaleNFLWeeks(t.Context(), 2026, now, held, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 2 || len(final) != 1 || final[0].Number != 9 {
		t.Fatalf("stale = %v, final = %v", stale, weekNumbers(final))
	}
}

func TestRefreshNFLWeekEvidenceReplacesStaleWeekWhenFinal(t *testing.T) {
	host, store, _ := scheduleRefreshFixture(t)
	ctx := t.Context()
	pastKickoff := time.Now().UTC().Add(-48 * time.Hour)
	held := leagueweek.Week{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: pastKickoff}}}
	next := leagueweek.Week{Number: 2, Games: []leagueweek.Game{{Teams: [2]string{"KC", "LV"}, Kickoff: pastKickoff.Add(72 * time.Hour)}}}
	body, pinned := footballEvidenceWithWeeks(t, "Schedule refresh", held, next)
	if err := store.ImportFootball(ctx, body, pinned); err != nil {
		t.Fatal(err)
	}
	if err := signInStore(host.files).PinFootball(ctx, "native.hosted", pinned); err != nil {
		t.Fatal(err)
	}
	beforeEvents := commissionerEvents(t, store)
	fetch := func(_ context.Context, _, week int) (leagueweek.Week, error) {
		return leagueweek.Week{Number: week, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: pastKickoff, Final: true}}}, nil
	}
	var output bytes.Buffer
	if err := refreshNFLWeekEvidence(ctx, store, signInStore(host.files), "native.hosted", 2026, pinned, []int{1}, fetch, &output); err != nil {
		t.Fatal(err)
	}
	entries, err := signInStore(host.files).Directory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Football == pinned {
		t.Fatalf("league pin did not move: %+v", entries)
	}
	updated, err := store.LoadFootball(ctx, entries[0].Football)
	if err != nil {
		t.Fatal(err)
	}
	weeks, err := updated.ScheduleDates(ctx, 2026)
	if err != nil {
		t.Fatal(err)
	}
	if len(weeks) != 2 || !weeks[0].Games[0].Final {
		t.Fatalf("week 1 not final in new pin: %+v", weeks)
	}
	if lineup, ok := leagueweek.LineupWeek(pastKickoff.Add(96*time.Hour), weeks...); !ok || lineup.Number != 2 {
		t.Fatalf("lineup after week-2 games = %d, want 2", lineup.Number)
	}
	if after := commissionerEvents(t, store); after != beforeEvents {
		t.Fatalf("league events changed from %d to %d", beforeEvents, after)
	}
	if !strings.Contains(output.String(), "Updated results for 1 past week(s): 1.") {
		t.Fatalf("report = %q", output.String())
	}
}

func TestRefreshNFLWeekEvidenceKeepsPinWhenStaleWeekStillNotFinal(t *testing.T) {
	host, store, _ := scheduleRefreshFixture(t)
	ctx := t.Context()
	pastKickoff := time.Now().UTC().Add(-48 * time.Hour)
	held := leagueweek.Week{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: pastKickoff}}}
	body, pinned := footballEvidenceWithWeeks(t, "Schedule refresh", held)
	if err := store.ImportFootball(ctx, body, pinned); err != nil {
		t.Fatal(err)
	}
	if err := signInStore(host.files).PinFootball(ctx, "native.hosted", pinned); err != nil {
		t.Fatal(err)
	}
	fetch := func(_ context.Context, _, week int) (leagueweek.Week, error) {
		return leagueweek.Week{Number: week, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: pastKickoff}}}, nil
	}
	var output bytes.Buffer
	if err := refreshNFLWeekEvidence(ctx, store, signInStore(host.files), "native.hosted", 2026, pinned, []int{1}, fetch, &output); err != nil {
		t.Fatal(err)
	}
	entries, err := signInStore(host.files).Directory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Football != pinned {
		t.Fatalf("pin moved for a week still not final: %+v", entries)
	}
	if !strings.Contains(output.String(), "no changes made") {
		t.Fatalf("report = %q", output.String())
	}
}
