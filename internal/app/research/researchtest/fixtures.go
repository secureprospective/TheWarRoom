package researchtest

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Data is the shared real-source Research fixture, independent of service and root contracts.
type Data struct {
	At       time.Time
	Lookup   normalize.Lookup
	Mirror   state.MirrorSnapshot
	Feeds    map[string]researchfeeds.Feed
	Depth    []researchfeeds.RawDepth
	Injuries []leaguefeed.Injury
	News     map[string][]researchfeeds.News
}

func Sample(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root, err := os.OpenRoot(filepath.Join(filepath.Dir(file), "..", "..", "..", "ingestion", "researchfeeds", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := root.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func Load(t *testing.T) Data {
	t.Helper()
	at := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	in := Data{At: at, Feeds: map[string]researchfeeds.Feed{}, News: map[string][]researchfeeds.News{}}
	in.Lookup = directory(t)
	var err error
	for _, kind := range []string{"playerScores", "topAdds", "topDrops", "topStarters", "topOwns", "topTrades", "pointsAllowed"} {
		name, week := kind+".json", ""
		if kind == "playerScores" {
			name, week = "playerScores_WYTD.json", "YTD"
		}
		feed, err := researchfeeds.Parse(Sample(t, name), kind, week)
		if err != nil {
			t.Fatal(err)
		}
		in.Feeds[kind+":"+week] = feed
	}
	for w := 1; w <= 5; w++ {
		feed, err := researchfeeds.Parse(Sample(t, "playerScores_W0"+strconv.Itoa(w)+".json"), "playerScores", strconv.Itoa(w))
		if err != nil {
			t.Fatal(err)
		}
		in.Feeds["playerScores:"+strconv.Itoa(w)] = feed
	}
	in.Depth, err = researchfeeds.ParseDepthCSV(Sample(t, "depth_charts_2026_latest.csv"))
	if err != nil {
		t.Fatal(err)
	}
	in.Injuries, err = injuries.ParseToInjuries(context.Background(), Sample(t, "injuries.json"))
	if err != nil {
		t.Fatal(err)
	}
	newsByID, err := researchfeeds.ParseNews(Sample(t, "playerProfile.json"), []string{"16162"})
	if err != nil {
		t.Fatal(err)
	}
	in.News["16162"] = newsByID["16162"]
	in.Mirror.Season = 2026
	in.Mirror.Players = []state.MirrorPlayer{{MFLID: "16162", FranchiseID: "0001"}, {MFLID: "13589", FranchiseID: "0002"}}
	return in
}

func directory(t *testing.T) normalize.Lookup {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(Sample(t, "players.json.gz")))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := players.Parse(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	lookup, err := normalize.NewLookup(raw)
	if err != nil {
		t.Fatal(err)
	}
	return lookup
}
