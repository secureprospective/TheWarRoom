package espnnews

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestParseLeagueNewsFixture(t *testing.T) {
	items, err := ParseLeagueNews(readFixture(t, "league_news.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 3 {
		t.Fatalf("got %d items, want >= 3", len(items))
	}
	for i := 1; i < len(items); i++ {
		if items[i].Published.After(items[i-1].Published) {
			t.Fatalf("items not sorted newest first at %d", i)
		}
	}
	var foundInjury, foundAthlete bool
	for _, it := range items {
		if it.Source != SourceESPN {
			t.Errorf("item %s source = %q, want ESPN", it.ID, it.Source)
		}
		if it.Headline == "" || it.Published.IsZero() {
			t.Errorf("item %+v missing headline or published time", it)
		}
		if strings.Contains(it.Headline, "Bowers") && it.Injury {
			foundInjury = true
		}
		if slices.Contains(it.AthleteIDs, "3919512") || slices.Contains(it.AthleteIDs, "4035687") {
			foundAthlete = true
		}
	}
	if !foundInjury {
		t.Error("expected Brock Bowers ankle/questionable headline to be tagged injury=true")
	}
	if !foundAthlete {
		t.Error("expected athlete IDs extracted from ESPN categories")
	}
}

func TestParseFantasyNewsFixture(t *testing.T) {
	items, err := ParseFantasyNews(readFixture(t, "player_news_fantasy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	rotowire := items[0]
	if rotowire.Type != "Rotowire" || !rotowire.Injury || rotowire.Source != SourceESPN {
		t.Fatalf("unexpected rotowire item: %+v", rotowire)
	}
	if rotowire.Description == "" || rotowire.Description == rotowire.Headline {
		t.Fatalf("expected rotowire description to carry spin story, got %q", rotowire.Description)
	}
	if !slices.Equal(rotowire.AthleteIDs, []string{"4432665"}) {
		t.Fatalf("rotowire athlete IDs = %v, want [4432665]", rotowire.AthleteIDs)
	}
	story := items[1]
	if story.URL == "" || !strings.HasPrefix(story.URL, "https://") {
		t.Fatalf("expected https web link on Story item, got %q", story.URL)
	}
}

func TestParseSitePlayerNewsFixture(t *testing.T) {
	items, err := ParseLeagueNews(readFixture(t, "player_news_site.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Source != SourceESPN || items[0].URL == "" {
		t.Fatalf("unexpected site player item: %+v", items[0])
	}
}

func TestParseBioFixtures(t *testing.T) {
	bio, err := ParseBio(readFixture(t, "athlete_bio.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bio.Height != `6' 2"` || bio.Weight != "225 lbs" || bio.Born != "1995-09-17" || bio.Age != 31 {
		t.Fatalf("unexpected Mahomes bio: %+v", bio)
	}
	if bio.College != "Texas Tech" || bio.Draft != "2017: Rd 1, Pk 10 (KC)" || bio.Team != "KC" ||
		bio.Position != "QB" || !strings.Contains(bio.Headshot, "3139477.png") {
		t.Fatalf("unexpected Mahomes bio details: %+v", bio)
	}

	injured, err := ParseBio(readFixture(t, "athlete_bio_injured.json"))
	if err != nil {
		t.Fatal(err)
	}
	if injured.Born != "2002-12-13" || injured.College != "Georgia" || injured.Injury == "" {
		t.Fatalf("unexpected injured bio: %+v", injured)
	}
}

func TestFetchPlayerNewsFallsBackToSiteEndpoint(t *testing.T) {
	siteBody := readFixture(t, "player_news_site.json")
	var calls []string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.URL.String())
		if strings.Contains(req.URL.Path, "/fantasy/") {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader("unavailable")),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(siteBody)),
		}, nil
	})
	_, items, kind, err := FetchPlayerNews(context.Background(), transport, DefaultEndpoints(), "3139477")
	if err != nil {
		t.Fatal(err)
	}
	if kind != "site" || len(items) != 2 || len(calls) != 2 {
		t.Fatalf("fallback result: kind=%q items=%d calls=%v", kind, len(items), calls)
	}
}

func TestValidationRejectsMalformedPayloadsAndEndpoints(t *testing.T) {
	if _, err := ParseLeagueNews([]byte(`{}`)); err == nil {
		t.Error("accepted league news without articles")
	}
	if _, err := ParseFantasyNews([]byte(`{}`)); err == nil {
		t.Error("accepted fantasy news without feed")
	}
	if _, err := ParseBio([]byte(`{"athlete":{}}`)); err == nil {
		t.Error("accepted bio without athlete id")
	}
	bad := DefaultEndpoints()
	bad.Bio = "http://insecure.example/{id}"
	if err := bad.Validate(); err == nil {
		t.Error("accepted non-HTTPS endpoint")
	}
	if isInjuryItem("Story", "Carmelo calls out player", "regular news") {
		t.Error("false positive injury tag on 'calls out'")
	}
}
