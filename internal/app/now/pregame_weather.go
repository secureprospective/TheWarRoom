package now

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

const (
	nwsUserAgent       = "TheWarRoom (lab)"
	defaultWeatherTTL  = 30 * time.Minute
	weatherFailureHold = 2 * time.Minute
)

//go:embed nfl_stadiums.json
var nflStadiumsJSON []byte

type Stadium struct {
	Team  string  `json:"team"`
	Name  string  `json:"name"`
	City  string  `json:"city"`
	State string  `json:"state"`
	Lat   float64 `json:"lat"`
	Lon   float64 `json:"lon"`
	Dome  bool    `json:"dome"`
}

func (s Stadium) Validate() error {
	if s.Team == "" || s.Name == "" || s.City == "" {
		return fmt.Errorf("stadium: team, name and city are required")
	}
	if s.Lat < -90 || s.Lat > 90 || s.Lon < -180 || s.Lon > 180 {
		return fmt.Errorf("stadium %s: coordinates out of range", s.Team)
	}
	return nil
}

// NFLStadiums decodes the embedded stadium table beside livegame_sources.json.
func NFLStadiums() ([]Stadium, error) {
	var raw []Stadium
	if err := json.Unmarshal(nflStadiumsJSON, &raw); err != nil {
		return nil, fmt.Errorf("nfl stadiums: %w", err)
	}
	for _, s := range raw {
		if err := s.Validate(); err != nil {
			return nil, err
		}
	}
	return raw, nil
}

type RawNWSPoints struct {
	Properties struct {
		ForecastHourly string `json:"forecastHourly"`
	} `json:"properties"`
}

func (r RawNWSPoints) Validate(expectedHost string) error {
	u, err := url.Parse(r.Properties.ForecastHourly)
	if err != nil {
		return fmt.Errorf("nws forecastHourly: %w", err)
	}
	if u.Scheme != "https" || u.Host == "" || u.User != nil {
		return fmt.Errorf("nws forecastHourly: requires HTTPS without credentials")
	}
	if expectedHost != "" && !strings.EqualFold(u.Host, expectedHost) {
		return fmt.Errorf("nws forecastHourly: unexpected host %q", u.Host)
	}
	return nil
}

type RawNWSPeriod struct {
	StartTime     string `json:"startTime"`
	EndTime       string `json:"endTime"`
	Temperature   *int   `json:"temperature"`
	TempUnit      string `json:"temperatureUnit"`
	WindSpeed     string `json:"windSpeed"`
	WindDirection string `json:"windDirection"`
	ShortForecast string `json:"shortForecast"`
	Precip        *struct {
		Value *int `json:"value"`
	} `json:"probabilityOfPrecipitation"`
}

func (p RawNWSPeriod) Validate() error {
	start, err := time.Parse(time.RFC3339, p.StartTime)
	if err != nil {
		return fmt.Errorf("nws period startTime: %w", err)
	}
	end, err := time.Parse(time.RFC3339, p.EndTime)
	if err != nil {
		return fmt.Errorf("nws period endTime: %w", err)
	}
	if !end.After(start) {
		return fmt.Errorf("nws period: end before start")
	}
	if p.Temperature == nil {
		return fmt.Errorf("nws period: missing temperature")
	}
	if p.TempUnit != "F" && p.TempUnit != "C" {
		return fmt.Errorf("nws period: unsupported unit %q", p.TempUnit)
	}
	return nil
}

type RawNWSForecast struct {
	Properties struct {
		Periods []RawNWSPeriod `json:"periods"`
	} `json:"properties"`
}

func (r RawNWSForecast) Validate() error {
	if r.Properties.Periods == nil {
		return fmt.Errorf("nws forecast: periods missing")
	}
	return nil
}

type weatherEntry struct {
	weather *livegame.Weather
	expires time.Time
}

// WeatherCache resolves pre-game stadium weather with a 30-minute NWS cache.
type WeatherCache struct {
	Transport livegame.Transport
	PointsURL string
	TTL       time.Duration
	Now       func() time.Time

	mu       sync.Mutex
	stadiums []Stadium
	gridURLs map[string]string
	entries  map[string]weatherEntry
}

