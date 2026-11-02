package nflschedule

import (
	"context"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

const ByeExport = "nflByeWeeks"

// RawBye is a validated team bye from a held schedule.
type RawBye = source.RawBye

// FetchByes reads the season's bye weeks from the api host, like the schedule.
func FetchByes(ctx context.Context, c *mfl.Client, year string) ([]RawBye, error) {
	resp, err := c.Do(ctx, mfl.Request{Type: ByeExport, Year: year})
	if err != nil {
		return nil, fmt.Errorf("nflschedule: fetch byes: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nflschedule: byes unexpected status %d", resp.StatusCode)
	}
	return ParseByes(resp.Body)
}

func ParseByes(body []byte) ([]RawBye, error) {
	rows, err := source.ParseByes(body)
	if err != nil {
		return nil, fmt.Errorf("held byes: %w", err)
	}
	return rows, nil
}
