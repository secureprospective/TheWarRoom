package research

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestArchiveFeedsChooseScope(t *testing.T) {
	t.Parallel()
	var calls []string
	bodies := func(_ context.Context, part string, _ func(string, []byte, time.Time) bool) (bool, error) {
		calls = append(calls, part)
		return false, nil
	}
	feeds := ArchiveFeeds{Bodies: bodies, Scoped: func(param, value string) ArchivedBodies { calls = append(calls, param+"="+value); return bodies }}
	feeds.Read(t.Context(), 2026, "playerScores", "YTD")
	feeds.Read(t.Context(), 2026, "topAdds", "")
	if !reflect.DeepEqual(calls, []string{"W=YTD", "TYPE=playerScores", "TYPE=topAdds"}) {
		t.Fatal(calls)
	}
}

func TestHydrateArchiveReadOrderMissingInputsAndRetries(t *testing.T) {
	t.Parallel()
	at := time.Unix(1800000000, 0)
	input := ArchiveInput{Feeds: map[string]Feed[[]researchfeeds.Feed]{Key("topAdds", ""): {FetchedAt: at}}, News: map[string]Feed[[]researchfeeds.News]{"0042": {FetchedAt: at}}}
	var calls []string
	var seen map[string]bool
	var logs []string
	bodies := func(_ context.Context, part string, _ func(string, []byte, time.Time) bool) (bool, error) {
		calls = append(calls, part)
		if part == "TYPE=playerProfile" {
			return false, errors.New("profile archive failed")
		}
		return false, nil
	}
	source := ArchiveSource{Feeds: ArchiveFeeds{Bodies: bodies, Scoped: func(param, value string) ArchivedBodies { calls = append(calls, param+"="+value); return bodies }}, Season: 2026, Week: func() int { return 2 }, FirstRead: func(key string) bool { return FirstArchiveRead(&seen, key) }, Roster: func() state.MirrorSnapshot {
		calls = append(calls, "roster")
		return state.MirrorSnapshot{Players: []state.MirrorPlayer{{MFLID: "0042"}, {MFLID: "0099"}}}
	}, Injuries: func() Feed[[]leaguefeed.Injury] {
		calls = append(calls, "injuries")
		return Feed[[]leaguefeed.Injury]{FetchedAt: at}
	}, Log: func(line string) { logs = append(logs, line) }}
	got := HydrateArchive(t.Context(), input, []string{"0099", "0055"}, source)
	want := []string{"W=YTD", "TYPE=playerScores", "TYPE=topDrops", "TYPE=topStarters", "TYPE=topOwns", "TYPE=topTrades", "TYPE=pointsAllowed", "W=1", "TYPE=playerScores", "W=2", "TYPE=playerScores", researchfeeds.DepthURL(2026), "injuries", "roster", "TYPE=playerProfile"}
	if !reflect.DeepEqual(calls, want) || len(logs) != 2 || !strings.HasPrefix(logs[0], "depth charts: no archived copy:") || !strings.Contains(logs[1], "profile archive failed") {
		t.Fatalf("calls %v logs %v", calls, logs)
	}
	if got.Injuries.FetchedAt != at || got.Injuries.RefreshError != "Archived copy; refresh pending" || got.Depth.RefreshError != "No archived copy held" || len(got.Feeds) != 9 || len(input.Feeds) != 1 || len(input.News) != 1 {
		t.Fatalf("hydration mutated input or lost results: %+v", got)
	}
	if seen["news:0042"] || !seen["news:0099"] || !seen["news:0055"] || !seen["depth"] {
		t.Fatalf("read markers: %v", seen)
	}
	calls = nil
	logs = nil
	latest, injury := MergeArchive(Values{News: input.News}, input.Injuries, got, nil, "", nil, func() time.Time { return at })
	HydrateArchive(t.Context(), ArchiveInput{Feeds: latest.Feeds, Depth: latest.Depth, News: latest.News, Injuries: injury}, []string{"0099", "0055"}, source)
	if !reflect.DeepEqual(calls, []string{"roster"}) || len(logs) != 0 {
		t.Fatalf("already attempted archive was retried: %v %v", calls, logs)
	}
}

