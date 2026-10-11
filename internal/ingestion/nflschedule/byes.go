package nflschedule

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

const ByeExport = "nflByeWeeks"

// RawBye is one NFL team's bye week for the season.
type RawBye struct {
	Team string `json:"id"`
	Week string `json:"bye_week"`
}

func (b RawBye) Validate() error {
	if strings.TrimSpace(b.Team) == "" {
		return fmt.Errorf("nflschedule: bye week without a team")
	}
	if week, err := strconv.Atoi(b.Week); err != nil || week < 1 {
		return fmt.Errorf("nflschedule: team %s bye week %q is not a week", b.Team, b.Week)
	}
	return nil
}

// ByeWeek is the RawBye's week; call it only on a validated RawBye.
func (b RawBye) ByeWeek() int {
	week, _ := strconv.Atoi(b.Week)
	return week
}

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
	if err := ingestion.CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("nflschedule: byes: %w", err)
	}
	var env struct {
		Byes struct {
			Teams ingestion.MFLList[RawBye] `json:"team"`
		} `json:"nflByeWeeks"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("nflschedule: decode byes: %w", err)
	}
	if len(env.Byes.Teams) == 0 {
		return nil, fmt.Errorf("nflschedule: byes list no teams")
	}
	for _, b := range env.Byes.Teams {
		if err := b.Validate(); err != nil {
			return nil, err
		}
	}
	return env.Byes.Teams, nil
}
