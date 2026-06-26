// Package collegedefense is the Layer 1 fetcher for the DEFENSIVE CollegeProductionShare
// signal — a defensive prospect's share of his college team's defensive production (the
// IDP analog of the offense collegeshare breakout predictor). It reads the
// CollegeFootballData (CFBD) season player-stats endpoint and returns RAW, gsis-keyed
// records; it does NOT score and does NOT combine the component shares into a position
// number (the engine's job — the scouting Profile is position-blind, Approach A).
//
// THE SCHEMA SLOT IS MONOLITHIC, THE DEFINITION IS POSITION-DEFINED. scouting.Profile
// carries ONE CollegeProductionShare float, "position-defined upstream". The locked IDP
// rubrics define that slot from DIFFERENT within-team market shares per position:
//   - CB: PD + INT market share          (CB_Rubric §4)
//   - S:  INT + Tackle market share       (S_Rubric §4)
//   - LB: Tackle + Sack + TFL market share(LB_Rubric §4)
//   - DT: TFL + Sack market share         (DT_Rubric §4)
//   - DE: TFL + Sack market share         (DE_Rubric §4)
//
// Composing those per-position combinations is engine work. This fetcher therefore emits
// each COMPONENT within-team market share RAW (tackle, sack, TFL, PD, INT) plus the raw
// counts, and the engine selects/combines per position. The within-team ratio itself is
// position-INDEPENDENT (player stat ÷ team stat — the signal's definition, exactly like
// collegeshare's offense shares), so computing it here does not usurp normalization.
//
// THE ROOKIE KEYING BRIDGE (identical to collegeshare). CFBD keys college players on an
// espn-style athlete id (its playerId); a prospect may have no NFL gsis yet.
// dynastyprocess db_playerids.csv carries both espn_id and gsis_id, so an INJECTED
// espn->gsis resolver (crosswalk.Map.GSISForESPN satisfies it) turns a CFBD player into
// the gsis the engine joins on. Injecting the resolver keeps this fetcher decoupled and
// fake-testable. A player whose espn id does not resolve is skipped (an ordinary miss —
// a college-only player never drafted has no gsis and is not a TheWarRoom player).
//
// ZERO-LEAK (hard constraint): only clean college defensive box-score counts (tackles,
// sacks, TFL, passes defended, interceptions) and their within-team shares are surfaced.
// CFBD's PPA (points-based predicted points) is a leak risk and is NEVER fetched or
// bound — this fetcher requests only the defensive and interceptions counting categories;
// RawCollegeDefense has no field to hold a fantasy or points value.
//
// CFBD IS AN AUTHED JSON API — it uses the shared ingestion/cfbd.go client (HTTP/1.1
// pinned; bearer; byte-capped) and a LENIENT decode into a concrete row type. The CFBD
// long-format plumbing (CFBDStatRow, FetchCFBDCategory, CFBDInt/CFBDFloat, Share, and
// the EmitDropAmbiguous espn->gsis poison-drop emit) lives in the shared ingestion
// package (extracted at this 2nd CFBD long-format consumer, Codex M17); this fetcher
// owns only its category list, its accumulator types, and how it builds RawCollegeDefense.
package collegedefense

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// SeasonStatsURL is the CFBD season player-stats endpoint. Fetch takes the base URL as
// an argument (tests point it at a fixture server); year and category are query params.
const SeasonStatsURL = "https://api.collegefootballdata.com/stats/player/season"

// CFBD categories and the long-format statTypes within them this fetcher binds.
// Confirmed live 2026-06-26: defensive = [PD QB HUR SACKS SOLO TD TFL TOT];
// interceptions = [AVG INT TD YDS]. INT lives in the interceptions category, NOT
// defensive, so two calls are required (as collegeshare makes two). Only the counting
// stats the IDP rubrics' CollegeProductionShare needs are read.
const (
	categoryDefensive     = "defensive"
	categoryInterceptions = "interceptions"
	statTotalTackles      = "TOT"   // total tackles (LB/S tackle share)
	statSacks             = "SACKS" // fractional (LB/DT/DE sack share)
	statTFL               = "TFL"   // tackles for loss, fractional (LB/DT/DE)
	statPassesDefended    = "PD"    // passes defended (CB/S PD share)
	statInterceptions     = "INT"   // interceptions (CB/S INT share)
)

// GSISResolver maps a CFBD/espn athlete id to an nflverse gsis id (ok=false on a miss).
// crosswalk.Map.GSISForESPN satisfies it; injecting it keeps this fetcher decoupled.
type GSISResolver func(espnID string) (gsis string, ok bool)

// errEmpty guards a fetch that resolved zero gsis-keyed records — for a played season the
// CFBD feed is never legitimately empty and the bridge resolves thousands, so an empty
// result means a truncated download, a wrong year, or a broken bridge, surfaced loudly.
var errEmpty = errors.New("collegedefense: resolved zero gsis-keyed records")

// RawCollegeDefense is one defender's raw college defensive production and within-team
// component shares for a season, keyed by nflverse gsis_id. Counts are the clean college
// box score; each share is a component the engine combines per position. No field
// references fantasy points, projected volume, or MFL scoring (zero-leak invariant).
// ESPNID and Player are carried for traceability/spot-checks, not as scoring inputs.
type RawCollegeDefense struct {
	GSISID string
	ESPNID string
	Player string
	Team   string
	Season int

	Tackles        int
	Sacks          float64
	TacklesForLoss float64
	PassesDefended int
	Interceptions  int

	TackleShare       float64 // tackles ÷ team tackles
	SackShare         float64 // sacks ÷ team sacks
	TFLShare          float64 // TFL ÷ team TFL
	PassDefShare      float64 // passes defended ÷ team passes defended
	InterceptionShare float64 // interceptions ÷ team interceptions
}

