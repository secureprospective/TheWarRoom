// Package college loads CollegeFootballData's season stats into the measure store. It is the one
// college reader: every FBS player's stats for a season arrive in one bearer-authed call, as one
// row per stat. Each row's field is "player_season.<category>.<statType>"; the team totals a
// college share needs are the team's rows summed, under "team_season.<category>.<statType>".
// Which of them become measures is source_fields' business.
package college

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/measures"
)

// Source is CFBD's id in sources.csv.
const Source = "cfbd"

// SeasonStatsURL is the season stats endpoint, without its query.
const SeasonStatsURL = "https://api.collegefootballdata.com/stats/player/season"

// IDType is the id CFBD keys players on: ESPN's, which carries from college into the NFL.
const IDType = "espn"

// maxBytes caps one season's body; a season is about 23 MB.
const maxBytes = 96 << 20

// The text fields each player row carries besides its stat.
const (
	fieldTeam       = "player_season.team"
	fieldConference = "player_season.conference"
)

type row struct {
	Season     int    `json:"season"`
	PlayerID   string `json:"playerId"`
	Team       string `json:"team"`
	Conference string `json:"conference"`
	Category   string `json:"category"`
	StatType   string `json:"statType"`
	Stat       string `json:"stat"`
}

// Fetch reads one college season from baseURL (SeasonStatsURL) and returns its facts for the players keep accepts. Most FBS
// players never reach the NFL; keeping only players the directory knows stops them waiting in
// the store forever. A player who joins the directory later is picked up when the season is next
// loaded.
func Fetch(ctx context.Context, client *http.Client, baseURL string, reg *measures.Registry, apiKey string,
	season int, keep func(espnID string) bool) (measures.Batch, error) {
	body, err := ingestion.GetCFBD(ctx, client, baseURL+"?year="+strconv.Itoa(season), apiKey, maxBytes)
	if err != nil {
		return measures.Batch{}, fmt.Errorf("college: %w", err)
	}
	return Map(reg, season, body, keep)
}

// Map reads one season's body as CFBD sent it, such as an archived copy, into the batch Fetch
// returns.
func Map(reg *measures.Registry, season int, body []byte, keep func(espnID string) bool) (measures.Batch, error) {
	var rows []row
	if err := json.Unmarshal(body, &rows); err != nil {
		return measures.Batch{}, fmt.Errorf("college: decode %d: %w", season, err)
	}
	sum := sha256.Sum256(body)
	facts, err := mapRows(reg, season, rows, keep)
	if err != nil {
		return measures.Batch{}, err
	}
	return measures.Batch{Source: Source, BodySHA256: hex.EncodeToString(sum[:]), Facts: facts}, nil
}

func mapRows(reg *measures.Registry, season int, rows []row, keep func(string) bool) ([]measures.Fact, error) {
	mapped := func(field string) bool { _, ok := reg.Field(Source, field); return ok }
	totals := map[string]map[string]float64{} // team → team field → sum
	for _, r := range rows {
		field := "team_season." + r.Category + "." + r.StatType
		if !mapped(field) {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(r.Stat), 64)
		if err != nil {
			return nil, fmt.Errorf("college: %d %s %s for %s is %q", season, r.Category, r.StatType, r.PlayerID, r.Stat)
		}
		if totals[r.Team] == nil {
			totals[r.Team] = map[string]float64{}
		}
		totals[r.Team][field] += v
	}

	type key struct{ id, field string }
	at := map[key]int{}
	var out []measures.Fact
	put := func(id, field, raw string) {
		f := measures.Fact{IDType: IDType, ID: id, Season: season, Field: field, Raw: raw}
		if i, dup := at[key{id, field}]; dup {
			out[i] = f
			return
		}
		at[key{id, field}] = len(out)
		out = append(out, f)
	}
	for _, r := range rows {
		if r.PlayerID == "" || !keep(r.PlayerID) {
			continue
		}
		if field := "player_season." + r.Category + "." + r.StatType; mapped(field) {
			put(r.PlayerID, field, strings.TrimSpace(r.Stat))
		}
		if r.Team != "" && mapped(fieldTeam) {
			put(r.PlayerID, fieldTeam, r.Team)
		}
		if r.Conference != "" && mapped(fieldConference) {
			put(r.PlayerID, fieldConference, r.Conference)
		}
		for field, v := range totals[r.Team] {
			put(r.PlayerID, field, strconv.FormatFloat(v, 'f', -1, 64))
		}
	}
	return out, nil
}
