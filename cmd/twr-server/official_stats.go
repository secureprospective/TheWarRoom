package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"log"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Official statistics: live scores are provisional, so without them no week after a league moves to the
// server ever gets a result. nflverse publishes official weekly statistics and play-by-play, rebuilt through
// the season (stat corrections included). The server reads both every officialRefresh and gives each league
// pinned to that season new football evidence holding every finished week (history.PinOfficialStats), then
// pins the league to it (setPin). Nothing changes while the finished weeks and their numbers are the same.
const (
	officialRefresh = 3 * time.Hour
	officialFirst   = 2 * time.Minute // after start, once live scores are back
	officialWeekly  = 16 << 20
	officialPlays   = 64 << 20
	officialRoster  = 32 << 20
	nflversePlayers = "https://github.com/nflverse/nflverse-data/releases/download/players/players.csv"
)

// seasonFiles is nflverse's compressed weekly statistics and play-by-play for a season, and its player list.
type seasonFiles struct{ weekly, plays, roster []byte }

// officialFiles fetches a season's files; tests serve their own.
type officialFiles func(ctx context.Context, season int) (seasonFiles, error)

func nflverseFiles(client *http.Client) officialFiles {
	return func(ctx context.Context, season int) (seasonFiles, error) {
		var out seasonFiles
		var err error
		if out.weekly, err = ingestion.Get(ctx, client, source.WeeklyURL(season), officialWeekly); err != nil {
			return out, fmt.Errorf("official weekly statistics: %w", err)
		}
		if out.plays, err = ingestion.Get(ctx, client, source.PlaysURL(season), officialPlays); err != nil {
			return out, fmt.Errorf("official play-by-play: %w", err)
		}
		if out.roster, err = ingestion.Get(ctx, client, nflversePlayers, officialRoster); err != nil {
			return out, fmt.Errorf("nflverse players: %w", err)
		}
		return out, nil
	}
}

// officialKeeper re-pins leagues to official statistics.
type officialKeeper struct {
	pins      *leaguePins
	signIns   *accountstore.Store
	directory *history.Store // holds the player directory (player_ids)
	files     officialFiles
	now       func() time.Time
	logf      func(string, ...any)
}

func startOfficialStats(ctx context.Context, pins *leaguePins, signIns *accountstore.Store) func() {
	scope, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	files, first, refresh := officialSource(nflverseFiles(&http.Client{Timeout: 5 * time.Minute}))
	k := &officialKeeper{pins: pins, signIns: signIns, directory: pins.files.family.Over(pins.files.reference),
		files: files, now: serverClock().Now, logf: log.Printf}
	go func() {
		defer close(done)
		delay := first
		for {
			timer := time.NewTimer(delay)
			select {
			case <-scope.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			k.pass(scope)
			delay = refresh
		}
	}()
	return func() { cancel(); <-done }
}

// pass reads each season some league is pinned to once, and offers it to every league on that season.
func (k *officialKeeper) pass(ctx context.Context) {
	bySeason := map[int][]string{}
	for _, league := range k.pins.files.leagues() {
		if pin := k.pins.pin(ctx, league); pin.Err == nil && pin.Football != nil {
			bySeason[pin.Football.Season()] = append(bySeason[pin.Football.Season()], league)
		}
	}
	links, err := k.directory.SourceIDs(ctx, "gsis")
	if err != nil {
		k.logf("server official stats: %v", err)
		return
	}
	for _, season := range slices.Sorted(maps.Keys(bySeason)) {
		files, err := k.files(ctx, season)
		if err != nil {
			k.logf("server official stats %d: %v; trying again in %s", season, err, officialRefresh)
			continue
		}
		at := k.now()
		for _, league := range bySeason[season] {
			k.offer(ctx, league, files, links, at)
		}
	}
}

// offer gives one league official statistics, pinning it to the new evidence when there is any.
func (k *officialKeeper) offer(ctx context.Context, league string, files seasonFiles, links map[string]string, at time.Time) {
	pin := k.pins.pin(ctx, league)
	if pin.Err != nil || pin.Football == nil {
		return
	}
	in, err := officialSeason(files, links, at)
	if err != nil {
		k.logf("server official stats %s: %v", league, err)
		return
	}
	got, err := pin.Store.PinOfficialStats(ctx, pin.Football, in)
	if err != nil {
		k.logf("server official stats %s: %v", league, err)
		return
	}
	if got.Digest == "" {
		return
	}
	if err := k.pins.advancePin(ctx, k.signIns, league, pin.Digest, got.Digest); err != nil {
		k.logf("server official stats %s: %v", league, err)
		return
	}
	k.logf("server official stats %s: pinned %s with official weeks %v%s", league, got.Digest, got.Weeks, skipped(got.Skipped))
}

func skipped(why []string) string {
	if len(why) == 0 {
		return ""
	}
	return "; not yet: " + strings.Join(why, "; ")
}

func officialSeason(files seasonFiles, links map[string]string, at time.Time) (history.OfficialSeason, error) {
	w, err := gzip.NewReader(bytes.NewReader(files.weekly))
	if err != nil {
		return history.OfficialSeason{}, fmt.Errorf("official weekly statistics: %w", err)
	}
	p, err := gzip.NewReader(bytes.NewReader(files.plays))
	if err != nil {
		return history.OfficialSeason{}, fmt.Errorf("official play-by-play: %w", err)
	}
	out := history.OfficialSeason{Weekly: w, Plays: p, At: at, GSIS: links}
	if len(files.roster) > 0 {
		out.Roster = bytes.NewReader(files.roster)
	}
	return out, nil
}
