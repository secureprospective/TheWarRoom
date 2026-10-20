// Package markettest holds the market feed fixtures that both the market service's tests and the
// root binding tests read, so neither copies them. They are plain values: each side wraps them in
// its own held-feed type.
package markettest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
)

// Read is when the fixture feeds were read: noon UTC on October 11, 2026.
func Read() time.Time { return time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC) }

// Kinds are the four MFL market reports, in the order the board shows them.
func Kinds() []string {
	return []string{marketfeeds.Projections, marketfeeds.Ranks, marketfeeds.ADP, marketfeeds.AAV}
}

// Feeds parses the real MFL export of every market report kind, from the ingestion testdata.
func Feeds(t testing.TB) map[string]marketfeeds.Feed {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("markettest: no source path")
	}
	testdata := filepath.Join(filepath.Dir(here), "..", "..", "..", "ingestion", "marketfeeds", "testdata")
	feeds := map[string]marketfeeds.Feed{}
	for _, kind := range Kinds() {
		body, err := os.ReadFile(filepath.Join(testdata, kind+".json"))
		if err != nil {
			t.Fatal(err)
		}
		feed, err := marketfeeds.Parse(kind, body)
		if err != nil {
			t.Fatal(err)
		}
		feeds[kind] = feed
	}
	return feeds
}