func NewWeatherCache(transport livegame.Transport, pointsURL string) (*WeatherCache, error) {
	stadiums, err := NFLStadiums()
	if err != nil {
		return nil, err
	}
	return &WeatherCache{
		Transport: transport,
		PointsURL: pointsURL,
		TTL:       defaultWeatherTTL,
		Now:       time.Now,
		stadiums:  stadiums,
		gridURLs:  map[string]string{},
		entries:   map[string]weatherEntry{},
	}, nil
}

// Resolve populates g.Weather following the indoor -> NWS US outdoor -> ESPN abroad -> omitted order.
func (c *WeatherCache) Resolve(ctx context.Context, g *livegame.Game) {
	if c == nil || g == nil {
		return
	}
	if g.Venue != nil && g.Venue.Indoor {
		g.Weather = &livegame.Weather{Text: "Indoors"}
		return
	}
	if isAbroadVenue(g.Venue) {
		g.Weather = validESPNWeather(g.Weather)
		return
	}
	st, matched := c.matchStadium(g)
	if !matched {
		g.Weather = validESPNWeather(g.Weather)
		return
	}
	if st.Dome {
		if g.Venue != nil {
			g.Venue.Indoor = true
		}
		g.Weather = &livegame.Weather{Text: "Indoors"}
		return
	}
	if g.Kickoff.IsZero() || c.Transport == nil || c.PointsURL == "" {
		g.Weather = nil
		return
	}
	g.Weather = c.nwsWeather(ctx, st, g.Kickoff)
}

func (c *WeatherCache) matchStadium(g *livegame.Game) (Stadium, bool) {
	if g.Venue != nil && (g.Venue.Name != "" || g.Venue.City != "") {
		if st, ok := c.matchByVenue(g.Venue); ok {
			return st, true
		}
		if g.Venue.State == "" || !isUSState(g.Venue.State) {
			return Stadium{}, false
		}
	}
	home := livegame.Canonical(g.Home.Abbreviation)
	for _, st := range c.stadiums {
		if st.Team == home {
			return st, true
		}
	}
	return Stadium{}, false
}

func (c *WeatherCache) matchByVenue(v *livegame.Venue) (Stadium, bool) {
	name := strings.ToLower(strings.TrimSpace(v.Name))
	city := strings.ToLower(strings.TrimSpace(v.City))
	state := strings.ToUpper(strings.TrimSpace(v.State))
	for _, st := range c.stadiums {
		if name != "" && strings.EqualFold(st.Name, name) {
			return st, true
		}
		if city != "" && strings.EqualFold(st.City, city) && (state == "" || st.State == state) {
			return st, true
		}
	}
	return Stadium{}, false
}

func (c *WeatherCache) nwsWeather(
	ctx context.Context, st Stadium, kickoff time.Time,
) *livegame.Weather {
	coord := fmt.Sprintf("%.4f,%.4f", st.Lat, st.Lon)
	hour := kickoff.UTC().Truncate(time.Hour).Format(time.RFC3339)
	key := coord + "|" + hour
	now := c.clock()

	c.mu.Lock()
	if hit, ok := c.entries[key]; ok && now.Before(hit.expires) {
		c.mu.Unlock()
		return cloneWeather(hit.weather)
	}
	gridURL := c.gridURLs[coord]
	c.mu.Unlock()

	weather, nextGrid, err := c.fetchNWS(ctx, st, gridURL, kickoff)
	ttl := c.ttl()
	if err != nil {
		ttl = weatherFailureHold
		weather = nil
	}

	c.mu.Lock()
	if nextGrid != "" {
		c.gridURLs[coord] = nextGrid
	}
	c.entries[key] = weatherEntry{weather: cloneWeather(weather), expires: now.Add(ttl)}
	c.mu.Unlock()
	return weather
}

