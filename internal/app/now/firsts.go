package now

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"maps"
	"net/url"
	"time"

	seasonapp "github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// Bodies is the history store's ArchivedBodies, as a seam for tests.
type Bodies func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error)

// ReadFirsts reads archived season results newer than prev.Through, newest first, and keeps
// for each week not already held the oldest copy that had it scored.
func ReadFirsts(ctx context.Context, prev FirstScores, season int, bodies Bodies) (FirstScores, error) {
	next := FirstScores{Weeks: maps.Clone(prev.Weeks), Through: prev.Through, Copies: prev.Copies}
	if next.Weeks == nil {
		next.Weeks = map[int]FirstScore{}
	}
	var lastSum [sha256.Size]byte
	var last []leaguefeed.WeekResult
	_, err := bodies(ctx, "TYPE="+weeklyresults.Export, func(src string, body []byte, at time.Time) bool {
		if !at.After(prev.Through) {
			return true
		}
		u, err := url.Parse(src)
		if err != nil || u.Query().Get("W") != "YTD" || !ingestion.SameExport(src, weeklyresults.Export, season) {
			return false
		}
		if sum := sha256.Sum256(body); last == nil || !bytes.Equal(sum[:], lastSum[:]) {
			weeks, perr := weeklyresults.Parse(body)
			if perr != nil {
				return false
			}
			last, lastSum = weeks, sum
		}
		next.Copies++
		if at.After(next.Through) {
			next.Through = at
		}
		for _, w := range last {
			if _, held := prev.Weeks[w.Week]; !held && seasonapp.WeekScored(w) {
				next.Weeks[w.Week] = FirstScore{At: at, Week: w}
			}
		}
		return false
	})
	if err != nil {
		return prev, fmt.Errorf("official stat changes: %w", err)
	}
	return next, nil
}

func RefreshFirsts(ctx context.Context, prev FirstScores, season int,
	bodies func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error), publish func(FirstScores), logFailure func(string)) {
	next, err := ReadFirsts(ctx, prev, season, bodies)
	if err != nil {
		logFailure("the war room: " + err.Error())
		return
	}
	publish(next)
}
