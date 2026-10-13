package ingestion

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

// calendarMFL answers the calendar only to a keyed request, as MFL does since 10-12.
type calendarMFL struct{ keyed []bool }

func (f *calendarMFL) RoundTrip(r *http.Request) (*http.Response, error) {
	q := r.URL.Query()
	body := `{"league":{"baseURL":"https://www47.myfantasyleague.com"}}`
	if q.Get("TYPE") == "calendar" {
		f.keyed = append(f.keyed, q.Get("APIKEY") != "")
		body = `{"error":{"$t":"API requires logged in user in league ID 14432"}}`
		if q.Get("APIKEY") != "" {
			body = `{"calendar":{"event":{"type":"DRAFT_START","id":"1","start_time":"1780000000",` +
				`"end_time":"","happens":"","title":""}}}`
		}
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
}

func TestFetchCalendarIsKeyed(t *testing.T) {
	transport := &calendarMFL{}
	client, err := mfl.New("api", 10000, mfl.WithTransport(transport), mfl.WithKeySource(
		func(context.Context) (mflkey.Key, error) { return "owner-key", nil }))
	if err != nil {
		t.Fatal(err)
	}
	events, err := FetchCalendar(context.Background(), client, "2026", "14432")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || len(transport.keyed) != 1 || !transport.keyed[0] {
		t.Fatalf("calendar read unkeyed: events %d, keyed %v", len(events), transport.keyed)
	}
}

func TestMFLRefusal(t *testing.T) {
	refusal := []byte(`{"version":"1.0","error":{"$t":"API requires logged in user in league ID 14432"}}`)
	mflURL := "https://www47.myfantasyleague.com/2026/export?TYPE=calendar&L=14432"
	if got := MFLRefusal(mflURL, refusal); !strings.Contains(got, "API requires logged in user") {
		t.Fatalf("MFL refusal not named: %q", got)
	}
	if got := MFLRefusal(mflURL, []byte(`{"calendar":{}}`)); got != "" {
		t.Fatalf("MFL data judged a refusal: %q", got)
	}
	if got := MFLRefusal("https://example.test/feed.json", refusal); got != "" {
		t.Fatalf("another host judged by MFL's envelope: %q", got)
	}
}
