// Package leaguetransfer moves replay-verified league history with explicit source fencing.
package leaguetransfer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// RawExport is versioned independently of SQLite, commands and transport. Content identifies an
// inactive import; SourceEpoch identifies the source authority generation, not a schema marker.
type RawExport struct {
	Identities  []playerid.RawIdentity     `json:"playerIdentities,omitempty"`
	Version     int                        `json:"version"`
	League      string                     `json:"league"`
	SourceEpoch string                     `json:"sourceEpoch"`
	FrozenAt    time.Time                  `json:"frozenAt"`
	Events      []history.RawTransferEvent `json:"events"`
	Football    string                     `json:"football"`
	Blobs       map[string][]byte          `json:"blobs"`
	Aliases     []governance.VerifiedAlias `json:"aliases"`
	Checkpoint  db.AuditHead               `json:"checkpoint"`
	Projection  ProjectionDigests          `json:"projection"`
	Content     string                     `json:"content"`
}

type ProjectionDigests struct {
	Roster, Cap, Lineup, Bid, Result, Standings string
}

// Capture does not seal or retire the source. A commissioner must use the same epoch until cutover.
func Capture(ctx context.Context, store *history.Store, league, epoch, football string, at time.Time, aliases []governance.VerifiedAlias, additional ...string) (RawExport, error) {
	out := RawExport{Version: 1, League: league, SourceEpoch: epoch, FrozenAt: at.UTC(), Football: football, Aliases: aliases}
	captured, err := store.CaptureTransferSnapshot(ctx, league, append([]string{football}, additional...))
	if err != nil {
		return out, fmt.Errorf("export league: %w", err)
	}
	out.Events, out.Checkpoint = captured.Events, captured.Checkpoint
	out.Blobs, out.Identities = captured.Blobs, captured.Identities
	out.Aliases = leagueAliases(captured.Events, aliases)
	if err := out.validateHistory(); err != nil {
		return out, err
	}
	if out.Projection, err = out.replay(ctx); err != nil {
		return out, err
	}
	if out.Content, err = out.contentDigest(); err != nil {
		return out, err
	}
	return out, out.Validate(ctx)
}

func digest(data []byte) string { hash := sha256.Sum256(data); return hex.EncodeToString(hash[:]) }
func jsonDigest[T any](value T) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("fingerprint league export: %w", err)
	}
	return digest(data), nil
}
func (r RawExport) contentDigest() (string, error) { r.Content = ""; return jsonDigest(r) }

func (r RawExport) validateHistory() error {
	if r.Version != 1 || !strings.HasPrefix(r.League, "native.") || r.League == "native." || strings.TrimSpace(r.League) != r.League || r.FrozenAt.IsZero() {
		return fmt.Errorf("this league export needs a supported version, league and comparison time")
	}
	// Reuse canonical UUID validation; an epoch is an opaque UUID, not an account grant.
	if _, err := governance.AccountPrincipal(r.SourceEpoch); err != nil {
		return fmt.Errorf("choose a source generation UUID: %w", err)
	}
	if _, err := governance.NewPrincipals(r.League, r.Aliases); err != nil {
		return fmt.Errorf("check exported linked identities: %w", err)
	}
	if err := r.validatePlayerIdentities(); err != nil {
		return err
	}
	if err := r.validateManifest(); err != nil {
		return err
	}
	if err := r.validateEvents(); err != nil {
		return err
	}
	body, ok := r.Blobs[r.Football]
	if !ok || digest(body) != r.Football {
		return fmt.Errorf("the league's football evidence is missing or has changed; export it again")
	}
	return nil
}

func (r RawExport) validateManifest() error {
	if len(r.Events) == 0 || len(r.Events) > 100000 || len(r.Blobs) < 1 || len(r.Blobs) > 100 {
		return fmt.Errorf("the league export is missing its history, evidence or saved checkpoint")
	}
	if r.Checkpoint.Version != 1 || r.Checkpoint.Source != "league_event" || r.Checkpoint.Size < int64(len(r.Events)) || len(r.Checkpoint.Hash) != 64 {
		return fmt.Errorf("the league checkpoint is incomplete")
	}
	return nil
}

func (r RawExport) validateEvents() error {
	revisions := map[string]int64{}
	ids := map[string]string{}
	for _, event := range r.Events {
		if err := event.Validate(); err != nil {
			return fmt.Errorf("check exported change: %w", err)
		}
		if event.League != r.League || ids[event.ID] != "" || event.Revision != revisions[event.Stream]+1 {
			return fmt.Errorf("the league export has a missing, repeated or unrelated change")
		}
		if event.Supersedes != nil && ids[*event.Supersedes] != event.Stream {
			return fmt.Errorf("the league export has a correction without its original change")
		}
		ids[event.ID] = event.Stream
		revisions[event.Stream] = event.Revision
	}
	return nil
}

// Validate authenticates content references and proves projection parity by replay, not a copied cache.
func (r RawExport) Validate(ctx context.Context) error {
	if err := r.validateHistory(); err != nil {
		return err
	}
	hash, err := r.contentDigest()
	if err != nil {
		return err
	}
	if hash != r.Content {
		return fmt.Errorf("the league export's contents have changed; export it again")
	}
	projection, err := r.replay(ctx)
	if err != nil {
		return err
	}
	if projection != r.Projection {
		return fmt.Errorf("this app cannot reproduce the exported league; do not activate it")
	}
	return nil
}

