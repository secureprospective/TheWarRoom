package history

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func TestDepthMembershipIsDatedAndDoesNotRetainDepartedPlayers(t *testing.T) {
	cutoff := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	rows := []researchfeeds.RawDepth{
		{DT: "2026-09-08T12:00:00Z", Team: "BUF", Name: "Departed"},
		{DT: "2026-09-09T12:00:00Z", Team: "NYJ", Name: "Current"},
		{DT: "2026-09-11T12:00:00Z", Team: "MIA", Name: "Future"},
	}
	got := depthAt(rows, cutoff)
	if len(got) != 1 || got[0].Name != "Current" {
		t.Fatalf("wrong historical affiliation: %+v", got)
	}
	if got := depthAt(rows, cutoff.AddDate(0, 0, 10)); len(got) != 0 {
		t.Fatalf("stale roster admitted: %+v", got)
	}
	if got := depthAt(rows, cutoff.AddDate(0, 0, -10)); len(got) != 0 {
		t.Fatalf("future roster admitted: %+v", got)
	}
}
