package history

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func TestExtendFootballScheduleKeepsOldEvidenceAndAddsOnlyNewWeeks(t *testing.T) {
	store, _, _ := newStore(t)
	ctx := t.Context()
	original, digest := footballBody(t)
	if err := store.ImportFootball(ctx, original, digest); err != nil {
		t.Fatal(err)
	}
	before, err := store.LoadFootball(ctx, digest)
	if err != nil {
		t.Fatal(err)
	}

	added := leagueweek.Week{Number: 2, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)}}}
	asOf := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	next, err := store.ExtendFootballSchedule(ctx, digest, []leagueweek.Week{added}, asOf)
	if err != nil {
		t.Fatal(err)
	}
	if next == digest {
		t.Fatal("schedule extension reused the immutable evidence digest")
	}
	after, err := store.LoadFootball(ctx, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.raw.Weeks) != 1 || before.raw.Weeks[0].Number != 1 {
		t.Fatalf("old digest changed: %+v", before.raw.Weeks)
	}
	if len(after.raw.Weeks) != 2 || after.raw.Weeks[0].Number != 1 || after.raw.Weeks[1].Number != 2 {
		t.Fatalf("new schedule = %+v", after.raw.Weeks)
	}
	if !reflect.DeepEqual(before.raw.Players, after.raw.Players) {
		t.Fatal("schedule extension changed player evidence")
	}
	const source = "; live MFL nflSchedule dates"
	if !strings.HasSuffix(after.raw.Source, source) {
		t.Fatalf("schedule provenance = %q", after.raw.Source)
	}
	if !after.raw.AsOf.Equal(asOf) {
		t.Fatalf("observation time = %s, want %s", after.raw.AsOf, asOf)
	}
}

func TestExtendFootballScheduleRejectsReplacementWithoutWriting(t *testing.T) {
	store, _, _ := newStore(t)
	ctx := t.Context()
	original, digest := footballBody(t)
	if err := store.ImportFootball(ctx, original, digest); err != nil {
		t.Fatal(err)
	}
	rows := footballRows(t, store)
	duplicate := leagueweek.Week{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)}}}
	if _, err := store.ExtendFootballSchedule(ctx, digest, []leagueweek.Week{duplicate}, time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("existing week was replaced")
	}
	if got := footballRows(t, store); got != rows {
		t.Fatalf("rejected extension wrote evidence: before=%v after=%v", rows, got)
	}
}
