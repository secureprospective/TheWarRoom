package now

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

type stubWeatherTransport struct {
	calls   []*http.Request
	replies map[string][]byte
	err     error
}

func (s *stubWeatherTransport) Do(req *http.Request) (*http.Response, error) {
	s.calls = append(s.calls, req)
	if s.err != nil {
		return nil, s.err
	}
	body, ok := s.replies[req.URL.String()]
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

func TestNFLStadiumsTableComplete(t *testing.T) {
	stadiums, err := NFLStadiums()
	if err != nil {
		t.Fatal(err)
	}
	if len(stadiums) != 32 {
		t.Fatalf("expected 32 clubs in stadium table, got %d", len(stadiums))
	}
	byTeam := map[string]Stadium{}
	for _, s := range stadiums {
		byTeam[s.Team] = s
	}
	if !byTeam["DET"].Dome || !byTeam["DAL"].Dome || !byTeam["NO"].Dome || !byTeam["LV"].Dome {
		t.Fatalf("expected dome/closed-roof stadiums flagged: %+v", byTeam["DET"])
	}
	if byTeam["BUF"].Dome || byTeam["GB"].Dome || byTeam["KC"].Dome {
		t.Fatalf("expected open-air stadiums not flagged as dome")
	}
}

func TestWeatherFallbackOrder(t *testing.T) {
	endpoints, err := LiveEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(endpoints.NWS, "api.weather.gov/points/") {
		t.Fatalf("expected NWS endpoint in livegame_sources.json, got %q", endpoints.NWS)
	}

	kickoff := time.Date(2026, 11, 22, 18, 0, 0, 0, time.UTC)
	pointsURL := "https://api.weather.gov/points/42.7738,-78.7870"
	hourlyURL := "https://api.weather.gov/gridpoints/BUF/39,46/forecast/hourly"
	pointsBody := readNowFixture(t, "nws-points-buf.json")
	hourlyBody := readNowFixture(t, "nws-hourly-buf.json")

	t.Run("ESPN indoor venue shows Indoors with no temperature and skips NWS", func(t *testing.T) {
		tr := &stubWeatherTransport{}
		cache, err := NewWeatherCache(tr, endpoints.NWS)
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
			t.Fatalf("indoor game should not call NWS, got %d calls", len(tr.calls))
		}
	})

	t.Run("stadium dome shows Indoors with no temp even when ESPN indoor false", func(t *testing.T) {
		tr := &stubWeatherTransport{}
		cache, err := NewWeatherCache(tr, endpoints.NWS)
		if err != nil {
			t.Fatal(err)
		}
		temp := 70
		g := livegame.Game{
			Status: "pre", Kickoff: kickoff,
			Home:    livegame.Team{Abbreviation: "DAL"},
			Venue:   &livegame.Venue{Name: "AT&T Stadium", City: "Arlington", State: "TX", Indoor: false},
			Weather: &livegame.Weather{Text: "Sunny", TempF: &temp, Source: "ESPN"},
		}
		cache.Resolve(context.Background(), &g)
		if g.Weather == nil || g.Weather.Text != "Indoors" || g.Weather.TempF != nil {
			t.Fatalf("want Indoors without temp, got %+v", g.Weather)
		}
		if !g.Venue.Indoor || len(tr.calls) != 0 {
			t.Fatalf("want indoor=true and 0 NWS calls, got indoor=%v calls=%d",
				g.Venue.Indoor, len(tr.calls))
		}
	})

	t.Run("US outdoor stadium fetches NWS hourly, sends User-Agent, and caches 30 min", func(t *testing.T) {
		nowAt := time.Date(2026, 11, 22, 16, 0, 0, 0, time.UTC)
		tr := &stubWeatherTransport{replies: map[string][]byte{
			pointsURL: pointsBody,
			hourlyURL: hourlyBody,
		}}
		cache, err := NewWeatherCache(tr, endpoints.NWS)
		if err != nil {
			t.Fatal(err)
		}
		cache.Now = func() time.Time { return nowAt }

		g := livegame.Game{
			Status: "pre", Kickoff: kickoff,
			Home:  livegame.Team{Abbreviation: "BUF"},
			Venue: &livegame.Venue{Name: "Highmark Stadium", City: "Orchard Park", State: "NY", Country: "USA"},
		}
		cache.Resolve(context.Background(), &g)
		if g.Weather == nil || g.Weather.TempF == nil || *g.Weather.TempF != 66 ||
			g.Weather.Text != "Mostly Cloudy" || g.Weather.Wind != "7 mph S" ||
			g.Weather.PrecipPct == nil || *g.Weather.PrecipPct != 0 || g.Weather.Source != "NWS" {
			t.Fatalf("unexpected NWS weather: %+v", g.Weather)
		}
		if len(tr.calls) != 2 {
			t.Fatalf("expected 2 NWS calls (points+hourly), got %d", len(tr.calls))
		}
		for _, req := range tr.calls {
			if ua := req.Header.Get("User-Agent"); ua != "TheWarRoom (lab)" {
				t.Fatalf("User-Agent = %q, want %q", ua, "TheWarRoom (lab)")
			}
		}

		// Within 30 min: cached, no new network calls.
		nowAt = nowAt.Add(29 * time.Minute)
		g2 := livegame.Game{Status: "pre", Kickoff: kickoff, Home: livegame.Team{Abbreviation: "BUF"}}
		cache.Resolve(context.Background(), &g2)
		if len(tr.calls) != 2 || g2.Weather == nil || *g2.Weather.TempF != 66 {
			t.Fatalf("expected cache hit without new NWS calls, got calls=%d weather=%+v", len(tr.calls), g2.Weather)
		}

		// After 30 min: refreshes hourly using cached gridURL (only 1 new call).
		nowAt = nowAt.Add(2 * time.Minute)
		cache.Resolve(context.Background(), &g2)
		if len(tr.calls) != 3 {
			t.Fatalf("expected 1 refreshed hourly call using cached grid URL, got %d total", len(tr.calls))
		}
	})

	t.Run("NWS failure or timeout omits weather without failing", func(t *testing.T) {
		tr := &stubWeatherTransport{err: errors.New("nws timeout")}
		cache, err := NewWeatherCache(tr, endpoints.NWS)
		if err != nil {
			t.Fatal(err)
		}
		temp := 55
		g := livegame.Game{
			Status: "pre", Kickoff: kickoff,
			Home:    livegame.Team{Abbreviation: "BUF"},
			Venue:   &livegame.Venue{Name: "Highmark Stadium", City: "Orchard Park", State: "NY", Country: "USA"},
			Weather: &livegame.Weather{Text: "Clear", TempF: &temp, Source: "ESPN"},
		}
		cache.Resolve(context.Background(), &g)
		if g.Weather != nil {
			t.Fatalf("expected nil weather on NWS failure, got %+v", g.Weather)
		}
	})

	t.Run("NWS foreign forecastHourly host is rejected", func(t *testing.T) {
		tr := &stubWeatherTransport{replies: map[string][]byte{
			pointsURL: []byte(`{"properties":{"forecastHourly":"https://evil.example/hourly"}}`),
		}}
		cache, err := NewWeatherCache(tr, endpoints.NWS)
		if err != nil {
			t.Fatal(err)
		}
		g := livegame.Game{Status: "pre", Kickoff: kickoff, Home: livegame.Team{Abbreviation: "BUF"}}
		cache.Resolve(context.Background(), &g)
		if g.Weather != nil || len(tr.calls) != 1 {
			t.Fatalf("expected rejected host to stop after points, got weather=%+v calls=%d", g.Weather, len(tr.calls))
		}
	})

	t.Run("game abroad uses ESPN weather and skips NWS", func(t *testing.T) {
		tr := &stubWeatherTransport{}
		cache, err := NewWeatherCache(tr, endpoints.NWS)
		if err != nil {
			t.Fatal(err)
		}
		temp := 54
		g := livegame.Game{
			Status: "pre", Kickoff: kickoff,
			Home:    livegame.Team{Abbreviation: "JAX"},
			Venue:   &livegame.Venue{Name: "Tottenham Hotspur Stadium", City: "London", Country: "United Kingdom"},
			Weather: &livegame.Weather{Text: "Light Rain", TempF: &temp, Source: "ESPN"},
		}
		cache.Resolve(context.Background(), &g)
		if g.Weather == nil || g.Weather.TempF == nil || *g.Weather.TempF != 54 || g.Weather.Text != "Light Rain" {
			t.Fatalf("expected ESPN weather for abroad game, got %+v", g.Weather)
		}
		if len(tr.calls) != 0 {
			t.Fatalf("abroad game must not call NWS, got %d calls", len(tr.calls))
		}
	})

	t.Run("game abroad with no ESPN weather omits weather", func(t *testing.T) {
		tr := &stubWeatherTransport{}
		cache, err := NewWeatherCache(tr, endpoints.NWS)
		if err != nil {
			t.Fatal(err)
		}
		g := livegame.Game{
			Status: "pre", Kickoff: kickoff,
			Home:  livegame.Team{Abbreviation: "JAX"},
			Venue: &livegame.Venue{Name: "Wembley Stadium", City: "London", Country: "United Kingdom"},
		}
		cache.Resolve(context.Background(), &g)
		if g.Weather != nil || len(tr.calls) != 0 {
			t.Fatalf("expected nil weather and 0 NWS calls, got weather=%+v calls=%d", g.Weather, len(tr.calls))
		}
	})

	t.Run("kickoff outside NWS window omits weather instead of inventing temperature", func(t *testing.T) {
		tr := &stubWeatherTransport{replies: map[string][]byte{
			pointsURL: pointsBody,
			hourlyURL: hourlyBody,
		}}
		cache, err := NewWeatherCache(tr, endpoints.NWS)
		if err != nil {
			t.Fatal(err)
		}
		farKickoff := kickoff.Add(10 * 24 * time.Hour)
		g := livegame.Game{Status: "pre", Kickoff: farKickoff, Home: livegame.Team{Abbreviation: "BUF"}}
		cache.Resolve(context.Background(), &g)
		if g.Weather != nil {
			t.Fatalf("expected nil weather when kickoff outside hourly window, got %+v", g.Weather)
		}
	})
}