func (c *WeatherCache) fetchNWS(
	ctx context.Context, st Stadium, gridURL string, kickoff time.Time,
) (*livegame.Weather, string, error) {
	headers := map[string]string{
		"User-Agent": nwsUserAgent,
		"Accept":     "application/geo+json",
	}
	pointsURL := strings.NewReplacer(
		"{lat}", fmt.Sprintf("%.4f", st.Lat),
		"{lon}", fmt.Sprintf("%.4f", st.Lon),
	).Replace(c.PointsURL)
	parsedPoints, err := url.Parse(pointsURL)
	if err != nil {
		return nil, "", fmt.Errorf("nws points url: %w", err)
	}
	if gridURL == "" {
		body, err := livegame.FetchHTTPSWithHeaders(ctx, c.Transport, pointsURL, headers)
		if err != nil {
			return nil, "", fmt.Errorf("nws points: %w", err)
		}
		var pts RawNWSPoints
		if err := json.Unmarshal(body, &pts); err != nil {
			return nil, "", fmt.Errorf("nws points decode: %w", err)
		}
		if err := pts.Validate(parsedPoints.Host); err != nil {
			return nil, "", err
		}
		gridURL = pts.Properties.ForecastHourly
	}
	body, err := livegame.FetchHTTPSWithHeaders(ctx, c.Transport, gridURL, headers)
	if err != nil {
		return nil, gridURL, fmt.Errorf("nws hourly: %w", err)
	}
	weather, err := ParseNWSHourly(body, kickoff)
	if err != nil {
		return nil, gridURL, err
	}
	return weather, gridURL, nil
}

// ParseNWSHourly selects the NWS hourly period covering kickoff and validates its fields.
func ParseNWSHourly(body []byte, kickoff time.Time) (*livegame.Weather, error) {
	var raw RawNWSForecast
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("nws hourly decode: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return nil, err
	}
	target := kickoff.UTC()
	for _, p := range raw.Properties.Periods {
		if p.Validate() != nil {
			continue
		}
		start, _ := time.Parse(time.RFC3339, p.StartTime)
		end, _ := time.Parse(time.RFC3339, p.EndTime)
		if !target.Before(start) && target.Before(end) {
			return nwsPeriodToWeather(p), nil
		}
	}
	return nil, nil
}

func nwsPeriodToWeather(p RawNWSPeriod) *livegame.Weather {
	tempF := *p.Temperature
	if p.TempUnit == "C" {
		tempF = int(math.Round(float64(tempF)*9.0/5.0 + 32.0))
	}
	wind := strings.TrimSpace(strings.TrimSpace(p.WindSpeed) + " " + strings.TrimSpace(p.WindDirection))
	w := &livegame.Weather{
		Text:   strings.TrimSpace(p.ShortForecast),
		TempF:  &tempF,
		Wind:   wind,
		Source: "NWS",
	}
	if p.Precip != nil && p.Precip.Value != nil {
		v := *p.Precip.Value
		if v >= 0 && v <= 100 {
			w.PrecipPct = &v
		}
	}
	return w
}

func (c *WeatherCache) clock() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *WeatherCache) ttl() time.Duration {
	if c.TTL > 0 {
		return c.TTL
	}
	return defaultWeatherTTL
}

func isAbroadVenue(v *livegame.Venue) bool {
	if v == nil || strings.TrimSpace(v.Country) == "" {
		return false
	}
	country := strings.ToUpper(strings.TrimSpace(v.Country))
	return !slices.Contains([]string{"US", "USA", "UNITED STATES", "UNITED STATES OF AMERICA"}, country)
}

func isUSState(state string) bool {
	code := strings.ToUpper(strings.TrimSpace(state))
	return slices.Contains([]string{
		"AL", "AK", "AZ", "AR", "CA", "CO", "CT", "DE", "DC", "FL", "GA", "HI", "ID", "IL", "IN",
		"IA", "KS", "KY", "LA", "ME", "MD", "MA", "MI", "MN", "MS", "MO", "MT", "NE", "NV", "NH",
		"NJ", "NM", "NY", "NC", "ND", "OH", "OK", "OR", "PA", "RI", "SC", "SD", "TN", "TX", "UT",
		"VT", "VA", "WA", "WV", "WI", "WY",
	}, code)
}

func validESPNWeather(w *livegame.Weather) *livegame.Weather {
	if w == nil || (w.Text == "" && w.TempF == nil) {
		return nil
	}
	return cloneWeather(w)
}

func cloneWeather(w *livegame.Weather) *livegame.Weather {
	if w == nil {
		return nil
	}
	cp := *w
	if w.TempF != nil {
		t := *w.TempF
		cp.TempF = &t
	}
	if w.PrecipPct != nil {
		p := *w.PrecipPct
		cp.PrecipPct = &p
	}
	return &cp
}
