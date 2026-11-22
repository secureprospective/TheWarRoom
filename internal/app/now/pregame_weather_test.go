package now

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

type stubWeatherTransport struct {
	mu      sync.Mutex
	calls   []*http.Request
	replies map[string][]byte
	err     error
	onDo    func(*http.Request)
}

func (s *stubWeatherTransport) Do(req *http.Request) (*http.Response, error) {
	if s.onDo != nil {
		s.onDo(req)
	}
	s.mu.Lock()
	s.calls = append(s.calls, req)
	err := s.err
	body, ok := s.replies[req.URL.String()]
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("not found")),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
	}, nil
}

func readNowFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func openMeteoURL(template string, lat, lon float64) string {
	return strings.NewReplacer(
		"{lat}", fmt.Sprintf("%.4f", lat),
		"{lon}", fmt.Sprintf("%.4f", lon),
	).Replace(template)
}

func TestNFLStadiumsTableComplete(t *testing.T) {
	stadiums, err := NFLStadiums()
	if err != nil {
		t.Fatal(err)
	}
	byTeam := map[string]Stadium{}
	byName := map[string]Stadium{}
	for _, s := range stadiums {
		if s.Team != "" {
			byTeam[s.Team] = s
		}
		byName[s.Name] = s
	}
	if len(byTeam) != 32 {
		t.Fatalf("expected 32 clubs in stadium table, got %d", len(byTeam))
	}
	for _, team := range []string{"DET", "NO", "LV", "MIN", "LAC", "LAR"} {
		st := byTeam[team]
		if !st.Dome || st.Retractable {
			t.Fatalf("expected fixed dome for %s: %+v", team, st)
		}
	}
	for _, team := range []string{"ARI", "ATL", "DAL", "HOU", "IND"} {
		st := byTeam[team]
		if st.Dome || !st.Retractable {
			t.Fatalf("expected retractable roof for %s: %+v", team, st)
		}
	}
	if byTeam["BUF"].Dome || byTeam["GB"].Dome || byTeam["KC"].Dome {
		t.Fatalf("expected open-air stadiums not flagged as dome")
	}
	abroad := []string{
		"Tottenham Hotspur Stadium",
		"Wembley Stadium",
		"Allianz Arena",
		"Olympiastadion",
		"Santiago Bernabéu",
		"Estadio Azteca",
		"Arena Corinthians",
		"Croke Park",
	}
	for _, name := range abroad {
		st, ok := byName[name]
		if !ok {
			t.Fatalf("missing abroad venue %q", name)
		}
		if st.Dome || (st.Lat == 0 && st.Lon == 0) {
			t.Fatalf("unexpected abroad stadium entry %q: %+v", name, st)
		}
	}
}

func TestWMOCodeMapping(t *testing.T) {
	cases := []struct {
		code int
		want string
	}{
		{0, "Clear"}, {1, "Clear"}, {2, "Partly cloudy"}, {3, "Cloudy"},
		{45, "Fog"}, {48, "Fog"},
		{51, "Drizzle"}, {55, "Drizzle"}, {57, "Drizzle"},
		{61, "Rain"}, {65, "Rain"}, {67, "Rain"},
		{71, "Snow"}, {77, "Snow"}, {85, "Snow"}, {86, "Snow"},
		{80, "Showers"}, {82, "Showers"},
		{95, "Thunderstorms"}, {99, "Thunderstorms"},
		{999, ""},
	}
	for _, tc := range cases {
		if got := wmoWeatherText(tc.code); got != tc.want {
			t.Errorf("wmoWeatherText(%d) = %q, want %q", tc.code, got, tc.want)
		}
	}
}

