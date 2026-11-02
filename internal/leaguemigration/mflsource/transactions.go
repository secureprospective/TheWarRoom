package mflsource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
)

// TransactionCommands translates checked feed evidence into ordinary commissioner
// corrections, including reserve statuses and LOAD_ROSTERS membership observations.
// An observation does not invent acquisition provenance or a contract.
// It does not append commands or confer authority on a source's by_commish flag.
func TransactionCommands(raw transactions.RawTransactions, league, reason string, season int, franchises, players map[string]string) ([][]commissioner.RawCommand, error) {
	rows, err := transactions.ToTransactions(raw)
	if err != nil {
		return nil, fmt.Errorf("check transaction feed: %w", err)
	}
	slices.SortStableFunc(rows, func(a, b leaguefeed.Transaction) int { return a.Time.Compare(b.Time) })
	out := make([][]commissioner.RawCommand, 0, len(rows))
	for _, row := range rows {
		commands, err := transactionCommands(row, league, reason, season, franchises, players)
		if err != nil {
			return nil, fmt.Errorf("translate %s at %s: %w", row.Kind, row.Time.Format(time.RFC3339Nano), err)
		}
		out = append(out, commands)
	}
	return out, nil
}

// BackfillTransactions uses the franchise directory recorded in the league, not
// a newly sorted source directory. Ordinary append checks still own admission.
func BackfillTransactions(ctx context.Context, ledger leaguemigration.Ledger, league, reason, principal string, season int, raw transactions.RawTransactions, heldPlayers map[string]string, draft ...[]draftreports.Pick) ([]commissioner.Receipt, error) {
	records, err := ledger.Records(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("read transaction identities: %w", err)
	}
	assets, err := commissioner.NativeAssetsReading(league, records)
	if err != nil {
		return nil, fmt.Errorf("read transaction franchises: %w", err)
	}
	if assets.Snapshot == nil || assets.Snapshot.Season != season {
		return nil, fmt.Errorf("record starting franchises for the transaction season")
	}
	franchises, players := map[string]string{}, map[string]string{}
	for _, f := range assets.Snapshot.Franchises {
		franchises[f.HeldID] = f.ID
	}
	for id := range heldPlayers {
		players[id] = id
	}
	seed := *assets.Snapshot
	for _, record := range records {
		if record.Command.Command == commissioner.NativeSeed && record.Command.Args.Native != nil {
			seed = *record.Command.Args.Native
			break
		}
	}
	groups, err := TransactionReplayCommands(raw, league, reason, seed, franchises, players, draft...)
	if err != nil {
		return nil, err
	}
	receipts, err := leaguemigration.SyncTransactions(ctx, ledger, league, principal, groups)
	if err != nil {
		return receipts, fmt.Errorf("backfill transactions: %w", err)
	}
	return receipts, nil
}

func transactionCommands(row leaguefeed.Transaction, league, reason string, season int, franchises, players map[string]string) ([]commissioner.RawCommand, error) {
	if row.Time.Year() != season {
		return nil, fmt.Errorf("transaction must belong to the imported season")
	}
	franchise := franchises[row.Franchise]
	if franchise == "" {
		return nil, fmt.Errorf("unknown franchise %s", row.Franchise)
	}
	body, err := json.Marshal(row)
	if err != nil {
		return nil, fmt.Errorf("identify transaction: %w", err)
	}
	sum := sha256.Sum256(body)
	key := "transactions.sync." + hex.EncodeToString(sum[:])
	var commands []commissioner.RawCommand
	add := func(command string, args commissioner.RawArgs) {
		commands = append(commands, commissioner.RawCommand{League: league, Command: command, Args: args,
			Request: fmt.Sprintf("%s.%d", key, len(commands)), Reason: reason,
			Effective: row.Time.UTC().Format(time.RFC3339Nano), Visibility: "league"})
	}
	switch row.Kind {
	case "FREE_AGENT", "LOAD_ROSTERS":
		if err := transactionRosterAssets(row.AddsDrops.Out, franchise, "", "remove", franchises, players, season, add); err != nil {
			return nil, err
		}
		if err := transactionRosterAssets(row.AddsDrops.In, franchise, "", "add", franchises, players, season, add); err != nil {
			return nil, err
		}
	case "TRADE":
		other := franchises[row.Trade.Counterparty]
		if other == "" {
			return nil, fmt.Errorf("unknown trade counterparty %s", row.Trade.Counterparty)
		}
		if err := transactionRosterAssets(row.Trade.Gave, franchise, other, "remove", franchises, players, season, add); err != nil {
			return nil, err
		}
		if err := transactionRosterAssets(row.Trade.CounterpartyGave, other, franchise, "remove", franchises, players, season, add); err != nil {
			return nil, err
		}
	case "IR", "TAXI":
		if err := transactionReserve(row, franchise, players, add); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported transaction kind %s", row.Kind)
	}
	for _, command := range commands {
		if err := command.Validate(); err != nil {
			return nil, fmt.Errorf("check translated command: %w", err)
		}
	}
	return commands, nil
}

