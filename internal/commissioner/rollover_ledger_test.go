package commissioner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const rolloverLeague = "native.test"

func rolloverMoment() time.Time { return time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC) }

// ledgerSnapshot is a seeded Legacy league's assets: franchise.02 holds one taxi player whose contract
// ends this season, one IR player and one active player; franchise.01 holds a pick owned by franchise.02.
func ledgerSnapshot() *RawNativeSnapshot {
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	native := &RawNativeSnapshot{Season: 2026, RosterAt: at, DraftAt: at, FutureAt: at,
		Players: []NativePlayer{
			{ID: "0001", Name: "Taxi Expiring", Franchise: "franchise.02", Status: domain.RosterTaxi, Salary: 100,
				ExpirationYear: 2026},
			{ID: "0002", Name: "Injured Veteran", Franchise: "franchise.02", Status: domain.RosterIR, Salary: 200,
				ExpirationYear: 2028},
			{ID: "0003", Name: "Starter", Franchise: "franchise.02", Status: domain.RosterActive, Salary: 300,
				ExpirationYear: 2027},
		},
		Picks:       []NativePick{{Season: 2028, Round: 1, Original: "franchise.01", Owner: "franchise.02"}},
		Adjustments: []NativeCharge{{ID: "held", Franchise: "franchise.02", Amount: 123}}}
	for i := 1; i <= 32; i++ {
		native.Franchises = append(native.Franchises, NativeFranchise{ID: fmt.Sprintf("franchise.%02d", i),
			HeldID: fmt.Sprintf("%04d", i), Name: fmt.Sprintf("Team %d", i), Division: fmt.Sprintf("div.%d", (i-1)/4)})
	}
	return native
}

// rolloverLedger is a seeded Legacy league whose calendar rulings are made; time moves one day per entry.
func rolloverLedger(t *testing.T) *ledgerBuilder {
	t.Helper()
	b := newLedger(t, PresetLegacy)
	b.add(RawCommand{League: rolloverLeague, Command: NativeSeed, Args: RawArgs{Native: ledgerSnapshot()},
		Request: "seed", Effective: "now", Visibility: "league"}, seasonMoment(2))
	b.set("calendar.boundary", `"start_of_day"`, "now", seasonMoment(3))
	b.set("calendar.tradeDeadline", `"2026-11-09T05:00:00Z"`, "now", seasonMoment(4))
	return b
}

func (b *ledgerBuilder) rolloverCommand(to int, dry bool) RawCommand {
	return RawCommand{League: rolloverLeague, Command: SeasonRollover, Gravity: "G3",
		Args:   RawArgs{Rollover: &RawRolloverArgs{To: to, DryRun: dry}},
		Reason: "Turn the league year", Expected: int64(len(b.records)), Request: fmt.Sprintf("rollover.%d", to),
		Effective: "now", Visibility: "league"}
}

// rollover records the act the way the store does: prepared under the ledger, then validated as read back.
func (b *ledgerBuilder) rollover(to int, now time.Time) string {
	b.t.Helper()
	raw := b.rolloverCommand(to, false)
	before, plan, err := PrepareRollover(raw, b.records, now)
	if err != nil {
		b.t.Fatal(err)
	}
	canonical, err := raw.Canonical("creator")
	if err != nil {
		b.t.Fatal(err)
	}
	record := Record{Command: raw, Principal: "creator", Role: "commissioner", Gravity: "G3", Canonical: canonical,
		Before: before, Receipt: Receipt{Entry: fmt.Sprintf("entry.%d", len(b.records)+1), League: rolloverLeague,
			Revision: int64(len(b.records) + 1), RecordedAt: before.At, Plan: plan}}
	if err := record.Validate(); err != nil {
		b.t.Fatal(err)
	}
	b.records = append(b.records, record)
	return record.Receipt.Entry
}

func (b *ledgerBuilder) waitingRule(field, value string, day int) string {
	return b.set(field, value, "next_rollover", seasonMoment(day))
}

func readRoster(t *testing.T, b *ledgerBuilder, now time.Time) NativeRosterCheck {
	t.Helper()
	check, err := NativeRosterCheckReading(rolloverLeague, b.records, now, nativecheck.Facts{})
	if err != nil {
		t.Fatal(err)
	}
	return check
}

func playersByID(t *testing.T, records []Record) map[string]NativePlayer {
	t.Helper()
	assets, err := NativeAssetsReading(rolloverLeague, records)
	if err != nil || !assets.Seeded {
		t.Fatalf("assets: %+v %v", assets, err)
	}
	out := map[string]NativePlayer{}
	for _, p := range assets.Snapshot.Players {
		out[p.ID] = p
	}
	return out
}

