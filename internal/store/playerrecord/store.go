// Package playerrecord stores the player record (internal/playerrecord) in the history file: one
// row per permanent player ID, written from the feeds and read with his production as a Book
// keyed by our ID. It shares the history file, not the history Store; the history migrations
// own its schema.
package playerrecord

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	record "github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// FeatureReader reads facts from history as of a date. history.Store satisfies it.
type FeatureReader interface {
	Features(ctx context.Context, q history.FeatureQuery) ([]history.Feature, error)
}

// Store holds the player record.
type Store struct {
	pools    *db.Pools
	features FeatureReader
}

// New uses the history-file migrations; it never independently creates schema.
func New(pools *db.Pools, features FeatureReader) *Store {
	return &Store{pools: pools, features: features}
}

// Write resolves every player's record from MFL's listing and history's facts as known at asOf,
// and stores them all in one transaction. A player's MFL link resolves to the permanent ID it
// already has; a link seen for the first time gets one, once. It returns how many it wrote.
func (s *Store) Write(ctx context.Context, listing record.Listing, asOf time.Time) (int, error) {
	obs, err := s.observations(ctx, []int{0}, asOf, model.RuntimeMeasures())
	if err != nil {
		return 0, err
	}
	known := model.Build(obs, nil).Players
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return 0, fmt.Errorf("playerrecord: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	records, err := record.Assemble(listing, known, func(mfl string) (string, error) {
		return db.EnsureMFLPlayer(ctx, tx, mfl)
	})
	if err != nil {
		return 0, fmt.Errorf("playerrecord: write: %w", err)
	}
	stamp := asOf.UTC().Format(time.RFC3339Nano)
	for _, r := range records {
		if err := upsert(ctx, tx, r, stamp); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("playerrecord: commit: %w", err)
	}
	return len(records), nil
}

func upsert(ctx context.Context, tx *sql.Tx, r record.Record, stamp string) error {
	sources, err := json.Marshal(r.Sources)
	if err != nil {
		return fmt.Errorf("playerrecord: encode sources: %w", err)
	}
	birth := ""
	if !r.Birth.IsZero() {
		birth = r.Birth.UTC().Format(time.RFC3339)
	}
	d := r.Draft
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_record(player,listed,name,position,birth_date,college,
 rookie_season,draft,draft_year,draft_round,draft_pick,draft_overall,draft_team,mislinked,sources,as_of)
 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
 ON CONFLICT(player) DO UPDATE SET listed=excluded.listed,name=excluded.name,position=excluded.position,
 birth_date=excluded.birth_date,college=excluded.college,rookie_season=excluded.rookie_season,draft=excluded.draft,
 draft_year=excluded.draft_year,draft_round=excluded.draft_round,draft_pick=excluded.draft_pick,
 draft_overall=excluded.draft_overall,draft_team=excluded.draft_team,mislinked=excluded.mislinked,
 sources=excluded.sources,as_of=excluded.as_of`,
		r.ID, r.Listed, r.Name, string(r.Position), birth, r.College, r.Rookie, string(d.Status), d.Year, d.Round, d.Pick,
		d.Overall, d.Team, r.Mislinked, string(sources), stamp); err != nil {
		return fmt.Errorf("playerrecord: save %s: %w", r.ID, err)
	}
	return nil
}

// Records reads every stored record with its MFL link.
func (s *Store) Records(ctx context.Context) ([]record.Record, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT r.player,COALESCE(e.value,''),r.listed,r.name,r.position,
 r.birth_date,r.college,r.rookie_season,r.draft,r.draft_year,r.draft_round,r.draft_pick,r.draft_overall,r.draft_team,
 r.mislinked,r.sources FROM player_record r LEFT JOIN player_external_id e ON e.player=r.player AND e.provider='mfl'
 ORDER BY r.player`)
	if err != nil {
		return nil, fmt.Errorf("playerrecord: read: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []record.Record
	for rows.Next() {
		var r record.Record
		var pos, birth, status, sources string
		if err := rows.Scan(&r.ID, &r.MFL, &r.Listed, &r.Name, &pos, &birth, &r.College, &r.Rookie, &status,
			&r.Draft.Year, &r.Draft.Round, &r.Draft.Pick, &r.Draft.Overall, &r.Draft.Team, &r.Mislinked, &sources); err != nil {
			return nil, fmt.Errorf("playerrecord: scan: %w", err)
		}
		r.Position, r.Draft.Status = domain.Position(pos), record.DraftStatus(status)
		if birth != "" {
			if r.Birth, err = time.Parse(time.RFC3339, birth); err != nil {
				return nil, fmt.Errorf("playerrecord: %s birth %q: %w", r.ID, birth, err)
			}
		}
		if err := json.Unmarshal([]byte(sources), &r.Sources); err != nil {
			return nil, fmt.Errorf("playerrecord: %s sources: %w", r.ID, err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("playerrecord: rows: %w", err)
	}
	return out, nil
}

// Book reads the stored records and the production q names, filed under our ID.
func (s *Store) Book(ctx context.Context, q record.Query) (record.Book, error) {
	records, err := s.Records(ctx)
	if err != nil {
		return record.Book{}, err
	}
	seasons := []int{0}
	for y := max(q.From, 1); y <= q.To; y++ {
		seasons = append(seasons, y)
	}
	obs, err := s.observations(ctx, seasons, q.AsOf, q.Measures)
	if err != nil {
		return record.Book{}, err
	}
	links, err := s.mflLinks(ctx)
	if err != nil {
		return record.Book{}, err
	}
	b, err := record.NewBook(records, obs, q.LastWeek, func(mfl string) (string, bool) {
		id, ok := links[mfl]
		return id, ok
	})
	if err != nil {
		return record.Book{}, fmt.Errorf("playerrecord: book: %w", err)
	}
	return b, nil
}

func (s *Store) observations(ctx context.Context, seasons []int, asOf time.Time, measures []string) ([]model.Obs, error) {
	var obs []model.Obs
	for _, season := range seasons {
		feats, err := s.features.Features(ctx, history.FeatureQuery{AsOf: asOf, Season: season, Measures: measures})
		if err != nil {
			return nil, fmt.Errorf("playerrecord: read season %d: %w", season, err)
		}
		for _, f := range feats {
			obs = append(obs, model.Obs{Player: f.PlayerID, Season: f.Season, Week: f.Week, Measure: f.Measure,
				Value: f.Value, Text: f.Text})
		}
	}
	return obs, nil
}

func (s *Store) mflLinks(ctx context.Context) (map[string]string, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT value,player FROM player_external_id WHERE provider='mfl'`)
	if err != nil {
		return nil, fmt.Errorf("playerrecord: read MFL links: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]string{}
	for rows.Next() {
		var mfl, id string
		if err := rows.Scan(&mfl, &id); err != nil {
			return nil, fmt.Errorf("playerrecord: scan MFL link: %w", err)
		}
		out[mfl] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("playerrecord: MFL links: %w", err)
	}
	return out, nil
}
