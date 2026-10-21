package main

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type refreshFailureTransport struct {
	base                   moveFeedTransport
	changed, rosterFailure bool
}

func (f *refreshFailureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if f.rosterFailure && r.URL.Query().Get("TYPE") == "rosters" {
		return nil, errors.New("injected roster failure")
	}
	response, err := f.base.RoundTrip(r)
	if err != nil || !f.changed || r.URL.Query().Get("TYPE") != "league" {
		return response, err
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if err := response.Body.Close(); err != nil {
		return nil, err
	}
	return seasonResponse(strings.ReplaceAll(string(body), `"salaryCapAmount":"120"`, `"salaryCapAmount":"130"`)), nil
}

func TestRefreshFailureLeavesRulesAndMirrorTogether(t *testing.T) {
	for _, failure := range []string{"rosters", "mirror write", "rules write"} {
		t.Run(failure, func(t *testing.T) {
			a := targetTestApp(t)
			feed := &refreshFailureTransport{}
			var err error
			a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(feed))
			if err != nil {
				t.Fatal(err)
			}
			if err := a.core.Rulebook.Initialize(a.core.Ctx, discoverSource{app: a}); err != nil {
				t.Fatal(err)
			}
			before, at, used := a.core.League.ContractsView()
			version, err := a.core.Rulebook.ActiveVersion(a.core.Ctx)
			if err != nil {
				t.Fatal(err)
			}
			feed.changed = true
			if failure == "rosters" {
				feed.rosterFailure = true
			} else {
				table := "league_mirror"
				if failure == "rules write" {
					table = "rulebook_active"
				}
				// The table names are fixed fixture choices, never external input.
				query := `CREATE TRIGGER reject_refresh BEFORE UPDATE ON ` + table + ` BEGIN SELECT RAISE(ABORT, 'injected refresh write failure'); END`
				if _, err := a.core.Pools.Write().ExecContext(a.core.Ctx, query); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := a.refreshLeagueLocked(a.core.Ctx, a.core.Rulebook, a.core.League); err == nil {
				t.Fatal("refresh failure was ignored")
			}
			after, afterAt, afterUsed := a.core.League.ContractsView()
			persisted, err := a.core.Rulebook.ActiveVersion(a.core.Ctx)
			if err != nil {
				t.Fatal(err)
			}
			if a.core.Rulebook.GetSalaryCap() != "120" || persisted != version || !reflect.DeepEqual(before, after) || at != afterAt || !reflect.DeepEqual(used, afterUsed) {
				t.Errorf("failed refresh published or persisted a partial revision: cap %s, version %d", a.core.Rulebook.GetSalaryCap(), persisted)
			}
			feed.rosterFailure = false
			if failure != "rosters" {
				if _, err := a.core.Pools.Write().ExecContext(a.core.Ctx, `DROP TRIGGER reject_refresh`); err != nil {
					t.Fatal(err)
				}
			}
			result, err := a.refreshLeagueLocked(a.core.Ctx, a.core.Rulebook, a.core.League)
			if err != nil || !result.RulesChanged || !result.Changed || a.core.Rulebook.GetSalaryCap() != "130" {
				t.Fatalf("refresh retry = %+v, %v", result, err)
			}
		})
	}
}