func TestWeatherFallbackOrder(t *testing.T) {
	endpoints, err := LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(endpoints.OpenMeteo, "https://api.open-meteo.com/v1/forecast?") {
		t.Fatalf("expected Open-Meteo endpoint in livegame_sources.json, got %q", endpoints.OpenMeteo)
	}

	kickoff := time.Date(2026, 11, 22, 18, 0, 0, 0, time.UTC)
	bufURL := openMeteoURL(endpoints.OpenMeteo, 42.7738, -78.7870)
	bufBody := readNowFixture(t, "openmeteo-hourly-buf.json")

	t.Run("indoor venue shows Indoors with no temperature and skips Open-Meteo", func(t *testing.T) {
		tr := &stubWeatherTransport{}
		cache, err := NewWeatherCache(tr, endpoints.OpenMeteo)
		if err != nil {
			t.Fatal(err)
		}
		temp := 72
		g := livegame.Game{
			Status: "pre", Kickoff: kickoff,
			Home:    livegame.Team{Abbreviation: "BUF"},
			Venue:   &livegame.Venue{Name: "Custom Dome", City: "Buffalo", State: "NY", Indoor: true},
			Weather: &livegame.Weather{Text: "Clear", TempF: &temp, Source: "ESPN"},
		}
		cache.Resolve(context.Background(), &g)
		if g.Weather == nil || g.Weather.Text != "Indoors" || g.Weather.TempF != nil {
			t.Fatalf("want Indoors without temp, got %+v", g.Weather)
		}
		if len(tr.calls) != 0 {
			t.Fatalf("indoor game should not call Open-Meteo, got %d calls", len(tr.calls))
		}
	})

	t.Run("fixed dome shows Indoors when venue indoor is false or absent", func(t *testing.T) {
		tr := &stubWeatherTransport{}
		cache, err := NewWeatherCache(tr, endpoints.OpenMeteo)
		if err != nil {
			t.Fatal(err)
		}
		temp := 70
		g := livegame.Game{
			Status: "pre", Kickoff: kickoff,
			Home:    livegame.Team{Abbreviation: "DET"},
			Venue:   &livegame.Venue{Name: "Ford Field", City: "Detroit", State: "MI"},
			Weather: &livegame.Weather{Text: "Sunny", TempF: &temp, Source: "ESPN"},
		}
		cache.Resolve(context.Background(), &g)
		if g.Weather == nil || g.Weather.Text != "Indoors" || g.Weather.TempF != nil {
			t.Fatalf("want Indoors without temp, got %+v", g.Weather)
		}
		if !g.Venue.Indoor || len(tr.calls) != 0 {
			t.Fatalf("want indoor=true and 0 calls, got indoor=%v calls=%d",
				g.Venue.Indoor, len(tr.calls))
		}
		// When ESPN sends no venue, fixed dome still shows Indoors with 0 calls.
		noVenue := livegame.Game{Status: "pre", Kickoff: kickoff, Home: livegame.Team{Abbreviation: "DET"}}
		cache.Resolve(context.Background(), &noVenue)
		if noVenue.Weather == nil || noVenue.Weather.Text != "Indoors" || len(tr.calls) != 0 {
			t.Fatalf("fixed dome without venue: weather=%+v calls=%d", noVenue.Weather, len(tr.calls))
		}
	})

	t.Run("retractable roof follows ESPN indoor flag and fetches weather when open or absent",
		func(t *testing.T) {
			dalURL := openMeteoURL(endpoints.OpenMeteo, 32.7473, -97.0945)
			ariURL := openMeteoURL(endpoints.OpenMeteo, 33.5276, -112.2626)
			retractBody := []byte(`{"hourly":{"time":["2026-11-22T18:00"],` +
				`"temperature_2m":[71.4],"weather_code":[0],"wind_speed_10m":[6.2],` +
				`"precipitation_probability":[5]}}`)
			tr := &stubWeatherTransport{replies: map[string][]byte{
				dalURL: retractBody,
				ariURL: retractBody,
			}}
			cache, err := NewWeatherCache(tr, endpoints.OpenMeteo)
			if err != nil {
				t.Fatal(err)
			}

			closed := livegame.Game{
				Status: "pre", Kickoff: kickoff,
				Home: livegame.Team{Abbreviation: "DAL"},
				Venue: &livegame.Venue{
					Name: "AT&T Stadium", City: "Arlington", State: "TX", Indoor: true,
				},
			}
			cache.Resolve(context.Background(), &closed)
			if closed.Weather == nil || closed.Weather.Text != "Indoors" || len(tr.calls) != 0 {
				t.Fatalf("retractable indoor=true: weather=%+v calls=%d",
					closed.Weather, len(tr.calls))
			}

			openRoof := livegame.Game{
				Status: "pre", Kickoff: kickoff,
				Home: livegame.Team{Abbreviation: "DAL"},
				Venue: &livegame.Venue{
					Name: "AT&T Stadium", City: "Arlington", State: "TX", Indoor: false,
				},
			}
			cache.Resolve(context.Background(), &openRoof)
			if openRoof.Weather == nil || openRoof.Weather.TempF == nil ||
				*openRoof.Weather.TempF != 71 || openRoof.Weather.Source != "Open-Meteo" {
				t.Fatalf("retractable indoor=false should fetch weather, got %+v", openRoof.Weather)
			}
			if len(tr.calls) != 1 {
				t.Fatalf("expected 1 Open-Meteo call for open retractable roof, got %d", len(tr.calls))
			}

			// ESPN says nothing (nil venue): retractable roof fetches weather, not Indoors.
			noVenue := livegame.Game{
				Status: "pre", Kickoff: kickoff,
				Home: livegame.Team{Abbreviation: "ARI"},
			}
			cache.Resolve(context.Background(), &noVenue)
			if noVenue.Weather == nil || noVenue.Weather.TempF == nil ||
				*noVenue.Weather.TempF != 71 || noVenue.Weather.Source != "Open-Meteo" {
				t.Fatalf("retractable without venue should fetch weather, got %+v", noVenue.Weather)
			}
			if len(tr.calls) != 2 {
				t.Fatalf("expected 2 Open-Meteo calls total, got %d", len(tr.calls))
			}
		})

	t.Run("outdoor stadium fetches Open-Meteo hourly, sends User-Agent, and caches 30 min", func(t *testing.T) {
		nowAt := time.Date(2026, 11, 22, 16, 0, 0, 0, time.UTC)
		tr := &stubWeatherTransport{replies: map[string][]byte{bufURL: bufBody}}
		cache, err := NewWeatherCache(tr, endpoints.OpenMeteo)
		if err != nil {
			t.Fatal(err)
		}
		cache.Now = func() time.Time { return nowAt }

		g := livegame.Game{
			Status: "pre", Kickoff: kickoff,
			Home:  livegame.Team{Abbreviation: "BUF"},
			Venue: &livegame.Venue{Name: "Highmark Stadium", City: "Orchard Park", State: "NY"},
		}
		cache.Resolve(context.Background(), &g)
		if g.Weather == nil || g.Weather.TempF == nil || *g.Weather.TempF != 48 ||
			g.Weather.Text != "Partly cloudy" || g.Weather.Wind != "9 mph" ||
			g.Weather.PrecipPct == nil || *g.Weather.PrecipPct != 20 ||
			g.Weather.Source != "Open-Meteo" {
			t.Fatalf("unexpected Open-Meteo weather: %+v", g.Weather)
		}
		if len(tr.calls) != 1 {
			t.Fatalf("expected 1 Open-Meteo call, got %d", len(tr.calls))
		}
		if ua := tr.calls[0].Header.Get("User-Agent"); ua != "TheWarRoom (lab)" {
			t.Fatalf("User-Agent = %q, want %q", ua, "TheWarRoom (lab)")
		}

		nowAt = nowAt.Add(29 * time.Minute)
		g2 := livegame.Game{Status: "pre", Kickoff: kickoff, Home: livegame.Team{Abbreviation: "BUF"}}
		cache.Resolve(context.Background(), &g2)
		if len(tr.calls) != 1 || g2.Weather == nil || *g2.Weather.TempF != 48 {
			t.Fatalf("expected cache hit, got calls=%d weather=%+v", len(tr.calls), g2.Weather)
		}

		nowAt = nowAt.Add(2 * time.Minute)
		cache.Resolve(context.Background(), &g2)
		if len(tr.calls) != 2 {
			t.Fatalf("expected refresh after 30 min TTL, got %d calls", len(tr.calls))
		}
	})

	t.Run("abroad venues fetch Open-Meteo and distinguish London stadiums", func(t *testing.T) {
		tottenhamURL := openMeteoURL(endpoints.OpenMeteo, 51.6042, -0.0662)
		wembleyURL := openMeteoURL(endpoints.OpenMeteo, 51.5560, -0.2795)
		bernabeuURL := openMeteoURL(endpoints.OpenMeteo, 40.4530, -3.6883)
		saoPauloURL := openMeteoURL(endpoints.OpenMeteo, -23.5453, -46.4742)
		abroadBody := []byte(`{"hourly":{"time":["2026-11-22T14:00"],` +
			`"temperature_2m":[54.2],"weather_code":[61],"wind_speed_10m":[11.6],` +
			`"precipitation_probability":[60]}}`)
		tr := &stubWeatherTransport{replies: map[string][]byte{
			tottenhamURL: abroadBody,
			wembleyURL:   abroadBody,
			bernabeuURL:  abroadBody,
			saoPauloURL:  abroadBody,
		}}
		cache, err := NewWeatherCache(tr, endpoints.OpenMeteo)
		if err != nil {
			t.Fatal(err)
		}
		londonKick := time.Date(2026, 11, 22, 14, 30, 0, 0, time.UTC)
		venues := []*livegame.Venue{
			{Name: "Tottenham Hotspur Stadium", City: "London", Country: "United Kingdom"},
			{Name: "Wembley Stadium", City: "London", Country: "United Kingdom"},
			{Name: "Estadio Santiago Bernabeu", City: "Madrid", Country: "Spain"},
			{Name: "Neo Quimica Arena", City: "Sao Paulo", Country: "Brazil"},
		}
		for _, v := range venues {
			g := livegame.Game{
				Status: "pre", Kickoff: londonKick,
				Home: livegame.Team{Abbreviation: "JAX"}, Venue: v,
			}
			cache.Resolve(context.Background(), &g)
			if g.Weather == nil || g.Weather.TempF == nil || *g.Weather.TempF != 54 ||
				g.Weather.Text != "Rain" || g.Weather.Wind != "12 mph" ||
				g.Weather.PrecipPct == nil || *g.Weather.PrecipPct != 60 ||
				g.Weather.Source != "Open-Meteo" {
				t.Fatalf("abroad venue %q weather = %+v", v.Name, g.Weather)
			}
		}
		if len(tr.calls) != 4 {
			t.Fatalf("expected 4 distinct abroad Open-Meteo calls, got %d", len(tr.calls))
		}
	})

	t.Run("Open-Meteo failure or timeout omits weather without failing", func(t *testing.T) {
		tr := &stubWeatherTransport{err: errors.New("open-meteo timeout")}
		cache, err := NewWeatherCache(tr, endpoints.OpenMeteo)
		if err != nil {
			t.Fatal(err)
		}
		temp := 55
		g := livegame.Game{
			Status: "pre", Kickoff: kickoff,
			Home:    livegame.Team{Abbreviation: "BUF"},
			Venue:   &livegame.Venue{Name: "Highmark Stadium", City: "Orchard Park", State: "NY"},
			Weather: &livegame.Weather{Text: "Clear", TempF: &temp, Source: "ESPN"},
		}
		cache.Resolve(context.Background(), &g)
		if g.Weather != nil {
			t.Fatalf("expected nil weather on fetch failure, got %+v", g.Weather)
		}
	})

	t.Run("Open-Meteo foreign host is rejected without network call", func(t *testing.T) {
		tr := &stubWeatherTransport{}
		cache, err := NewWeatherCache(tr, "https://evil.example/v1/forecast?latitude={lat}&longitude={lon}")
		if err != nil {
			t.Fatal(err)
		}
		g := livegame.Game{Status: "pre", Kickoff: kickoff, Home: livegame.Team{Abbreviation: "BUF"}}
		cache.Resolve(context.Background(), &g)
		if g.Weather != nil || len(tr.calls) != 0 {
			t.Fatalf("expected host pin rejection, got weather=%+v calls=%d", g.Weather, len(tr.calls))
		}
	})

	t.Run("unmatched venue omits weather and does not fall back to ESPN or home club", func(t *testing.T) {
		tr := &stubWeatherTransport{replies: map[string][]byte{bufURL: bufBody}}
		cache, err := NewWeatherCache(tr, endpoints.OpenMeteo)
		if err != nil {
			t.Fatal(err)
		}
		temp := 61
		g := livegame.Game{
			Status: "pre", Kickoff: kickoff,
			Home:    livegame.Team{Abbreviation: "BUF"},
			Venue:   &livegame.Venue{Name: "Unlisted Arena", City: "Tokyo", Country: "Japan"},
			Weather: &livegame.Weather{Text: "Clear", TempF: &temp, Source: "ESPN"},
		}
		cache.Resolve(context.Background(), &g)
		if g.Weather != nil || len(tr.calls) != 0 {
			t.Fatalf("expected omitted weather and 0 calls, got weather=%+v calls=%d",
				g.Weather, len(tr.calls))
		}
	})

	t.Run("kickoff outside Open-Meteo window omits weather", func(t *testing.T) {
		tr := &stubWeatherTransport{replies: map[string][]byte{bufURL: bufBody}}
		cache, err := NewWeatherCache(tr, endpoints.OpenMeteo)
		if err != nil {
			t.Fatal(err)
		}
		farKickoff := kickoff.Add(20 * 24 * time.Hour)
		g := livegame.Game{Status: "pre", Kickoff: farKickoff, Home: livegame.Team{Abbreviation: "BUF"}}
		cache.Resolve(context.Background(), &g)
		if g.Weather != nil {
			t.Fatalf("expected nil weather outside forecast window, got %+v", g.Weather)
		}
	})
}

