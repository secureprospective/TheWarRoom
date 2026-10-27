package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// RawTransferEvent preserves SQL text exactly; importing must not re-execute its command.
type RawTransferEvent struct {
	ID, League, Stream                                    string
	Revision                                              int64
	EffectiveAt, RecordedAt                               string
	Actor, Scope, Kind, Payload, InputHash, CorrelationID string
	Supersedes                                            *string
}

func (r RawTransferEvent) event() (LeagueEvent, error) {
	e := LeagueEvent{ID: r.ID, League: r.League, Stream: r.Stream, Revision: r.Revision, Actor: r.Actor, Scope: r.Scope, Kind: r.Kind, InputHash: r.InputHash, CorrelationID: r.CorrelationID}
	var err error
	if e.EffectiveAt, err = parseEventTime(r.EffectiveAt); err != nil {
		return e, err
	}
	if e.RecordedAt, err = parseEventTime(r.RecordedAt); err != nil {
		return e, err
	}
	if r.Supersedes != nil {
		e.Supersedes = *r.Supersedes
	}
	if err := json.Unmarshal([]byte(r.Payload), &e.Payload); err != nil {
		return e, fmt.Errorf("read exported change: %w", err)
	}
	return e, e.Validate()
}

// Validate rejects broken stream order and mismatched receipts without upcasting durable bytes.
func (r RawTransferEvent) Validate() error {
	e, err := r.event()
	if err != nil {
		return err
	}
	if e.Revision < 1 || e.RecordedAt.IsZero() {
		return fmt.Errorf("exported change has no revision or recording time")
	}
	if e.Stream == commissioner.Stream {
		_, err = commissionerRecords([]LeagueEvent{e})
		if err != nil {
			return err
		}
		record := e.Payload.Commissioner
		hash := sha256.Sum256([]byte(record.Canonical))
		if hex.EncodeToString(hash[:]) != e.InputHash || e.Kind != record.Command.Command {
			return fmt.Errorf("exported change does not match its saved request")
		}
	}
	return err
}

