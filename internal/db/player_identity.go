package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// CreatePlayerIdentity separates durable identity and ownership from fetched facts.
// Legacy evidence keeps its original MFL strings; player_ids is a read adapter.
func CreatePlayerIdentity(ctx context.Context, tx *sql.Tx) error {
	const ddl = `CREATE TABLE canonical_player (
 id TEXT PRIMARY KEY CHECK(length(id)=39 AND substr(id,1,7)='player.')
 ) STRICT;
 CREATE TRIGGER canonical_player_no_update BEFORE UPDATE ON canonical_player BEGIN SELECT RAISE(ABORT,'player identity is immutable'); END;
 CREATE TRIGGER canonical_player_no_delete BEFORE DELETE ON canonical_player BEGIN SELECT RAISE(ABORT,'player identity is permanent'); END;
 CREATE TABLE player_external_id (
 provider TEXT NOT NULL CHECK(length(provider)>0),
 value TEXT NOT NULL CHECK(length(value)>0),
 player TEXT NOT NULL REFERENCES canonical_player(id),
 PRIMARY KEY(provider,value)
 ) STRICT;
 CREATE INDEX external_ids_by_player ON player_external_id(player);
 CREATE UNIQUE INDEX one_mfl_id_per_player ON player_external_id(player) WHERE provider='mfl';
 CREATE TABLE player_ownership (
 league TEXT NOT NULL CHECK(length(league)>0),
 side TEXT NOT NULL,
 player TEXT NOT NULL REFERENCES canonical_player(id),
 franchise TEXT NOT NULL CHECK(length(franchise)>0),
 PRIMARY KEY(league,side,player)
 ) STRICT;
 CREATE TEMP TABLE migration_player_ids(mfl TEXT PRIMARY KEY, player TEXT NOT NULL);
 INSERT INTO migration_player_ids
 SELECT mfl,'player.'||lower(hex(randomblob(16))) FROM (
 SELECT player_id AS mfl FROM player_ids UNION SELECT player_id FROM observations
 UNION SELECT mfl_id FROM run_scores UNION SELECT mfl_id FROM model_scores
 UNION SELECT mfl_id FROM quarantine_model_scores
 UNION SELECT CAST(j.atom AS TEXT) FROM league_event e,json_tree(e.payload) j
 WHERE j.type='text' AND (j.key IN ('player','playerId') OR j.fullkey LIKE '%.players[%].id') AND j.atom<>''
 );
 INSERT INTO canonical_player SELECT player FROM migration_player_ids;
 INSERT INTO player_external_id SELECT 'mfl',mfl,player FROM migration_player_ids;
 INSERT INTO player_external_id SELECT p.id_type,p.id_value,m.player FROM player_ids p JOIN migration_player_ids m ON m.mfl=p.player_id
 WHERE p.id_type<>'mfl';
 DROP TABLE player_ids;
 DROP TABLE migration_player_ids;
 CREATE VIEW player_ids AS SELECT e.provider AS id_type,e.value AS id_value,m.value AS player_id
 FROM player_external_id e JOIN player_external_id m ON m.player=e.player AND m.provider='mfl'
 WHERE e.provider<>'mfl';`
	if _, err := tx.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate canonical player identities: %w", err)
	}
	return nil
}

// EnsureMFLPlayer is the compatibility boundary: an MFL value resolves to an
// opaque permanent identity. It never normalizes or strips provider bytes.
func EnsureMFLPlayer(ctx context.Context, tx *sql.Tx, mfl string) (string, error) {
	if mfl == "" {
		return "", fmt.Errorf("empty MFL player link")
	}
	var id string
	err := tx.QueryRowContext(ctx, `SELECT player FROM player_external_id WHERE provider='mfl' AND value=?`, mfl).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("resolve MFL player: %w", err)
	}
	return NewPlayerIdentity(ctx, tx, []playerid.RawOutsideID{{Provider: "mfl", Value: mfl}})
}

