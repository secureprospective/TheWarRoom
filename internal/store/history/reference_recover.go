package history

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// IdentityRecovery is what RecoverLeagueIdentities did: players put back into the reference file,
// and players it left out because an outside id there already names a different player.
type IdentityRecovery struct {
	Installed int
	Conflicts []string
}

// RecoverLeagueIdentities puts back into the reference file every player identity a league owns that
// the reference file lacks. Each league's roster restore point carries the permanent id and every
// outside id of each player it owns, rewritten in the same transaction as every league change, so a
// reference file older than the league file is completed from the league's own record, never by
// minting. A player whose outside id the reference file already gives to another player is reported
// and left out; nothing is merged. Running it again changes nothing.
func RecoverLeagueIdentities(ctx context.Context, league, reference *db.Pools) (IdentityRecovery, error) {
	var out IdentityRecovery
	identities, err := restorePointIdentities(ctx, league)
	if err != nil || len(identities) == 0 {
		return out, err
	}
	tx, err := reference.BeginWrite(ctx)
	if err != nil {
		return out, fmt.Errorf("history: begin player recovery: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var install []playerid.RawIdentity
	for _, identity := range identities {
		held, conflict, err := identityState(ctx, tx, identity)
		if err != nil {
			return out, err
		}
		switch {
		case held:
		case conflict != "":
			out.Conflicts = append(out.Conflicts, conflict)
		default:
			install = append(install, identity)
		}
	}
	if len(install) == 0 {
		return out, nil
	}
	if err := db.InstallRosterIdentities(ctx, tx, install); err != nil {
		return out, fmt.Errorf("history: put players back: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return out, fmt.Errorf("history: commit player recovery: %w", err)
	}
	out.Installed = len(install)
	return out, nil
}

// restorePointIdentities reads every identity the league file's validated roster restore points
// carry, one per permanent id, in id order.
func restorePointIdentities(ctx context.Context, league *db.Pools) ([]playerid.RawIdentity, error) {
	rows, err := league.Read().QueryContext(ctx, `SELECT payload FROM main.league_roster_restore ORDER BY league`)
	if err != nil {
		return nil, fmt.Errorf("history: read roster restore points: %w", err)
	}
	defer func() { _ = rows.Close() }()
	byID := map[string]playerid.RawIdentity{}
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, fmt.Errorf("history: read roster restore point: %w", err)
		}
		var point commissioner.RawRosterRestorePoint
		if err := json.Unmarshal([]byte(body), &point); err != nil {
			return nil, fmt.Errorf("history: decode roster restore point: %w", err)
		}
		if err := point.Validate(); err != nil {
			return nil, fmt.Errorf("history: roster restore point for %s: %w", point.League, err)
		}
		for _, identity := range point.Identities {
			byID[identity.Canonical] = identity
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history: read roster restore points: %w", err)
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	out := make([]playerid.RawIdentity, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out, nil
}

// identityState says whether the reference file already holds identity, or names the outside id it
// gives to a different player.
func identityState(ctx context.Context, tx *sql.Tx, identity playerid.RawIdentity) (held bool, conflict string, err error) {
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM canonical_player WHERE id=?)`, identity.Canonical).Scan(&held); err != nil {
		return false, "", fmt.Errorf("history: read reference players: %w", err)
	}
	if held {
		return true, "", nil
	}
	for _, link := range identity.Links {
		var other string
		err := tx.QueryRowContext(ctx, `SELECT player FROM player_external_id WHERE provider=? AND value=?`, link.Provider, link.Value).Scan(&other)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return false, "", fmt.Errorf("history: read reference player links: %w", err)
		}
		return false, fmt.Sprintf("%s (%s %s is already %s)", identity.Canonical, link.Provider, link.Value, other), nil
	}
	return false, "", nil
}

// ArchiveBody is one raw capture as the reference file keeps it (gzip), for PutArchiveBody on another
// server; held is false when the file does not have it.
func ArchiveBody(ctx context.Context, reference *db.Pools, key string) ([]byte, bool, error) {
	var stored []byte
	err := reference.Read().QueryRowContext(ctx, `SELECT body FROM raw_archive WHERE sha256 = ? AND size > 0`, key).Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("history: read raw capture %s: %w", key, err)
	}
	return stored, true, nil
}

// PutArchiveBody adds one raw capture to the reference file from its off-site evidence object; stored
// is the body exactly as the archive keeps it (gzip). It is refused unless it unpacks to content whose
// sha256 is key. A key already held is left as it is.
func PutArchiveBody(ctx context.Context, reference *db.Pools, key string, stored []byte) error {
	reader, err := gzip.NewReader(bytes.NewReader(stored))
	if err != nil {
		return fmt.Errorf("history: open raw capture %s: %w", key, err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("history: unpack raw capture %s: %w", key, err)
	}
	if footballDigest(body) != key {
		return fmt.Errorf("history: raw capture %s does not match its name", key)
	}
	if _, err := reference.Write().ExecContext(ctx, `INSERT OR IGNORE INTO raw_archive(sha256,size,body) VALUES(?,?,?)`, key, len(body), stored); err != nil {
		return fmt.Errorf("history: save raw capture %s: %w", key, err)
	}
	return nil
}
