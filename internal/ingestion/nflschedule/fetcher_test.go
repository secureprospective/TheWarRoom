package nflschedule

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestRealWeeks(t *testing.T) {
	at := time.Date(2026, 10, 7, 13, 38, 0, 0, time.UTC)
	weeks := make([]leagueweek.Week, 0, 2)
	for _, tc := range []struct {
		file          string
		number, games int
		final         bool
	}{
		{"nflSchedule.json", 4, 16, true},
		{"nflSchedule-w5.json", 5, 15, false},
	} {
		raw, err := Parse(fixture(t, tc.file))
		if err != nil {
			t.Fatal(err)
		}
		w, err := ToWeek(raw, at)
		if err != nil {
			t.Fatal(err)
		}
		if w.Number != tc.number || len(w.Games) != tc.games {
			t.Fatalf("week mapping: %+v", w)
		}
		for _, g := range w.Games {
			if g.Final != tc.final || g.Kickoff.Location() != time.UTC {
				t.Fatalf("game mapping: %+v", g)
			}
		}
		weeks = append(weeks, w)
	}
	if _, ok := leagueweek.LineupWeek(at, weeks[0]); ok {
		t.Fatal("completed week must require another fetch")
	}
	w, ok := leagueweek.LineupWeek(at, weeks[1], weeks[0])
	if !ok || w.Number != 5 {
		t.Fatalf("lineup week: %+v, %v", w, ok)
	}
	want := time.Date(2026, 10, 9, 0, 15, 0, 0, time.UTC)
	if got := leagueweek.FirstLock(w); !got.Equal(want) {
		t.Fatalf("first lock = %v, want %v", got, want)
	}
	locks := leagueweek.Locks(w)
	if len(locks) != 30 || !locks["TBB"].Equal(want) || !locks["DAL"].Equal(want) {
		t.Fatal(locks)
	}
	if _, present := locks["CAR"]; present {
		t.Fatal("bye team CAR must have no lock")
	}
}

func TestParseRejectsBadData(t *testing.T) {
	for _, body := range []string{
		`{"error":{"$t":"API unavailable"}}`,
		`{"nflSchedule":{"week":"5","matchup":[]}}`,
		`{"nflSchedule":{"week":"five","matchup":[]}}`,
		`{"nflSchedule":{"week":"0","matchup":[]}}`,
		`{"nflSchedule":{"week":"5","matchup":{"kickoff":"bad","team":[{"id":"A"},{"id":"B"}]}}}`,
		`{"nflSchedule":{"week":"5","matchup":{"kickoff":"1","team":[{"id":"A"}]}}}`,
		`{"nflSchedule":{"week":"5","matchup":{"kickoff":"1","team":[{"id":"A"},{"id":"A"}]}}}`,
		`{"nflSchedule":{"week":"5","matchup":{"kickoff":"1","team":[{"id":""},{"id":"B"}]}}}`,
		`{`,
	} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestFinalityRequiresKickoff(t *testing.T) {
	raw, err := Parse(fixture(t, "nflSchedule-w5.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw.Matchups[0].GameSecondsRemaining = "0"
	kickoff := time.Unix(1791504900, 0).UTC()
	for _, tc := range []struct {
		at    time.Time
		final bool
	}{
		{kickoff.Add(-time.Second), false},
		{kickoff, true},
		{kickoff.Add(time.Second), true},
	} {
		w, err := ToWeek(raw, tc.at)
		if err != nil || w.Games[0].Final != tc.final {
			t.Fatalf("at %v: %+v, %v", tc.at, w, err)
		}
	}
	raw.Matchups[0].Kickoff = "bad"
	if _, err := ToWeek(raw, kickoff); err == nil {
		t.Fatal("converter accepted invalid raw input")
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestFetchUsesAPIHostWithoutLeague(t *testing.T) {
	for _, week := range []int{0, 5} {
		t.Run(strconv.Itoa(week), func(t *testing.T) {
			calls := 0
			rt := transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				q := r.URL.Query()
				wantWeek := ""
				if week != 0 {
					wantWeek = "5"
				}
				if r.URL.Host != "api.myfantasyleague.com" || r.URL.Path != "/2026/export" ||
					q.Get("TYPE") != Export || q.Get("JSON") != "1" || q.Has("L") || q.Get("W") != wantWeek {
					t.Fatalf("unexpected request: %s", r.URL)
				}
				if week == 0 && q.Has("W") {
					t.Fatal("default week must omit W")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(fixture(t, "nflSchedule-w5.json")))),
					Header:     make(http.Header),
				}, nil
			})
			c, err := mfl.New("www47", 100, mfl.WithTransport(rt))
			if err != nil {
				t.Fatal(err)
			}
			raw, err := Fetch(context.Background(), c, "2026", week)
			if err != nil || raw.Week != "5" || calls != 1 {
				t.Fatalf("fetch: %+v, %v, %d calls", raw, err, calls)
			}
		})
	}
}

func TestFetchFailures(t *testing.T) {
	failure := errors.New("transport failed")
	for _, tc := range []struct {
		name         string
		status       int
		body         string
		transportErr error
	}{
		{"status", http.StatusBadGateway, "", nil},
		{"API error", http.StatusOK, `{"error":{"$t":"unavailable"}}`, nil},
		{"transport", 0, "", failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := mfl.New("", 100, mfl.WithTransport(transportFunc(
				func(*http.Request) (*http.Response, error) {
					if tc.transportErr != nil {
						return nil, tc.transportErr
					}
					return &http.Response{
						StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)),
					}, nil
				},
			)))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Fetch(context.Background(), c, "2026", 5); err == nil {
				t.Fatal("fetch should fail")
			} else if tc.transportErr != nil && !errors.Is(err, failure) {
				t.Fatal("transport error lost", err)
			}
		})
	}
	if _, err := Fetch(context.Background(), nil, "2026", -1); err == nil {
		t.Fatal("negative week accepted")
	}
}