func rosterMaxNow(t *testing.T, b *ledgerBuilder, now time.Time) int {
	t.Helper()
	rules, err := RulesNow(b.records, now)
	if err != nil {
		t.Fatal(err)
	}
	return rules.Roster.Limits.Value.Max.Value
}

func TestRolloverActMovesEveryReadingToTheNewSeason(t *testing.T) {
	b := rolloverLedger(t)
	now := rolloverMoment()
	if got := readRoster(t, b, now); got.RosterSeason != 2026 || got.RulesSeason != 2026 {
		t.Fatalf("before the act: %+v", got)
	}
	b.rollover(2027, now)
	check := readRoster(t, b, now)
	if check.RosterSeason != 2027 || check.RulesSeason != 2027 {
		t.Fatalf("the roster check still reads the old season: %+v", check)
	}
	directory, err := NativeDirectoryReading(rolloverLeague, b.records, now)
	if err != nil || directory.RosterSeason != 2027 {
		t.Fatalf("the directory still reads the old season: %+v %v", directory, err)
	}
	if season, err := SeasonAt(b.records, now); err != nil || season != 2027 {
		t.Fatalf("season resolver: %d %v", season, err)
	}
	deadline, err := RulesNow(b.records, now)
	if err != nil || deadline.Calendar.TradeDeadline.Value.Year() != 2027 ||
		deadline.Calendar.TradeDeadline.Value.Month() != time.November {
		t.Fatalf("the trade deadline was not re-dated to the new season: %+v %v", deadline.Calendar.TradeDeadline, err)
	}
	clock, err := CalendarReading(rolloverLeague, b.records, now, 2027, "trade", "franchise.02", time.UTC)
	if err != nil || len(clock.Clock.Windows) == 0 || clock.Clock.Windows[0].Start.Year() != 2027 ||
		clock.Clock.TradeDeadline.Year() != 2027 {
		t.Fatalf("the calendar was not re-dated: %+v %v", clock.Clock, err)
	}
}

func TestRolloverActReturnsReservesAndMarksExpiringPending(t *testing.T) {
	b := rolloverLedger(t)
	b.rollover(2027, rolloverMoment())
	players := playersByID(t, b.records)
	for id, p := range players {
		if p.Status != domain.RosterActive || p.Franchise != "franchise.02" {
			t.Fatalf("player %s left on reserve or moved: %+v", id, p)
		}
	}
	if !players["0001"].Pending || players["0002"].Pending || players["0003"].Pending {
		t.Fatalf("only the expiring player is pending: %+v", players)
	}
	assets, _ := NativeAssetsReading(rolloverLeague, b.records)
	if !reflect.DeepEqual(assets.Snapshot.Picks, ledgerSnapshot().Picks) {
		t.Fatal("picks were issued or changed at rollover")
	}
}

func TestRolloverActAppliesOnlyTheNewestWaitingChange(t *testing.T) {
	b := rolloverLedger(t)
	b.waitingRule("roster.limits", `{"min":20,"max":70}`, 5)
	b.waitingRule("roster.limits", `{"min":22,"max":75}`, 6)
	b.waitingRule("cap.amount", `9000000000`, 7)
	if got := rosterMaxNow(t, b, rolloverMoment()); got != 80 {
		t.Fatalf("a waiting change counted before its rollover: %d", got)
	}
	b.rollover(2027, rolloverMoment())
	if got := rosterMaxNow(t, b, rolloverMoment()); got != 75 {
		t.Fatalf("roster maximum after the rollover = %d, want the newest waiting change (75)", got)
	}
	rules, err := RulesNow(b.records, rolloverMoment())
	if err != nil || rules.Cap.Amount.Value != 9000000000 || rules.Roster.Limits.Value.Min.Value != 22 {
		t.Fatalf("waiting rules after the rollover: %+v %v", rules.Roster.Limits, err)
	}
	reading, err := RulesReading(rolloverLeague, b.records, rolloverMoment())
	if err != nil || rules.Season != 2027 {
		t.Fatalf("rules reading: %d %v", rules.Season, err)
	}
	for _, section := range reading.Sections {
		for _, row := range section.Rows {
			for _, pending := range row.Pending {
				if pending.When == waitingWords {
					t.Fatalf("%s still waits after the rollover took it", row.Label)
				}
			}
		}
	}
}

