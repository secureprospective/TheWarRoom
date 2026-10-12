package ingestion

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type RawCalendarEvent struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Start   string `json:"start_time"`
	End     string `json:"end_time"`
	Happens string `json:"happens"`
}

func (r RawCalendarEvent) Validate() error {
	if r.ID == "" || r.Type == "" {
		return fmt.Errorf("calendar: missing event identity")
	}
	for _, v := range []string{r.Start, r.End} {
		if v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err != nil || n < 0 {
				return fmt.Errorf("calendar: invalid event time %q", v)
			}
		}
	}
	return nil
}
func ParseCalendar(body []byte) ([]RawCalendarEvent, error) {
	if err := CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("calendar: %w", err)
	}
	var raw struct {
		Calendar *struct {
			Events MFLList[RawCalendarEvent] `json:"event"`
		} `json:"calendar"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("calendar: decode: %w", err)
	}
	if raw.Calendar == nil {
		return nil, fmt.Errorf("calendar: missing section")
	}
	out := []RawCalendarEvent(raw.Calendar.Events)
	for _, r := range out {
		if err := r.Validate(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

type RawTradeBait struct {
	Franchise string `json:"franchise_id"`
	Give      string `json:"willGiveUp"`
	Want      string `json:"inExchangeFor"`
	At        string `json:"timestamp"`
}

func (r RawTradeBait) Validate() error {
	if _, err := strconv.ParseUint(r.Franchise, 10, 64); err != nil {
		return fmt.Errorf("trade bait: franchise: %w", err)
	}
	if r.At != "" {
		if _, err := strconv.ParseInt(r.At, 10, 64); err != nil {
			return fmt.Errorf("trade bait: time: %w", err)
		}
	}
	for _, id := range strings.Split(r.Give, ",") {
		if id != "" {
			if _, err := ValidatePlayerID(id); err != nil {
				return fmt.Errorf("trade bait: %w", err)
			}
		}
	}
	return nil
}
func ParseTradeBait(body []byte) ([]RawTradeBait, error) {
	if err := CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("trade bait: %w", err)
	}
	var raw struct {
		Baits *struct {
			Rows MFLList[RawTradeBait] `json:"tradeBait"`
		} `json:"tradeBaits"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("trade bait: decode: %w", err)
	}
	if raw.Baits == nil {
		return nil, fmt.Errorf("trade bait: missing section")
	}
	rows := []RawTradeBait(raw.Baits.Rows)
	for _, r := range rows {
		if err := r.Validate(); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

type RawPlayerScore struct {
	ID    string `json:"id"`
	Week  string `json:"week"`
	Score string `json:"score"`
}

func (r RawPlayerScore) Validate() error {
	if _, err := ValidatePlayerID(r.ID); err != nil {
		return fmt.Errorf("player scores: %w", err)
	}
	if r.Week != "YTD" {
		if w, err := strconv.Atoi(r.Week); err != nil || w < 1 || w > 22 {
			return fmt.Errorf("player scores: invalid week")
		}
	}
	if f, err := strconv.ParseFloat(r.Score, 64); err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("player scores: invalid score")
	}
	return nil
}
func ParsePlayerScores(body []byte) ([]RawPlayerScore, error) {
	if err := CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("player scores: %w", err)
	}
	var raw struct {
		Scores *struct {
			Week string                  `json:"week"`
			Rows MFLList[RawPlayerScore] `json:"playerScore"`
		} `json:"playerScores"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("player scores: decode: %w", err)
	}
	if raw.Scores == nil {
		return nil, fmt.Errorf("player scores: missing section")
	}
	rows := []RawPlayerScore(raw.Scores.Rows)
	for i := range rows {
		if rows[i].Week == "" {
			rows[i].Week = raw.Scores.Week
		}
		if err := rows[i].Validate(); err != nil {
			return nil, err
		}
	}
	return rows, nil
}