func TestHydrateDepthPreservesLiveFailure(t *testing.T) {
	t.Parallel()
	for _, note := range []string{"", "live depth failed"} {
		depth := Feed[[]researchfeeds.RawDepth]{RefreshError: note}
		var logs []string
		HydrateDepth(t.Context(), 2026, noNewsArchive, &depth, func(line string) { logs = append(logs, line) })
		if note == "" {
			if depth.RefreshError != "No archived copy held" || len(logs) != 1 {
				t.Fatalf("missing: %+v %v", depth, logs)
			}
		} else if depth.RefreshError != note || len(logs) != 0 {
			t.Fatalf("overwrote live error: %+v %v", depth, logs)
		}
	}
}

func TestArchiveLinksRefreshOnlyWhenMissingOrVersionChanged(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		held    map[string]string
		version string
		reads   int
	}{
		{"absent", nil, "loaded", 1}, {"same", map[string]string{}, "loaded", 0}, {"changed", map[string]string{"old": "0042"}, "old", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reads := 0
			sentinel := errors.New("crosswalk failed")
			got, err := ArchiveLinks(tc.held, tc.version, "loaded", func() (map[string]string, error) { reads++; return nil, sentinel })
			if reads != tc.reads {
				t.Fatal(reads)
			}
			if tc.reads == 0 {
				if err != nil || !reflect.DeepEqual(got, tc.held) {
					t.Fatalf("reuse: %v %v", got, err)
				}
			} else if !errors.Is(err, sentinel) {
				t.Fatal(err)
			}
		})
	}
}

func TestMergeArchivePreservesLiveAndOtherHeldState(t *testing.T) {
	t.Parallel()
	at := time.Unix(1800000000, 0)
	old := at.Add(-48 * time.Hour)
	live := Feed[[]researchfeeds.Feed]{FetchedAt: at, RefreshError: "live warning", Value: []researchfeeds.Feed{{Players: []researchfeeds.Entry{{Value: 42}}}}}
	archivedFeed := Feed[[]researchfeeds.Feed]{FetchedAt: old}
	current := Values{Feeds: map[string]Feed[[]researchfeeds.Feed]{"existing": live}, Depth: Feed[[]researchfeeds.RawDepth]{FetchedAt: at}, News: map[string]Feed[[]researchfeeds.News]{"existing": {FetchedAt: at, RefreshError: "live news warning"}}, Links: map[string]string{"old": "0042"}, LinkVersion: "old", Status: "in progress", FinishedAt: map[int]time.Time{1: at}, ArchiveRead: map[string]bool{"key": true}}
	injuries := Feed[[]leaguefeed.Injury]{FetchedAt: at, RefreshError: "live injuries warning"}
	archived := ArchiveInput{Feeds: map[string]Feed[[]researchfeeds.Feed]{"existing": archivedFeed, "missing": archivedFeed}, Depth: Feed[[]researchfeeds.RawDepth]{FetchedAt: old}, Injuries: Feed[[]leaguefeed.Injury]{FetchedAt: old}, News: map[string]Feed[[]researchfeeds.News]{"existing": {FetchedAt: old}, "old": {FetchedAt: old, RefreshError: "prior note"}, "today": {FetchedAt: at, RefreshError: "today note"}}}
	reads := 0
	got, injury := MergeArchive(current, injuries, archived, nil, "new", errors.New("crosswalk failed"), func() time.Time { reads++; return at })
	if !reflect.DeepEqual(got.Feeds["existing"], live) || got.Feeds["missing"].FetchedAt != old || got.Depth.FetchedAt != at || !reflect.DeepEqual(injury, injuries) {
		t.Fatalf("live values overwritten: %+v %+v", got, injury)
	}
	if got.News["existing"].RefreshError != "live news warning" || got.News["old"].RefreshError != "Archived copy; refresh pending" || got.News["today"].RefreshError != "today note" || reads != 2 {
		t.Fatalf("news: %+v clock reads %d", got.News, reads)
	}
	if got.LinkError != "crosswalk failed" || got.LinkVersion != "old" || got.Links["old"] != "0042" || got.Status != "in progress" || got.FinishedAt[1] != at || !got.ArchiveRead["key"] {
		t.Fatalf("other held state changed: %+v", got)
	}
	replacement := map[string]string{"new": "0099"}
	got, injury = MergeArchive(Values{}, Feed[[]leaguefeed.Injury]{}, archived, replacement, "new", nil, func() time.Time { return at })
	if !reflect.DeepEqual(got.Links, replacement) || got.LinkVersion != "new" || got.LinkError != "" || got.Depth.FetchedAt != old || injury.FetchedAt != old {
		t.Fatalf("empty merge: %+v %+v", got, injury)
	}
}
