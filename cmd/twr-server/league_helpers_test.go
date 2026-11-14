package main

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// fixtureLeague is the league the hosted test fixtures create (hostedfixture.CreateLeague).
const fixtureLeague = "native.hosted"

// hostedLeague is the fixture league's own store, its file made on first use.
func hostedLeague(t *testing.T, files stateFiles) *history.Store {
	t.Helper()
	return leagueOn(t, files, fixtureLeague)
}

// leagueOn is league's own store on the test server, its file made on first use.
func leagueOn(t *testing.T, files stateFiles, league string) *history.Store {
	t.Helper()
	if store, ok := files.leagues.store(league); ok {
		return store
	}
	store, err := files.leagues.create(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	return store
}

// hostedPools is the fixture league's own file.
func hostedPools(t *testing.T, files stateFiles) *db.Pools {
	t.Helper()
	hostedLeague(t, files)
	pools, _ := files.leagues.pools(fixtureLeague)
	return pools
}
