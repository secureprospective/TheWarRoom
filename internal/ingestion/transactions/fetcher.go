// Package transactions adapts MFL's season transaction feed.
package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

const Export = "transactions"

type RawTransaction struct {
	Kind             string  `json:"type"`
	Timestamp        string  `json:"timestamp"`
	Franchise        string  `json:"franchise"`
	ByCommish        string  `json:"by_commish"`
	Counterparty     string  `json:"franchise2"`
	Gave             *string `json:"franchise1_gave_up"`
	CounterpartyGave *string `json:"franchise2_gave_up"`
	Comments         string  `json:"comments"`
	Expires          string  `json:"expires"`
	Transaction      *string `json:"transaction"`
	Activated        *string `json:"activated"`
	Deactivated      *string `json:"deactivated"`
	Promoted         *string `json:"promoted"`
	Demoted          *string `json:"demoted"`
}

type RawTransactions struct {
	Transactions []RawTransaction
}

func (r RawTransactions) Validate() error {
	for i, row := range r.Transactions {
		if err := row.Validate(); err != nil {
			return fmt.Errorf("transactions: index %d: %w", i, err)
		}
	}
	return nil
}

func (r RawTransaction) Validate() error {
	_, err := convert(r)
	return err
}

func Fetch(ctx context.Context, c *mfl.Client, year, league string) (RawTransactions, error) {
	if err := c.DiscoverHost(ctx, year, league); err != nil {
		return RawTransactions{}, fmt.Errorf("transactions: discover: %w", err)
	}
	body, err := ingestion.LeagueExport(ctx, c, Export, year, league, nil)
	if err != nil {
		return RawTransactions{}, fmt.Errorf("transactions: fetch: %w", err)
	}
	return Parse(body)
}

func Parse(body []byte) (RawTransactions, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return RawTransactions{}, fmt.Errorf("transactions: %w", err)
	}
	var env struct {
		Transactions *struct {
			Rows ingestion.MFLList[RawTransaction] `json:"transaction"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return RawTransactions{}, fmt.Errorf("transactions: decode: %w", err)
	}
	if env.Transactions == nil {
		return RawTransactions{}, fmt.Errorf("transactions: missing transactions object")
	}
	raw := RawTransactions{Transactions: env.Transactions.Rows}
	if err := raw.Validate(); err != nil {
		return RawTransactions{}, err
	}
	return raw, nil
}

func ToTransactions(raw RawTransactions) ([]leaguefeed.Transaction, error) {
	rows := make([]leaguefeed.Transaction, 0, len(raw.Transactions))
	for i, r := range raw.Transactions {
		row, err := convert(r)
		if err != nil {
			return nil, fmt.Errorf("transactions: index %d: %w", i, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func convert(r RawTransaction) (leaguefeed.Transaction, error) {
	at, err := ingestion.FeedTime(r.Timestamp)
	if err != nil {
		return leaguefeed.Transaction{}, fmt.Errorf("transactions: time: %w", err)
	}
	if err := ingestion.FeedFranchise(r.Franchise); err != nil {
		return leaguefeed.Transaction{}, fmt.Errorf("transactions: franchise: %w", err)
	}
	if r.Kind == "" {
		return leaguefeed.Transaction{}, fmt.Errorf("transactions: missing type")
	}
	if r.ByCommish != "" && r.ByCommish != "0" && r.ByCommish != "1" {
		return leaguefeed.Transaction{}, fmt.Errorf("transactions: invalid by_commish %q", r.ByCommish)
	}
	row := leaguefeed.Transaction{
		Kind: r.Kind, Time: at, Franchise: r.Franchise, ByCommish: r.ByCommish == "1",
	}
	switch r.Kind {
	case "TRADE":
		row.Trade, err = convertTrade(r)
	case "FREE_AGENT", "LOAD_ROSTERS":
		row.AddsDrops, err = convertAddsDrops(r.Transaction)
	case "IR":
		row.IR, err = ingestion.FeedChange(r.Activated, r.Deactivated)
	case "TAXI":
		row.Taxi, err = ingestion.FeedChange(r.Promoted, r.Demoted)
	default:
		row.Unparsed = true
	}
	if err != nil {
		return leaguefeed.Transaction{}, fmt.Errorf("transactions: %s: %w", r.Kind, err)
	}
	return row, nil
}

func convertAddsDrops(raw *string) (*leaguefeed.RosterChange, error) {
	if raw == nil || strings.Count(*raw, "|") != 1 {
		return nil, fmt.Errorf("adds/drops needs exactly one pipe")
	}
	in, out, _ := strings.Cut(*raw, "|")
	change, err := ingestion.FeedChange(&in, &out)
	if err != nil {
		return nil, fmt.Errorf("adds/drops: %w", err)
	}
	return change, nil
}

func convertTrade(r RawTransaction) (*leaguefeed.Trade, error) {
	if err := ingestion.FeedFranchise(r.Counterparty); err != nil {
		return nil, fmt.Errorf("counterparty: %w", err)
	}
	if r.Counterparty == r.Franchise {
		return nil, fmt.Errorf("trade needs distinct franchises")
	}
	gave, err := ingestion.FeedAssets(r.Gave)
	if err != nil {
		return nil, fmt.Errorf("franchise1_gave_up: %w", err)
	}
	counterpartyGave, err := ingestion.FeedAssets(r.CounterpartyGave)
	if err != nil {
		return nil, fmt.Errorf("franchise2_gave_up: %w", err)
	}
	expires, err := ingestion.FeedTime(r.Expires)
	if err != nil {
		return nil, fmt.Errorf("expires: %w", err)
	}
	return &leaguefeed.Trade{
		Counterparty: r.Counterparty, Gave: gave, CounterpartyGave: counterpartyGave,
		Comments: r.Comments, Expires: expires,
	}, nil
}
