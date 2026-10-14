package commissioner

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func seasonMoment(day int) time.Time { return time.Date(2026, 10, day, 12, 0, 0, 0, time.UTC) }

// ledgerBuilder appends records the way the store does: ordered, each stamped with its own entry.
type ledgerBuilder struct {
	t       *testing.T
	records []Record
}

func newLedger(t *testing.T, preset string) *ledgerBuilder {
	t.Helper()
	create, err := RawCommand{League: "native.test", Command: Create, Args: RawArgs{Name: "Ours", Preset: preset,
		FranchiseCount: map[string]int{PresetLegacy: 32, PresetEmpty: 4}[preset]}, Request: "create",
		Effective: "now", Visibility: "league"}.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	b := &ledgerBuilder{t: t}
	b.add(create, seasonMoment(1))
	return b
}

func (b *ledgerBuilder) add(raw RawCommand, at time.Time) string {
	b.t.Helper()
	entry := fmt.Sprintf("entry.%d", len(b.records)+1)
	b.records = append(b.records, Record{Command: raw, Principal: "creator", Role: "commissioner",
		Receipt: Receipt{Entry: entry, League: raw.League, Revision: int64(len(b.records) + 1), RecordedAt: at}})
	return entry
}

func (b *ledgerBuilder) set(field, value, effective string, at time.Time) string {
	b.t.Helper()
	return b.add(RawCommand{League: "native.test", Command: RulesSet, Request: fmt.Sprintf("r%d", len(b.records)),
		Args: RawArgs{Field: field, Value: json.RawMessage(value)}, Effective: effective,
		Visibility: "league"}, at)
}

func (b *ledgerBuilder) revert(entry string, at time.Time) {
	b.t.Helper()
	b.add(RawCommand{League: "native.test", Command: Revert, Args: RawArgs{Entry: entry},
		Request: "undo." + entry, Effective: "now", Visibility: "league"}, at)
}

func (b *ledgerBuilder) seed(season int, divisions map[string]string, at time.Time) {
	b.t.Helper()
	native := &RawNativeSnapshot{Season: season}
	for i := 1; i <= 32; i++ {
		id := fmt.Sprintf("franchise.%02d", i)
		native.Franchises = append(native.Franchises, NativeFranchise{ID: id, Division: divisions[id]})
	}
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: native},
		Request: "seed", Effective: "now", Visibility: "league"}, at)
}

func rosterMax(t *testing.T, records []Record, season int, at time.Time) int {
	t.Helper()
	rules, err := RulesInForce(records, season, at)
	if err != nil {
		t.Fatal(err)
	}
	return rules.Roster.Limits.Value.Max.Value
}

const rolloverAt = `"2027-03-01T05:00:00Z"`

func TestNextRolloverChangeCountsOnlyFromItsRollover(t *testing.T) {
	rollover := time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC)
	b := newLedger(t, PresetLegacy)
	b.set("calendar.rollover", rolloverAt, "now", seasonMoment(2))
	b.set("roster.limits", `{"min":20,"max":70}`, "next_rollover", seasonMoment(3))
	first := firstSeason(b.records)
	for _, c := range []struct {
		name   string
		season int
		at     time.Time
		want   int
	}{
		{"waiting before its rollover", first, rollover.Add(-time.Nanosecond), 80},
		{"in force at the rollover", first + 1, rollover, 70},
		{"in force long after", first + 1, rollover.AddDate(0, 6, 0), 70},
		{"the season that ended keeps its own rules", first, rollover.AddDate(0, 6, 0), 80},
	} {
		if got := rosterMax(t, b.records, c.season, c.at); got != c.want {
			t.Errorf("%s: roster maximum %d, want %d", c.name, got, c.want)
		}
	}
	if season, err := SeasonAt(b.records, rollover.Add(-time.Nanosecond)); err != nil || season != first {
		t.Fatalf("season before the rollover: %d %v", season, err)
	}
	if season, err := SeasonAt(b.records, rollover); err != nil || season != first+1 {
		t.Fatalf("season at the rollover: %d %v", season, err)
	}
}

func TestNextRolloverWithoutADateNeverCounts(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set("roster.limits", `{"min":20,"max":70}`, "next_rollover", seasonMoment(3))
	if got := rosterMax(t, b.records, firstSeason(b.records), seasonMoment(30)); got != 80 {
		t.Fatalf("a change with no rollover to wait for counted: maximum %d", got)
	}
}

