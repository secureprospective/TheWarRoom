// Package personal owns the GM's private lists, separate from MFL's mirror.
package personal

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

type Entry struct {
	Kind   string `json:"kind"`
	Entity string `json:"entity"`
	ID     string `json:"id"`
	Text   string `json:"text"`
	At     string `json:"at"`
	Order  int    `json:"order"`
}

// Thread is one player's dated notes, newest first, with the total held.
type Thread struct {
	Notes []Entry `json:"notes"`
	Count int     `json:"count"`
}

type RawChange struct {
	Mine   string `json:"mine"`
	Kind   string `json:"kind"`
	Entity string `json:"entity"`
	ID     string `json:"id"`
	Text   string `json:"text"`
	Action string `json:"action"`
	// At identifies one dated note when removing it; notes are otherwise appended.
	At string `json:"at,omitempty"`
}

func (r RawChange) Validate() error {
	if r.Mine == "" || strings.IndexFunc(r.Mine, func(c rune) bool { return c < '0' || c > '9' }) >= 0 {
		return fmt.Errorf("personal: invalid franchise")
	}
	if !slices.Contains([]string{"watch", "bait", "note", "draft"}, r.Kind) {
		return fmt.Errorf("personal: invalid list")
	}
	if err := r.validateEntity(); err != nil {
		return err
	}
	if !r.validAction() {
		return fmt.Errorf("personal: invalid action")
	}
	if r.invalidText() {
		return fmt.Errorf("personal: note must be plain text, nonempty and at most 8000 bytes")
	}
	return nil
}

func (r RawChange) validAction() bool {
	if !slices.Contains([]string{"add", "remove", "up", "down"}, r.Action) {
		return false
	}
	if r.Kind == "note" {
		switch r.Action {
		case "add":
			return r.At == ""
		case "remove":
			return r.validNoteAt()
		default:
			return false
		}
	}
	if r.At != "" {
		return false
	}
	return r.Action != "up" && r.Action != "down" || r.Kind == "draft"
}

func (r RawChange) validNoteAt() bool {
	if r.At == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, r.At)
	return err == nil
}

func (r RawChange) invalidText() bool {
	return len(r.Text) > 8000 || strings.IndexByte(r.Text, 0) >= 0 ||
		r.Kind == "note" && r.Action == "add" && strings.TrimSpace(r.Text) == ""
}

func (r RawChange) validateEntity() error {
	if r.Entity == "player" {
		pid, err := playerid.New(r.ID)
		if err != nil {
			return fmt.Errorf("personal: player: %w", err)
		}
		if pid.String() != r.ID {
			return fmt.Errorf("personal: player id must be canonical")
		}
	} else if r.Entity != "franchise" || r.Kind != "note" || r.ID == "" ||
		strings.IndexFunc(r.ID, func(c rune) bool { return c < '0' || c > '9' }) >= 0 {
		return fmt.Errorf("personal: invalid entity")
	}
	return nil
}

type Store struct{ pools *db.Pools }

func New(pools *db.Pools) *Store { return &Store{pools: pools} }

func (s *Store) Initialize(ctx context.Context) error {
	if err := s.pools.InitializeMain(ctx); err != nil {
		return fmt.Errorf("personal: initialize schema: %w", err)
	}
	return nil
}

func (s *Store) Read(ctx context.Context, mine string) ([]Entry, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT kind,entity,entity_id,text,at,ordinal
 FROM personal_entries WHERE mine=? ORDER BY kind,ordinal,at,entity_id`, mine)
	if err != nil {
		return nil, fmt.Errorf("personal: read: %w", err)
	}
	defer func() { _ = rows.Close() }()
	entries := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.Kind, &e.Entity, &e.ID, &e.Text, &e.At, &e.Order); err != nil {
			return nil, fmt.Errorf("personal: scan: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("personal: rows: %w", err)
	}
	return entries, nil
}

// PlayerNotes returns one franchise's notes for a player, newest first.
func (s *Store) PlayerNotes(ctx context.Context, mine, playerID string) (Thread, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT kind,entity,entity_id,text,at,ordinal
 FROM personal_entries WHERE mine=? AND kind='note' AND entity='player' AND entity_id=?
 ORDER BY at DESC,ordinal DESC`, mine, playerID)
	if err != nil {
		return Thread{}, fmt.Errorf("personal: player notes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	notes := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.Kind, &e.Entity, &e.ID, &e.Text, &e.At, &e.Order); err != nil {
			return Thread{}, fmt.Errorf("personal: player note scan: %w", err)
		}
		notes = append(notes, e)
	}
	if err := rows.Err(); err != nil {
		return Thread{}, fmt.Errorf("personal: player note rows: %w", err)
	}
	return Thread{Notes: notes, Count: len(notes)}, nil
}

func (s *Store) Change(ctx context.Context, r RawChange, at time.Time) error {
	if err := r.Validate(); err != nil {
		return err
	}
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("personal: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	switch r.Action {
	case "add":
		_, err = tx.ExecContext(ctx, `INSERT INTO personal_entries(mine,kind,entity,entity_id,text,at,ordinal)
  SELECT ?,?,?,?,?,?,COALESCE(MAX(ordinal),0)+1 FROM personal_entries WHERE mine=? AND kind=?
  ON CONFLICT DO NOTHING`, r.Mine, r.Kind, r.Entity, r.ID, r.Text, at.UTC().Format(time.RFC3339Nano), r.Mine, r.Kind)
	case "remove":
		if r.Kind == "note" {
			_, err = tx.ExecContext(ctx, `DELETE FROM personal_entries
 WHERE mine=? AND kind='note' AND entity=? AND entity_id=? AND at=?`,
				r.Mine, r.Entity, r.ID, r.At)
		} else {
			_, err = tx.ExecContext(ctx, `DELETE FROM personal_entries WHERE mine=? AND kind=? AND entity=? AND entity_id=?`,
				r.Mine, r.Kind, r.Entity, r.ID)
		}
	default:
		err = reorder(ctx, tx, r)
	}
	if err != nil {
		return fmt.Errorf("personal: %s: %w", r.Action, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("personal: commit: %w", err)
	}
	return nil
}

func reorder(ctx context.Context, tx *sql.Tx, r RawChange) error {
	rows, err := tx.QueryContext(ctx, `SELECT entity_id FROM personal_entries WHERE mine=? AND kind='draft'
 ORDER BY ordinal,at,entity_id`, r.Mine)
	if err != nil {
		return fmt.Errorf("personal: order: %w", err)
	}
	defer func() { _ = rows.Close() }()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return fmt.Errorf("personal: order scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("personal: order rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("personal: order close: %w", err)
	}
	index := slices.Index(ids, r.ID)
	if index < 0 {
		return fmt.Errorf("personal: draft player is not listed")
	}
	next := index + 1
	if r.Action == "up" {
		next = index - 1
	}
	if next < 0 || next >= len(ids) {
		return nil
	}
	ids[index], ids[next] = ids[next], ids[index]
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE personal_entries SET ordinal=? WHERE mine=? AND kind='draft' AND entity_id=?`,
			i+1, r.Mine, id); err != nil {
			return fmt.Errorf("personal: reorder: %w", err)
		}
	}
	return nil
}
