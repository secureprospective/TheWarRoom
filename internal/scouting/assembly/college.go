package assembly

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/agetrajectory"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegedefense"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegeshare"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// The college signals come from two CFBD feeds of within-team production shares, offense and
// defense. Each gives two signals: the college production share for one season, and the breakout
// age, the player's age at the first scanned season a share crossed the breakout line. The feeds
// differ only in their row type and in how a position's share is read, so one routine serves
// both. A fetch failure is an error; a player with no gsis, row, birthdate or position-defined
// share is an ordinary miss and is left out, so the rubric reads him as neutral.

// The breakout lines: offense on yardage share, IDP on the lower averaged component share.
const (
	BreakoutThreshold    = 0.20
	BreakoutThresholdIDP = 0.12
)

// collegeFeed is one CFBD production feed.
type collegeFeed[T any] struct {
	name      string
	fetch     func(ctx context.Context, client *http.Client, url, key string, year int, resolve func(string) (string, bool)) (map[string]T, error)
	share     func(T, domain.Position) (float64, bool) // the college production share
	breakout  func(T, domain.Position) (float64, bool) // the share the breakout line reads
	threshold float64
}

func offenseFeed() collegeFeed[collegeshare.RawCollegeShare] {
	return collegeFeed[collegeshare.RawCollegeShare]{
		name: "college share",
		fetch: func(ctx context.Context, c *http.Client, url, key string, year int, resolve func(string) (string, bool)) (map[string]collegeshare.RawCollegeShare, error) {
			return collegeshare.Fetch(ctx, c, url, key, year, resolve)
		},
		share:     collapseCollegeShare,
		breakout:  offenseBreakoutShare,
		threshold: BreakoutThreshold,
	}
}

func defenseFeed() collegeFeed[collegedefense.RawCollegeDefense] {
	return collegeFeed[collegedefense.RawCollegeDefense]{
		name: "college defense",
		fetch: func(ctx context.Context, c *http.Client, url, key string, year int, resolve func(string) (string, bool)) (map[string]collegedefense.RawCollegeDefense, error) {
			return collegedefense.Fetch(ctx, c, url, key, year, resolve)
		},
		share:     collapseCollegeDefense,
		breakout:  collapseCollegeDefense,
		threshold: BreakoutThresholdIDP,
	}
}

// BuildCollegeShare returns each rostered WR, TE and RB's offense production share for year.
func BuildCollegeShare(ctx context.Context, client *http.Client, statsURL, apiKey string, cw crosswalk.Map,
	year int, rosterMFLIDs []string, pos PositionLookup) (map[playerid.PlayerID]float64, error) {
	return buildShare(ctx, client, statsURL, apiKey, cw, year, rosterMFLIDs, pos, offenseFeed())
}

// BuildCollegeDefense returns each rostered defender's production share for year.
func BuildCollegeDefense(ctx context.Context, client *http.Client, statsURL, apiKey string, cw crosswalk.Map,
	year int, rosterMFLIDs []string, pos PositionLookup) (map[playerid.PlayerID]float64, error) {
	return buildShare(ctx, client, statsURL, apiKey, cw, year, rosterMFLIDs, pos, defenseFeed())
}

// BuildBreakoutAge returns each rostered WR, TE and RB's breakout age over seasons.
func BuildBreakoutAge(ctx context.Context, client *http.Client, statsURL, apiKey string, cw crosswalk.Map,
	ages map[string]agetrajectory.RawAge, seasons []int, rosterMFLIDs []string, pos PositionLookup) (map[playerid.PlayerID]float64, error) {
	return buildBreakout(ctx, client, statsURL, apiKey, cw, ages, seasons, rosterMFLIDs, pos, offenseFeed())
}

// BuildBreakoutAgeIDP returns each rostered defender's breakout age over seasons.
func BuildBreakoutAgeIDP(ctx context.Context, client *http.Client, statsURL, apiKey string, cw crosswalk.Map,
	ages map[string]agetrajectory.RawAge, seasons []int, rosterMFLIDs []string, pos PositionLookup) (map[playerid.PlayerID]float64, error) {
	return buildBreakout(ctx, client, statsURL, apiKey, cw, ages, seasons, rosterMFLIDs, pos, defenseFeed())
}

func buildShare[T any](ctx context.Context, client *http.Client, url, key string, cw crosswalk.Map, year int,
	rosterMFLIDs []string, pos PositionLookup, feed collegeFeed[T]) (map[playerid.PlayerID]float64, error) {
	if client == nil || pos == nil {
		return nil, fmt.Errorf("assembly: %s needs a client and a position lookup", feed.name)
	}
	rows, err := feed.fetch(ctx, client, url, key, year, cw.GSISForESPN)
	if err != nil {
		return nil, fmt.Errorf("assembly: fetch %s: %w", feed.name, err)
	}
	return rosterValues(cw, rosterMFLIDs, pos, func(gsis string, p domain.Position) (float64, bool) {
		row, ok := rows[gsis]
		if !ok {
			return 0, false
		}
		return feed.share(row, p)
	}), nil
}