// playerAgg accumulates one CFBD player's defensive counts across the two calls before
// shares are computed. Keyed by CFBD playerId (espn id).
type playerAgg struct {
	espn, name, team                string
	tackles, passDef, interceptions int
	sacks, tfl                      float64
}

// teamTotals accumulates a college team's defensive production — the share denominators.
type teamTotals struct {
	tackles, passDef, interceptions int
	sacks, tfl                      float64
}

// Fetch retrieves the season's defensive and interceptions player stats from baseURL
// (two calls, no team param — each returns all FBS players in one long-format response),
// sums each team's production locally, computes per-player within-team component shares,
// and returns them keyed by the gsis the resolver maps each CFBD player to. A player
// whose espn id does not resolve is skipped. apiKey must be TrimSpace'd (the CT105 env
// carries trailing newlines). A malformed stat cell fails loud.
func Fetch(ctx context.Context, client *http.Client, baseURL, apiKey string, year int, resolve GSISResolver) (map[string]RawCollegeDefense, error) {
	players := map[string]*playerAgg{}
	teams := map[string]*teamTotals{}

	if err := accumulate(ctx, client, baseURL, apiKey, year, categoryDefensive, players, teams); err != nil {
		return nil, err
	}
	if err := accumulate(ctx, client, baseURL, apiKey, year, categoryInterceptions, players, teams); err != nil {
		return nil, err
	}

	out := emit(players, teams, year, resolve)
	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}

// accumulate fetches one category and folds its rows into the player and team
// accumulators.
func accumulate(ctx context.Context, client *http.Client, baseURL, apiKey string, year int, category string,
	players map[string]*playerAgg, teams map[string]*teamTotals) error {
	rows, err := ingestion.FetchCFBDCategory(ctx, client, baseURL, apiKey, year, category)
	if err != nil {
		return fmt.Errorf("collegedefense: %w", err)
	}
	for i := range rows {
		if err := foldRow(&rows[i], players, teams); err != nil {
			return err
		}
	}
	return nil
}

// foldRow attributes one long-format stat row to its player and team accumulators,
// creating either on first sight. A row with no player or team key is skipped. Only the
// statTypes the rubrics need are folded; all others (SOLO, QB HUR, TD, YDS, AVG) ignored.
func foldRow(r *ingestion.CFBDStatRow, players map[string]*playerAgg, teams map[string]*teamTotals) error {
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

	switch r.StatType {
	case statTotalTackles:
		return foldInt(r, &p.tackles, &t.tackles)
	case statPassesDefended:
		return foldInt(r, &p.passDef, &t.passDef)
	case statInterceptions:
		return foldInt(r, &p.interceptions, &t.interceptions)
	case statSacks:
		return foldFloat(r, &p.sacks, &t.sacks)
	case statTFL:
		return foldFloat(r, &p.tfl, &t.tfl)
	}
	return nil
}

// foldInt adds an integer counting stat to both the player and team accumulators.
func foldInt(r *ingestion.CFBDStatRow, pDst, tDst *int) error {
	n, err := ingestion.CFBDInt(*r)
	if err != nil {
		return fmt.Errorf("collegedefense: %w", err)
	}
	*pDst += n
	*tDst += n
	return nil
}

// foldFloat adds a fractional counting stat (sacks/TFL come in half increments) to both
// the player and team accumulators.
func foldFloat(r *ingestion.CFBDStatRow, pDst, tDst *float64) error {
	f, err := ingestion.CFBDFloat(*r)
	if err != nil {
		return fmt.Errorf("collegedefense: %w", err)
	}
	*pDst += f
	*tDst += f
	return nil
}

// emit resolves each accumulated player to a gsis and builds the component-share records.
// A player whose espn id does not resolve is skipped. If two distinct CFBD players
// resolve to the SAME gsis the record is ambiguous and dropped entirely (collegeshare /
// crosswalk drop-ambiguous precedent — a clean miss beats a mis-attributed line).
func emit(players map[string]*playerAgg, teams map[string]*teamTotals, year int, resolve GSISResolver) map[string]RawCollegeDefense {
	return ingestion.EmitDropAmbiguous(players,
		func(p *playerAgg) string { return p.espn },
		resolve,
		func(p *playerAgg, gsis string) RawCollegeDefense {
			t := teams[p.team]
			return RawCollegeDefense{
				GSISID:            gsis,
				ESPNID:            p.espn,
				Player:            p.name,
				Team:              p.team,
				Season:            year,
				Tackles:           p.tackles,
				Sacks:             p.sacks,
				TacklesForLoss:    p.tfl,
				PassesDefended:    p.passDef,
				Interceptions:     p.interceptions,
				TackleShare:       ingestion.Share(float64(p.tackles), float64(t.tackles)),
				SackShare:         ingestion.Share(p.sacks, t.sacks),
				TFLShare:          ingestion.Share(p.tfl, t.tfl),
				PassDefShare:      ingestion.Share(float64(p.passDef), float64(t.passDef)),
				InterceptionShare: ingestion.Share(float64(p.interceptions), float64(t.interceptions)),
			}
		})
}
