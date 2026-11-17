// Package personal owns the GM's private lists, separate from MFL's mirror.
package personal

import (
	"context"
	"database/sql"
	"errors"
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
	// With is the other watched player in a "swap": the screen ranks targets within a position
	// lane, so it names the lane neighbour and the two trade ranks.
	With string `json:"with,omitempty"`
}

func (r RawChange) Validate() error {
	if r.Mine == "" || strings.IndexFunc(r.Mine, func(c rune) bool { return c < '0' || c > '9' }) >= 0 {
		return fmt.Errorf("personal: invalid franchise")
	}
	return r.validateFields()
}

func (r RawChange) validateFields() error {
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
	if r.Action == "swap" {
		return r.Kind == "watch" && r.Entity == "player" && r.At == "" && r.validWith()
	}
	if r.With != "" || !slices.Contains([]string{"add", "remove", "up", "down"}, r.Action) {
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

func (r RawChange) validWith() bool {
	pid, err := playerid.New(r.With)
	return err == nil && pid.String() == r.With && r.With != r.ID
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

type Store struct {
	pools  *db.Pools
	league bool
}

func New(pools *db.Pools) *Store { return &Store{pools: pools} }

// NewLeague uses the hosted league's private_entries table. Its caller must supply the
// authenticated franchise ID to every franchise-scoped method.
func NewLeague(pools *db.Pools) *Store { return &Store{pools: pools, league: true} }

func (s *Store) Initialize(ctx context.Context) error {
	if s.league {
		return fmt.Errorf("personal: hosted schema is owned by history.MigrateLeague")
	}
	if err := s.pools.InitializeMain(ctx); err != nil {
		return fmt.Errorf("personal: initialize schema: %w", err)
	}
	return nil
}

func (s *Store) Read(ctx context.Context, mine string) ([]Entry, error) {
	if s.league {
		return nil, fmt.Errorf("personal: use ReadFranchise for hosted entries")
	}
	return s.read(ctx, mine)
}

// ReadFranchise reads private entries for the authenticated caller's franchise only.
func (s *Store) ReadFranchise(ctx context.Context, franchiseID string) ([]Entry, error) {
	if !s.league {
		return nil, fmt.Errorf("personal: franchise reads require a league store")
	}
	if err := validateFranchiseScope(franchiseID); err != nil {
		return nil, err
	}
	return s.read(ctx, franchiseID)
}

func (s *Store) read(ctx context.Context, owner string) ([]Entry, error) {
	query := `SELECT kind,entity,entity_id,text,at,ordinal FROM personal_entries WHERE mine=? ORDER BY kind,ordinal,at,entity_id`
	args := s.scopeArgs(owner)
	if s.league {
		query = `SELECT kind,entity,entity_id,text,at,ordinal FROM private_entries WHERE franchise_id=? AND mine=? ORDER BY kind,ordinal,at,entity_id`
	}
	rows, err := s.pools.Read().QueryContext(ctx, query, args...)
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
	if s.league {
		return Thread{}, fmt.Errorf("personal: use PlayerNotesForFranchise for hosted entries")
	}
	return s.playerNotes(ctx, mine, playerID)
}

// PlayerNotesForFranchise reads a private player-note thread for the caller's authenticated franchise.
func (s *Store) PlayerNotesForFranchise(ctx context.Context, franchiseID, playerID string) (Thread, error) {
	if !s.league {
		return Thread{}, fmt.Errorf("personal: franchise reads require a league store")
	}
	if err := validateFranchiseScope(franchiseID); err != nil {
		return Thread{}, err
	}
	return s.playerNotes(ctx, franchiseID, playerID)
}

func (s *Store) playerNotes(ctx context.Context, owner, playerID string) (Thread, error) {
	query := `SELECT kind,entity,entity_id,text,at,ordinal FROM personal_entries WHERE mine=? AND kind='note' AND entity='player' AND entity_id=? ORDER BY at DESC,ordinal DESC`
	args := s.scopeArgs(owner, playerID)
	if s.league {
		query = `SELECT kind,entity,entity_id,text,at,ordinal FROM private_entries WHERE franchise_id=? AND mine=? AND kind='note' AND entity='player' AND entity_id=? ORDER BY at DESC,ordinal DESC`
	}
	rows, err := s.pools.Read().QueryContext(ctx, query, args...)
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
	if s.league {
		return fmt.Errorf("personal: use ChangeForFranchise for hosted entries")
	}
	if err := r.Validate(); err != nil {
		return err
	}
	return s.change(ctx, r, at)
}

// ChangeForFranchise scopes writes to the caller's authenticated franchise. RawChange.Mine is
// deliberately overwritten; request data cannot select another franchise's entries.
func (s *Store) ChangeForFranchise(ctx context.Context, franchiseID string, r RawChange, at time.Time) error {
	if !s.league {
		return fmt.Errorf("personal: franchise writes require a league store")
	}
	if err := validateFranchiseScope(franchiseID); err != nil {
		return err
	}
	r.Mine = franchiseID
	if err := r.validateFields(); err != nil {
		return err
	}
	return s.change(ctx, r, at)
}

func validateFranchiseScope(franchiseID string) error {
	if franchiseID == "" || strings.TrimSpace(franchiseID) != franchiseID || len(franchiseID) > 128 || strings.IndexByte(franchiseID, 0) >= 0 {
		return fmt.Errorf("personal: invalid franchise")
	}
	return nil
}

func (s *Store) scopeArgs(owner string, extra ...any) []any {
	args := make([]any, 0, 2+len(extra))
	args = append(args, owner)
	if s.league {
		args = append(args, owner)
	}
	return append(args, extra...)
}

func (s *Store) change(ctx context.Context, r RawChange, at time.Time) error {
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("personal: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	switch r.Action {
	case "add":
		err = s.add(ctx, tx, r, at.UTC().Format(time.RFC3339Nano))
	case "remove":
		err = s.remove(ctx, tx, r)
	case "swap":
		err = s.swapWatch(ctx, tx, r)
	default:
		err = s.reorder(ctx, tx, r)
	}
	if err != nil {
		return fmt.Errorf("personal: %s: %w", r.Action, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("personal: commit: %w", err)
	}
	return nil
}

func (s *Store) add(ctx context.Context, tx *sql.Tx, r RawChange, at string) error {
	args := []any{r.Mine, r.Kind, r.Entity, r.ID, r.Text, at, r.Mine, r.Kind}
	query := `INSERT INTO personal_entries(mine,kind,entity,entity_id,text,at,ordinal)
SELECT ?,?,?,?,?,?,COALESCE(MAX(ordinal),0)+1 FROM personal_entries WHERE mine=? AND kind=?
ON CONFLICT DO NOTHING`
	if s.league {
		query = `INSERT INTO private_entries(franchise_id,mine,kind,entity,entity_id,text,at,ordinal)
SELECT ?,?,?,?,?,?,?,COALESCE(MAX(ordinal),0)+1 FROM private_entries WHERE franchise_id=? AND mine=? AND kind=?
ON CONFLICT DO NOTHING`
		args = []any{r.Mine, r.Mine, r.Kind, r.Entity, r.ID, r.Text, at, r.Mine, r.Mine, r.Kind}
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("personal: add: %w", err)
	}
	return nil
}

func (s *Store) remove(ctx context.Context, tx *sql.Tx, r RawChange) error {
	args := s.scopeArgs(r.Mine, r.Entity, r.ID, r.At)
	query := `DELETE FROM personal_entries WHERE mine=? AND kind='note' AND entity=? AND entity_id=? AND at=?`
	if r.Kind != "note" {
		args = s.scopeArgs(r.Mine, r.Kind, r.Entity, r.ID)
		query = `DELETE FROM personal_entries WHERE mine=? AND kind=? AND entity=? AND entity_id=?`
	}
	if s.league {
		args = s.scopeArgs(r.Mine, r.Entity, r.ID, r.At)
		query = `DELETE FROM private_entries WHERE franchise_id=? AND mine=? AND kind='note' AND entity=? AND entity_id=? AND at=?`
		if r.Kind != "note" {
			args = s.scopeArgs(r.Mine, r.Kind, r.Entity, r.ID)
			query = `DELETE FROM private_entries WHERE franchise_id=? AND mine=? AND kind=? AND entity=? AND entity_id=?`
		}
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("personal: remove: %w", err)
	}
	return nil
}

// swapWatch trades the ranks of two watched players; both must be on the owner's list.
func (s *Store) swapWatch(ctx context.Context, tx *sql.Tx, r RawChange) error {
	ordinal := func(id string) (int, error) {
		var n int
		query := `SELECT ordinal FROM personal_entries WHERE mine=? AND kind='watch' AND entity='player' AND entity_id=?`
		if s.league {
			query = `SELECT ordinal FROM private_entries WHERE franchise_id=? AND mine=? AND kind='watch' AND entity='player' AND entity_id=?`
		}
		err := tx.QueryRowContext(ctx, query, s.scopeArgs(r.Mine, id)...).Scan(&n)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("personal: watched player is not listed")
		}
		if err != nil {
			return 0, fmt.Errorf("personal: swap read: %w", err)
		}
		return n, nil
	}
	a, err := ordinal(r.ID)
	if err != nil {
		return err
	}
	b, err := ordinal(r.With)
	if err != nil {
		return err
	}
	for id, n := range map[string]int{r.ID: b, r.With: a} {
		query := `UPDATE personal_entries SET ordinal=? WHERE mine=? AND kind='watch' AND entity='player' AND entity_id=?`
		if s.league {
			query = `UPDATE private_entries SET ordinal=? WHERE franchise_id=? AND mine=? AND kind='watch' AND entity='player' AND entity_id=?`
		}
		args := append([]any{n}, s.scopeArgs(r.Mine, id)...)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("personal: swap: %w", err)
		}
	}
	return nil
}

func (s *Store) reorder(ctx context.Context, tx *sql.Tx, r RawChange) error {
	query := `SELECT entity_id FROM personal_entries WHERE mine=? AND kind='draft' ORDER BY ordinal,at,entity_id`
	if s.league {
		query = `SELECT entity_id FROM private_entries WHERE franchise_id=? AND mine=? AND kind='draft' ORDER BY ordinal,at,entity_id`
	}
	rows, err := tx.QueryContext(ctx, query, s.scopeArgs(r.Mine)...)
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
		query := `UPDATE personal_entries SET ordinal=? WHERE mine=? AND kind='draft' AND entity_id=?`
		if s.league {
			query = `UPDATE private_entries SET ordinal=? WHERE franchise_id=? AND mine=? AND kind='draft' AND entity_id=?`
		}
		args := append([]any{i + 1}, s.scopeArgs(r.Mine, id)...)
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("personal: reorder: %w", err)
		}
	}
	return nil
}
