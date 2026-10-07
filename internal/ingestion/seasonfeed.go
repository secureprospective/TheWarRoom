package ingestion

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func digits(s string) bool {
	return s != "" && strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' }) < 0
}

// FeedID checks an MFL numeric identifier such as a trade id.
func FeedID(raw string) error {
	if !digits(raw) {
		return fmt.Errorf("invalid feed ID %q", raw)
	}
	return nil
}

// FeedFranchise checks an MFL franchise id: always four digits ("0025").
func FeedFranchise(raw string) error {
	if len(raw) != 4 || !digits(raw) {
		return fmt.Errorf("invalid franchise ID %q", raw)
	}
	return nil
}

// FeedTime reads MFL's epoch-seconds string as a UTC time.
func FeedTime(raw string) (time.Time, error) {
	if !digits(raw) {
		return time.Time{}, fmt.Errorf("invalid timestamp %q", raw)
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("timestamp %q: %w", raw, err)
	}
	return time.Unix(seconds, 0).UTC(), nil
}

// FeedAssets distinguishes a missing required field from MFL's valid empty list.
func FeedAssets(raw *string) ([]leaguefeed.Asset, error) {
	if raw == nil {
		return nil, fmt.Errorf("missing asset list")
	}
	assets, err := leaguefeed.ParseAssets(*raw)
	if err != nil {
		return nil, fmt.Errorf("feed assets: %w", err)
	}
	return assets, nil
}

// FeedChange reads a roster move's two lists: what arrived, then what left.
func FeedChange(in, out *string) (*leaguefeed.RosterChange, error) {
	added, err := FeedAssets(in)
	if err != nil {
		return nil, fmt.Errorf("in: %w", err)
	}
	removed, err := FeedAssets(out)
	if err != nil {
		return nil, fmt.Errorf("out: %w", err)
	}
	return &leaguefeed.RosterChange{In: added, Out: removed}, nil
}
