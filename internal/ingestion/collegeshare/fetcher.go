// Package collegeshare is the Layer 1 fetcher for the CollegeProductionShare scouting
// signal — a player's share of his college team's production (a well-established
// breakout predictor for skill-position prospects). It reads the CollegeFootballData
// (CFBD) season player-stats endpoint and returns RAW, gsis-keyed records; it does
// NOT score and does NOT apply position-specific normalization (the engine's job —
// the scouting Profile is position-blind, Approach A).
//
// THE ROOKIE KEYING BRIDGE. Every other offense fetcher keys on nflverse gsis ids,
// but CFBD keys college players on an espn-style athlete id (its playerId), and a
// college prospect may have no NFL gsis yet. The bridge (verified live 2026-06-21):
// CFBD playerId == espn_id, and dynastyprocess db_playerids.csv carries both espn_id
// and gsis_id — so an espn->gsis resolver (crosswalk.Map.GSISForESPN) turns a CFBD
// player into the same gsis the engine joins every other offense signal on. The
// resolver is INJECTED (a GSISResolver func), not imported, so this fetcher stays
// decoupled from the crosswalk's concrete type and is unit-testable with a fake. A
// player whose espn id does not resolve is skipped — an ordinary miss (a college-only
// player never drafted has no gsis, and is therefore not a TheWarRoom player).
//
// THE SHARE IS THIS FETCHER'S DEFINED JOB, not engine normalization. CFBD returns
// per-player season counting stats in LONG format (one row per stat); the source map
// assigns the team-sum to this fetcher ("+ local Go team-sum"): a player's share =
// Σ(his REC|YDS) ÷ Σ(his team's REC|YDS). That within-team ratio is position-
// INDEPENDENT (it is the signal's definition, like nflproduction's counting stats),
// so computing it here does not usurp the engine's position-specific normalization.
//
// ZERO-LEAK (hard constraint): only clean college box-score counts and their within-
// team shares are surfaced. CFBD's PPA (points-based predicted-points) is a leak risk
// and is NEVER fetched or bound (this fetcher requests only the rushing/receiving
// counting categories); RawCollegeShare has no field to hold a fantasy or points value.
//
// TWO LIVE DATA PROPERTIES the engine inherits (verified 2026-06-21, year 2023):
//  1. PLACEHOLDER gsis. dynastyprocess constructs a non-standard gsis (e.g. "ALL092954"
//     instead of "00-00…") for a player with no official NFL id yet — i.e. an incoming
//     rookie, this signal's PRIMARY target. It is a consistent join key (a rostered
//     rookie maps MFL->the same placeholder through the same crosswalk column), so it
//     is kept, not dropped. Reconciliation to the official gsis when the NFL assigns one
//     is the already-tracked OQ-013 ramp, not this fetcher's concern.
//  2. ReceptionShare is a count ratio and is structurally in [0,1]. The YARDAGE shares
//     can be slightly NEGATIVE — negative rushing yards (trick-play losses) are real —
//     and are emitted RAW (no clamp). In practice they sit in roughly [-0.05, 1] because
//     an FBS team nets strongly positive yardage over a season; the engine clamps/ignores
//     the tiny negatives for positions where the share is irrelevant.
//
// CFBD IS AN AUTHED JSON API — it uses the shared ingestion/cfbd.go client (HTTP/1.1
// pinned; bearer; byte-capped) and a LENIENT decode into a concrete row type.
package collegeshare

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// SeasonStatsURL is the CFBD season player-stats endpoint. Fetch takes the base URL as
// an argument (tests point it at a fixture server); the year and category are query
// params appended per call.
const SeasonStatsURL = "https://api.collegefootballdata.com/stats/player/season"

// CFBD stat categories and the long-format statTypes within them this fetcher binds.
// Confirmed live 2026-06-21: receiving = [REC YDS TD LONG YPR], rushing = [CAR YDS TD
// LONG YPC]. Only the counting stats that define a production share are read; TD/LONG/
// per-attempt rates are ignored.
const (
	categoryReceiving = "receiving"
	categoryRushing   = "rushing"
	statReceptions    = "REC"
	statYards         = "YDS"
	statCarries       = "CAR"
)

// GSISResolver maps a CFBD/espn athlete id to an nflverse gsis id (ok=false on a
// miss). crosswalk.Map.GSISForESPN satisfies it; injecting the resolver instead of
// importing crosswalk keeps this fetcher decoupled and lets tests pass a fake.
type GSISResolver func(espnID string) (gsis string, ok bool)

// errEmpty guards a fetch that resolved zero gsis-keyed records — for a played season
// the CFBD feed is never legitimately empty and the bridge resolves thousands, so an
// empty result means a truncated download, a wrong year, or a silently-broken bridge,
// surfaced loudly rather than as "no college production".
var errEmpty = errors.New("collegeshare: resolved zero gsis-keyed records")

// RawCollegeShare is one player's raw college production and within-team shares for a
// season, keyed by nflverse gsis_id. Counts are the clean college box score; shares
// are the signal (team-summed here). No field references fantasy points, projected
// volume, or MFL scoring (zero-leak invariant). ESPNID and Player are carried for
// traceability/spot-checks, not as scoring inputs.
type RawCollegeShare struct {
	GSISID string
	ESPNID string
	Player string
	Team   string
	Season int

	Receptions     int
	ReceivingYards float64
	RushingYards   float64
	RushingCarries int

	ReceptionShare     float64 // receptions ÷ team receptions
	ReceivingYardShare float64 // receiving yards ÷ team receiving yards
	RushingYardShare   float64 // rushing yards ÷ team rushing yards
}