func transactionRosterAssets(assets []leaguefeed.Asset, from, to, action string, franchises, players map[string]string, season int, add func(string, commissioner.RawArgs)) error {
	for _, asset := range assets {
		switch {
		case asset.Player != nil:
			player := players[asset.Player.String()]
			if player == "" {
				return fmt.Errorf("unknown player %s", asset.Player.String())
			}
			add(commissioner.CorrectRoster, commissioner.RawArgs{Correction: &commissioner.RawCorrection{Player: player, Franchise: from, Action: action}})
			if to != "" {
				add(commissioner.CorrectRoster, commissioner.RawArgs{Correction: &commissioner.RawCorrection{Player: player, Franchise: to, Action: "add"}})
			}
		case asset.CurrentPick != nil && to != "":
			p := asset.CurrentPick
			add(commissioner.CorrectPick, commissioner.RawArgs{Correction: &commissioner.RawCorrection{Owner: to, Pick: &commissioner.CorrectionPick{Season: season, Round: p.Round, Slot: p.Pick}}})
		case asset.FuturePick != nil && to != "":
			p := asset.FuturePick
			original := franchises[p.Franchise]
			if original == "" {
				return fmt.Errorf("unknown original pick franchise %s", p.Franchise)
			}
			add(commissioner.CorrectPick, commissioner.RawArgs{Correction: &commissioner.RawCorrection{Owner: to, Pick: &commissioner.CorrectionPick{Season: p.Year, Round: p.Round, Original: original}}})
		default:
			return fmt.Errorf("asset cannot be represented as a roster or future-pick correction")
		}
	}
	return nil
}

func transactionReserveAssets(assets []leaguefeed.Asset, franchise, bucket string, players map[string]string, add func(string, commissioner.RawArgs)) error {
	for _, asset := range assets {
		if asset.Player == nil || players[asset.Player.String()] == "" {
			return fmt.Errorf("reserve move needs a known player")
		}
		add(commissioner.CorrectRoster, commissioner.RawArgs{Correction: &commissioner.RawCorrection{Player: players[asset.Player.String()], Franchise: franchise, Action: "status", Status: commissioner.ReserveStatus(bucket)}})
	}
	return nil
}

func transactionReserve(row leaguefeed.Transaction, franchise string, players map[string]string, add func(string, commissioner.RawArgs)) error {
	change, bucket := row.IR, "ir"
	if row.Kind == "TAXI" {
		change, bucket = row.Taxi, "taxi"
	}
	if err := transactionReserveAssets(change.In, franchise, "active", players, add); err != nil {
		return err
	}
	return transactionReserveAssets(change.Out, franchise, bucket, players, add)
}

// DraftArchive offers only held, successful reference fetches.
type DraftArchive interface {
	ArchivedBodies(context.Context, string, func(string, []byte, time.Time) bool) (bool, error)
}

// BackfillHeldTransactions uses checked held selections before leaving a pick unverified.
func BackfillHeldTransactions(ctx context.Context, ledger leaguemigration.Ledger, archive DraftArchive, league, reason, principal string, season int, raw transactions.RawTransactions, players map[string]string) ([]commissioner.Receipt, error) {
	var draft []draftreports.Pick
	records, err := ledger.Records(ctx, league)
	if err != nil {
		return nil, fmt.Errorf("read held draft identity: %w", err)
	}
	source, _, _, ok := ReferenceLeague(records)
	if !ok {
		return nil, fmt.Errorf("record a reference league identity before reading held draft evidence")
	}
	_, err = archive.ArchivedBodies(ctx, "TYPE=draftResults", func(rawURL string, body []byte, _ time.Time) bool {
		u, err := url.Parse(rawURL)
		if err != nil || u.Query().Get("L") != source || !strings.HasPrefix(u.Path, "/"+strconv.Itoa(season)+"/") {
			return false
		}
		parsed, parseErr := draftreports.ParseResults(body)
		if parseErr != nil {
			return false
		}
		draft = parsed
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("read held draft evidence: %w", err)
	}
	return BackfillTransactions(ctx, ledger, league, reason, principal, season, raw, players, draft)
}
