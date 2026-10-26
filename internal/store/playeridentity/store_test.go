package playeridentity

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestIndependentIdentitiesLinksAndConflictRollback(t *testing.T) {
	t.Parallel()
	p, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "identity.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = p.Close() }()
	tx, err := p.BeginWrite(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(t.Context(), `CREATE TABLE player_ids(id_type TEXT,id_value TEXT,player_id TEXT);
 CREATE TABLE observations(player_id TEXT);CREATE TABLE run_scores(mfl_id TEXT);
 CREATE TABLE model_scores(mfl_id TEXT);CREATE TABLE quarantine_model_scores(mfl_id TEXT);
 CREATE TABLE league_event(payload TEXT);`); err != nil {
		t.Fatal(err)
	}
	if err := db.CreatePlayerIdentity(t.Context(), tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	s := New(p)
	firstLinks := []playerid.RawOutsideID{{Provider: "gsis", Value: "00-0000001"}, {Provider: "sleeper", Value: "000123"}}
	first, err := s.Create(t.Context(), firstLinks)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Create(t.Context(), []playerid.RawOutsideID{{Provider: "espn", Value: "456"}})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("distinct people share a canonical ID")
	}
	for _, link := range firstLinks {
		got, err := s.Resolve(t.Context(), link.Provider, link.Value)
		if err != nil || got != first {
			t.Fatalf("outside link: %s %v", got, err)
		}
	}
	links := playerid.RawIdentity{Canonical: first, Links: []playerid.RawOutsideID{{Provider: "pfr", Value: "Person00"}, {Provider: "espn", Value: "456"}}}
	if err := s.Link(t.Context(), links); err == nil {
		t.Fatal("conflicting outside link reassigned")
	}
	if _, err := s.Resolve(t.Context(), "pfr", "Person00"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("failed batch wrote partial link: %v", err)
	}
	got, err := s.Resolve(t.Context(), "espn", "456")
	if err != nil || got != second {
		t.Fatal("conflict changed original owner")
	}
	links.Links = links.Links[:1]
	if err := s.Link(t.Context(), links); err != nil {
		t.Fatal(err)
	}
	if err := s.Link(t.Context(), links); err != nil {
		t.Fatal("same link was not idempotent", err)
	}
	for _, invalid := range []playerid.RawIdentity{{Canonical: "0099", Links: firstLinks}, {Canonical: first, Links: []playerid.RawOutsideID{{Provider: "mfl", Value: "99"}}}} {
		if err := s.Link(t.Context(), invalid); err == nil {
			t.Fatal("invalid raw identity accepted")
		}
	}
}
