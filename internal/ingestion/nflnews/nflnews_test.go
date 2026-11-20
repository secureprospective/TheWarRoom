package nflnews

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func reply(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}
}

const sleeperFixture = `{
 "4046": {"player_id":"4046","full_name":"Ja'Marr Chase","position":"WR","team":"CIN",
   "injury_status":"Questionable","injury_body_part":"Ankle","news_updated":1763600000000,"espn_id":"4362628"},
 "5000": {"player_id":"5000","full_name":"Jane Doe","position":"RB","team":null,
   "injury_status":null,"injury_body_part":null,"news_updated":null,"espn_id":4040715}
}`

const espnFixture = `{"status":"success","feed":[
 {"headline":"<b>Chase</b> <i>questionable</i> with ankle","description":"ignored","published":"2026-11-19T10:00:00Z"},
 {"headline":"","description":"Description used when no headline","published":"2026-11-18T10:00:00Z"},
 {"headline":"   ","description":"","published":"2026-11-18T11:00:00Z"}
]}`

func TestParseSleeperKeepsTheFieldsTheReadUses(t *testing.T) {
	players, err := ParseSleeper(strings.NewReader(sleeperFixture))
	if err != nil {
		t.Fatal(err)
	}
	chase := players["4046"]
	if chase.Name != "Ja'Marr Chase" || chase.Team != "CIN" || chase.Status != "Questionable" || chase.BodyPart != "Ankle" ||
		chase.ESPNID != "4362628" || !chase.NewsAt.Equal(time.UnixMilli(1763600000000).UTC()) {
		t.Fatalf("Chase = %+v", chase)
	}
	doe := players["5000"]
	if doe.ESPNID != "4040715" || !doe.NewsAt.IsZero() || doe.Status != "" {
		t.Fatalf("numeric espn_id and nulls = %+v", doe)
	}
	if _, err := ParseSleeper(strings.NewReader("[")); err == nil {
		t.Fatal("malformed Sleeper table accepted")
	}
}

func TestParseESPNStripsHTMLAndFallsBackToDescription(t *testing.T) {
	notes, err := ParseESPN([]byte(espnFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 || notes[0].Text != "Chase questionable with ankle" || notes[1].Text != "Description used when no headline" {
		t.Fatalf("notes = %+v", notes)
	}
	long := `{"feed":[{"headline":"` + strings.Repeat("é", 300) + `","published":"2026-11-19T10:00:00Z"}]}`
	notes, err = ParseESPN([]byte(long))
	if err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(notes[0].Text)); n != maxText {
		t.Fatalf("capped text has %d runes, want %d", n, maxText)
	}
}

func TestTagMapping(t *testing.T) {
	for status, want := range map[string]string{"Questionable": "INJ", "Doubtful": "INJ", "Out": "OUT",
		"IR": "IR", "Active": "ACT", "": "", "Sus": ""} {
		if got := Tag(status); got != want {
			t.Errorf("Tag(%q) = %q, want %q", status, got, want)
		}
	}
	if tagOr("Sus") != "NOTE" {
		t.Fatal("an unnamed status with an ESPN note should be NOTE")
	}
}

// fixtureFeed serves the Sleeper table and one ESPN note list per player id.
func fixtureFeed(t *testing.T, notes map[string]string, now *time.Time) *Feed {
	t.Helper()
	transport := rtFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.String() == SleeperURL:
			return reply(200, sleeperFixture), nil
		case strings.Contains(r.URL.String(), "playerId="):
			id := r.URL.Query().Get("playerId")
			body, ok := notes[id]
			if !ok {
				return reply(404, ""), nil
			}
			return reply(200, body), nil
		}
		return reply(500, ""), nil
	})
	f := NewFeed(transport, func() time.Time { return *now })
	f.logf = func(string, ...any) {}
	return f
}

