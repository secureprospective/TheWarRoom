package pendingtrades

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type RawPendingTrade struct {
	ID          string  `json:"trade_id"`
	Offering    string  `json:"offeringteam"`
	OfferedTo   string  `json:"offeredto"`
	Gives       *string `json:"will_give_up"`
	Gets        *string `json:"will_receive"`
	Timestamp   string  `json:"timestamp"`
	Expires     string  `json:"expires"`
	Comments    string  `json:"comments"`
	Description string  `json:"description"`
}

type RawPendingTrades struct {
	Trades []RawPendingTrade
}

func (r RawPendingTrades) Validate() error {
	for i, trade := range r.Trades {
		if err := trade.Validate(); err != nil {
			return fmt.Errorf("pendingTrades: index %d: %w", i, err)
		}
	}
	return nil
}

func (r RawPendingTrade) Validate() error {
	_, err := convert(r)
	return err
}

func Fetch(ctx context.Context, c *mfl.Client, year, league string) (RawPendingTrades, error) {
	env, err := fetchEnvelope(ctx, c, year, league)
	if err != nil {
		return RawPendingTrades{}, err
	}
	return parseTrades(env)
}

func Parse(body []byte) (RawPendingTrades, error) {
	env, err := decodeEnvelope(body, http.StatusOK)
	if err != nil {
		return RawPendingTrades{}, err
	}
	return parseTrades(env)
}

func parseTrades(env RawEnvelope) (RawPendingTrades, error) {
	var block struct {
		Trades json.RawMessage `json:"pendingTrade"`
	}
	if err := json.Unmarshal(env.PendingTrades, &block); err != nil {
		return RawPendingTrades{}, fmt.Errorf("pendingTrades: decode block: %w", err)
	}
	var trades ingestion.MFLList[RawPendingTrade]
	data := bytes.TrimSpace(block.Trades)
	if len(data) != 0 && !bytes.Equal(data, []byte("{}")) {
		if err := json.Unmarshal(data, &trades); err != nil {
			return RawPendingTrades{}, fmt.Errorf("pendingTrades: decode trades: %w", err)
		}
	}
	raw := RawPendingTrades{Trades: trades}
	if err := raw.Validate(); err != nil {
		return RawPendingTrades{}, err
	}
	return raw, nil
}

func ToPendingTrades(raw RawPendingTrades) ([]leaguefeed.PendingTrade, error) {
	trades := make([]leaguefeed.PendingTrade, 0, len(raw.Trades))
	for i, r := range raw.Trades {
		trade, err := convert(r)
		if err != nil {
			return nil, fmt.Errorf("pendingTrades: index %d: %w", i, err)
		}
		trades = append(trades, trade)
	}
	return trades, nil
}

func convert(r RawPendingTrade) (leaguefeed.PendingTrade, error) {
	if err := ingestion.FeedID(r.ID); err != nil {
		return leaguefeed.PendingTrade{}, fmt.Errorf("pendingTrades: trade_id: %w", err)
	}
	for _, id := range []string{r.Offering, r.OfferedTo} {
		if err := ingestion.FeedFranchise(id); err != nil {
			return leaguefeed.PendingTrade{}, fmt.Errorf("pendingTrades: %w", err)
		}
	}
	if r.Offering == r.OfferedTo {
		return leaguefeed.PendingTrade{}, fmt.Errorf("pendingTrades: needs distinct franchises")
	}
	gives, err := ingestion.FeedAssets(r.Gives)
	if err != nil {
		return leaguefeed.PendingTrade{}, fmt.Errorf("pendingTrades: will_give_up: %w", err)
	}
	gets, err := ingestion.FeedAssets(r.Gets)
	if err != nil {
		return leaguefeed.PendingTrade{}, fmt.Errorf("pendingTrades: will_receive: %w", err)
	}
	proposed, err := ingestion.FeedTime(r.Timestamp)
	if err != nil {
		return leaguefeed.PendingTrade{}, fmt.Errorf("pendingTrades: proposed: %w", err)
	}
	expires, err := ingestion.FeedTime(r.Expires)
	if err != nil {
		return leaguefeed.PendingTrade{}, fmt.Errorf("pendingTrades: expires: %w", err)
	}
	return leaguefeed.PendingTrade{
		ID: r.ID, Offering: r.Offering, OfferedTo: r.OfferedTo, Gives: gives, Gets: gets,
		Proposed: proposed, Expires: expires, Comments: r.Comments, Description: r.Description,
	}, nil
}
