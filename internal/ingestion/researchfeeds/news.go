package researchfeeds

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

type RawArticle struct {
	ID        string `json:"id"`
	Headline  string `json:"headline"`
	Published string `json:"published"`
}
type RawProfile struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Player struct {
		Age string `json:"age"`
	} `json:"player"`
	News struct {
		Articles ingestion.MFLList[RawArticle] `json:"article"`
	} `json:"news"`
}
type News struct {
	ID                                      playerid.PlayerID
	Name, Age, Article, Headline, Published string
}

func (r RawProfile) Validate(want string) error {
	id, err := playerid.New(r.ID)
	if err != nil {
		return fmt.Errorf("profile id: %w", err)
	}
	if id.String() != want || strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("profile: wrong or missing player")
	}
	if r.Player.Age != "" {
		n, err := strconv.Atoi(r.Player.Age)
		if err != nil || n < 0 || n > 100 {
			return fmt.Errorf("profile: invalid age")
		}
	}
	for _, a := range r.News.Articles {
		if strings.TrimSpace(a.ID) == "" || strings.TrimSpace(a.Headline) == "" {
			return fmt.Errorf("profile: missing article identity or headline")
		}
	}
	return nil
}

// ParseNews leaves unanswered IDs absent; an empty slice means an answered player with no news.
func ParseNews(body []byte, want []string) (map[string][]News, error) {
	if err := validateNewsIDs(want); err != nil {
		return nil, err
	}
	if err := ingestion.CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("playerProfile: %w", err)
	}
	var raw struct {
		Profiles struct {
			Players ingestion.MFLList[RawProfile] `json:"playerProfile"`
		} `json:"playerProfiles"`
		Legacy ingestion.MFLList[RawProfile] `json:"playerProfile"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("profile decode: %w", err)
	}
	raw.Profiles.Players = append(raw.Profiles.Players, raw.Legacy...)
	profiles := raw.Profiles.Players
	out := make(map[string][]News, len(want))
	for _, profile := range profiles {
		if !slices.Contains(want, profile.ID) {
			return nil, fmt.Errorf("profile: unexpected player %s", profile.ID)
		}
		if _, duplicate := out[profile.ID]; duplicate {
			return nil, fmt.Errorf("profile: duplicate player %s", profile.ID)
		}
		if err := profile.Validate(profile.ID); err != nil {
			return nil, fmt.Errorf("profile %s: %w", profile.ID, err)
		}
		id, err := playerid.New(profile.ID)
		if err != nil {
			return nil, fmt.Errorf("profile: %w", err)
		}
		out[profile.ID] = []News{}
		for _, a := range profile.News.Articles {
			out[profile.ID] = append(out[profile.ID], News{ID: id, Name: profile.Name, Age: profile.Player.Age, Article: a.ID, Headline: a.Headline, Published: a.Published})
		}
	}
	return out, nil
}
func validateNewsIDs(ids []string) error {
	if len(ids) == 0 || len(ids) > 100 {
		return fmt.Errorf("playerProfile: request must contain 1–100 players")
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if _, err := playerid.New(id); err != nil {
			return fmt.Errorf("playerProfile request: %w", err)
		}
		if seen[id] {
			return fmt.Errorf("playerProfile: duplicate requested player %s", id)
		}
		seen[id] = true
	}
	return nil
}
func FetchNews(ctx context.Context, c *mfl.Client, season int, ids []string) (map[string][]News, error) {
	if err := validateNewsIDs(ids); err != nil {
		return nil, err
	}
	response, err := c.Do(ctx, mfl.Request{Type: "playerProfile", Year: strconv.Itoa(season), Params: map[string]string{"P": strings.Join(ids, ",")}})
	if err != nil {
		return nil, fmt.Errorf("playerProfile: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("playerProfile: status %d", response.StatusCode)
	}
	return ParseNews(response.Body, ids)
}

// TeamUnit uses the directory position, not IDs: leagues can create unit IDs of their own.
func TeamUnit(position string) bool {
	p := strings.ToUpper(strings.TrimSpace(position))
	return slices.Contains([]string{"DEF", "DST", "ST", "OFF"}, p) || strings.HasPrefix(p, "TM")
}
