package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// The hosted server keeps sign-in data in its own file beside the league file: accounts, sessions,
// helper tokens, aliases, join links, the security audit chain and the league directory. One person
// can own teams in several leagues, so this file is server-wide. Its version is its user_version.

// NewServerFile is the store over the server's sign-in file.
func NewServerFile(pools *db.Pools) *Store { s := New(pools); s.server = true; return s }

// ServerVersion is the newest sign-in file version this build can read; db.Open refuses newer.
func ServerVersion() int { return len(serverMigrations()) }

// serverMigrations is append-only.
func serverMigrations() []db.Migration {
	return []db.Migration{createServerFile, addLeagueFootball, addSeatClaims}
}

// addLeagueFootball gives each league its own pinned football evidence digest; empty means the league
// uses the server's -football pin, as every league did before.
func addLeagueFootball(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `ALTER TABLE league_directory ADD COLUMN football TEXT NOT NULL DEFAULT '' CHECK(football = '' OR length(football) = 64)`); err != nil {
		return fmt.Errorf("accounts: league football pin: %w", err)
	}
	return nil
}

func createServerFile(ctx context.Context, tx *sql.Tx) error {
	for _, step := range []db.Migration{adoptAccounts, db.CreateAuditTables, db.CreateServerRelease, createDirectory} {
		if err := step(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

// PrepareServerFile brings the sign-in file's tables to this build without touching the audit chain,
// so a conversion can copy chain rows in before the chain is first verified.
func PrepareServerFile(ctx context.Context, pools *db.Pools) error {
	if _, err := pools.Migrate(ctx, serverMigrations()...); err != nil {
		return fmt.Errorf("accounts: sign-in file: %w", err)
	}
	return nil
}

// SignInTables are the sign-in tables, parents first. A conversion copies them in this order and
// drops them from the league file in the reverse order.
func SignInTables() []string {
	return []string{"accounts", "account_enrollments", "account_totp", "account_recovery_codes", "account_auth_generation",
		"account_security_audit", "account_auth_epoch", "account_sessions", "account_agent_tokens", "account_aliases", "league_join_links"}
}

// SecurityAuditSource is the security chain's name in audit_chain and audit_chain_head.
const SecurityAuditSource = securityAuditSource

// The league directory lists the leagues on this server: one row per league, written only after the
// league's own first commit. It is a listing, never the truth: the league file is authoritative for
// whether a league exists and for its state; a row here pointing at nothing makes no league appear.

// DirectoryEntry is one league as last recorded from its league file.
type DirectoryEntry struct {
	League  string
	State   string
	Created string
	// Football is the league's pinned football evidence digest; empty uses the server's pin.
	Football string
}

func createDirectory(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE TABLE league_directory (
 league TEXT PRIMARY KEY NOT NULL CHECK(length(league) > 0),
 state TEXT NOT NULL CHECK(length(state) > 0),
 created TEXT NOT NULL
) STRICT`)
	if err != nil {
		return fmt.Errorf("accounts: league directory schema: %w", err)
	}
	return nil
}

// RecordLeagues adds missing rows and refreshes the state of existing ones, in one write.
func (s *Store) RecordLeagues(ctx context.Context, leagues []DirectoryEntry) error {
	if len(leagues) == 0 {
		return nil
	}
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: record leagues: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, l := range leagues {
		if _, err := tx.ExecContext(ctx, `INSERT INTO league_directory(league, state, created) VALUES (?, ?, ?)
 ON CONFLICT(league) DO UPDATE SET state = excluded.state WHERE state != excluded.state`, l.League, l.State, l.Created); err != nil {
			return fmt.Errorf("accounts: record league: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: record leagues: %w", err)
	}
	return nil
}

// Directory lists the recorded leagues in league order.
func (s *Store) Directory(ctx context.Context) ([]DirectoryEntry, error) {
	rows, err := s.pools.Read().QueryContext(ctx, `SELECT league, state, created, football FROM league_directory ORDER BY league`)
	if err != nil {
		return nil, fmt.Errorf("accounts: read league directory: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []DirectoryEntry
	for rows.Next() {
		var e DirectoryEntry
		if err := rows.Scan(&e.League, &e.State, &e.Created, &e.Football); err != nil {
			return nil, fmt.Errorf("accounts: read league directory: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("accounts: read league directory: %w", err)
	}
	return out, nil
}

// ForgetLeague removes a deleted league's directory row and join link; accounts stay (they are
// server-wide). A league already gone from the directory is not an error.
func (s *Store) ForgetLeague(ctx context.Context, league string) error {
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: forget league: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, query := range []string{`DELETE FROM league_directory WHERE league = ?`, `DELETE FROM league_join_links WHERE league = ?`,
		`DELETE FROM league_seat_claims WHERE league = ?`} {
		if _, err := tx.ExecContext(ctx, query, league); err != nil {
			return fmt.Errorf("accounts: forget league: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: forget league: %w", err)
	}
	return nil
}

// PinFootball sets one league's pinned football evidence digest; empty returns it to the server's pin.
// It changes that league's row only. The league must already be in the directory.
func (s *Store) PinFootball(ctx context.Context, league, digest string) error {
	tx, err := s.pools.BeginWrite(ctx)
	if err != nil {
		return fmt.Errorf("accounts: pin football: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE league_directory SET football = ? WHERE league = ?`, digest, league)
	if err != nil {
		return fmt.Errorf("accounts: pin football: %w", err)
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		return errors.Join(fmt.Errorf("accounts: no league %q on this server", league), err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("accounts: pin football: %w", err)
	}
	return nil
}