func buildBreakout[T any](ctx context.Context, client *http.Client, url, key string, cw crosswalk.Map,
	ages map[string]agetrajectory.RawAge, seasons []int, rosterMFLIDs []string, pos PositionLookup,
	feed collegeFeed[T]) (map[playerid.PlayerID]float64, error) {
	if client == nil || pos == nil || len(seasons) == 0 {
		return nil, fmt.Errorf("assembly: %s breakout needs a client, a position lookup and a season", feed.name)
	}
	scan := slices.Sorted(slices.Values(seasons))
	bySeason := make(map[int]map[string]T, len(scan))
	for _, yr := range scan {
		rows, err := feed.fetch(ctx, client, url, key, yr, cw.GSISForESPN)
		if err != nil {
			return nil, fmt.Errorf("assembly: fetch %s season %d: %w", feed.name, yr, err)
		}
		bySeason[yr] = rows
	}
	return rosterValues(cw, rosterMFLIDs, pos, func(gsis string, p domain.Position) (float64, bool) {
		age, ok := ages[gsis]
		if !ok {
			return 0, false
		}
		share := func(row T) (float64, bool) { return feed.breakout(row, p) }
		return earliestBreakout(scan, bySeason, gsis, age.BirthDate, share, feed.threshold)
	}), nil
}

// rosterValues maps each rostered player with a gsis and a position through value.
func rosterValues(cw crosswalk.Map, rosterMFLIDs []string, pos PositionLookup,
	value func(gsis string, p domain.Position) (float64, bool)) map[playerid.PlayerID]float64 {
	out := make(map[playerid.PlayerID]float64, len(rosterMFLIDs))
	for _, mfl := range rosterMFLIDs {
		pid, err := playerid.New(mfl)
		if err != nil {
			continue
		}
		gsis, ok := cw.Lookup(pid)
		if !ok {
			continue
		}
		p, ok := pos.Position(mfl)
		if !ok {
			continue
		}
		if v, ok := value(gsis, p); ok {
			out[pid] = v
		}
	}
	return out
}

// earliestBreakout is the player's age on September 1 of the first season, in ascending order,
// whose share reaches threshold. A position the feed has no share for never breaks out.
func earliestBreakout[T any](seasons []int, bySeason map[int]map[string]T, gsis string, birth time.Time,
	share func(T) (float64, bool), threshold float64) (float64, bool) {
	for _, yr := range seasons {
		row, ok := bySeason[yr][gsis]
		if !ok {
			continue
		}
		s, ok := share(row)
		if !ok {
			return 0, false
		}
		if s < threshold {
			continue
		}
		age := time.Date(yr, time.September, 1, 0, 0, 0, 0, time.UTC).Sub(birth).Hours() / 24 / 365.25
		if math.IsNaN(age) || math.IsInf(age, 0) || age < 0 {
			return 0, false
		}
		return age, true
	}
	return 0, false
}

// collapseCollegeShare is the offense production share: receiving yards at WR and TE, and
// 0.70 rushing + 0.30 receiving yards at RB. Other positions have none in this feed.
func collapseCollegeShare(rc collegeshare.RawCollegeShare, pos domain.Position) (float64, bool) {
	var share float64
	switch pos {
	case domain.PosWR, domain.PosTE:
		share = rc.ReceivingYardShare
	case domain.PosRB:
		share = 0.70*rc.RushingYardShare + 0.30*rc.ReceivingYardShare
	case domain.PosQB, domain.PosK, domain.PosDE, domain.PosDT,
		domain.PosLB, domain.PosCB, domain.PosS, domain.PosFlag:
		return 0, false
	}
	if math.IsNaN(share) || math.IsInf(share, 0) {
		return 0, false
	}
	return share, true
}

// offenseBreakoutShare is the share the offense breakout line reads: receiving yards at WR and
// TE, rushing yards at RB.
func offenseBreakoutShare(rc collegeshare.RawCollegeShare, pos domain.Position) (float64, bool) {
	switch pos {
	case domain.PosWR, domain.PosTE:
		return rc.ReceivingYardShare, true
	case domain.PosRB:
		return rc.RushingYardShare, true
	case domain.PosQB, domain.PosK, domain.PosCB, domain.PosS,
		domain.PosLB, domain.PosDT, domain.PosDE, domain.PosFlag:
		return 0, false
	}
	return 0, false
}

// collapseCollegeDefense is the defensive production share, the mean of each position's
// components: CB passes defended and interceptions; S interceptions and tackles; LB tackles,
// sacks and tackles for loss; DT and DE tackles for loss and sacks.
func collapseCollegeDefense(rc collegedefense.RawCollegeDefense, pos domain.Position) (float64, bool) {
	var share float64
	switch pos {
	case domain.PosCB:
		share = mean(rc.PassDefShare, rc.InterceptionShare)
	case domain.PosS:
		share = mean(rc.InterceptionShare, rc.TackleShare)
	case domain.PosLB:
		share = mean(rc.TackleShare, rc.SackShare, rc.TFLShare)
	case domain.PosDT, domain.PosDE:
		share = mean(rc.TFLShare, rc.SackShare)
	case domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE,
		domain.PosK, domain.PosFlag:
		return 0, false
	}
	if math.IsNaN(share) || math.IsInf(share, 0) {
		return 0, false
	}
	return share, true
}

func mean(vals ...float64) float64 {
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
