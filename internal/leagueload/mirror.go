// Package leagueload puts a league held only as an MFL mirror onto a hosted server: it builds the
// native league from held MFL reads (never a live MFL call), then moves it with the existing
// export → inactive import → confirmed seal/activate path. MFL is only ever read.
package leagueload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Mirror is a desktop history database holding MFL reads. It is only read.
type Mirror struct{ Store *history.Store }

// Lookup returns the newest held body whose archive.HeldKey equals key.
func (m Mirror) Lookup(ctx context.Context, key string) ([]byte, bool, error) {
	u, err := url.Parse(key)
	if err != nil {
		return nil, false, fmt.Errorf("held read key: %w", err)
	}
	part := "TYPE=" + url.QueryEscape(u.Query().Get("TYPE"))
	var body []byte
	found, err := m.Store.ArchivedBodies(ctx, part, func(fetched string, held []byte, _ time.Time) bool {
		if k, kerr := archive.HeldKey(fetched); kerr == nil && k == key {
			body = held
			return true
		}
		return false
	})
	if err != nil {
		return nil, false, fmt.Errorf("read held copy: %w", err)
	}
	return body, found, nil
}

// Players is the newest held MFL player list by MFL id.
func (m Mirror) Players(ctx context.Context) ([]players.RawPlayer, error) {
	var list []players.RawPlayer
	found, err := m.Store.ArchivedBodies(ctx, "TYPE=players", func(_ string, body []byte, _ time.Time) bool {
		parsed, perr := players.Parse(ctx, body)
		if perr != nil {
			return false
		}
		list = parsed
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("read held player list: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("the mirror holds no MFL player list")
	}
	return list, nil
}

// Names is the held player list as MFL id → name, the shape the MFL league reader takes.
func Names(list []players.RawPlayer) map[string]string {
	out := make(map[string]string, len(list))
	for _, p := range list {
		out[p.ID] = p.Name
	}
	return out
}

// footballPositions maps MFL's position codes onto the scoring positions football evidence lists;
// team units, punters and coaches are not people the evidence describes.
var footballPositions = map[string]leaguerules.Position{ //nolint:gochecknoglobals // fixed lookup
	"QB": leaguerules.QB, "RB": leaguerules.RB, "WR": leaguerules.WR, "TE": leaguerules.TE, "PK": leaguerules.PK,
	"DT": leaguerules.DT, "DE": leaguerules.DE, "LB": leaguerules.LB, "CB": leaguerules.CB, "S": leaguerules.S,
}

// Football builds the season's football evidence from held MFL reads: the player directory (name,
// position, NFL team, draft year) and the NFL schedule's kickoffs. It invents no statistics.
func (m Mirror) Football(ctx context.Context, season int, at time.Time) ([]byte, string, error) {
	list, err := m.Players(ctx)
	if err != nil {
		return nil, "", err
	}
	weeks, err := m.weeks(ctx, season, at)
	if err != nil {
		return nil, "", err
	}
	return BuildFootball(list, weeks, season, at)
}

// BuildFootball turns a held player list and NFL weeks into validated football evidence bytes.
func BuildFootball(list []players.RawPlayer, weeks []leagueweek.Week, season int, at time.Time) ([]byte, string, error) {
	raw := history.RawFootballEvidence{Version: 1, Season: season, AsOf: at.UTC(),
		Source: "MFL player directory and NFL schedule, read from the commissioner's held copies", Players: map[string]history.RawFootballPlayer{}, Weeks: weeks}
	for _, p := range list {
		position, ok := footballPositions[p.Position]
		id, idErr := playerid.New(p.ID)
		if !ok || idErr != nil || strings.TrimSpace(p.Name) == "" {
			continue
		}
		facts := nativecheck.PlayerFacts{Name: p.Name, Position: position, Team: footballTeam(p.Team)}
		if year, yerr := strconv.Atoi(p.DraftYear); yerr == nil && year >= 1920 && year <= season+1 && year != 1970 {
			facts.DraftYear, facts.DraftYearHeld = year, true
		}
		player := history.RawFootballPlayer{Identity: "mfl." + id.String(), Facts: facts}
		if player.Validate() != nil {
			continue
		}
		raw.Players[id.String()] = player
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return nil, "", fmt.Errorf("encode football evidence: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return nil, "", fmt.Errorf("check football evidence: %w", err)
	}
	sum := sha256.Sum256(body)
	return body, hex.EncodeToString(sum[:]), nil
}

// footballTeam keeps MFL's NFL code when it is one; anything else is no team.
func footballTeam(team string) string {
	if len(team) < 2 || len(team) > 3 {
		return ""
	}
	for _, ch := range team {
		if ch < 'A' || ch > 'Z' {
			return ""
		}
	}
	return team
}

func (m Mirror) weeks(ctx context.Context, season int, at time.Time) ([]leagueweek.Week, error) {
	var weeks []leagueweek.Week
	for number := 1; number <= 22; number++ {
		key := fmt.Sprintf("/%d/export?JSON=1&TYPE=nflSchedule&W=%d", season, number)
		body, found, err := m.Lookup(ctx, key)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		raw, err := nflschedule.Parse(body)
		if err != nil || len(raw.Matchups) == 0 {
			continue
		}
		week, err := nflschedule.ToWeek(raw, at)
		if err != nil {
			return nil, fmt.Errorf("NFL week %d: %w", number, err)
		}
		weeks = append(weeks, week)
	}
	if len(weeks) == 0 {
		return nil, fmt.Errorf("the mirror holds no NFL schedule for %d", season)
	}
	return weeks, nil
}