func (r RawExport) replay(ctx context.Context) (ProjectionDigests, error) {
	var out ProjectionDigests
	records, err := history.TransferRecords(r.Events)
	if err != nil {
		return out, fmt.Errorf("read league for replay: %w", err)
	}
	principals, err := governance.NewPrincipals(r.League, r.Aliases)
	if err != nil {
		return out, fmt.Errorf("read exported identities: %w", err)
	}
	records, err = commissioner.ResolvePrincipals(commissioner.WithPrincipals(ctx, principals), r.League, records)
	if err != nil {
		return out, fmt.Errorf("resolve exported identities: %w", err)
	}
	projections, err := r.projections(ctx, records)
	if err != nil {
		return out, err
	}
	return projectionDigests(projections)
}

func (r RawExport) projections(ctx context.Context, records []commissioner.Record) ([]history.TransferProjection, error) {
	keys := make([]string, 0, len(r.Blobs))
	for key := range r.Blobs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]history.TransferProjection, 0, len(keys))
	seasons := map[int]bool{}
	for _, key := range keys {
		if digest(r.Blobs[key]) != key {
			return nil, fmt.Errorf("exported evidence has changed")
		}
		football, err := history.TransferFootball(r.Blobs[key])
		if err != nil {
			return nil, fmt.Errorf("read exported football evidence: %w", err)
		}
		if seasons[football.Season()] {
			return nil, fmt.Errorf("choose only one pinned football bundle per season")
		}
		seasons[football.Season()] = true
		projection, err := history.TransferReadings(ctx, r.League, records, football, r.FrozenAt)
		if err != nil {
			return nil, fmt.Errorf("replay exported league: %w", err)
		}
		out = append(out, projection)
	}
	current, err := commissioner.SeasonAt(records, commissioner.ReadingInstant(records, r.FrozenAt))
	if err != nil {
		return nil, fmt.Errorf("read exported season: %w", err)
	}
	active, err := history.TransferFootball(r.Blobs[r.Football])
	if err != nil {
		return nil, fmt.Errorf("read active football evidence: %w", err)
	}
	if active.Season() != current {
		return nil, fmt.Errorf("select the current season's football evidence")
	}
	first, err := commissioner.SeasonAt(records, records[0].Receipt.RecordedAt)
	if err != nil {
		return nil, fmt.Errorf("read first exported season: %w", err)
	}
	for season := first; season <= current; season++ {
		if !seasons[season] {
			return nil, fmt.Errorf("include pinned football evidence for season %d", season)
		}
	}
	return out, nil
}

func projectionDigests(projections []history.TransferProjection) (ProjectionDigests, error) {
	var out ProjectionDigests
	roster := make([]commissioner.NativeAssets, 0, len(projections))
	capReadings := make([]commissioner.NativeCapCheck, 0, len(projections))
	lineups := [][]commissioner.NativeLineup{}
	bids := [][]commissioner.NativeBids{}
	results := [][]commissioner.NativeResults{}
	standings := []commissioner.NativeStandings{}
	for _, p := range projections {
		roster = append(roster, p.Assets)
		capReadings = append(capReadings, p.Cap)
		lineups = append(lineups, p.Lineups)
		bids = append(bids, p.Bids)
		results = append(results, p.Results)
		standings = append(standings, p.Standings)
	}
	var err error
	if out.Roster, err = jsonDigest(roster); err != nil {
		return out, err
	}
	if out.Cap, err = jsonDigest(capReadings); err != nil {
		return out, err
	}
	if out.Lineup, err = jsonDigest(lineups); err != nil {
		return out, err
	}
	if out.Bid, err = jsonDigest(bids); err != nil {
		return out, err
	}
	if out.Result, err = jsonDigest(results); err != nil {
		return out, err
	}
	out.Standings, err = jsonDigest(standings)
	return out, err
}

func leagueAliases(events []history.RawTransferEvent, aliases []governance.VerifiedAlias) []governance.VerifiedAlias {
	out := []governance.VerifiedAlias{}
	for _, alias := range aliases {
		for _, event := range events {
			if event.Actor == string(alias.Alias) || strings.Contains(event.Payload, strconv.Quote(string(alias.Alias))) {
				out = append(out, alias)
				break
			}
		}
	}
	return out
}

func decode(ctx context.Context, data []byte) (RawExport, error) {
	var raw RawExport
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("read league export: %w", err)
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return raw, fmt.Errorf("the league export must contain one document")
	}
	return raw, raw.Validate(ctx)
}

// Import authenticates and replays before opening the destination writer transaction.
// It preserves durable bytes without executing commands, enrolling accounts or activating authority.
func Import(ctx context.Context, store *history.Store, input io.Reader, key [32]byte) (history.ImportReceipt, error) {
	raw, err := Decrypt(ctx, input, key)
	if err != nil {
		return history.ImportReceipt{}, fmt.Errorf("check league before importing: %w", err)
	}
	document, err := json.Marshal(raw)
	if err != nil {
		return history.ImportReceipt{}, fmt.Errorf("save league import document: %w", err)
	}
	receipt, err := store.StageTransfer(ctx, history.ImportReceipt{League: raw.League, SourceEpoch: raw.SourceEpoch, Content: raw.Content}, document, raw.Events, raw.Blobs, raw.Identities)
	if err != nil {
		return receipt, fmt.Errorf("import inactive league: %w", err)
	}
	return receipt, nil
}

func (r RawExport) validatePlayerIdentities() error {
	seen := make(map[string]bool)
	links := make(map[playerid.RawOutsideID]bool)
	for _, identity := range r.Identities {
		if err := identity.Validate(); err != nil {
			return fmt.Errorf("check exported player identity: %w", err)
		}
		if seen[identity.Canonical] {
			return fmt.Errorf("repeated exported player identity")
		}
		seen[identity.Canonical] = true
		for _, link := range identity.Links {
			if links[link] {
				return fmt.Errorf("repeated exported outside player link")
			}
			links[link] = true
		}
	}
	return nil
}