// LinkPlayerExternalID keeps ingestion's explicit correction behavior. MFL links
// themselves cannot be reassigned: they anchor the legacy evidence adapter.
func LinkPlayerExternalID(ctx context.Context, tx *sql.Tx, provider, value, mfl string) error {
	if provider == "" || provider == "mfl" || value == "" {
		return fmt.Errorf("invalid outside player link")
	}
	id, err := EnsureMFLPlayer(ctx, tx, mfl)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_external_id(provider,value,player) VALUES(?,?,?) ON CONFLICT(provider,value) DO UPDATE SET player=excluded.player`, provider, value, id); err != nil {
		return fmt.Errorf("link outside player identity: %w", err)
	}
	return nil
}

// RosterOwner is the stored ownership key. Empty side is the single pool; a league with a
// player pool per side owns each player once per side. The caller names the side.
type RosterOwner struct{ MFL, Side, Franchise string }

// ReplaceRosterOwnership replaces every side's ownership of the league in the caller's transaction, so
// both sides are saved together or not at all.
func ReplaceRosterOwnership(ctx context.Context, tx *sql.Tx, league string, owners []RosterOwner) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_ownership WHERE league=?`, league); err != nil {
		return fmt.Errorf("replace roster ownership: %w", err)
	}
	for _, owner := range owners {
		id, err := EnsureMFLPlayer(ctx, tx, owner.MFL)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO player_ownership(league,side,player,franchise) VALUES(?,?,?,?)`, league, owner.Side, id, owner.Franchise); err != nil {
			return fmt.Errorf("save roster ownership: %w", err)
		}
	}
	return nil
}

// GuardPlayerEvidence maintains links for new legacy-shaped fact and score rows.
// Evidence bodies remain byte-for-byte unchanged.
func GuardPlayerEvidence(ctx context.Context, tx *sql.Tx) error {
	for _, column := range []struct{ table, name string }{{"observations", "player_id"}, {"run_scores", "mfl_id"}, {"model_scores", "mfl_id"}} {
		if !sqlIdentifier(column.table) || !sqlIdentifier(column.name) {
			return fmt.Errorf("invalid player identity guard identifier")
		}
		//nolint:gosec // SQLite cannot bind DDL identifiers; these checked names are fixed migration constants.
		ddl := fmt.Sprintf(`CREATE TRIGGER %s_player_identity AFTER INSERT ON %s BEGIN
 INSERT INTO canonical_player(id) SELECT 'player.'||lower(hex(randomblob(16)))
 WHERE NOT EXISTS(SELECT 1 FROM player_external_id WHERE provider='mfl' AND value=NEW.%s);
 INSERT INTO player_external_id(provider,value,player)
 SELECT 'mfl',NEW.%s,id FROM canonical_player WHERE rowid=last_insert_rowid()
 AND NOT EXISTS(SELECT 1 FROM player_external_id WHERE provider='mfl' AND value=NEW.%s);
 END;`, column.table, column.table, column.name, column.name, column.name)
		if _, err := tx.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("guard evidence player identity: %w", err)
		}
	}
	return nil
}

// RosterIdentities is a transaction-consistent reading for a private restore point.
func RosterIdentities(ctx context.Context, tx *sql.Tx, league string) ([]playerid.RawIdentity, error) {
	// A player owned on two sides is one identity: his links are read once.
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT o.player,e.provider,e.value FROM player_ownership o JOIN player_external_id e ON e.player=o.player WHERE o.league=? ORDER BY o.player,e.provider,e.value`, league)
	if err != nil {
		return nil, fmt.Errorf("read roster identities: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanPlayerIdentities(rows)
}

func InstallRosterIdentities(ctx context.Context, tx *sql.Tx, identities []playerid.RawIdentity) error {
	for _, identity := range identities {
		if err := identity.Validate(); err != nil {
			return fmt.Errorf("validate restored identity: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO canonical_player(id) VALUES(?)`, identity.Canonical); err != nil {
			return fmt.Errorf("restore canonical player: %w", err)
		}
		for _, link := range identity.Links {
			if _, err := tx.ExecContext(ctx, `INSERT INTO player_external_id(provider,value,player) VALUES(?,?,?)`, link.Provider, link.Value, identity.Canonical); err != nil {
				return fmt.Errorf("restore outside player link: %w", err)
			}
		}
	}
	return nil
}

func NewPlayerIdentity(ctx context.Context, tx *sql.Tx, links []playerid.RawOutsideID) (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate player identity: %w", err)
	}
	identity := playerid.RawIdentity{Canonical: "player." + hex.EncodeToString(random[:]), Links: links}
	if err := InstallRosterIdentities(ctx, tx, []playerid.RawIdentity{identity}); err != nil {
		return "", err
	}
	return identity.Canonical, nil
}

// LeagueIdentities includes historical players as well as the current roster;
// moving a league must not assign a new identity to a player it later reacquires.
func LeagueIdentities(ctx context.Context, tx *sql.Tx, league string) ([]playerid.RawIdentity, error) {
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT e.player,e.provider,e.value FROM player_external_id e
 JOIN player_external_id m ON m.player=e.player AND m.provider='mfl'
 WHERE m.value IN (
 SELECT CAST(j.atom AS TEXT) FROM league_event l,json_tree(l.payload) j WHERE l.league=?
 AND j.type='text' AND (j.key IN ('player','playerId') OR j.fullkey LIKE '%.players[%].id') AND j.atom<>''
 ) ORDER BY e.player,e.provider,e.value`, league)
	if err != nil {
		return nil, fmt.Errorf("read league player identities: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanPlayerIdentities(rows)
}

// MergePlayerIdentities accepts the same immutable links on a reverse transfer,
// but never guesses that conflicting permanent identities are the same person.
func MergePlayerIdentities(ctx context.Context, tx *sql.Tx, identities []playerid.RawIdentity) error {
	for _, identity := range identities {
		if err := identity.Validate(); err != nil {
			return fmt.Errorf("validate transferred player identity: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO canonical_player(id) VALUES(?) ON CONFLICT(id) DO NOTHING`, identity.Canonical); err != nil {
			return fmt.Errorf("receive canonical player: %w", err)
		}
		for _, link := range identity.Links {
			if _, err := tx.ExecContext(ctx, `INSERT INTO player_external_id(provider,value,player) VALUES(?,?,?) ON CONFLICT(provider,value) DO NOTHING`, link.Provider, link.Value, identity.Canonical); err != nil {
				return fmt.Errorf("receive outside player link: %w", err)
			}
			var held string
			if err := tx.QueryRowContext(ctx, `SELECT player FROM player_external_id WHERE provider=? AND value=?`, link.Provider, link.Value).Scan(&held); err != nil {
				return fmt.Errorf("check transferred player link: %w", err)
			}
			if held != identity.Canonical {
				return fmt.Errorf("destination holds a conflicting permanent player identity; use a fresh destination")
			}
		}
	}
	return nil
}

