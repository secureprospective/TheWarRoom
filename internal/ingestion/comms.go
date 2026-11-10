package ingestion

import (
	"fmt"
	"strings"
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
