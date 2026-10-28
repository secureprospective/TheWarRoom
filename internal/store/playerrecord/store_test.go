package playerrecord

import (
	"context"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	record "github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type listing map[string]normalize.PlayerFacts

func (l listing) Facts(id string) (normalize.PlayerFacts, bool) {
	f, ok := l[id]
	return f, ok
}
func (l listing) IDs() []string { return slices.Sorted(maps.Keys(l)) }

type features map[int][]history.Feature

func (f features) Features(_ context.Context, q history.FeatureQuery) ([]history.Feature, error) {
	var out []history.Feature
	for _, row := range f[q.Season] {
		if slices.Contains(q.Measures, row.Measure) {
			out = append(out, row)
		}
	}
	return out, nil
}

func open(t *testing.T) *db.Pools {
	t.Helper()
	p, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
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
	if err := db.CreatePlayerRecord(t.Context(), tx); err != nil {
		t.Fatal(err)
	}
	if err := db.AddPlayerCareer(t.Context(), tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return p
}

func feeds() (listing, features) {
	l := listing{
		"0001": {Name: "Veteran", Position: domain.PosDE, DraftYear: 2020, HasDraftYear: true, DraftRound: 1, DraftPick: 5,
			DraftTeam: "NYJ", College: "Ohio State"},
		"0002": {Name: "Rookie", Position: domain.PosWR, DraftYear: 2026, HasDraftYear: true, DraftRound: 7, DraftPick: 29,
			DraftTeam: "NEP", HasBirthdate: true, Birthdate: time.Date(2004, 4, 29, 4, 0, 0, 0, time.UTC).Unix()},
		"0003": {Name: "Undrafted", Position: domain.PosLB, DraftYear: 2026, HasDraftYear: true},
	}
	f := features{
		0: {
			{PlayerID: "0001", Measure: "context.nfl_position", Text: "LB"},
			{PlayerID: "0001", Measure: "context.birth_date", Text: "1999-02-03"},
			{PlayerID: "0001", Measure: "context.rookie_season", Value: 2020},
			{PlayerID: "0001", Measure: "prior.draft_pick", Value: 5},
			{PlayerID: "0004", Measure: "context.rookie_season", Value: 2026},
			{PlayerID: "0004", Measure: "prior.draft_pick", Value: 220},
		},
		2025: {
			{PlayerID: "0001", Season: 2025, Week: 1, Measure: "exposure.defense_snaps", Value: 50},
			{PlayerID: "0001", Season: 2025, Measure: "outcome.fantasy_points", Value: 12},
			{PlayerID: "0005", Season: 2025, Week: 1, Measure: "exposure.defense_snaps", Value: 50},
		},
	}
	l["0004"] = normalize.PlayerFacts{Name: "Placer", Position: domain.PosRB, DraftYear: 2026, HasDraftYear: true,
		DraftRound: 7, DraftPick: 4}
	return l, f
}

// A refresh never forks a second identity: a link already held keeps its ID, a new one gets one
// once, and writing again changes nothing but the facts.
func TestWriteKeepsOneIdentityPerPlayer(t *testing.T) {
	p := open(t)
	l, f := feeds()
	tx, err := p.BeginWrite(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	held, err := db.NewPlayerIdentity(t.Context(), tx, []playerid.RawOutsideID{{Provider: "mfl", Value: "0001"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	s := New(p, f)
	for range 2 {
		if n, err := s.Write(t.Context(), l, time.Now()); err != nil || n != 4 {
			t.Fatalf("wrote %d: %v", n, err)
		}
	}
	var players, rows int
	if err := p.Read().QueryRow(`SELECT (SELECT count(*) FROM canonical_player),(SELECT count(*) FROM player_record)`).Scan(&players, &rows); err != nil {
		t.Fatal(err)
	}
	if players != 4 || rows != 4 {
		t.Errorf("%d identities and %d records after two refreshes, want 4 and 4", players, rows)
	}
	got, err := s.Records(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	byMFL := map[string]record.Record{}
	for _, r := range got {
		byMFL[r.MFL] = r
	}
	if byMFL["0001"].ID != held {
		t.Errorf("the held link got a new identity: %s, want %s", byMFL["0001"].ID, held)
	}
	if d := byMFL["0002"].Draft; d.Status != record.Drafted || d.Overall != 217+29-1 || d.Team != "NEP" {
		t.Errorf("rookie draft = %+v", d)
	}
	if d := byMFL["0003"].Draft; d.Status != record.Undrafted {
		t.Errorf("undrafted = %+v", d)
	}
}

// The stored record reads back as the record the feeds resolve to, and the book a live run reads
// from it is the book an offline replay builds from the same held feeds.
func TestStoredBookIsTheHeldBook(t *testing.T) {
	p := open(t)
	l, f := feeds()
	s := New(p, f)
	if _, err := s.Write(t.Context(), l, time.Now()); err != nil {
		t.Fatal(err)
	}
	// 0005 has production but no identity facts, so no record: his facts still resolve through
	// his link, as history's evidence trigger guarantees every observed link has one.
	tx, err := p.BeginWrite(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.EnsureMFLPlayer(t.Context(), tx, "0005"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	q := record.Query{From: 2025, To: 2025, AsOf: time.Now(), Measures: model.RuntimeMeasures(), LastWeek: map[int]int{2025: 17}}
	stored, err := s.Book(t.Context(), q)
	if err != nil {
		t.Fatal(err)
	}
	var obs []model.Obs
	for _, season := range []int{0, 2025} {
		for _, r := range f[season] {
			obs = append(obs, model.Obs{Player: r.PlayerID, Season: r.Season, Week: r.Week, Measure: r.Measure, Value: r.Value, Text: r.Text})
		}
	}
	held, err := record.HeldBook(l, obs, q.LastWeek)
	if err != nil {
		t.Fatal(err)
	}
	for _, mfl := range []string{"0001", "0002", "0003", "0004"} {
		sid, ok1 := stored.ID(mfl)
		hid, ok2 := held.ID(mfl)
		if !ok1 || !ok2 {
			t.Fatalf("%s missing: stored %t held %t", mfl, ok1, ok2)
		}
		sp, hp := stored.Player(sid), held.Player(hid)
		sp.ID, hp.ID = "", ""
		if !sp.Birth.Equal(hp.Birth) {
			t.Errorf("%s birth %v != %v", mfl, sp.Birth, hp.Birth)
		}
		sp.Birth, hp.Birth = time.Time{}, time.Time{}
		if !reflect.DeepEqual(sp, hp) {
			t.Errorf("%s: stored %+v, held %+v", mfl, sp, hp)
		}
		if !reflect.DeepEqual(seasons(stored.Data.Seasons[sid]), seasons(held.Data.Seasons[hid])) {
			t.Errorf("%s seasons differ", mfl)
		}
	}
	if pl := stored.Player(mustID(t, stored, "0001")); pl.Position != domain.PosDE {
		t.Errorf("the league's DE reads as %s", pl.Position)
	}
	if len(stored.Data.Seasons) != len(held.Data.Seasons) {
		t.Errorf("stored book holds %d players' seasons, held %d", len(stored.Data.Seasons), len(held.Data.Seasons))
	}
}

// seasons is a player's seasons without the ID each carries (the two books mint different IDs).
func seasons(in map[int]*model.Season) map[int]model.Season {
	out := map[int]model.Season{}
	for y, s := range in {
		c := *s
		c.Player = ""
		out[y] = c
	}
	return out
}

func mustID(t *testing.T, b record.Book, mfl string) string {
	t.Helper()
	id, ok := b.ID(mfl)
	if !ok {
		t.Fatalf("no record for %s", mfl)
	}
	return id
}

func TestCareerCountsRoundTripAndRejectWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		count float64
		valid bool
	}{
		{"known zero", 0, true}, {"supplied counts", 5, true},
		{"negative", -1, false}, {"fractional", 1.5, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := open(t)
			l, f := feeds()
			s := New(p, f)
			if _, err := s.Write(t.Context(), l, time.Now()); err != nil {
				t.Fatal(err)
			}
			before, err := s.Records(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			for _, measure := range record.CareerMeasures() {
				f[0] = append(f[0], history.Feature{PlayerID: "0001", Measure: measure, Value: tc.count})
			}
			_, err = s.Write(t.Context(), l, time.Now())
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%t: %v", tc.valid, err)
			}
			after, err := s.Records(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if !tc.valid {
				if !reflect.DeepEqual(before, after) {
					t.Fatal("invalid career facts changed stored records")
				}
				return
			}
			for _, r := range after {
				if r.Career.Wins != nil {
					t.Fatal("invented career wins")
				}
				if r.MFL != "0001" {
					if r.Career.StartingSeasons != nil {
						t.Fatal("invented starts for an absent player")
					}
					continue
				}
				for _, count := range []*int{r.Career.StartingSeasons, r.Career.AllPro, r.Career.ProBowls} {
					if count == nil || *count != int(tc.count) {
						t.Fatalf("career: %+v", r.Career)
					}
				}
				for _, measure := range record.CareerMeasures() {
					if r.Sources[measure] != record.SourceNFLverse {
						t.Fatal("lost career provenance")
					}
				}
			}
		})
	}
}
