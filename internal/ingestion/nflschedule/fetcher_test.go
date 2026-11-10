package nflschedule

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

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

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
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

func TestParseByesRejectsMalformed(t *testing.T) {
	for _, body := range []string{
		`{"nflByeWeeks":{"team":[]}}`,
		`{"nflByeWeeks":{"team":{"id":"ARI","bye_week":""}}}`,
		`{"nflByeWeeks":{"team":{"id":"","bye_week":"7"}}}`,
		`{"error":{"$t":"bad"}}`,
	} {
		if _, err := ParseByes([]byte(body)); err == nil {
			t.Fatal("accepted", body)
		}
	}
	if _, err := Parse([]byte(strings.Replace(string(fixture(t, "nflSchedule-2025-w05.json")), `"score":"20"`, `"score":"x"`, 1))); err == nil {
		t.Fatal("accepted a non-numeric score")
	}
}
