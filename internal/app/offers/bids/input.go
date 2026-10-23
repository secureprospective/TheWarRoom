// Package bids reads and prices the owner's bid plans and the window closes they watch.
package bids

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const maxEvidenceRunes = 500

type Input struct {
	ConditionalDrop string `json:"conditionalDrop"`
	FranchiseID     string `json:"franchiseId"`
	Player          string `json:"player"`
	Salary          string `json:"salary"`
	Years           int    `json:"years"`
	Snipe           bool   `json:"snipe"`
	Beat            string `json:"beat"`
	Evidence        string `json:"evidence"`
	Closes          string `json:"closes"`
}

type Entry struct {
	Franchise string
	Entered   domain.Money // before the $10k snap
	Facts     envelope.BidFacts
}

// entry reads the typed figures into exact values; the salary snaps to $10k here (§0).
func (r Input) Entry() (Entry, error) {
	if err := ingestion.FeedFranchise(r.FranchiseID); err != nil {
		return Entry{}, fmt.Errorf("franchise: %w", err)
	}
	pid, err := playerid.New(r.Player)
	if err != nil {
		return Entry{}, fmt.Errorf("player: %w", err)
	}
	if strings.TrimSpace(r.Salary) == "" {
		return Entry{}, fmt.Errorf("enter the annual salary in millions, such as 5.25")
	}
	entered, err := domain.ParseMoneyMillions(r.Salary)
	if err != nil {
		return Entry{}, fmt.Errorf("salary %q is not an amount in millions, such as 5.25", r.Salary)
	}
	facts := envelope.BidFacts{Player: pid, Salary: domain.RoundToNearest10k(entered), Years: r.Years,
		Snipe: r.Snipe, Evidence: strings.TrimSpace(r.Evidence)}
	if r.ConditionalDrop != "" {
		facts.ConditionalDrop, err = playerid.New(r.ConditionalDrop)
		if err != nil {
			return Entry{}, fmt.Errorf("conditional drop: %w", err)
		}
	}
	facts.Points, _ = bid.Score(facts.Salary, facts.Years) // a bad length fails the length rule instead
	if strings.TrimSpace(r.Beat) != "" {
		beat, err := bid.ParsePoints(r.Beat)
		if err != nil {
			return Entry{}, fmt.Errorf("bid to beat: %w", err)
		}
		facts.Beat = &beat
	}
	if facts.Closes, err = ClosesText(r.Closes); err != nil {
		return Entry{}, err
	}
	if utf8.RuneCountInString(facts.Evidence) > maxEvidenceRunes {
		return Entry{}, fmt.Errorf("evidence is longer than %d characters", maxEvidenceRunes)
	}
	return Entry{Franchise: r.FranchiseID, Entered: entered, Facts: facts}, nil
}

func ClosesText(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("window close time %q is not a date and time", raw)
	}
	return at.UTC().Format(time.RFC3339), nil
}

// ClosesNote is what the watcher does with the window close the owner typed.
func ClosesNote(closes string) string {
	if closes == "" {
		return "No closing time is set: the app keeps watching the rosters until the player shows up on one."
	}
	return "After this time, the next roster check that does not show him on your roster ends the plan " +
		"as awarded off-app."
}