func TestWithdrawnNextRolloverChangeDoesNotCount(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set("calendar.rollover", rolloverAt, "now", seasonMoment(2))
	entry := b.set("roster.limits", `{"min":20,"max":70}`, "next_rollover", seasonMoment(3))
	b.revert(entry, seasonMoment(4))
	if got := rosterMax(t, b.records, firstSeason(b.records)+1, time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC)); got != 80 {
		t.Fatalf("withdrawn change counted: maximum %d", got)
	}
}

func TestScheduledChangeBeatsALaterNowChangeAtItsRollover(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set("calendar.rollover", rolloverAt, "now", seasonMoment(2))
	b.set("roster.limits", `{"min":20,"max":70}`, "next_rollover", seasonMoment(3))
	b.set("roster.limits", `{"min":18,"max":75}`, "now", seasonMoment(4))
	first := firstSeason(b.records)
	if got := rosterMax(t, b.records, first, seasonMoment(5)); got != 75 {
		t.Fatalf("this season's own change: maximum %d", got)
	}
	if got := rosterMax(t, b.records, first+1, time.Date(2027, 3, 2, 0, 0, 0, 0, time.UTC)); got != 70 {
		t.Fatalf("scheduled change lost to an earlier now change: maximum %d", got)
	}
}

func TestRolloverDateMovesUntilItPasses(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set("calendar.rollover", rolloverAt, "now", seasonMoment(2))
	b.set("calendar.rollover", `"2027-03-05T05:00:00Z"`, "now", seasonMoment(3))
	starts, err := rolloverInstants(b.records)
	if err != nil || len(starts) != 1 || !starts[0].Equal(time.Date(2027, 3, 5, 5, 0, 0, 0, time.UTC)) {
		t.Fatalf("a corrected date became a second season: %v %v", starts, err)
	}
	b.set("calendar.rollover", `"2028-03-01T05:00:00Z"`, "now", time.Date(2027, 3, 6, 0, 0, 0, 0, time.UTC))
	starts, err = rolloverInstants(b.records)
	if err != nil || len(starts) != 2 {
		t.Fatalf("a date set after the rollover passed did not schedule the next: %v %v", starts, err)
	}
	if season, err := SeasonAt(b.records, time.Date(2028, 3, 2, 0, 0, 0, 0, time.UTC)); err != nil ||
		season != firstSeason(b.records)+2 {
		t.Fatalf("season after two rollovers: %d %v", season, err)
	}
}

func TestSeasonReadingsRefuseSeasonsOutsideTheLeague(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set("calendar.rollover", rolloverAt, "now", seasonMoment(2))
	first := firstSeason(b.records)
	if _, err := RulesInForce(b.records, first-1, seasonMoment(5)); err == nil {
		t.Fatal("a season before the league began was read")
	}
	if _, err := RulesInForce(b.records, first+2, seasonMoment(5)); err == nil {
		t.Fatal("a season past the next rollover was read")
	}
	if _, err := RulesInForce(b.records, first+1, seasonMoment(5)); err == nil {
		t.Fatal("a season that has not started was read")
	}
}

func alignmentOf(t *testing.T, records []Record) map[string]string {
	t.Helper()
	rules, err := RulesNow(records, seasonMoment(30))
	if err != nil {
		t.Fatal(err)
	}
	if rules.League.Alignment.State != leaguerules.IsKnown {
		return nil
	}
	return rules.League.Alignment.Value
}

func TestSeedDivisionsCarryIntoTheResolvedAlignment(t *testing.T) {
	held := map[string]string{"franchise.01": "East", "franchise.02": "West"}
	b := newLedger(t, PresetLegacy)
	if got := alignmentOf(t, b.records); got != nil {
		t.Fatalf("an unseeded league has an alignment: %v", got)
	}
	b.seed(2026, held, seasonMoment(2))
	if got := alignmentOf(t, b.records); got["franchise.01"] != "East" || got["franchise.02"] != "West" || len(got) != 2 {
		t.Fatalf("seeded divisions were not carried: %v", got)
	}
	set := b.set("league.alignment", `{"franchise.01":"North"}`, "now", seasonMoment(3))
	if got := alignmentOf(t, b.records); got["franchise.01"] != "North" || len(got) != 1 {
		t.Fatalf("the commissioner's alignment did not replace the carried one: %v", got)
	}
	b.revert(set, seasonMoment(4))
	if got := alignmentOf(t, b.records); got["franchise.01"] != "East" || len(got) != 2 {
		t.Fatalf("compensation did not restore the carried alignment: %v", got)
	}
}