func TestRolloverActKeepsTheClosedSeasonsOwnRules(t *testing.T) {
	b := rolloverLedger(t)
	b.waitingRule("roster.limits", `{"min":22,"max":75}`, 5)
	b.rollover(2027, rolloverMoment())
	b.set("roster.limits", `{"min":10,"max":90}`, "now", rolloverMoment().Add(time.Hour))
	later := rolloverMoment().Add(2 * time.Hour)
	if got := rosterMax(t, b.records, 2026, later); got != 80 {
		t.Fatalf("the closed season's rules moved to %d", got)
	}
	if got := rosterMax(t, b.records, 2027, later); got != 90 {
		t.Fatalf("a change made in the new season does not apply to it: %d", got)
	}
	if _, err := RulesInForce(b.records, 2028, later); err == nil {
		t.Fatal("a season the league has not reached was readable")
	}
}

func TestRolloverRevertRestoresTheBeforeState(t *testing.T) {
	b := rolloverLedger(t)
	b.waitingRule("roster.limits", `{"min":22,"max":75}`, 5)
	before := snapshotReadings(t, b)
	act := b.rollover(2027, rolloverMoment())
	if reflect.DeepEqual(snapshotReadings(t, b), before) {
		t.Fatal("the act changed nothing")
	}
	b.revert(act, rolloverMoment().Add(time.Hour))
	got := snapshotReadings(t, b)
	if got.season != 2026 || got.rosterMax != 80 || got.deadline.Year() != 2026 || got.taxi != 1 || got.ir != 1 ||
		got.pending != 0 || got.rosterSeason != 2026 {
		t.Fatalf("the revert left the new season in force: %+v", got)
	}
	if !reflect.DeepEqual(got, before) {
		t.Fatalf("revert did not restore the before-state:\n got %+v\nwant %+v", got, before)
	}
	restored := b.rollover(2027, rolloverMoment().Add(2*time.Hour))
	if season, _ := SeasonAt(b.records, rolloverMoment().Add(3*time.Hour)); season != 2027 || restored == "" {
		t.Fatal("the league could not roll over again after the revert")
	}
}

// readings are what a GM can see of the league in one season.
type readings struct {
	season, rosterMax, taxi, ir, pending, rosterSeason int
	deadline                                           time.Time
}

func snapshotReadings(t *testing.T, b *ledgerBuilder) readings {
	t.Helper()
	now := rolloverMoment().Add(10 * time.Hour)
	season, err := SeasonAt(b.records, now)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := RulesNow(b.records, now)
	if err != nil {
		t.Fatal(err)
	}
	out := readings{season: season, rosterMax: rules.Roster.Limits.Value.Max.Value,
		deadline: rules.Calendar.TradeDeadline.Value, rosterSeason: readRoster(t, b, now).RosterSeason}
	for _, p := range playersByID(t, b.records) {
		switch p.Status {
		case domain.RosterTaxi:
			out.taxi++
		case domain.RosterIR:
			out.ir++
		case domain.RosterActive:
		}
		if p.Pending {
			out.pending++
		}
	}
	return out
}

// reload sends every record through JSON, and every rollover through validation, as a reopened database does.
func reload(t *testing.T, records []Record) []Record {
	t.Helper()
	out := make([]Record, len(records))
	for i, r := range records {
		body, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, &out[i]); err != nil {
			t.Fatal(err)
		}
		if r.Command.Command != SeasonRollover {
			continue // the builder's other entries carry no reason or canonical form
		}
		if err := out[i].Validate(); err != nil {
			t.Fatalf("entry %d does not validate after reload: %v", i+1, err)
		}
	}
	return out
}

func TestRolloverReplaysIdenticallyAfterReload(t *testing.T) {
	b := rolloverLedger(t)
	b.waitingRule("roster.limits", `{"min":22,"max":75}`, 5)
	act := b.rollover(2027, rolloverMoment())
	live := snapshotReadings(t, b)
	reloaded := &ledgerBuilder{t: t, records: reload(t, b.records)}
	if got := snapshotReadings(t, reloaded); !reflect.DeepEqual(got, live) {
		t.Fatalf("replay differs:\n got %+v\nwant %+v", got, live)
	}
	b.revert(act, rolloverMoment().Add(time.Hour))
	reverted := snapshotReadings(t, b)
	reloaded = &ledgerBuilder{t: t, records: reload(t, b.records)}
	if got := snapshotReadings(t, reloaded); !reflect.DeepEqual(got, reverted) {
		t.Fatalf("replay of the revert differs:\n got %+v\nwant %+v", got, reverted)
	}
}

