package espnnews

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

const SourceESPN = "ESPN"

type Endpoints struct {
	LeagueNews         string
	PlayerNews         string
	PlayerNewsFallback string
	Bio                string
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		LeagueNews: "https://site.api.espn.com/apis/site/v2/sports/football/nfl/news?limit=50",
		PlayerNews: "https://site.api.espn.com/apis/fantasy/v2/games/ffl/news/players" +
			"?playerId={id}&limit=10",
		PlayerNewsFallback: "https://site.api.espn.com/apis/site/v2/sports/football/nfl/news?athlete={id}",
		Bio:                "https://site.web.api.espn.com/apis/common/v3/sports/football/nfl/athletes/{id}",
	}
}

func (e Endpoints) Validate() error {
	urls := map[string]string{
		"leagueNews":         e.LeagueNews,
		"playerNews":         e.PlayerNews,
		"playerNewsFallback": e.PlayerNewsFallback,
		"bio":                e.Bio,
	}
	for name, raw := range urls {
		u, err := url.Parse(strings.ReplaceAll(raw, "{id}", "1"))
		if err != nil {
			return fmt.Errorf("espnnews %s endpoint: %w", name, err)
		}
		if u.Scheme != "https" || u.Host == "" || u.User != nil {
			return fmt.Errorf("espnnews %s endpoint: requires HTTPS without credentials", name)
		}
	}
	return nil
}

type Bio struct {
	Height   string `json:"height,omitempty"`
	Weight   string `json:"weight,omitempty"`
	Born     string `json:"born,omitempty"`
	Age      int    `json:"age,omitempty"`
	College  string `json:"college,omitempty"`
	Draft    string `json:"draft,omitempty"`
	Headshot string `json:"headshot,omitempty"`
	Jersey   string `json:"jersey,omitempty"`
	Team     string `json:"team,omitempty"`
	Position string `json:"position,omitempty"`
	Status   string `json:"status,omitempty"`
	Injury   string `json:"injury,omitempty"`
}

type Item struct {
	ID          string    `json:"id"`
	Headline    string    `json:"headline"`
	Description string    `json:"description,omitempty"`
	Published   time.Time `json:"published"`
	URL         string    `json:"url,omitempty"`
	Source      string    `json:"source"`
	Type        string    `json:"type,omitempty"`
	Injury      bool      `json:"injury,omitempty"`
	AthleteIDs  []string  `json:"-"`
}

type RawLink struct {
	Href string `json:"href"`
}

type RawLinks struct {
	Web    RawLink `json:"web"`
	Mobile RawLink `json:"mobile"`
}

type RawCategory struct {
	Type      string      `json:"type"`
	AthleteID json.Number `json:"athleteId"`
	Athlete   struct {
		ID json.Number `json:"id"`
	} `json:"athlete"`
}

type RawSiteArticle struct {
	ID           json.Number   `json:"id"`
	Type         string        `json:"type"`
	Headline     string        `json:"headline"`
	Description  string        `json:"description"`
	Published    string        `json:"published"`
	LastModified string        `json:"lastModified"`
	Links        RawLinks      `json:"links"`
	Categories   []RawCategory `json:"categories"`
}

type RawLeagueNews struct {
	Articles []RawSiteArticle `json:"articles"`
}

func (r RawLeagueNews) Validate() error {
	if r.Articles == nil {
		return fmt.Errorf("espnnews: articles array is missing")
	}
	return nil
}

type RawFantasyEntry struct {
	ID               json.Number `json:"id"`
	Type             string      `json:"type"`
	Headline         string      `json:"headline"`
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	Story            string      `json:"story"`
	Published        string      `json:"published"`
	LastModified     string      `json:"lastModified"`
	OriginallyPosted string      `json:"originallyPosted"`
	Links            RawLinks    `json:"links"`
	PlayerID         json.Number `json:"playerId"`
}

type RawFantasyNews struct {
	Feed []RawFantasyEntry `json:"feed"`
}

func (r RawFantasyNews) Validate() error {
	if r.Feed == nil {
		return fmt.Errorf("espnnews: fantasy feed array is missing")
	}
	return nil
}

type RawAthleteBio struct {
	Athlete struct {
		ID            string `json:"id"`
		DisplayHeight string `json:"displayHeight"`
		DisplayWeight string `json:"displayWeight"`
		DisplayDOB    string `json:"displayDOB"`
		Age           int    `json:"age"`
		DisplayDraft  string `json:"displayDraft"`
		Jersey        string `json:"jersey"`
		College       struct {
			Name      string `json:"name"`
			ShortName string `json:"shortName"`
		} `json:"college"`
		Headshot struct {
			Href string `json:"href"`
		} `json:"headshot"`
		Team struct {
			Abbreviation string `json:"abbreviation"`
		} `json:"team"`
		Position struct {
			Abbreviation string `json:"abbreviation"`
		} `json:"position"`
		Status struct {
			Name string `json:"name"`
		} `json:"status"`
		Injuries []struct {
			Status string `json:"status"`
		} `json:"injuries"`
	} `json:"athlete"`
}

