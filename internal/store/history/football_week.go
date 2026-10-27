package history

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
)

// maxFootballItem caps every pinned football body: the evidence itself and each week part.
const maxFootballItem = 8 << 20

// RawFootballWeek is one NFL week of a statistic revision, imported as its own pinned item so a
// full season of weekly statistics, play-by-play and advanced rushing fits under the per-item cap.
// A revision lists its weeks by digest (RawFootballStats.Parts) instead of embedding the bodies.
type RawFootballWeek struct {
	Version int                     `json:"version"`
	Season  int                     `json:"season"`
	Week    int                     `json:"week"`
	Weekly  string                  `json:"weekly"`
	Plays   string                  `json:"plays"`
	Rushing string                  `json:"rushing,omitempty"`
	Sleeper []source.RawSleeperWeek `json:"sleeper,omitempty"`
}

// partSource returns a held week part's exact body by digest; nil means none is reachable.
type partSource func(digest string) ([]byte, error)

func decodeFootballWeek(body []byte) (RawFootballWeek, error) {
	var raw RawFootballWeek
	if len(body) > maxFootballItem {
		return raw, fmt.Errorf("a football week part exceeds 8 MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return raw, fmt.Errorf("decode football week part: %w", err)
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return raw, fmt.Errorf("a football week part must contain one JSON value")
	}
	if raw.Version != 1 || raw.Season < 1920 || raw.Season > 9999 || raw.Week < 1 || raw.Week > 22 ||
		raw.Weekly == "" || raw.Plays == "" {
		return raw, fmt.Errorf("a football week part needs version 1, a season, a week, weekly and play-by-play CSV")
	}
	if err := raw.validateSleeper(); err != nil {
		return raw, err
	}
	return raw, nil
}

// weekFiles reads one part's weekly rows and plays, refusing any row from another season or week.
// Week 0 is an inline revision's whole body, which spans weeks.
func (w RawFootballWeek) weekFiles() ([]source.PlayerWeek, []source.RawPlay, error) {
	weeks, err := source.ReadWeeks(strings.NewReader(w.Weekly))
	if err != nil {
		return nil, nil, fmt.Errorf("read pinned weekly statistics: %w", err)
	}
	if w.Week == 0 {
		plays, err := source.ReadPlays(strings.NewReader(w.Plays))
		if err != nil {
			return nil, nil, fmt.Errorf("read pinned plays: %w", err)
		}
		return weeks, plays, nil
	}
	for _, row := range weeks {
		if row.Season != w.Season || row.Week != w.Week {
			return nil, nil, fmt.Errorf("week %d part holds a row from season %d week %d", w.Week, row.Season, row.Week)
		}
	}
	plays, err := source.ReadPlays(strings.NewReader(w.Plays))
	if err != nil {
		return nil, nil, fmt.Errorf("read week %d plays: %w", w.Week, err)
	}
	season, week := strconv.Itoa(w.Season), strconv.Itoa(w.Week)
	for _, play := range plays {
		if play.Season != season || play.Week != week {
			return nil, nil, fmt.Errorf("week %d part holds a play from season %s week %s", w.Week, play.Season, play.Week)
		}
	}
	return weeks, plays, nil
}

// readParts loads a revision's week parts in order: one per week, each the evidence's season.
func (r RawFootballEvidence) readParts(raw RawFootballStats, parts partSource) ([]RawFootballWeek, error) {
	if parts == nil {
		return nil, fmt.Errorf("split football evidence needs its week parts")
	}
	out := make([]RawFootballWeek, 0, len(raw.Parts))
	seen := make(map[int]bool, len(raw.Parts))
	for _, digest := range raw.Parts {
		body, err := parts(digest)
		if err != nil {
			return nil, fmt.Errorf("load football week part: %w", err)
		}
		if footballDigest(body) != digest {
			return nil, fmt.Errorf("held football week part digest mismatch")
		}
		week, err := decodeFootballWeek(body)
		if err != nil {
			return nil, err
		}
		if week.Season != r.Season || seen[week.Week] {
			return nil, fmt.Errorf("week parts must be the evidence's season, one per week")
		}
		seen[week.Week] = true
		out = append(out, week)
	}
	return out, nil
}

// storeParts reads week parts from the history archive.
func (s *Store) storeParts(ctx context.Context) partSource {
	return func(digest string) ([]byte, error) { return s.Body(ctx, digest) }
}

// ImportFootballWeek pins one week part. It must be imported before the evidence that lists it.
func (s *Store) ImportFootballWeek(ctx context.Context, body []byte, digest string) error {
	week, err := decodeFootballWeek(body)
	if err != nil {
		return err
	}
	if _, _, err := week.weekFiles(); err != nil {
		return err
	}
	if footballDigest(body) != digest {
		return fmt.Errorf("football week part digest mismatch")
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(body); err != nil {
		return fmt.Errorf("compress football week part: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close football week part: %w", err)
	}
	if err := s.Record(ctx, archive.Fetch{URL: "urn:twr:football-week:" + digest, SHA256: digest,
		Size: int64(len(body)), Gzip: compressed.Bytes(), Status: 200, FetchedAt: s.now()}); err != nil {
		return fmt.Errorf("import football week part: %w", err)
	}
	return nil
}
