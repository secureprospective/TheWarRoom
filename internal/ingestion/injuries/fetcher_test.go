package injuries

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// Synthetic bodies only: no values are derived from an authenticated league feed.
func TestParseSingletonListAndEmpty(t *testing.T) {
	one := `{"id":"10249","status":"IR","details":"Knee - PCL","exp_return":"Feb 15, 2027"}`
	cases := []struct {
		name, block string
		count       int
		wantErr     string
	}{
		{"one row", `{"injuries":{"injury":` + one + `}}`, 1, ""},
		{"two rows", `{"injuries":{"injury":[` + one + `,` +
			`{"id":"10250","status":"Out","details":"Ankle"}]}}`, 2, ""},
		{"empty list", `{"injuries":{"injury":[]}}`, 0, "zero players"},
		{"absent", `{"injuries":{}}`, 0, "zero players"},
		{"missing status", `{"injuries":{"injury":{"id":"10249","status":""}}}`, 0, "missing status"},
		{"bad id", `{"injuries":{"injury":{"id":"ID","status":"IR"}}}`, 0, "player"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := Parse(context.Background(), []byte(tc.block))
			if tc.wantErr == "" {
				if err != nil || len(rows) != tc.count {
					t.Fatalf("Parse(%s) = %d rows, %v; want %d, nil", tc.name, len(rows), err, tc.count)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Parse(%s) err = %v, want it to contain %q", tc.name, err, tc.wantErr)
			}
		})
	}
}

// urlTransport records the URL one request was sent to and answers one body, so tests can
// assert how a fetch is routed without leaving the machine.
type urlTransport struct {
	body []byte
	got  *url.URL
}

func (u *urlTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	cloned := *r.URL
	u.got = &cloned
	return &http.Response{StatusCode: http.StatusOK,
		Body: io.NopCloser(bytes.NewReader(u.body)), Header: http.Header{}}, nil
}

// TestFetchRoutesToTheAPIHostWithoutL proves the league-less routing the 10-10 live gate showed
// MFL requires: the injuries request must go to api.myfantasyleague.com and carry no L, even
// when the client holds the league's own discovered host (www47 refused the league-scoped
// form with MFL's HTTP-200 error envelope, so the feed could never go live).
func TestFetchRoutesToTheAPIHostWithoutL(t *testing.T) {
	const feed = `{"injuries":{"injury":[{"id":"10249","status":"IR"},{"id":"10250","status":"Out"}]}}`
	transport := &urlTransport{body: []byte(feed)}
	// The client holds a discovered league host, as after any league-scoped fetch.
	c, err := mfl.New("www47", 10000, mfl.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := Fetch(context.Background(), c, "2026")
	if err != nil {
		t.Fatal(err)
	}
	if transport.got == nil {
		t.Fatal("no request was made")
	}
	if transport.got.Host != "api.myfantasyleague.com" {
		t.Fatalf("routed to %s, want api.myfantasyleague.com", transport.got.Host)
	}
	q := transport.got.Query()
	if q.Get("L") != "" {
		t.Fatalf("request carried L=%q; the injuries export is league-less", q.Get("L"))
	}
	if q.Get("TYPE") != Export || transport.got.Path != "/2026/export" {
		t.Fatalf("request was %s, want the injuries export", transport.got)
	}
	if len(rows) != 2 || rows[0].ID != "10249" || rows[0].Status != "IR" {
		t.Fatalf("rows = %+v, want the two validated rows", rows)
	}
}

// TestParseNamesMflErrorEnvelope locks the guard behind the 10-10 live failure: MFL answers
// a misplaced injuries request with HTTP 200 and {"error":{"$t":…}}, and that body must read
// as MFL's own message, never as zero injured players.
func TestParseNamesMflErrorEnvelope(t *testing.T) {
	const refused = `{"error":{"$t":"Invalid request. This API request must go to api.myfantasyleague.com"}}`
	_, err := Parse(context.Background(), []byte(refused))
	if err == nil || !strings.Contains(err.Error(), "MFL API error") ||
		!strings.Contains(err.Error(), "must go to api.myfantasyleague.com") {
		t.Fatalf("Parse(error envelope) = %v, want MFL's own message", err)
	}
	if _, err := ParseToInjuries(context.Background(), []byte(refused)); err == nil ||
		!strings.Contains(err.Error(), "must go to api.myfantasyleague.com") {
		t.Fatalf("ParseToInjuries(error envelope) = %v, want MFL's own message", err)
	}
}

func TestToInjuriesTypesAndRejectsDuplicates(t *testing.T) {
	typed, err := ToInjuries([]RawInjury{
		{ID: "10249", Status: " IR-PUP ", Details: " Hamstring "},
		{ID: "10250", Status: "Out"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(typed) != 2 {
		t.Fatalf("typed %d rows, want 2", len(typed))
	}
	if typed[0].ID.String() != "10249" || typed[0].Status != "IR-PUP" || typed[0].Details != "Hamstring" ||
		typed[0].ExpReturn != "" {
		t.Fatalf("row 0 = %+v, want typed and trimmed IR-PUP", typed[0])
	}
	if typed[1].ID.String() != "10250" || typed[1].Status != "Out" || typed[1].Details != "" {
		t.Fatalf("row 1 = %+v, want Out with absent details", typed[1])
	}
	if _, err := ToInjuries([]RawInjury{{ID: "10249", Status: "IR"}, {ID: "10249", Status: "Out"}}); err == nil ||
		!strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate id err = %v, want duplicate rejection", err)
	}
}
