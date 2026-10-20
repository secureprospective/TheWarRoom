package db

import (
	"context"
	"database/sql"
	"fmt"
)

// MainVersion is the newest schema understood for thewarroom.db.
func MainVersion() int { return len(mainMigrations()) }

// InitializeMain adopts the complete main-file schema, never one store at a time.
func (p *Pools) InitializeMain(ctx context.Context) error {
	_, err := p.Migrate(ctx, mainMigrations()...)
	return err
}

func mainMigrations() []Migration { return []Migration{adoptMain} }

func adoptMain(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, mainRulebookDDL+mainParamsDDL+mainMovesDDL+mainBidFactsDDL+mainReleaseFactsDDL+mainContractFactsDDL+mainPersonalDDL+mainOffersDDL+mainMirrorDDL); err != nil {
		return fmt.Errorf("db: main schema: %w", err)
	}
	return nil
}

const mainRulebookDDL = `
CREATE TABLE IF NOT EXISTS rulebook_versions (
	version    INTEGER PRIMARY KEY AUTOINCREMENT,
	source     TEXT NOT NULL,
	payload    TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS rulebook_active (
	singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
	version   INTEGER NOT NULL REFERENCES rulebook_versions(version)
);
CREATE TABLE IF NOT EXISTS rulebook_overrides (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	scope      TEXT NOT NULL,
	rule_key   TEXT NOT NULL,
	value      TEXT NOT NULL,
	note       TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	UNIQUE (scope, rule_key)
);`

const mainParamsDDL = `
CREATE TABLE IF NOT EXISTS param_defaults (
	param_key     TEXT    NOT NULL,
	position      TEXT    NOT NULL DEFAULT '',
	value_type    TEXT    NOT NULL,
	default_val   TEXT    NOT NULL,
	min_val       TEXT    NOT NULL,
	max_val       TEXT    NOT NULL,
	is_calibrated INTEGER NOT NULL DEFAULT 0,
	description   TEXT    NOT NULL DEFAULT '',
	PRIMARY KEY (param_key, position)
);
CREATE TABLE IF NOT EXISTS param_overrides (
	param_key  TEXT NOT NULL,
	position   TEXT NOT NULL DEFAULT '',
	value      TEXT NOT NULL,
	note       TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL,
	PRIMARY KEY (param_key, position)
);`

const mainMovesDDL = `
CREATE TABLE IF NOT EXISTS move_envelopes (
 correlation_id TEXT PRIMARY KEY,
 created TEXT NOT NULL,
 spec TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS move_audit (
 correlation_id TEXT NOT NULL REFERENCES move_envelopes(correlation_id),
 sequence INTEGER NOT NULL CHECK (sequence >= 0),
 at TEXT NOT NULL,
 from_state TEXT NOT NULL,
 event TEXT NOT NULL,
 to_state TEXT NOT NULL,
 note TEXT NOT NULL,
 PRIMARY KEY (correlation_id, sequence)
);
CREATE TRIGGER IF NOT EXISTS move_envelopes_no_update
BEFORE UPDATE ON move_envelopes
BEGIN SELECT RAISE(ABORT, 'move_envelopes is append-only'); END;
CREATE TRIGGER IF NOT EXISTS move_envelopes_no_delete
BEFORE DELETE ON move_envelopes
BEGIN SELECT RAISE(ABORT, 'move_envelopes is append-only'); END;
CREATE TRIGGER IF NOT EXISTS move_audit_no_update
BEFORE UPDATE ON move_audit
BEGIN SELECT RAISE(ABORT, 'move_audit is append-only'); END;
CREATE TRIGGER IF NOT EXISTS move_audit_no_delete
BEFORE DELETE ON move_audit
BEGIN SELECT RAISE(ABORT, 'move_audit is append-only'); END;`

const mainPersonalDDL = `CREATE TABLE IF NOT EXISTS personal_entries (
 mine TEXT NOT NULL, kind TEXT NOT NULL, entity TEXT NOT NULL, entity_id TEXT NOT NULL,
 text TEXT NOT NULL, at TEXT NOT NULL, ordinal INTEGER NOT NULL,
 PRIMARY KEY (mine, kind, entity, entity_id, at));
 CREATE UNIQUE INDEX IF NOT EXISTS personal_marker ON personal_entries(mine,kind,entity,entity_id)
 WHERE kind != 'note';`