func scanPlayerIdentities(rows *sql.Rows) ([]playerid.RawIdentity, error) {
	var out []playerid.RawIdentity
	for rows.Next() {
		var id string
		var link playerid.RawOutsideID
		if err := rows.Scan(&id, &link.Provider, &link.Value); err != nil {
			return nil, fmt.Errorf("read player link: %w", err)
		}
		if len(out) == 0 || out[len(out)-1].Canonical != id {
			out = append(out, playerid.RawIdentity{Canonical: id})
		}
		out[len(out)-1].Links = append(out[len(out)-1].Links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read roster identities: %w", err)
	}
	return out, nil
}

// CreatePlayerRecord adds the player record: one row per permanent player holding his NFL draft
// capital and bio, each field's winning source beside it. Outside IDs stay in player_external_id
// and production in observations; the record is how both are read by our ID.
func CreatePlayerRecord(ctx context.Context, tx *sql.Tx) error {
	const ddl = `CREATE TABLE player_record (
 player TEXT PRIMARY KEY REFERENCES canonical_player(id),
 listed INTEGER NOT NULL CHECK(listed IN (0,1)),
 name TEXT NOT NULL,
 position TEXT NOT NULL,
 birth_date TEXT NOT NULL,
 college TEXT NOT NULL,
 rookie_season INTEGER NOT NULL CHECK(rookie_season>=0),
 draft TEXT NOT NULL CHECK(draft IN ('drafted','undrafted','unknown')),
 draft_year INTEGER NOT NULL CHECK(draft_year>=0),
 draft_round INTEGER NOT NULL CHECK(draft_round>=0),
 draft_pick INTEGER NOT NULL CHECK(draft_pick>=0),
 draft_overall INTEGER NOT NULL CHECK(draft_overall>=0),
 draft_team TEXT NOT NULL,
 mislinked INTEGER NOT NULL CHECK(mislinked IN (0,1)),
 sources TEXT NOT NULL CHECK(json_valid(sources)),
 as_of TEXT NOT NULL
 ) STRICT;`
	if _, err := tx.ExecContext(ctx, ddl); err != nil {
		return fmt.Errorf("migrate player record: %w", err)
	}
	return nil
}

// AddPlayerCareer preserves existing records; missing career facts stay explicitly unknown.
func AddPlayerCareer(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `ALTER TABLE player_record ADD COLUMN career TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(career))`); err != nil {
		return fmt.Errorf("migrate player career: %w", err)
	}
	return nil
}