func TestCommissionersAlignmentBeforeSeedingIsNotOverridden(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set("league.alignment", `{"franchise.01":"North"}`, "now", seasonMoment(2))
	b.seed(2026, map[string]string{"franchise.01": "East", "franchise.02": "West"}, seasonMoment(3))
	if got := alignmentOf(t, b.records); got["franchise.01"] != "North" || len(got) != 1 {
		t.Fatalf("seeding replaced an alignment the commissioner chose: %v", got)
	}
}

func TestRulesReadingAgreesWithTheResolverAboutWaitingChanges(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set("calendar.rollover", rolloverAt, "now", seasonMoment(2))
	b.set("roster.limits", `{"min":20,"max":70}`, "next_rollover", seasonMoment(3))
	row := func(now time.Time) RuleRow {
		reading, err := RulesReading("native.test", b.records, now)
		if err != nil {
			t.Fatal(err)
		}
		for _, section := range reading.Sections {
			for _, r := range section.Rows {
				if r.Field == "roster.limits" {
					return r
				}
			}
		}
		t.Fatal("no roster limits row")
		return RuleRow{}
	}
	before := row(seasonMoment(20))
	if len(before.Pending) != 1 || before.Value != row(seasonMoment(21)).Value {
		t.Fatalf("a waiting change is not shown as waiting: %+v", before)
	}
	after := row(time.Date(2027, 3, 2, 0, 0, 0, 0, time.UTC))
	if len(after.Pending) != 0 || after.Value == before.Value || after.ChangedAt == "" {
		t.Fatalf("a change whose rollover has come still waits or is not in force: %+v", after)
	}
}

func TestHistoricalRulesIgnoreLaterCompensation(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set("calendar.rollover", rolloverAt, "now", seasonMoment(2))
	entry := b.set("roster.limits", `{"min":18,"max":75}`, "now", seasonMoment(3))
	b.revert(entry, time.Date(2027, 3, 2, 0, 0, 0, 0, time.UTC))
	first := firstSeason(b.records)
	if got := rosterMax(t, b.records, first, seasonMoment(2)); got != 80 {
		t.Fatalf("future change affected past instant: %d", got)
	}
	if got := rosterMax(t, b.records, first, time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC)); got != 75 {
		t.Fatalf("later compensation rewrote ended season: %d", got)
	}
	if got := rosterMax(t, b.records, first+1, time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC)); got != 80 {
		t.Fatalf("compensation did not restore current season: %d", got)
	}
}

func TestFutureRolloverCorrectionDoesNotRedateThePast(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set("calendar.rollover", rolloverAt, "now", seasonMoment(2))
	b.set("calendar.rollover", `"2027-04-01T05:00:00Z"`, "now", time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC))
	at := seasonMoment(3)
	starts, err := rolloverInstants(recordsThrough(b.records, at))
	if err != nil || len(starts) != 1 || starts[0].Month() != time.March {
		t.Fatalf("future correction changed the date held at a past instant: %v %v", starts, err)
	}
}

func TestReversingAnElapsedRolloverCannotEraseASeason(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	rollover := b.set("calendar.rollover", rolloverAt, "now", seasonMoment(2))
	b.set("roster.limits", `{"min":18,"max":75}`, "now", seasonMoment(3))
	b.revert(rollover, time.Date(2027, 3, 2, 0, 0, 0, 0, time.UTC))
	b.set("roster.limits", `{"min":18,"max":90}`, "now", time.Date(2027, 3, 3, 0, 0, 0, 0, time.UTC))
	at := time.Date(2027, 4, 1, 0, 0, 0, 0, time.UTC)
	first := firstSeason(b.records)
	if got, err := SeasonAt(b.records, at); err != nil || got != first+1 {
		t.Fatalf("elapsed season erased by compensation: %d %v", got, err)
	}
	if got := rosterMax(t, b.records, first, at); got != 75 {
		t.Fatalf("ended season absorbed later rules: %d", got)
	}
	if got := rosterMax(t, b.records, first+1, at); got != 90 {
		t.Fatalf("current season failed to apply later rules: %d", got)
	}
}
