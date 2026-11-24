package history

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func TestPinnedKickoffsReturnACopy(t *testing.T) {
	kickoff := time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC)
	pinned := &PinnedFootball{raw: RawFootballEvidence{Weeks: []leagueweek.Week{{Games: []leagueweek.Game{{Kickoff: kickoff}}}}}}
	first := pinned.Kickoffs()
	first[0] = kickoff.Add(time.Hour)
	if got := pinned.Kickoffs()[0]; !got.Equal(kickoff) {
		t.Fatalf("caller changed pinned kickoff: got %s, want %s", got, kickoff)
	}
}

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

func TestReviseFootballScheduleReplacesHeldWeekAndKeepsOldEvidence(t *testing.T) {
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
	final := leagueweek.Week{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC), Final: true}}}
	added := leagueweek.Week{Number: 2, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)}}}
	asOf := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	next, err := store.ReviseFootballSchedule(ctx, digest, []leagueweek.Week{final}, []leagueweek.Week{added}, asOf)
	if err != nil {
		t.Fatal(err)
	}
	if next == digest {
		t.Fatal("revision reused the immutable evidence digest")
	}
	after, err := store.LoadFootball(ctx, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.raw.Weeks) != 1 || before.raw.Weeks[0].Games[0].Final {
		t.Fatalf("old digest changed: %+v", before.raw.Weeks)
	}
	if len(after.raw.Weeks) != 2 || !after.raw.Weeks[0].Games[0].Final || after.raw.Weeks[1].Number != 2 {
		t.Fatalf("revised schedule = %+v", after.raw.Weeks)
	}
	if !reflect.DeepEqual(before.raw.Players, after.raw.Players) {
		t.Fatal("revision changed player evidence")
	}
}

func TestReviseFootballScheduleRejectsReplacingUnheldWeekWithoutWriting(t *testing.T) {
	store, _, _ := newStore(t)
	ctx := t.Context()
	original, digest := footballBody(t)
	if err := store.ImportFootball(ctx, original, digest); err != nil {
		t.Fatal(err)
	}
	rows := footballRows(t, store)
	ghost := leagueweek.Week{Number: 7, Games: []leagueweek.Game{{Teams: [2]string{"BUF", "MIA"}, Kickoff: time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC), Final: true}}}
	if _, err := store.ReviseFootballSchedule(ctx, digest, []leagueweek.Week{ghost}, nil, time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("unheld week was replaced")
	}
	if got := footballRows(t, store); got != rows {
		t.Fatalf("rejected revision wrote evidence: before=%v after=%v", rows, got)
	}
}