// playerAgg accumulates one CFBD player's counts across the receiving and rushing
// calls before shares are computed. Keyed by CFBD playerId (espn id).
type playerAgg struct {
	espn, name, team             string
	receptions, carries          int
	receivingYards, rushingYards float64
}

// teamTotals accumulates a college team's production, the denominator of each share.
type teamTotals struct {
	receptions     int
	receivingYards float64
	rushingYards   float64
}

// Fetch retrieves the season's receiving and rushing player stats from baseURL (two
// calls, no team param — each returns all FBS players in one long-format response),
// sums each team's production locally, computes per-player within-team shares, and
// returns them keyed by the gsis the resolver maps each CFBD player to. A player whose
// espn id does not resolve is skipped (ordinary miss). apiKey must be TrimSpace'd (the
// CT105 env carries trailing newlines). A malformed stat cell fails loud.
func Fetch(ctx context.Context, client *http.Client, baseURL, apiKey string, year int, resolve GSISResolver) (map[string]RawCollegeShare, error) {
	players := map[string]*playerAgg{}
	teams := map[string]*teamTotals{}

	if err := accumulate(ctx, client, baseURL, apiKey, year, categoryReceiving, players, teams); err != nil {
		return nil, err
	}
	if err := accumulate(ctx, client, baseURL, apiKey, year, categoryRushing, players, teams); err != nil {
		return nil, err
	}

	out := emit(players, teams, year, resolve)
	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}

// accumulate fetches one category and folds its rows into the player and team
// accumulators. Splitting the network+decode from the aggregation keeps each concern
// testable and the functions short.
func accumulate(ctx context.Context, client *http.Client, baseURL, apiKey string, year int, category string,
	players map[string]*playerAgg, teams map[string]*teamTotals) error {
	rows, err := ingestion.FetchCFBDCategory(ctx, client, baseURL, apiKey, year, category)
	if err != nil {
		return fmt.Errorf("collegeshare: %w", err)
	}
	for i := range rows {
		if err := foldRow(&rows[i], category, players, teams); err != nil {
			return err
		}
	}
	return nil
}

// foldRow attributes one long-format stat row to its player and team accumulators,
// creating either on first sight, then delegating to the per-category fold. A row with
// no player or team key cannot be attributed and is skipped.
func foldRow(r *ingestion.CFBDStatRow, category string, players map[string]*playerAgg, teams map[string]*teamTotals) error {
	espn := strings.TrimSpace(r.PlayerID)
	if ingestion.IsMissing(espn) || ingestion.IsMissing(strings.TrimSpace(r.Team)) {
		return nil
	}

	p := players[espn]
	if p == nil {
		p = &playerAgg{espn: espn, name: r.Player, team: r.Team}
		players[espn] = p
	}
	t := teams[r.Team]
	if t == nil {
		t = &teamTotals{}
		teams[r.Team] = t
	}

	if category == categoryReceiving {
		return foldReceiving(r, p, t)
	}
	return foldRushing(r, p, t)
}

// foldReceiving folds a receiving row's REC/YDS into the player and team totals; other
// receiving statTypes (TD/LONG/YPR) are ignored. A malformed stat fails loud.
func foldReceiving(r *ingestion.CFBDStatRow, p *playerAgg, t *teamTotals) error {
	switch r.StatType {
	case statReceptions:
		n, err := ingestion.CFBDInt(*r)
		if err != nil {
			return fmt.Errorf("collegeshare: %w", err)
		}
		p.receptions += n
		t.receptions += n
	case statYards:
		f, err := ingestion.CFBDFloat(*r)
		if err != nil {
			return fmt.Errorf("collegeshare: %w", err)
		}
		p.receivingYards += f
		t.receivingYards += f
	}
	return nil
}

// foldRushing folds a rushing row's CAR/YDS; carries accumulate per player only (the
// rushing share is yardage-based), team rushing yards form the share denominator.
func foldRushing(r *ingestion.CFBDStatRow, p *playerAgg, t *teamTotals) error {
	switch r.StatType {
	case statCarries:
		n, err := ingestion.CFBDInt(*r)
		if err != nil {
			return fmt.Errorf("collegeshare: %w", err)
		}
		p.carries += n
	case statYards:
		f, err := ingestion.CFBDFloat(*r)
		if err != nil {
			return fmt.Errorf("collegeshare: %w", err)
		}
		p.rushingYards += f
		t.rushingYards += f
	}
	return nil
}

// emit resolves each accumulated player to a gsis and builds the share records. A
// player whose espn id does not resolve is skipped (ordinary miss). If two distinct
// CFBD players resolve to the SAME gsis the record is ambiguous (the espn->gsis bridge
// is many-to-one in rare MFL-duplicate cases) and is dropped from the output entirely,
// mirroring the crosswalk's drop-ambiguous policy — a clean miss beats a silently
// mis-attributed college line.
func emit(players map[string]*playerAgg, teams map[string]*teamTotals, year int, resolve GSISResolver) map[string]RawCollegeShare {
	return ingestion.EmitDropAmbiguous(players,
		func(p *playerAgg) string { return p.espn },
		resolve,
		func(p *playerAgg, gsis string) RawCollegeShare {
			t := teams[p.team]
			return RawCollegeShare{
				GSISID:             gsis,
				ESPNID:             p.espn,
				Player:             p.name,
				Team:               p.team,
				Season:             year,
				Receptions:         p.receptions,
				ReceivingYards:     p.receivingYards,
				RushingYards:       p.rushingYards,
				RushingCarries:     p.carries,
				ReceptionShare:     ingestion.Share(float64(p.receptions), float64(t.receptions)),
				ReceivingYardShare: ingestion.Share(p.receivingYards, t.receivingYards),
				RushingYardShare:   ingestion.Share(p.rushingYards, t.rushingYards),
			}
		})
}
