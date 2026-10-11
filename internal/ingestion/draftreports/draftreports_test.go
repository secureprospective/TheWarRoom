package draftreports

import (
	"os"
	"testing"
	"time"
)

func TestRealSamples(t *testing.T) {
	for _, year := range []string{"2025", "2026"} {
		t.Run(year, func(t *testing.T) {
			body, err := os.ReadFile("testdata/results-" + year + ".json")
			if err != nil {
				t.Fatal(err)
			}
			picks, err := ParseResults(body)
			if err != nil || len(picks) != 160 {
				t.Fatalf("picks %d: %v", len(picks), err)
			}
			first, last := picks[0], picks[159]
			wantPlayer, wantOwner, wantTime := "17042", "0024", int64(1752494431)
			if year == "2026" {
				wantPlayer, wantOwner, wantTime = "17462", "0004", 1784030451
			}
			if first.Round != 1 || first.Slot != 1 || first.Player.String() != wantPlayer || first.Franchise != wantOwner ||
				!first.At.Equal(time.Unix(wantTime, 0)) || first.At.Location() != time.UTC || last.Round != 5 || last.Slot != 32 {
				t.Fatalf("first %+v last %+v", first, last)
			}
			body, err = os.ReadFile("testdata/future-" + year + ".json")
			if err != nil {
				t.Fatal(err)
			}
			future, err := ParseFuture(body)
			want := 320
			if year == "2026" {
				want = 320
			}
			if err != nil || len(future) != want {
				t.Fatalf("future %d: %v", len(future), err)
			}
			if future[0].Owner != "0002" || future[0].Original != "0002" || future[0].Round != 1 {
				t.Fatal(future[0])
			}
			traded := false
			for _, p := range future {
				if p.Owner != p.Original {
					traded = true
				}
			}
			if !traded {
				t.Fatal("lost ownership transfers")
			}
		})
	}
}

func TestShapesEmptyAndLeadingZeros(t *testing.T) {
	picks, err := ParseResults([]byte(`{"draftResults":{"draftUnit":{"draftPick":{"round":"01","pick":"02","player":"0531","franchise":"0002","timestamp":"0"}}}}`))
	if err != nil || len(picks) != 1 || picks[0].Player.String() != "0531" || !picks[0].At.IsZero() {
		t.Fatal(picks, err)
	}
	for _, body := range []string{`{"draftResults":{}}`, `{"draftResults":{"draftUnit":{"unit":"LEAGUE"}}}`} {
		rows, err := ParseResults([]byte(body))
		if err != nil || len(rows) != 0 {
			t.Fatal(rows, err)
		}
	}
	rows, err := ParseFuture([]byte(`{"futureDraftPicks":{"franchise":{"id":"0002","futureDraftPick":{"year":"2027","round":"1","originalPickFor":"0001"}}}}`))
	if err != nil || len(rows) != 1 || rows[0].Owner != "0002" || rows[0].Original != "0001" {
		t.Fatal(rows, err)
	}
	future, err := ParseFuture([]byte(`{"futureDraftPicks":{"franchise":{"id":"0001"}}}`))
	if err != nil || len(future) != 0 {
		t.Fatal(future, err)
	}
}

func TestRejectsMalformedExports(t *testing.T) {
	for _, body := range []string{"", `{}`, `{"draftResults":null}`, `{"error":{"$t":"offline"}}`,
		`{"draftResults":{"draftUnit":{"draftPick":{"round":"x","pick":"1","player":"0531","franchise":"0001"}}}}`,
		`{"draftResults":{"draftUnit":{"draftPick":{"round":"1","pick":"1","player":"x","franchise":"0001"}}}}`,
		`{"draftResults":{"draftUnit":{"draftPick":{"round":"1","pick":"1","player":"0531","franchise":"1"}}}}`,
		`{"draftResults":{"draftUnit":{"draftPick":{"round":"1","pick":"1","player":"0531","franchise":"0001","timestamp":"bad"}}}}`,
		`{"draftResults":{"draftUnit":{"unit":"DIVISION"}}}`} {
		if _, err := ParseResults([]byte(body)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	for _, body := range []string{`{}`, `{"futureDraftPicks":null}`,
		`{"futureDraftPicks":{"franchise":{"id":"1"}}}`,
		`{"futureDraftPicks":{"franchise":{"id":"0001","futureDraftPick":{"year":"2027","round":"0","originalPickFor":"0002"}}}}`} {
		if _, err := ParseFuture([]byte(body)); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestSkippedPickIsNotAPlayerOrRemainingSlot(t *testing.T) {
	body, err := os.ReadFile("testdata/results-2026.json")
	if err != nil {
		t.Fatal(err)
	}
	picks, err := ParseResults(body)
	if err != nil {
		t.Fatal(err)
	}
	skipped := picks[151]
	if !skipped.Skipped || skipped.Player != nil || skipped.Round != 5 || skipped.Slot != 24 || skipped.At.IsZero() {
		t.Fatalf("lost skipped-pick evidence: %+v", skipped)
	}
}

func TestDuplicateAssetsAreRefused(t *testing.T) {
	body := `{"draftResults":{"draftUnit":{"draftPick":[{"round":"01","pick":"01","player":"0531","franchise":"0001"},{"round":"1","pick":"1","player":"0532","franchise":"0002"}]}}}`
	if _, err := ParseResults([]byte(body)); err == nil {
		t.Fatal("accepted duplicate numeric slot")
	}
	body = `{"futureDraftPicks":{"franchise":[{"id":"0001","futureDraftPick":{"year":"2027","round":"1","originalPickFor":"0002"}},{"id":"0002","futureDraftPick":{"year":"2027","round":"01","originalPickFor":"0002"}}]}}`
	if _, err := ParseFuture([]byte(body)); err == nil {
		t.Fatal("accepted duplicate ownership")
	}
}
