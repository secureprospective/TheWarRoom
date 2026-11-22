package now

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

const (
	weatherUserAgent   = "TheWarRoom (lab)"
	openMeteoHost      = "api.open-meteo.com"
	defaultWeatherTTL  = 30 * time.Minute
	weatherFailureHold = 2 * time.Minute
)

//go:embed nfl_stadiums.json
var nflStadiumsJSON []byte

type Stadium struct {
	Team        string  `json:"team,omitempty"`
	Name        string  `json:"name"`
	City        string  `json:"city"`
	State       string  `json:"state,omitempty"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Dome        bool    `json:"dome"`
	Retractable bool    `json:"retractable,omitempty"`
}

func (s Stadium) Validate() error {
	if s.Name == "" || s.City == "" {
		return fmt.Errorf("stadium: name and city are required")
	}
	if s.Lat < -90 || s.Lat > 90 || s.Lon < -180 || s.Lon > 180 {
		return fmt.Errorf("stadium %s: coordinates out of range", s.Name)
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

type RawOpenMeteoHourly struct {
	Time          []string   `json:"time"`
	Temperature2m []*float64 `json:"temperature_2m"`
	WeatherCode   []*int     `json:"weather_code"`
	WindSpeed10m  []*float64 `json:"wind_speed_10m"`
	PrecipProb    []*int     `json:"precipitation_probability"`
}

type RawOpenMeteoForecast struct {
	Hourly *RawOpenMeteoHourly `json:"hourly"`
}

func (r RawOpenMeteoForecast) Validate() error {
	if r.Hourly == nil || len(r.Hourly.Time) == 0 {
		return fmt.Errorf("open-meteo forecast: hourly data missing")
	}
	n := len(r.Hourly.Time)
	if len(r.Hourly.Temperature2m) != n || len(r.Hourly.WeatherCode) != n ||
		len(r.Hourly.WindSpeed10m) != n {
		return fmt.Errorf("open-meteo forecast: hourly column length mismatch")
	}
	if len(r.Hourly.PrecipProb) != 0 && len(r.Hourly.PrecipProb) != n {
		return fmt.Errorf("open-meteo forecast: precipitation_probability length mismatch")
	}
	return nil
}

type weatherEntry struct {
	forecast *RawOpenMeteoForecast
	expires  time.Time
}

type weatherFlight struct {
	done     chan struct{}
	forecast *RawOpenMeteoForecast
}

// WeatherCache resolves pre-game stadium weather via Open-Meteo with a 30-minute per-stadium cache.
type WeatherCache struct {
	Transport   livegame.Transport
	ForecastURL string
	TTL         time.Duration
	Now         func() time.Time

	mu       sync.Mutex
	stadiums []Stadium
	entries  map[string]weatherEntry
	inflight map[string]*weatherFlight
}

func NewWeatherCache(transport livegame.Transport, forecastURL string) (*WeatherCache, error) {
	stadiums, err := NFLStadiums()
	if err != nil {
		return nil, err
	}
	return &WeatherCache{
		Transport:   transport,
		ForecastURL: forecastURL,
		TTL:         defaultWeatherTTL,
		Now:         time.Now,
		stadiums:    stadiums,
		entries:     map[string]weatherEntry{},
		inflight:    map[string]*weatherFlight{},
	}, nil
}

// Resolve populates g.Weather following the indoor -> Open-Meteo -> omit order.
func (c *WeatherCache) Resolve(ctx context.Context, g *livegame.Game) {
	if c == nil || g == nil {
		return
	}
	if g.Venue != nil && g.Venue.Indoor {
		g.Weather = &livegame.Weather{Text: "Indoors"}
		return
	}
	st, matched := c.matchStadium(g)
	if !matched {
		g.Weather = nil
		return
	}
	if st.Dome && !st.Retractable {
		if g.Venue != nil {
			g.Venue.Indoor = true
		}
		g.Weather = &livegame.Weather{Text: "Indoors"}
		return
	}
	if g.Kickoff.IsZero() || c.Transport == nil || c.ForecastURL == "" {
		g.Weather = nil
		return
	}
	g.Weather = c.openMeteoWeather(ctx, st, g.Kickoff)
}

func (c *WeatherCache) matchStadium(g *livegame.Game) (Stadium, bool) {
	if g.Venue != nil && (strings.TrimSpace(g.Venue.Name) != "" || strings.TrimSpace(g.Venue.City) != "") {
		return c.matchByVenue(g.Venue)
	}
	home := livegame.Canonical(g.Home.Abbreviation)
	for _, st := range c.stadiums {
		if st.Team != "" && st.Team == home {
			return st, true
		}
	}
	return Stadium{}, false
}

func (c *WeatherCache) matchByVenue(v *livegame.Venue) (Stadium, bool) {
	name := normalizePlace(v.Name)
	if name != "" {
		if st, ok := c.matchByName(name); ok {
			return st, true
		}
	}
	city := normalizePlace(v.City)
	if city == "" {
		return Stadium{}, false
	}
	state := strings.ToUpper(strings.TrimSpace(v.State))
	var match Stadium
	matched := false
	for _, st := range c.stadiums {
		if normalizePlace(st.City) != city {
			continue
		}
		if state != "" && st.State != "" && st.State != state {
			continue
		}
		if matched && !strings.EqualFold(match.Name, st.Name) {
			return Stadium{}, false
		}
		match, matched = st, true
	}
	return match, matched
}

func (c *WeatherCache) matchByName(name string) (Stadium, bool) {
	for _, st := range c.stadiums {
		if normalizePlace(st.Name) == name {
			return st, true
		}
	}
	for _, st := range c.stadiums {
		stName := normalizePlace(st.Name)
		if len(stName) >= 8 && strings.Contains(name, stName) {
			return st, true
		}
	}
	return Stadium{}, false
}

func (c *WeatherCache) openMeteoWeather(
	ctx context.Context, st Stadium, kickoff time.Time,
) *livegame.Weather {
	if ctx.Err() != nil {
		return nil
	}
	key := fmt.Sprintf("%.4f,%.4f", st.Lat, st.Lon)
	forecast, flight, ok := c.claimWeather(ctx, key)
	if flight != nil {
		forecast, ok = c.fillWeather(ctx, key, st, flight), true
	}
	if !ok {
		return nil
	}
	return selectOpenMeteoSlot(forecast, kickoff)
}

// claimWeather answers from the cache or from another caller's fetch already in flight. When neither has
// it, it registers a new flight that the caller must fill. ok is false when ctx ended while waiting.
func (c *WeatherCache) claimWeather(
	ctx context.Context, key string,
) (forecast *RawOpenMeteoForecast, flight *weatherFlight, ok bool) {
	c.mu.Lock()
	if hit, found := c.entries[key]; found && c.clock().Before(hit.expires) {
		c.mu.Unlock()
		return hit.forecast, nil, true
	}
	if other, found := c.inflight[key]; found {
		c.mu.Unlock()
		select {
		case <-other.done:
			return other.forecast, nil, true
		case <-ctx.Done():
			return nil, nil, false
		}
	}
	flight = &weatherFlight{done: make(chan struct{})}
	if c.inflight == nil {
		c.inflight = map[string]*weatherFlight{}
	}
	c.inflight[key] = flight
	c.mu.Unlock()
	return nil, flight, false
}

// fillWeather fetches for a claimed flight and always releases its waiters, even on panic. A failure is
// held briefly; a failure caused by this caller's own cancelled ctx is never cached.
func (c *WeatherCache) fillWeather(
	ctx context.Context, key string, st Stadium, flight *weatherFlight,
) (forecast *RawOpenMeteoForecast) {
	defer func() {
		c.mu.Lock()
		flight.forecast = forecast
		delete(c.inflight, key)
		close(flight.done)
		c.mu.Unlock()
	}()
	fetched, err := c.fetchOpenMeteoForecast(ctx, st)
	ttl := c.ttl()
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		fetched, ttl = nil, weatherFailureHold
	}
	c.mu.Lock()
	c.storeWeatherLocked(key, fetched, ttl)
	c.mu.Unlock()
	return fetched
}

func (c *WeatherCache) storeWeatherLocked(
	key string, forecast *RawOpenMeteoForecast, ttl time.Duration,
) {
	writeAt := c.clock()
	if c.entries == nil {
		c.entries = map[string]weatherEntry{}
	}
	for k, entry := range c.entries {
		if !writeAt.Before(entry.expires) {
			delete(c.entries, k)
		}
	}
	c.entries[key] = weatherEntry{forecast: forecast, expires: writeAt.Add(ttl)}
}

func (c *WeatherCache) fetchOpenMeteoForecast(
	ctx context.Context, st Stadium,
) (*RawOpenMeteoForecast, error) {
	reqURL := strings.NewReplacer(
		"{lat}", fmt.Sprintf("%.4f", st.Lat),
		"{lon}", fmt.Sprintf("%.4f", st.Lon),
	).Replace(c.ForecastURL)
	if err := validateOpenMeteoURL(reqURL); err != nil {
		return nil, err
	}
	headers := map[string]string{"User-Agent": weatherUserAgent}
	body, err := livegame.FetchHTTPSWithHeaders(ctx, c.Transport, reqURL, headers)
	if err != nil {
		return nil, fmt.Errorf("open-meteo hourly: %w", err)
	}
	return parseOpenMeteoForecast(body)
}

func validateOpenMeteoURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("open-meteo url: %w", err)
	}
	if u.Scheme != "https" || u.User != nil || !strings.EqualFold(u.Host, openMeteoHost) {
		return fmt.Errorf("open-meteo url: requires https://%s without credentials", openMeteoHost)
	}
	return nil
}

func parseOpenMeteoForecast(body []byte) (*RawOpenMeteoForecast, error) {
	var raw RawOpenMeteoForecast
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("open-meteo decode: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return nil, err
	}
	return &raw, nil
}

func selectOpenMeteoSlot(raw *RawOpenMeteoForecast, kickoff time.Time) *livegame.Weather {
	if raw == nil || raw.Hourly == nil {
		return nil
	}
	w, err := openMeteoWeatherForKickoff(raw.Hourly, kickoff)
	if err != nil {
		return nil
	}
	return w
}

// ParseOpenMeteoHourly selects the UTC hourly slot covering kickoff and validates its fields.
func ParseOpenMeteoHourly(body []byte, kickoff time.Time) (*livegame.Weather, error) {
	raw, err := parseOpenMeteoForecast(body)
	if err != nil {
		return nil, err
	}
	return openMeteoWeatherForKickoff(raw.Hourly, kickoff)
}

func openMeteoWeatherForKickoff(
	hourly *RawOpenMeteoHourly, kickoff time.Time,
) (*livegame.Weather, error) {
	target := kickoff.UTC().Truncate(time.Hour)
	want := target.Format("2006-01-02T15:04")
	for i, ts := range hourly.Time {
		if ts == want {
			return openMeteoSlotToWeather(hourly, i)
		}
		if parsed, ok := parseOpenMeteoTime(ts); ok && parsed.Equal(target) {
			return openMeteoSlotToWeather(hourly, i)
		}
	}
	return nil, nil
}

func parseOpenMeteoTime(ts string) (time.Time, bool) {
	if t, err := time.Parse("2006-01-02T15:04", ts); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

func finiteInRange(v *float64, lo, hi float64) (float64, bool) {
	if v == nil || math.IsNaN(*v) || math.IsInf(*v, 0) || *v < lo || *v > hi {
		return 0, false
	}
	return *v, true
}

func openMeteoSlotToWeather(h *RawOpenMeteoHourly, idx int) (*livegame.Weather, error) {
	temp, ok := finiteInRange(h.Temperature2m[idx], -60, 140)
	if !ok {
		return nil, fmt.Errorf("open-meteo: invalid temperature")
	}
	tempF := int(math.Round(temp))
	w := &livegame.Weather{TempF: &tempF, Source: "Open-Meteo"}
	if code := h.WeatherCode[idx]; code != nil {
		w.Text = wmoWeatherText(*code)
	}
	if h.WindSpeed10m[idx] != nil {
		wind, ok := finiteInRange(h.WindSpeed10m[idx], 0, 200)
		if !ok {
			return nil, fmt.Errorf("open-meteo: invalid wind speed")
		}
		w.Wind = fmt.Sprintf("%d mph", int(math.Round(wind)))
	}
	if idx < len(h.PrecipProb) && h.PrecipProb[idx] != nil {
		if v := *h.PrecipProb[idx]; v >= 0 && v <= 100 {
			w.PrecipPct = &v
		}
	}
	return w, nil
}

func wmoWeatherText(code int) string {
	switch code {
	case 0, 1:
		return "Clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Cloudy"
	case 45, 48:
		return "Fog"
	case 51, 53, 55, 56, 57:
		return "Drizzle"
	case 61, 63, 65, 66, 67:
		return "Rain"
	case 71, 73, 75, 77, 85, 86:
		return "Snow"
	case 80, 81, 82:
		return "Showers"
	case 95, 96, 99:
		return "Thunderstorms"
	default:
		return ""
	}
}

func normalizePlace(s string) string {
	folded := strings.Map(func(r rune) rune {
		switch r {
		case 'á', 'à', 'â', 'ã', 'ä', 'Á', 'À', 'Â', 'Ã', 'Ä':
			return 'a'
		case 'é', 'è', 'ê', 'ë', 'É', 'È', 'Ê', 'Ë':
			return 'e'
		case 'í', 'ì', 'î', 'ï', 'Í', 'Ì', 'Î', 'Ï':
			return 'i'
		case 'ó', 'ò', 'ô', 'õ', 'ö', 'Ó', 'Ò', 'Ô', 'Õ', 'Ö':
			return 'o'
		case 'ú', 'ù', 'û', 'ü', 'Ú', 'Ù', 'Û', 'Ü':
			return 'u'
		case 'ñ', 'Ñ':
			return 'n'
		case 'ç', 'Ç':
			return 'c'
		default:
			return r
		}
	}, strings.TrimSpace(s))
	return strings.ToLower(folded)
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
