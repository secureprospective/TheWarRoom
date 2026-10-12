package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

type RawLeaguePost struct {
	Title     string `json:"title"`
	Subject   string `json:"subject"`
	Franchise string `json:"franchise"`
	Timestamp string `json:"timestamp"`
}

func (r RawLeaguePost) Validate() error {
	if strings.TrimSpace(r.Title) == "" && strings.TrimSpace(r.Subject) == "" {
		return fmt.Errorf("league board: missing title")
	}
	if err := FeedFranchise(r.Franchise); err != nil {
		return fmt.Errorf("league board: author: %w", err)
	}
	if _, err := FeedTime(r.Timestamp); err != nil {
		return fmt.Errorf("league board: date: %w", err)
	}
	return nil
}

type RawLeagueBoard struct {
	Messages MFLList[RawLeaguePost] `json:"message"`
	Threads  MFLList[RawLeaguePost] `json:"thread"`
	Polls    MFLList[RawLeaguePost] `json:"poll"`
}

func (r RawLeagueBoard) Validate() error {
	for _, rows := range [][]RawLeaguePost{r.Messages, r.Threads, r.Polls} {
		for i, row := range rows {
			if err := row.Validate(); err != nil {
				return fmt.Errorf("league board: row %d: %w", i, err)
			}
		}
	}
	return nil
}

func ParseLeagueBoard(body []byte, export string) ([]RawLeaguePost, error) {
	if export != "messageBoard" && export != "polls" {
		return nil, fmt.Errorf("league board: invalid export %q", export)
	}
	if err := CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("league board: not available: %w", err)
	}
	var raw struct {
		Board *RawLeagueBoard `json:"messageBoard"`
		Polls *RawLeagueBoard `json:"polls"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("league board: decode: %w", err)
	}
	board := raw.Board
	if export == "polls" {
		board = raw.Polls
	}
	if board == nil {
		return nil, fmt.Errorf("league board: missing %s object", export)
	}
	if err := board.Validate(); err != nil {
		return nil, err
	}
	if board.Messages == nil && board.Threads == nil && board.Polls == nil {
		if err := checkEmptyBoard(body, export); err != nil {
			return nil, err
		}
	}
	rows := append([]RawLeaguePost{}, board.Messages...)
	rows = append(rows, board.Threads...)
	return append(rows, board.Polls...), nil
}

func checkEmptyBoard(body []byte, export string) error {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("league board: envelope: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(envelope[export], &fields); err != nil {
		return fmt.Errorf("league board: object: %w", err)
	}
	for key := range fields {
		if key != "message" && key != "thread" && key != "poll" {
			return fmt.Errorf("league board: not available: unsupported post fields")
		}
	}
	return nil
}

// FetchLeagueBoard reads the league message board or polls. MFL answers both only to a logged-in league member,
// so the request is keyed.
func FetchLeagueBoard(ctx context.Context, client *mfl.Client, year, league, export string) ([]RawLeaguePost, error) {
	if export != "messageBoard" && export != "polls" {
		return nil, fmt.Errorf("league board: invalid export %q", export)
	}
	if err := client.DiscoverHost(ctx, year, league); err != nil {
		return nil, fmt.Errorf("league board: discover: %w", err)
	}
	body, err := KeyedLeagueExport(ctx, client, export, year, league, nil)
	if err != nil {
		return nil, fmt.Errorf("league board: fetch: %w", err)
	}
	return ParseLeagueBoard(body, export)
}