func TestWeatherCacheEvictsExpiredEntriesOnWrite(t *testing.T) {
	endpoints, err := LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	kickoff := time.Date(2026, 11, 22, 18, 0, 0, 0, time.UTC)
	bufURL := openMeteoURL(endpoints.OpenMeteo, 42.7738, -78.7870)
	gbURL := openMeteoURL(endpoints.OpenMeteo, 44.5013, -88.0622)
	kcURL := openMeteoURL(endpoints.OpenMeteo, 39.0489, -94.4839)
	body := readNowFixture(t, "openmeteo-hourly-buf.json")
	tr := &stubWeatherTransport{replies: map[string][]byte{
		bufURL: body,
		gbURL:  body,
		kcURL:  body,
	}}
	cache, err := NewWeatherCache(tr, endpoints.OpenMeteo)
	if err != nil {
		t.Fatal(err)
	}
	nowAt := time.Date(2026, 11, 22, 16, 0, 0, 0, time.UTC)
	cache.Now = func() time.Time { return nowAt }

	g1 := livegame.Game{Status: "pre", Kickoff: kickoff, Home: livegame.Team{Abbreviation: "BUF"}}
	cache.Resolve(context.Background(), &g1)
	if len(cache.entries) != 1 {
		t.Fatalf("want 1 cached entry, got %d", len(cache.entries))
	}

	nowAt = nowAt.Add(20 * time.Minute)
	g2 := livegame.Game{Status: "pre", Kickoff: kickoff, Home: livegame.Team{Abbreviation: "GB"}}
	cache.Resolve(context.Background(), &g2)
	if len(cache.entries) != 2 {
		t.Fatalf("want 2 cached entries before expiry, got %d", len(cache.entries))
	}

	// After 15 more minutes, BUF (35m old) is expired while GB (15m old) is still valid.
	nowAt = nowAt.Add(15 * time.Minute)
	g3 := livegame.Game{Status: "pre", Kickoff: kickoff, Home: livegame.Team{Abbreviation: "KC"}}
	cache.Resolve(context.Background(), &g3)

	bufKey := fmt.Sprintf("%.4f,%.4f", 42.7738, -78.7870)
	gbKey := fmt.Sprintf("%.4f,%.4f", 44.5013, -88.0622)
	kcKey := fmt.Sprintf("%.4f,%.4f", 39.0489, -94.4839)
	if len(cache.entries) != 2 {
		t.Fatalf("want expired BUF evicted leaving 2 entries, got %d", len(cache.entries))
	}
	if _, ok := cache.entries[bufKey]; ok {
		t.Fatalf("expired entry %q was not evicted on write", bufKey)
	}
	if _, ok := cache.entries[gbKey]; !ok {
		t.Fatalf("unexpired entry %q was wrongly evicted", gbKey)
	}
	if _, ok := cache.entries[kcKey]; !ok {
		t.Fatalf("new entry %q missing after write", kcKey)
	}
}