// TransferRecords decodes a validated stream into disposable replay state only.
func TransferRecords(rows []RawTransferEvent) ([]commissioner.Record, error) {
	events := make([]LeagueEvent, 0, len(rows))
	for _, row := range rows {
		if err := row.Validate(); err != nil {
			return nil, err
		}
		if row.Stream != commissioner.Stream {
			continue
		}
		e, err := row.event()
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return commissionerRecords(events)
}

// CaptureTransfer reads audit, original event text and selected self-contained evidence from one
// SQLite snapshot. It never copies schema/authority markers, credentials or unrelated archives.
type TransferCapture struct {
	Events     []RawTransferEvent
	Blobs      map[string][]byte
	Checkpoint db.AuditHead
	Identities []playerid.RawIdentity
}

func (s *Store) CaptureTransfer(ctx context.Context, league string, digests []string) ([]RawTransferEvent, map[string][]byte, db.AuditHead, error) {
	captured, err := s.CaptureTransferSnapshot(ctx, league, digests)
	return captured.Events, captured.Blobs, captured.Checkpoint, err
}

func (s *Store) CaptureTransferSnapshot(ctx context.Context, league string, digests []string) (TransferCapture, error) {
	var out TransferCapture
	tx, err := s.pools.Read().BeginTx(ctx, nil)
	if err != nil {
		return out, fmt.Errorf("start league export: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	out.Events, out.Blobs, out.Checkpoint, err = captureTransferRows(ctx, tx, league, digests)
	if err != nil {
		return out, err
	}
	out.Identities, err = db.LeagueIdentities(ctx, tx, league)
	if err != nil {
		return out, fmt.Errorf("capture league player identities: %w", err)
	}
	return out, nil
}

func captureTransferRows(ctx context.Context, tx *sql.Tx, league string, digests []string) ([]RawTransferEvent, map[string][]byte, db.AuditHead, error) {
	head, err := db.VerifyAudit(ctx, tx, ledgerAuditSource, ledgerAuditQuery)
	if err != nil {
		return nil, nil, head, fmt.Errorf("check league history before exporting: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,league,stream,revision,effective_at,recorded_at,actor,scope,kind,payload,input_hash,correlation_id,supersedes FROM league_event WHERE league=? ORDER BY stream,revision`, league)
	if err != nil {
		return nil, nil, head, fmt.Errorf("read league for export: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []RawTransferEvent{}
	for rows.Next() {
		var r RawTransferEvent
		if err := rows.Scan(&r.ID, &r.League, &r.Stream, &r.Revision, &r.EffectiveAt, &r.RecordedAt, &r.Actor, &r.Scope, &r.Kind, &r.Payload, &r.InputHash, &r.CorrelationID, &r.Supersedes); err != nil {
			return nil, nil, head, fmt.Errorf("read exported change: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, head, fmt.Errorf("read exported changes: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, nil, head, fmt.Errorf("finish exported changes: %w", err)
	}
	blobs := make(map[string][]byte, len(digests))
	for _, digest := range digests {
		var compressed []byte
		var size int64
		if err := tx.QueryRowContext(ctx, `SELECT size,body FROM raw_archive WHERE sha256=?`, digest).Scan(&size, &compressed); err != nil {
			return nil, nil, head, fmt.Errorf("the selected football evidence is missing: %w", err)
		}
		body, err := archive.Gunzip(compressed, size)
		if err != nil {
			return nil, nil, head, fmt.Errorf("read selected football evidence: %w", err)
		}
		if footballDigest(body) != digest {
			return nil, nil, head, fmt.Errorf("the selected football evidence has changed")
		}
		blobs[digest] = body
	}
	return out, blobs, head, nil
}

// TransferFootball uses the same validation and scoring decoder as the running authority.
// A transfer does not yet carry week parts, so split evidence is refused here.
func TransferFootball(body []byte) (*PinnedFootball, error) {
	raw, err := decodeFootball(body, nil)
	if err != nil {
		return nil, err
	}
	stats, err := raw.scoringFiles(nil)
	if err != nil {
		return nil, err
	}
	return &PinnedFootball{digest: footballDigest(body), raw: raw, stats: stats}, nil
}

// TransferReadings replays without any database or archived-source access.
func TransferReadings(ctx context.Context, league string, records []commissioner.Record, evidence *PinnedFootball, at time.Time) (TransferProjection, error) {
	s := &Store{now: func() time.Time { return at }, lineups: ledgerLineups{}, lineupEnv: evidence}
	out := TransferProjection{}
	var err error
	if out.Assets, err = commissioner.NativeAssetsReading(league, records, at); err != nil {
		return out, fmt.Errorf("replay league reading: %w", err)
	}
	if out.Cap, err = commissioner.NativeCapCheckReading(commissioner.RawNativeCapScope{League: league, ViewerZone: "UTC"}, records, at); err != nil {
		return out, fmt.Errorf("replay league reading: %w", err)
	}
	directory, err := commissioner.NativeDirectoryReading(league, records, at)
	if err != nil {
		return out, fmt.Errorf("replay league reading: %w", err)
	}
	src := s.nativeSources(league, evidence.raw.Season)
	if out.Standings, err = commissioner.NativeStandingsReading(ctx, league, records, src, evidence.raw.Season, at); err != nil {
		return out, fmt.Errorf("replay league reading: %w", err)
	}
	current, err := commissioner.SeasonAt(records, commissioner.ReadingInstant(records, at))
	if err != nil {
		return out, fmt.Errorf("replay league reading: %w", err)
	}
	for week := 1; week <= 22; week++ {
		result, err := commissioner.NativeResultsReading(ctx, league, records, src, evidence.raw.Season, week, at)
		if err != nil {
			return out, fmt.Errorf("replay league reading: %w", err)
		}
		out.Results = append(out.Results, result)
	}
	for _, week := range evidence.raw.Weeks {
		if current != evidence.raw.Season || !out.Assets.Seeded {
			continue
		}
		lineups, err := transferLineups(ctx, league, records, evidence, directory, week.Number, at)
		if err != nil {
			return out, fmt.Errorf("replay league reading: %w", err)
		}
		out.Lineups = append(out.Lineups, lineups...)
	}
	if current != evidence.raw.Season {
		return out, nil
	}
	out.Bids, err = transferBids(league, records, evidence, directory, at)
	return out, err
}

func transferBids(league string, records []commissioner.Record, evidence *PinnedFootball, directory commissioner.NativeDirectory, at time.Time) ([]market.NativeBids, error) {
	out := []market.NativeBids{}
	people, err := commissioner.GovernanceState(records)
	if err != nil {
		return nil, fmt.Errorf("replay league owners: %w", err)
	}
	for _, seat := range directory.Franchises {
		owner, ok := people.Franchise(governance.FranchiseID(seat.Franchise))
		if !ok || owner.Seat == "" {
			continue
		}
		bids, err := market.NativeBidsReading(league, seat.Franchise, string(owner.Seat), records, at, time.UTC, evidence.PlayerFacts())
		if err != nil {
			return nil, fmt.Errorf("replay league reading: %w", err)
		}
		out = append(out, bids)
	}
	return out, nil
}

func transferLineups(ctx context.Context, league string, records []commissioner.Record, evidence *PinnedFootball, directory commissioner.NativeDirectory, week int, at time.Time) ([]commissioner.NativeLineup, error) {
	facts, err := evidence.LineupFacts(ctx, evidence.raw.Season, week)
	if err != nil {
		return nil, fmt.Errorf("replay league lineup: %w", err)
	}
	out := make([]commissioner.NativeLineup, 0, len(directory.Franchises))
	for _, seat := range directory.Franchises {
		lineup, err := commissioner.NativeLineupReading(league, seat.Franchise, commissioner.PrincipalFor(records, commissioner.Replay(records).Creator), week, records, at, facts)
		if err != nil {
			return nil, fmt.Errorf("replay league lineup: %w", err)
		}
		out = append(out, lineup)
	}
	return out, nil
}

// Season identifies the source season in a validated, self-contained evidence bundle.
func (p *PinnedFootball) Season() int { return p.raw.Season }

type TransferProjection struct {
	Assets    commissioner.NativeAssets
	Cap       commissioner.NativeCapCheck
	Lineups   []commissioner.NativeLineup
	Bids      []market.NativeBids
	Results   []commissioner.NativeResults
	Standings commissioner.NativeStandings
}
