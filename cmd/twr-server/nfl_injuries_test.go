package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type injuryRoundTripper func(*http.Request) (*http.Response, error)

type allowNFLStatusRead struct{}

func (allowNFLStatusRead) Admit(context.Context, commissioner.Access) error { return nil }

func (rt injuryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return rt(req) }

func newStoredNFLInjuryFeed(t *testing.T) nflInjuryFeed {
	t.Helper()
	base := newOneFileServer(t)
	files, err := openSignInFiles(t.Context(), base.path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := files.Close(); err != nil {
			t.Errorf("close injury fixture: %v", err)
		}
	})
	return newNFLInjuryFeed(files.leagues.family.Over(files.reference))
}

func TestRefreshNFLInjuriesIsAnOperatorSubcommand(t *testing.T) {
	handled, err := subcommand(t.Context(), "refresh-nfl-injuries", nil, io.Discard)
	if !handled || err == nil || !strings.Contains(err.Error(), "needs -db") {
		t.Fatalf("operator command dispatch: handled=%t err=%v", handled, err)
	}
}

func TestNFLInjuryRefreshStoresMemberReadFactsAndPreservesLastGoodOnFailure(t *testing.T) {
	feed := newStoredNFLInjuryFeed(t)
	season := nflSeasonAt(time.Now())
	body := `{"injuries":{"injury":[{"id":"0099","status":"Out","details":"Ankle - PCL","exp_return":"Oct 15, 2026"}]}}`
	base := injuryRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("TYPE") != "injuries" || req.URL.Query().Get("L") != "" || req.URL.Host != "api.myfantasyleague.com" {
			return nil, errors.New("injury export request did not use the MFL league-less API")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})
	var output strings.Builder
	if err := feed.Refresh(t.Context(), season, base, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Stored 1 NFL injury status fact(s)") {
		t.Fatalf("refresh confirmation: %q", output.String())
	}
	stored, err := feed.ReadNFLInjuryFeed(t.Context())
	if err != nil || !stored.Available || len(stored.Injuries) != 1 || stored.Injuries[0].ID.String() != "0099" ||
		stored.Injuries[0].Status != "Out" || stored.Injuries[0].Details != "Ankle - PCL" || stored.FetchedAt.IsZero() {
		t.Fatalf("fetched status was not held as a readable fact: %+v, %v", stored, err)
	}
	lastGood := stored.FetchedAt
	service := leagueservice.NewHosted(leagueservice.StoredHost{Store: feed.store,
		Context: func() context.Context { return t.Context() }, Principal: "member"}, allowNFLStatusRead{}).WithNFLInjuryFeed(feed)
	reading, err := service.TargetNFLPlayerStatus("native.hosted")
	if err != nil || reading.State != "available" || len(reading.Players) != 1 || reading.Players[0].Designation != "Out" ||
		reading.Players[0].BodyPart != "Ankle" || reading.FetchedAt == nil || !reading.FetchedAt.Equal(lastGood) {
		t.Fatalf("member service did not expose the fetched facts: %+v, %v", reading, err)
	}

	offline := injuryRoundTripper(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("upstream temporarily offline")
	})
	if err := feed.Refresh(t.Context(), season, offline, io.Discard); err == nil {
		t.Fatal("failed upstream fetch reported success")
	}
	after, err := feed.ReadNFLInjuryFeed(t.Context())
	if err != nil || !after.Available || len(after.Injuries) != 1 || after.Injuries[0] != stored.Injuries[0] ||
		!after.FetchedAt.Equal(lastGood) || !after.LastAttemptAt.After(lastGood) {
		t.Fatalf("failed refresh lost last good data or its chronology: before=%+v after=%+v err=%v", stored, after, err)
	}
	stale, err := service.TargetNFLPlayerStatus("native.hosted")
	if err != nil || stale.State != "stale" || stale.FetchedAt == nil || !stale.FetchedAt.Equal(lastGood) ||
		len(stale.Players) != 1 || stale.Players[0].Designation != "Out" {
		t.Fatalf("member read did not mark the preserved facts stale: %+v, %v", stale, err)
	}
}