func TestWeatherCacheFetchesSameStadiumOncePerRefresh(t *testing.T) {
	endpoints, err := LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	// MetLife Stadium (NYG and NYJ share 40.8135, -74.0745).
	metLifeURL := openMeteoURL(endpoints.OpenMeteo, 40.8135, -74.0745)
	body := readNowFixture(t, "openmeteo-hourly-buf.json")
	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	tr := &stubWeatherTransport{
		replies: map[string][]byte{metLifeURL: body},
		onDo: func(*http.Request) {
			startOnce.Do(func() { close(started) })
			<-release
		},
	}
	cache, err := NewWeatherCache(tr, endpoints.OpenMeteo)
	if err != nil {
		t.Fatal(err)
	}
	early := time.Date(2026, 11, 22, 17, 0, 0, 0, time.UTC)
	late := time.Date(2026, 11, 22, 18, 0, 0, 0, time.UTC)
	g1 := livegame.Game{Status: "pre", Kickoff: early, Home: livegame.Team{Abbreviation: "NYG"}}
	g2 := livegame.Game{Status: "pre", Kickoff: late, Home: livegame.Team{Abbreviation: "NYJ"}}

	var wg sync.WaitGroup
	wg.Go(func() { cache.Resolve(context.Background(), &g1) })
	<-started
	wg.Go(func() { cache.Resolve(context.Background(), &g2) })
	time.Sleep(15 * time.Millisecond)
	close(release)
	wg.Wait()

	if len(tr.calls) != 1 {
		t.Fatalf("expected 1 Open-Meteo fetch for shared stadium, got %d", len(tr.calls))
	}
	if g1.Weather == nil || g1.Weather.TempF == nil || *g1.Weather.TempF != 46 {
		t.Fatalf("early game weather = %+v, want 46F", g1.Weather)
	}
	if g2.Weather == nil || g2.Weather.TempF == nil || *g2.Weather.TempF != 48 {
		t.Fatalf("late game weather = %+v, want 48F", g2.Weather)
	}
}
