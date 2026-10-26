package control

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestMergeMFLCalendarHeldOnlyOrderingAndLabels(t *testing.T) {
	t.Parallel()
	now := time.Unix(1800000000, 0)
	later := now.Add(time.Hour)
	held := report.HeldFeed[[]ingestion.RawCalendarEvent]{FetchedAt: now, Value: []ingestion.RawCalendarEvent{
		{ID: "past", Title: "Past", Start: strconv.FormatInt(now.Add(-time.Second).Unix(), 10)},
		{ID: "next", Title: "Dated title", Start: strconv.FormatInt(later.Unix(), 10)},
		{ID: "undated", Type: "ROOKIE_DRAFT"},
		{ID: "now", Type: "LINEUP_LOCK", Start: strconv.FormatInt(now.Unix(), 10)},
	}}
	before := append([]ingestion.RawCalendarEvent(nil), held.Value...)
	clock := snapshot.Sourced[leagueclock.Reading]{Value: leagueclock.Reading{Season: 2026, Deadlines: []leagueclock.Deadline{{ID: "app", At: &later, Urgency: leagueclock.U3}}}, Provenance: snapshot.Provenance{Source: "app", Freshness: domain.Freshness{Note: "base"}}}
	MergeMFLCalendar(now, &clock, held)
	ids := []string{}
	labels := map[string]string{}
	for _, d := range clock.Value.Deadlines {
		ids = append(ids, d.ID)
		labels[d.ID] = d.Label
	}
	if !reflect.DeepEqual(ids, []string{"mfl-calendar:now", "app", "mfl-calendar:next", "mfl-calendar:undated"}) {
		t.Fatalf("order: %v", ids)
	}
	if labels["mfl-calendar:next"] != "MFL · Dated title" || labels["mfl-calendar:undated"] != "MFL · rookie draft" {
		t.Fatalf("labels: %v", labels)
	}
	if clock.Value.Season != 2026 || clock.Provenance.Source != "app+held-mfl-calendar" || !strings.HasPrefix(clock.Provenance.Freshness.Note, "base; MFL calendar ") || !reflect.DeepEqual(held.Value, before) {
		t.Fatalf("preserved fields/inputs: %+v", clock)
	}
}

func TestMergeMFLCalendarUnavailable(t *testing.T) {
	t.Parallel()
	clock := snapshot.Sourced[leagueclock.Reading]{Provenance: snapshot.Provenance{Source: "app"}}
	MergeMFLCalendar(time.Unix(1800000000, 0), &clock, report.HeldFeed[[]ingestion.RawCalendarEvent]{})
	if clock.Provenance.Source != "app+held-mfl-calendar" || clock.Provenance.Freshness.Note != "; MFL calendar unavailable; showing app clocks only" || len(clock.Value.Deadlines) != 0 {
		t.Fatalf("unavailable: %+v", clock)
	}
}

func TestSeasonGuess(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 12, 31, 23, 59, 0, 0, time.FixedZone("owner", -6*60*60))
	if SeasonGuess(2025, now) != 2025 || SeasonGuess(0, now) != 2026 {
		t.Fatal("held season must win, fallback uses caller calendar year")
	}
}