func (r RawAthleteBio) Validate() error {
	if strings.TrimSpace(r.Athlete.ID) == "" || !allDigits(r.Athlete.ID) {
		return fmt.Errorf("espnnews: athlete.id is missing or invalid")
	}
	return nil
}

func ParseLeagueNews(body []byte) ([]Item, error) {
	var raw RawLeagueNews
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("espnnews: decode league news: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return nil, err
	}
	return buildSiteItems(raw.Articles), nil
}

func ParseFantasyNews(body []byte) ([]Item, error) {
	var raw RawFantasyNews
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("espnnews: decode fantasy news: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(raw.Feed))
	seen := make(map[string]bool, len(raw.Feed))
	for _, entry := range raw.Feed {
		item, ok := fantasyEntryToItem(entry)
		if !ok || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		out = append(out, item)
	}
	sortItemsNewest(out)
	return out, nil
}

func ParseBio(body []byte) (Bio, error) {
	var raw RawAthleteBio
	if err := json.Unmarshal(body, &raw); err != nil {
		return Bio{}, fmt.Errorf("espnnews: decode athlete bio: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return Bio{}, err
	}
	a := raw.Athlete
	college := strings.TrimSpace(a.College.Name)
	if college == "" {
		college = strings.TrimSpace(a.College.ShortName)
	}
	injury := ""
	if len(a.Injuries) > 0 {
		injury = strings.TrimSpace(a.Injuries[0].Status)
	}
	age := a.Age
	if age < 0 || age > 99 {
		age = 0
	}
	return Bio{
		Height:   strings.TrimSpace(a.DisplayHeight),
		Weight:   strings.TrimSpace(a.DisplayWeight),
		Born:     parseDOB(a.DisplayDOB),
		Age:      age,
		College:  college,
		Draft:    strings.TrimSpace(a.DisplayDraft),
		Headshot: safeHTTPSURL(a.Headshot.Href),
		Jersey:   jerseyNumber(a.Jersey),
		Team:     strings.TrimSpace(a.Team.Abbreviation),
		Position: strings.TrimSpace(a.Position.Abbreviation),
		Status:   strings.TrimSpace(a.Status.Name),
		Injury:   injury,
	}, nil
}

func FetchLeagueNews(
	ctx context.Context, transport livegame.Transport, ep Endpoints,
) ([]byte, []Item, error) {
	if err := ep.Validate(); err != nil {
		return nil, nil, err
	}
	body, err := livegame.FetchHTTPS(ctx, transport, ep.LeagueNews)
	if err != nil {
		return nil, nil, fmt.Errorf("espnnews: fetch league news: %w", err)
	}
	items, err := ParseLeagueNews(body)
	if err != nil {
		return nil, nil, err
	}
	return body, items, nil
}

// FetchPlayerNews queries the fantasy player feed first, falling back to the NFL news athlete feed.
func FetchPlayerNews(
	ctx context.Context, transport livegame.Transport, ep Endpoints, espnID string,
) ([]byte, []Item, string, error) {
	if err := ep.Validate(); err != nil {
		return nil, nil, "", err
	}
	if !allDigits(espnID) {
		return nil, nil, "", fmt.Errorf("espnnews: invalid espn player id %q", espnID)
	}
	primaryURL := strings.ReplaceAll(ep.PlayerNews, "{id}", espnID)
	body, err := livegame.FetchHTTPS(ctx, transport, primaryURL)
	if err == nil {
		if items, parseErr := ParseFantasyNews(body); parseErr == nil && len(items) > 0 {
			return body, items, "fantasy", nil
		}
	}
	fallbackURL := strings.ReplaceAll(ep.PlayerNewsFallback, "{id}", espnID)
	fbBody, fbErr := livegame.FetchHTTPS(ctx, transport, fallbackURL)
	if fbErr != nil {
		if err == nil {
			return body, []Item{}, "fantasy", nil
		}
		return nil, nil, "", fmt.Errorf("espnnews: fetch player news: %w", fbErr)
	}
	items, parseErr := ParseLeagueNews(fbBody)
	if parseErr != nil {
		return nil, nil, "", parseErr
	}
	return fbBody, items, "site", nil
}

func FetchBio(
	ctx context.Context, transport livegame.Transport, ep Endpoints, espnID string,
) ([]byte, Bio, error) {
	if err := ep.Validate(); err != nil {
		return nil, Bio{}, err
	}
	if !allDigits(espnID) {
		return nil, Bio{}, fmt.Errorf("espnnews: invalid espn player id %q", espnID)
	}
	rawURL := strings.ReplaceAll(ep.Bio, "{id}", espnID)
	body, err := livegame.FetchHTTPS(ctx, transport, rawURL)
	if err != nil {
		return nil, Bio{}, fmt.Errorf("espnnews: fetch bio: %w", err)
	}
	bio, err := ParseBio(body)
	if err != nil {
		return nil, Bio{}, err
	}
	return body, bio, nil
}

func buildSiteItems(articles []RawSiteArticle) []Item {
	out := make([]Item, 0, len(articles))
	seen := make(map[string]bool, len(articles))
	for _, a := range articles {
		headline := cleanText(a.Headline)
		if headline == "" {
			continue
		}
		id := strings.TrimSpace(a.ID.String())
		if id == "" {
			id = headline
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		desc := cleanText(a.Description)
		if desc == headline {
			desc = ""
		}
		kind := strings.TrimSpace(a.Type)
		out = append(out, Item{
			ID:          id,
			Headline:    headline,
			Description: desc,
			Published:   parseTime(a.Published, a.LastModified),
			URL:         bestLink(a.Links),
			Source:      SourceESPN,
			Type:        kind,
			Injury:      isInjuryItem(kind, headline, desc),
			AthleteIDs:  extractAthletes(a.Categories),
		})
	}
	sortItemsNewest(out)
	return out
}

func fantasyEntryToItem(e RawFantasyEntry) (Item, bool) {
	headline := cleanText(e.Headline)
	if headline == "" {
		headline = cleanText(e.Title)
	}
	if headline == "" {
		return Item{}, false
	}
	id := strings.TrimSpace(e.ID.String())
	if id == "" {
		id = headline
	}
	desc := cleanText(e.Description)
	if desc == "" || desc == headline {
		story := stripHTML(e.Story)
		if len(story) > 0 && len(story) <= 900 && story != headline {
			desc = story
		} else if desc == headline {
			desc = ""
		}
	}
	var athletes []string
	if pid := cleanAthleteID(e.PlayerID.String()); pid != "" {
		athletes = []string{pid}
	}
	kind := strings.TrimSpace(e.Type)
	return Item{
		ID:          id,
		Headline:    headline,
		Description: desc,
		Published:   parseTime(e.Published, e.LastModified, e.OriginallyPosted),
		URL:         bestLink(e.Links),
		Source:      SourceESPN,
		Type:        kind,
		Injury:      isInjuryItem(kind, headline, desc),
		AthleteIDs:  athletes,
	}, true
}

func extractAthletes(cats []RawCategory) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range cats {
		if c.Type != "athlete" {
			continue
		}
		id := cleanAthleteID(c.AthleteID.String())
		if id == "" {
			id = cleanAthleteID(c.Athlete.ID.String())
		}
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

func cleanAthleteID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || s == "0" || !allDigits(s) {
		return ""
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return ""
	}
	return strconv.FormatInt(n, 10)
}

func parseTime(values ...string) time.Time {
	for _, raw := range values {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04Z"} {
			if t, err := time.Parse(layout, s); err == nil {
				return t.UTC()
			}
		}
	}
	return time.Time{}
}

// parseDOB converts ESPN's "D/M/YYYY" (or ISO "YYYY-MM-DD") displayDOB to canonical "YYYY-MM-DD".
func parseDOB(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	for _, layout := range []string{"2/1/2006", "2006-01-02", "1/2/2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

func bestLink(links RawLinks) string {
	if u := safeWebURL(links.Web.Href); u != "" {
		return u
	}
	return safeWebURL(links.Mobile.Href)
}

func safeWebURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	return s
}

func safeHTTPSURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.Scheme != "https" {
		return ""
	}
	return s
}

func isInjuryItem(kind, headline, desc string) bool {
	if strings.Contains(strings.ToLower(kind), "injur") {
		return true
	}
	text := strings.ToLower(headline + " " + desc)
	for _, phrase := range []string{
		"ruled out", "injured reserve", "did not practice", "limited practice",
		"limited participant", "listed as questionable", "listed as doubtful",
		"questionable to play", "doubtful to play", "injury report", "concussion protocol",
	} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	for _, tok := range wordTokens(text) {
		switch tok {
		case "injury", "injuries", "injured", "questionable", "doubtful", "inactives",
			"inactive", "concussion", "hamstring", "ankle", "knee", "acl", "mcl",
			"achilles", "groin", "quad", "quadriceps", "shoulder", "calf", "surgery",
			"sprain", "sprained", "strain", "strained", "fracture", "fractured", "ir", "pup":
			return true
		}
	}
	return false
}

func wordTokens(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
}

func cleanText(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(s)), " ")
}

func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
			b.WriteByte(' ')
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return cleanText(b.String())
}

func sortItemsNewest(items []Item) {
	slices.SortFunc(items, func(a, b Item) int {
		if cmp := b.Published.Compare(a.Published); cmp != 0 {
			return cmp
		}
		return strings.Compare(b.ID, a.ID)
	})
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// jerseyNumber keeps ESPN's jersey only when it is a real one- or two-digit number.
func jerseyNumber(raw string) string {
	n := strings.TrimSpace(raw)
	if len(n) == 0 || len(n) > 2 || strings.Trim(n, "0123456789") != "" {
		return ""
	}
	return n
}
