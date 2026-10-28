package db

import (
	"context"
	"database/sql"
	"fmt"
)

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
