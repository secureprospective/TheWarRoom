package commissioner

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// midseasonLeague is the league every Midseason proof creates: a 32-team Default, season 2026.
const midseasonLeague = "native.midseason"

// midseasonRaw is a schedule change given a reason, placed now and visible to the league, as the store needs.
func midseasonRaw(sched RawScheduleArgs, command string) RawCommand {
	return RawCommand{League: midseasonLeague, Command: command, Args: RawArgs{Schedule: &sched},
		Request: command + " midseason", Reason: "Midseason proof", Effective: "now", Visibility: "league"}
}

// midseasonPrepare runs what the store runs before it records a schedule change: prepare, validate, then the
// ledger check. The first refusal is returned verbatim.
func midseasonPrepare(raw RawCommand, records []Record) (RawCommand, error) {
	prepared, err := raw.Prepared()
	if err != nil {
		return raw, err
	}
	if err := prepared.Validate(); err != nil {
		return prepared, err
	}
	return prepared, CheckScheduleAppend(prepared, records)
}

// midseasonAppend records an accepted schedule change the way the store does: gravity and the canonical
// request are filled in, a generate freezes its weeks on the entry, and the record must read back.
func midseasonAppend(t *testing.T, b *ledgerBuilder, raw RawCommand, at time.Time) {
	t.Helper()
	prepared, err := midseasonPrepare(raw, b.records)
	if err != nil {
		t.Fatalf("%s refused: %v", raw.Command, err)
	}
	definition, err := prepared.EffectiveDefinition()
	if err != nil {
		t.Fatalf("define %s: %v", raw.Command, err)
	}
	prepared.Gravity = definition.Gravity
	canonical, err := prepared.Canonical("creator")
	if err != nil {
		t.Fatalf("canonical %s: %v", raw.Command, err)
	}
	receipt := Receipt{Entry: fmt.Sprintf("entry.%d", len(b.records)+1), League: midseasonLeague,
		Revision: int64(len(b.records) + 1), RecordedAt: at, Message: "Recorded."}
	record := Record{CommandVersion: DurableCommandVersion, Command: prepared, Principal: "creator",
		Role: "commissioner", Gravity: definition.Gravity, Canonical: canonical, Receipt: receipt}
	if prepared.Command == ScheduleGenerate {
		if record.GeneratedSchedule, err = FreezeSchedule(prepared, b.records); err != nil {
			t.Fatalf("freeze the generated weeks: %v", err)
		}
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("recorded %s does not read back: %v", raw.Command, err)
	}
	b.records = append(b.records, record)
}

// midseasonTeams is the league's 32 franchises, in the ledger's order.
func midseasonTeams(t *testing.T, b *ledgerBuilder) []string {
	t.Helper()
	teams := LeagueFranchises(b.records)
	if len(teams) != 32 {
		t.Fatalf("league has %d franchises, want 32", len(teams))
	}
	return teams
}

// midseasonWeek is one schedule week as a full round of 16 games. The circle method pairs every franchise
// once, so nobody rests and the game IDs are unique across weeks.
func midseasonWeek(teams []string, week int) ScheduleWeek {
	n := len(teams)
	rot := (week - 1) % (n - 1)
	at := func(j int) string { return teams[1+(j+rot)%(n-1)] }
	games := []ScheduleGame{{ID: fmt.Sprintf("w%d-g00", week), Home: teams[0], Away: at(0)}}
	for i := 1; i < n/2; i++ {
		games = append(games, ScheduleGame{ID: fmt.Sprintf("w%d-g%02d", week, i), Home: at(i), Away: at(n - 1 - i)})
	}
	return ScheduleWeek{Week: week, Games: games}
}

// midseasonRange is the full rounds for schedule weeks first..last, numbered as the schedule numbers them.
func midseasonRange(teams []string, first, last int) []ScheduleWeek {
	var weeks []ScheduleWeek
	for w := first; w <= last; w++ {
		weeks = append(weeks, midseasonWeek(teams, w))
	}
	return weeks
}

// midseasonSummary is each week as week:games, so a reader can see the numbering and the games together.
func midseasonSummary(weeks []ScheduleWeek) string {
	parts := make([]string, 0, len(weeks))
	for _, w := range weeks {
		parts = append(parts, fmt.Sprintf("%d:%d", w.Week, len(w.Games)))
	}
	return strings.Join(parts, " ")
}

// midseasonPublished reads the published schedule the way a franchise's schedule reading does.
func midseasonPublished(t *testing.T, b *ledgerBuilder) []ScheduleWeek {
	t.Helper()
	reading, err := ScheduleReading(midseasonLeague, b.records)
	if err != nil {
		t.Fatalf("read published schedule: %v", err)
	}
	return reading.Weeks
}

// TestMidseasonLeadingEmptyWeeks is the rehearsal shape: Weeks=18, weeks 1..13 empty, 14..18 full
// (drafts Tue Dec 8; week 14 is the first week after the drafts).
// The schedule accepts empty weeks (validateFixtures), so this is the shape that matches the need.
func TestMidseasonLeadingEmptyWeeks(t *testing.T) {
	b := twin32Start(t, midseasonLeague, "")
	teams := midseasonTeams(t, b)
	var weeks []ScheduleWeek
	for w := 1; w <= 13; w++ {
		weeks = append(weeks, ScheduleWeek{Week: w, Games: []ScheduleGame{}})
	}
	weeks = append(weeks, midseasonRange(teams, 14, 18)...)
	gen := midseasonRaw(RawScheduleArgs{Weeks: 18, Fixtures: weeks}, ScheduleGenerate)
	midseasonAppend(t, b, gen, seasonMoment(5))
	midseasonAppend(t, b, midseasonRaw(RawScheduleArgs{}, SchedulePublish), seasonMoment(6))
	published := midseasonPublished(t, b)
	t.Logf("(d) published weeks week:games = %s", midseasonSummary(published))
	if len(published) != 18 {
		t.Fatalf("published %d weeks, want 18", len(published))
	}
	for _, w := range published {
		want := 0
		if w.Week >= 14 {
			want = 16
		}
		if len(w.Games) != want {
			t.Fatalf("week %d has %d games, want %d", w.Week, len(w.Games), want)
		}
	}
}