func TestReadMergesNewestFirstAndCapsAt60(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	var body strings.Builder
	body.WriteString(`{"feed":[`)
	for i := 0; i < 70; i++ {
		if i > 0 {
			body.WriteString(",")
		}
		at := now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339)
		body.WriteString(`{"headline":"note ` + strings.Repeat("x", i%3) + `","published":"` + at + `"}`)
	}
	body.WriteString(`]}`)
	f := fixtureFeed(t, map[string]string{"4362628": body.String()}, &now)
	players := []Player{{ID: "4046", Name: "Ja'Marr Chase", Position: "WR", ESPNID: "4362628"}}
	f.Read(players)
	f.Refresh(context.Background())
	got := f.Read(players)
	if len(got.Items) != MaxItems {
		t.Fatalf("items = %d, want %d", len(got.Items), MaxItems)
	}
	for i := 1; i < len(got.Items); i++ {
		if got.Items[i].At.After(got.Items[i-1].At) {
			t.Fatalf("items not newest first at %d", i)
		}
	}
	if got.Items[0].Tag != "INJ" || got.Items[0].NFLTeam != "CIN" || got.Items[0].Source != "espn" {
		t.Fatalf("first item = %+v", got.Items[0])
	}
}

func TestSleeperStatusWithoutNoteBecomesItem(t *testing.T) {
	// The fixture's news_updated is 2025-11-19T22:13:20Z; read an hour later so it is inside the window.
	now := time.UnixMilli(1763600000000).UTC().Add(time.Hour)
	f := fixtureFeed(t, map[string]string{}, &now)
	players := []Player{{ID: "4046", Name: "Ja'Marr Chase", Position: "WR"}}
	f.Read(players)
	f.Refresh(context.Background())
	got := f.Read(players)
	if len(got.Items) != 1 || got.Items[0].Text != "Questionable (Ankle)" || got.Items[0].Source != "sleeper" || got.Items[0].Tag != "INJ" {
		t.Fatalf("sleeper item = %+v", got.Items)
	}
}

func TestFetchFailureKeepsLastGoodCopyAndMarksStale(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	var fail atomic.Bool
	transport := rtFunc(func(r *http.Request) (*http.Response, error) {
		if fail.Load() {
			return reply(503, ""), nil
		}
		if r.URL.String() == SleeperURL {
			return reply(200, sleeperFixture), nil
		}
		return reply(200, espnFixture), nil
	})
	f := NewFeed(transport, func() time.Time { return now })
	f.logf = func(string, ...any) {}
	players := []Player{{ID: "4046", Name: "Ja'Marr Chase", Position: "WR", ESPNID: "4362628"}}
	f.Read(players)
	f.Refresh(context.Background())
	first := f.Read(players)
	if first.Stale || len(first.Items) == 0 {
		t.Fatalf("first read = %+v", first)
	}
	fail.Store(true)
	now = now.Add(time.Hour)
	f.Refresh(context.Background())
	after := f.Read(players)
	if !after.Stale || len(after.Items) != len(first.Items) || after.Items[0].Text != first.Items[0].Text {
		t.Fatalf("after failure: stale=%t items=%d, want the last good copy", after.Stale, len(after.Items))
	}
}

func TestESPNRateLimits(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	var inFlight, peak atomic.Int32
	var mu sync.Mutex
	calls := map[string]int{}
	transport := rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() == SleeperURL {
			return reply(200, `{}`), nil
		}
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		inFlight.Add(-1)
		mu.Lock()
		calls[r.URL.Query().Get("playerId")]++
		mu.Unlock()
		return reply(200, `{"feed":[]}`), nil
	})
	f := NewFeed(transport, func() time.Time { return now })
	f.logf = func(string, ...any) {}
	var players []Player
	for i := 0; i < 10; i++ {
		id := strings.Repeat("1", i+1)
		players = append(players, Player{ID: id, Name: "P" + id, Position: "WR", ESPNID: id})
	}
	f.Read(players)
	f.Refresh(context.Background())
	if peak.Load() > MaxInFlight {
		t.Fatalf("peak in flight = %d, want at most %d", peak.Load(), MaxInFlight)
	}
	if len(calls) != 10 {
		t.Fatalf("players fetched = %d, want 10", len(calls))
	}
	now = now.Add(29 * time.Minute)
	f.Refresh(context.Background())
	for id, n := range calls {
		if n != 1 {
			t.Fatalf("player %s fetched %d times inside 30 minutes", id, n)
		}
	}
	now = now.Add(2 * time.Minute)
	f.Refresh(context.Background())
	for id, n := range calls {
		if n != 2 {
			t.Fatalf("player %s fetched %d times after 30 minutes, want 2", id, n)
		}
	}
}

func TestNothingFetchedUntilARead(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	var calls atomic.Int32
	f := NewFeed(rtFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return reply(200, `{}`), nil
	}), func() time.Time { return now })
	f.Refresh(context.Background())
	if calls.Load() != 0 {
		t.Fatalf("refresh with no read made %d requests", calls.Load())
	}
}
