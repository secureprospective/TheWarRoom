package research

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/research/researchtest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func noNewsArchive(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
	return false, nil
}

// The daily cache reads the archive once per refresh, newest first, and stops at the first body from an earlier
// day: the cost does not grow with the number of players or with the archive's age.
func TestResearchNewsDailyCacheIsOneArchivePass(t *testing.T) {
	t.Parallel()
	now := time.Now()
	body := researchtest.Sample(t, "playerProfile-batch2.json")
	scans, read := 0, 0
	archive := func(_ context.Context, _ string, accept func(string, []byte, time.Time) bool) (bool, error) {
		scans++
		for day := range 400 {
			read++
			if accept("/2026/export?TYPE=playerProfile&P=13116,15761", body, now.AddDate(0, 0, -day)) {
				return true, nil
			}
		}
		return false, nil
	}
	ids := []string{"13116", "15761"}
	for n := 20000; n < 21452; n++ {
		ids = append(ids, strconv.Itoa(n))
	}
	held := map[string]Feed[[]researchfeeds.News]{}
	asked := 0
	fetch := func(_ context.Context, batch []string) (map[string][]researchfeeds.News, error) {
		asked += len(batch)
		out := map[string][]researchfeeds.News{}
		for _, id := range batch {
			out[id] = []researchfeeds.News{}
		}
		return out, nil
	}
	err := ReadNewsBatches(context.Background(), held, 2026, ids, now, archive,
		func(context.Context, time.Duration) error { return nil }, fetch, func(string) {})
	if err != nil || scans != 1 || read != 2 || asked != len(ids)-2 || len(held["15761"].Value) != 1 {
		t.Fatal(err, scans, read, asked)
	}
}

func TestResearchNewsMissingPlayerIsHeldNoNews(t *testing.T) {
	t.Parallel()
	at := time.Now().Truncate(time.Second)
	parsed, err := researchfeeds.ParseNews(researchtest.Sample(t, "playerProfile-batch2.json"), []string{"13116", "15761", "16162"})
	if err != nil {
		t.Fatal(err)
	}
	held := map[string]Feed[[]researchfeeds.News]{"16162": {Value: []researchfeeds.News{{Headline: "held article"}}, FetchedAt: at.AddDate(0, 0, -1)}}
	lines := []string{}
	fetch := func(context.Context, []string) (map[string][]researchfeeds.News, error) { return parsed, nil }
	err = ReadNewsBatches(context.Background(), held, 2026, []string{"13116", "15761", "16162"}, at, noNewsArchive,
		func(context.Context, time.Duration) error { return nil }, fetch, func(s string) { lines = append(lines, s) })
	if err != nil || len(held["15761"].Value) != 1 || held["13116"].FetchedAt.IsZero() || len(held["16162"].Value) != 0 || held["16162"].RefreshError != "" || !held["16162"].FetchedAt.Equal(at) || len(lines) != 1 || !strings.Contains(lines[0], "16162") {
		t.Fatal(held, lines, err)
	}
	asked := false
	err = ReadNewsBatches(context.Background(), held, 2026, []string{"16162"}, at, noNewsArchive,
		func(context.Context, time.Duration) error { return nil },
		func(context.Context, []string) (map[string][]researchfeeds.News, error) {
			asked = true
			return nil, nil
		}, func(string) {})
	if err != nil || asked {
		t.Fatal("omitted profile re-asked today", err)
	}

}

func TestResearchNewsBatchArchiveFallbackPreservesProvenance(t *testing.T) {
	t.Parallel()
	at := time.Now().AddDate(0, 0, -1).Truncate(time.Second)
	archive := func(_ context.Context, _ string, accept func(string, []byte, time.Time) bool) (bool, error) {
		return accept("/2026/export?TYPE=playerProfile&P=13116,15761", researchtest.Sample(t, "playerProfile-batch2.json"), at), nil
	}
	held := map[string]Feed[[]researchfeeds.News]{}
	fetch := func(context.Context, []string) (map[string][]researchfeeds.News, error) {
		return nil, errors.New("fixture offline")
	}
	err := ReadNewsBatches(context.Background(), held, 2026, []string{"15761"}, time.Now(), archive,
		func(context.Context, time.Duration) error { return nil }, fetch, func(string) {})
	if err != nil || len(held["15761"].Value) != 1 || !held["15761"].FetchedAt.Equal(at) || held["15761"].RefreshError == "" {
		t.Fatal(held, err)
	}
}