func TestRolloverRefusesAnythingButTheNextSeason(t *testing.T) {
	b := rolloverLedger(t)
	for _, to := range []int{2025, 2026, 2028, 2030} {
		_, _, err := PrepareRollover(b.rolloverCommand(to, false), b.records, rolloverMoment())
		var refused *Refusal
		if !errors.As(err, &refused) || !strings.Contains(refused.Reason(), "Move only from season 2026 to season 2027") {
			t.Fatalf("to %d: %v", to, err)
		}
	}
	b.rollover(2027, rolloverMoment())
	if _, _, err := PrepareRollover(b.rolloverCommand(2027, false), b.records, rolloverMoment()); err == nil {
		t.Fatal("the same season was rolled into twice")
	}
	if _, _, err := PrepareRollover(b.rolloverCommand(2028, false), b.records, rolloverMoment().Add(time.Hour)); err != nil {
		t.Fatalf("the next season after a rollover was refused: %v", err)
	}
}

func TestRolloverBlockedPlanPreviewsButNeverRecords(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.add(RawCommand{League: rolloverLeague, Command: NativeSeed, Args: RawArgs{Native: ledgerSnapshot()},
		Request: "seed", Effective: "now", Visibility: "league"}, seasonMoment(2))
	_, plan, err := PrepareRollover(b.rolloverCommand(2027, true), b.records, rolloverMoment())
	if err != nil || !plan.Blocked || len(plan.Reasons) == 0 {
		t.Fatalf("a dry run must show the blocked plan: %+v %v", plan, err)
	}
	_, _, err = PrepareRollover(b.rolloverCommand(2027, false), b.records, rolloverMoment())
	var refused *Refusal
	if !errors.As(err, &refused) || !strings.Contains(refused.Reason(), "The rollover is blocked") {
		t.Fatalf("a real run of a blocked plan was not refused: %v", err)
	}
}

func TestRolloverRecordIsTamperProof(t *testing.T) {
	b := rolloverLedger(t)
	b.rollover(2027, rolloverMoment())
	last := len(b.records) - 1
	for name, mutate := range map[string]func(*Record){
		"digest":      func(r *Record) { r.Receipt.Plan.BeforeDigest = "0" },
		"assets":      func(r *Record) { r.Before.Assets.Players[0].Salary++ },
		"rules":       func(r *Record) { r.Before.Rules.Roster.Limits.Value.Max.Value++ },
		"no before":   func(r *Record) { r.Before = nil },
		"no plan":     func(r *Record) { r.Receipt.Plan = nil },
		"other time":  func(r *Record) { r.Before.At = r.Before.At.Add(time.Second) },
		"wrong to":    func(r *Record) { r.Receipt.Plan.To++ },
		"blocked":     func(r *Record) { r.Receipt.Plan.Blocked = true },
		"on a change": func(r *Record) { r.Command.Command = Revert },
	} {
		clone := reload(t, b.records)[last]
		mutate(&clone)
		if clone.Validate() == nil {
			t.Fatalf("a tampered rollover (%s) validated", name)
		}
	}
	other := reload(t, b.records)[1]
	other.Receipt.Plan = b.records[last].Receipt.Plan
	if other.Validate() == nil {
		t.Fatal("a plan on another change validated")
	}
}

func TestRolloverReversalsAreOrderedAroundTheAct(t *testing.T) {
	b := rolloverLedger(t)
	waiting := b.waitingRule("roster.limits", `{"min":22,"max":75}`, 5)
	act := b.rollover(2027, rolloverMoment())
	check := func(entry string) error {
		return CheckRolloverAppend(RawCommand{Command: Revert, Args: RawArgs{Entry: entry}}, b.records)
	}
	if err := check(waiting); err == nil || !strings.Contains(err.Error(), "already part of a season") {
		t.Fatalf("a captured change could be reversed around the rollover: %v", err)
	}
	later := b.set("roster.limits", `{"min":10,"max":90}`, "now", rolloverMoment().Add(time.Hour))
	if err := check(act); err == nil || !strings.Contains(err.Error(), "Reverse the later changes first") {
		t.Fatalf("a rollover with later changes could be reversed: %v", err)
	}
	b.revert(later, rolloverMoment().Add(2*time.Hour))
	if err := check(act); err != nil {
		t.Fatalf("a rollover whose later changes are reversed was refused: %v", err)
	}
	b.revert(act, rolloverMoment().Add(3*time.Hour))
	if err := check(waiting); err != nil {
		t.Fatalf("a change before a reversed rollover was refused: %v", err)
	}
	if err := CheckRolloverAppend(RawCommand{Command: Revert, Args: RawArgs{Entry: "missing"}}, b.records); err != nil {
		t.Fatalf("an unknown entry is another check's refusal: %v", err)
	}
	if rolloverReversalRefusal(b.records, b.records[len(b.records)-1].Receipt.Entry) != "" {
		t.Fatal("reversing the reversal of a rollover with no later changes was refused")
	}
}

