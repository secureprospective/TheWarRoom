package draftreports

import (
	"testing"
)

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
