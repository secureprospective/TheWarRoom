// Package nowtest supplies real-source fixtures to Now's service and root tests.
package nowtest

import (
	"compress/gzip"
	"io"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/race/racetest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

type Data struct {
	Race        racetest.Data
	Schedule    nflschedule.RawWeek
	Byes        []nflschedule.RawBye
	Projections marketfeeds.Feed
}

func Compressed(t *testing.T, feed, name string) string {
	t.Helper()
	r, e := gzip.NewReader(strings.NewReader(racetest.Sample(t, feed, name)))
	if e != nil {
		t.Fatal(e)
	}
	b, e := io.ReadAll(r)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.Close(); e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func Load(t *testing.T) Data {
	t.Helper()
	schedule, e := nflschedule.Parse([]byte(racetest.Sample(t, "nflschedule", "nflSchedule-2025-w05.json")))
	if e != nil {
		t.Fatal(e)
	}
	byes, e := nflschedule.ParseByes([]byte(racetest.Sample(t, "nflschedule", "nflByeWeeks-2025.json")))
	if e != nil {
		t.Fatal(e)
	}
	projections, e := marketfeeds.Parse(marketfeeds.Projections, []byte(Compressed(t, "marketfeeds", "projectedScores-2025-w05.json.gz")))
	if e != nil {
		t.Fatal(e)
	}
	return Data{Race: racetest.Load(t, "2025"), Schedule: schedule, Byes: byes, Projections: projections}
}