const mainBidFactsDDL = `
CREATE TABLE IF NOT EXISTS bid_facts (
 correlation_id TEXT PRIMARY KEY REFERENCES move_envelopes(correlation_id),
 player TEXT NOT NULL,
 salary_cents INTEGER NOT NULL,
 years INTEGER NOT NULL,
 points_milli INTEGER NOT NULL,
 snipe INTEGER NOT NULL CHECK (snipe IN (0, 1)),
 beat_milli INTEGER,
 evidence TEXT NOT NULL,
 closes TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS bid_facts_no_delete
BEFORE DELETE ON bid_facts
BEGIN SELECT RAISE(ABORT, 'bid_facts is append-only'); END;
CREATE TRIGGER IF NOT EXISTS bid_facts_frozen
BEFORE UPDATE ON bid_facts
WHEN (
 SELECT to_state FROM move_audit WHERE correlation_id = OLD.correlation_id
 ORDER BY sequence DESC LIMIT 1
) NOT IN ('draft', 'blocked', 'ready', 'stale')
BEGIN SELECT RAISE(ABORT, 'bid_facts is frozen once the bid is handed off'); END;
CREATE TABLE IF NOT EXISTS bid_conditions (
 correlation_id TEXT NOT NULL REFERENCES move_envelopes(correlation_id),
 sequence INTEGER NOT NULL,
 player TEXT NOT NULL,
 name TEXT NOT NULL,
 PRIMARY KEY (correlation_id, sequence)
);
CREATE TRIGGER IF NOT EXISTS bid_conditions_no_update
BEFORE UPDATE ON bid_conditions
BEGIN SELECT RAISE(ABORT, 'bid conditions are append-only'); END;
CREATE TRIGGER IF NOT EXISTS bid_conditions_no_delete
BEFORE DELETE ON bid_conditions
BEGIN SELECT RAISE(ABORT, 'bid conditions are append-only'); END;
CREATE TRIGGER IF NOT EXISTS bid_conditions_frozen
BEFORE INSERT ON bid_conditions
WHEN (SELECT to_state FROM move_audit WHERE correlation_id = NEW.correlation_id
 ORDER BY sequence DESC LIMIT 1) NOT IN ('draft', 'blocked', 'ready')
BEGIN SELECT RAISE(ABORT, 'bid conditions are frozen once handed off'); END;`

const mainReleaseFactsDDL = `
CREATE TABLE IF NOT EXISTS release_facts (
 correlation_id TEXT PRIMARY KEY REFERENCES move_envelopes(correlation_id),
 facts TEXT NOT NULL CHECK (json_valid(facts))
);
CREATE TRIGGER IF NOT EXISTS release_facts_no_delete
BEFORE DELETE ON release_facts
BEGIN SELECT RAISE(ABORT, 'release_facts cannot be deleted'); END;`

const mainContractFactsDDL = `
CREATE TABLE IF NOT EXISTS contract_facts (
 correlation_id TEXT PRIMARY KEY REFERENCES move_envelopes(correlation_id),
 facts TEXT NOT NULL CHECK(json_valid(facts))
);
CREATE TRIGGER IF NOT EXISTS contract_facts_no_delete
BEFORE DELETE ON contract_facts
BEGIN SELECT RAISE(ABORT, 'contract_facts cannot be deleted'); END;`

const mainOffersDDL = `CREATE TABLE IF NOT EXISTS personal_offers (
 mine TEXT NOT NULL, id TEXT NOT NULL, year INTEGER NOT NULL, ts INTEGER NOT NULL,
 sender TEXT NOT NULL, receiver TEXT NOT NULL, first_seen INTEGER NOT NULL,
 last_seen INTEGER NOT NULL, first_missed INTEGER NOT NULL DEFAULT 0,
 outcome TEXT NOT NULL, PRIMARY KEY(mine,year,id,ts));
 CREATE TABLE IF NOT EXISTS personal_offer_checks (mine TEXT PRIMARY KEY, checked INTEGER NOT NULL);`

const mainMirrorDDL = `
CREATE TABLE IF NOT EXISTS league_mirror (
	id           INTEGER PRIMARY KEY CHECK (id = 1),
	season       INTEGER NOT NULL,
	sha256       TEXT NOT NULL,
	snapshot     TEXT NOT NULL,
	refreshed_at TEXT NOT NULL
);`