func TestRolloverStartsTheNewSeasonWithoutTheOldSchedule(t *testing.T) {
	b := rolloverLedger(t)
	for i, command := range []string{ScheduleGenerate, SchedulePublish} {
		b.add(RawCommand{League: rolloverLeague, Command: command, Args: RawArgs{Schedule: &RawScheduleArgs{}},
			Request: fmt.Sprintf("schedule.%d", i), Effective: "now", Visibility: "league"}, seasonMoment(5+i))
	}
	published, err := ScheduleReading(rolloverLeague, b.records)
	if err != nil || len(published.Weeks) == 0 {
		t.Fatalf("the league has no schedule to carry: %+v %v", published, err)
	}
	act := b.rollover(2027, rolloverMoment())
	if reading, err := ScheduleReading(rolloverLeague, b.records); err != nil || len(reading.Weeks) != 0 ||
		reading.Season != 2027 {
		t.Fatalf("the new season kept the old schedule: %+v %v", reading, err)
	}
	b.revert(act, rolloverMoment().Add(time.Hour))
	if reading, err := ScheduleReading(rolloverLeague, b.records); err != nil ||
		!reflect.DeepEqual(reading.Weeks, published.Weeks) || reading.Season != 2026 {
		t.Fatalf("the revert did not bring the schedule back: %+v %v", reading, err)
	}
}

func TestRolloverTwiceKeepsEachSeasonsRulesAndThePickHistory(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	native := ledgerSnapshot()
	native.Picks = []NativePick{
		{Season: 2026, Round: 1, Slot: 1, Owner: "franchise.02", Player: "0003"},
		{Season: 2027, Round: 2, Original: "franchise.01", Owner: "franchise.02"}}
	b.add(RawCommand{League: rolloverLeague, Command: NativeSeed, Args: RawArgs{Native: native},
		Request: "seed", Effective: "now", Visibility: "league"}, seasonMoment(2))
	b.set("calendar.boundary", `"start_of_day"`, "now", seasonMoment(3))
	b.set("calendar.tradeDeadline", `"2026-11-09T05:00:00Z"`, "now", seasonMoment(4))
	first := rolloverMoment()
	second := first.AddDate(1, 0, 0)
	b.rollover(2027, first)
	last := b.rollover(2028, second)
	if season, err := SeasonAt(b.records, second); err != nil || season != 2028 {
		t.Fatalf("two rollovers did not reach season 2028: %d %v", season, err)
	}
	assets, err := NativeAssetsReading(rolloverLeague, b.records)
	if err != nil || assets.Snapshot.Season != 2028 || !reflect.DeepEqual(assets.Snapshot.Picks, native.Picks) {
		t.Fatalf("assets after two rollovers: %+v %v", assets.Snapshot, err)
	}
	for season, year := range map[int]int{2026: 2026, 2027: 2027, 2028: 2028} {
		rules, err := RulesInForce(b.records, season, second)
		if err != nil || rules.Season != season || rules.Calendar.TradeDeadline.Value.Year() != year {
			t.Fatalf("season %d rules: deadline %v, %v", season, rules.Calendar.TradeDeadline.Value, err)
		}
	}
	b.revert(last, second.Add(time.Hour))
	if season, err := SeasonAt(b.records, second.Add(2*time.Hour)); err != nil || season != 2027 {
		t.Fatalf("reversing the latest rollover left season %d: %v", season, err)
	}
}

func TestRolloverEntryWire(t *testing.T) {
	b := rolloverLedger(t)
	b.waitingRule("roster.limits", `{"min":22,"max":75}`, 5)
	b.rollover(2027, rolloverMoment())
	body, err := json.Marshal(b.records[len(b.records)-1])
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := "testdata/rollover-entry.wire.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, want) {
		t.Fatal("rollover entry wire differs from Go fixture")
	}
	var back Record
	if err := json.Unmarshal(want, &back); err != nil || back.Validate() != nil {
		t.Fatalf("the written entry does not read back: %v", err)
	}
}
