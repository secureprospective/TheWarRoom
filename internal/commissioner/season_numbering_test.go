package commissioner

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestCreationNeedsTheNFLSeasonItStartsIn(t *testing.T) {
	raw := RawCommand{League: "native.test", Command: Create, Args: RawArgs{Name: "Ours", Preset: PresetEmpty,
		FranchiseCount: 4}, Reason: "start", Request: "create", Effective: "now", Visibility: "league"}
	if _, err := raw.Prepared(); err == nil || !strings.Contains(err.Error(), "NFL season") {
		t.Fatalf("a league with no starting season was prepared: %v", err)
	}
	for _, season := range []int{1919, 10000, -1} {
		raw.Args.Season = season
		prepared, err := raw.Prepared()
		if err == nil {
			err = prepared.Validate()
		}
		if err == nil {
			t.Fatalf("season %d accepted", season)
		}
	}
	raw.Args.Season = 2026
	prepared, err := raw.Prepared()
	if err != nil || prepared.Validate() != nil {
		t.Fatalf("a league with a season was refused: %v", err)
	}
	other := RawCommand{League: "native.test", Command: Revert, Args: RawArgs{Entry: "entry.1", Season: 2026},
		Reason: "undo", Request: "undo", Effective: "now", Visibility: "league", Expected: 1}
	if other.Validate() == nil {
		t.Fatal("a season was accepted on a change that is not a creation")
	}
}

func TestLedgerRecordedWithoutASeasonRefusesToGuessOne(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.records[0].Command.Args.Season = 0
	b.records[0].Command.Reason = "start"
	if err := b.records[0].Command.Validate(); err != nil {
		t.Fatalf("an already recorded creation no longer replays: %v", err)
	}
	_, err := RulesNow(b.records, seasonMoment(5))
	if err == nil || !strings.Contains(err.Error(), "started before its first season was recorded") {
		t.Fatalf("no plain refusal for a league with no season: %v", err)
	}
	if _, err := SeasonAt(b.records, seasonMoment(5)); err == nil {
		t.Fatal("a season was guessed for a league with none recorded")
	}
	// The clocks run every second for every league: an unseeded one has nothing due and is not an error.
	if due, err := DueWaiverWindows("native.test", b.records, seasonMoment(5)); err != nil || len(due) != 0 {
		t.Fatalf("the waiver clock failed on a league with nothing to process: %v %v", due, err)
	}
	if due, err := DueBids("native.test", b.records, seasonMoment(5), nativecheck.Facts{}); err != nil || len(due) != 0 {
		t.Fatalf("the bid clock failed on a league with nothing to process: %v %v", due, err)
	}
	b.seed(2026, nil, seasonMoment(2))
	if season, err := SeasonAt(b.records, seasonMoment(5)); err != nil || season != 2026 {
		t.Fatalf("a seeded league keeps the seed's season: %d %v", season, err)
	}
}

func TestRulesScheduleAndRosterNameTheSameSeasonBeforeAndAfterRollover(t *testing.T) {
	b := rolloverLedger(t)
	now := rolloverMoment()
	names := func() [4]int {
		rules, err := RulesNow(b.records, now)
		if err != nil {
			t.Fatal(err)
		}
		schedule, err := ScheduleReading(rolloverLeague, b.records)
		if err != nil {
			t.Fatal(err)
		}
		house, err := NativeHouseRulesReading(rolloverLeague, b.records, now)
		if err != nil {
			t.Fatal(err)
		}
		season, err := SeasonAt(b.records, ReadingInstant(b.records, now))
		if err != nil {
			t.Fatal(err)
		}
		return [4]int{rules.Season, schedule.Season, house.Season, season}
	}
	if got := names(); got != [4]int{2026, 2026, 2026, 2026} {
		t.Fatalf("before the rollover the readings name %v", got)
	}
	act := b.rollover(2027, now)
	if got := names(); got != [4]int{2027, 2027, 2027, 2027} {
		t.Fatalf("after the rollover the readings name %v", got)
	}
	b.revert(act, now.Add(time.Hour))
	if got := names(); got != [4]int{2026, 2026, 2026, 2026} {
		t.Fatalf("after the revert the readings name %v", got)
	}
}

func TestUnseededLeagueNamesItsCreationSeasonThroughARollover(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	b.set("calendar.boundary", `"start_of_day"`, "now", seasonMoment(3))
	now := rolloverMoment()
	act := b.rollover(2027, now)
	rules, err := RulesNow(b.records, now)
	if err != nil || rules.Season != 2027 {
		t.Fatalf("unseeded rules reading after the rollover: %d %v", rules.Season, err)
	}
	schedule, err := ScheduleReading(rolloverLeague, b.records)
	if err != nil || schedule.Season != 2027 {
		t.Fatalf("unseeded schedule after the rollover: %d %v", schedule.Season, err)
	}
	if assets, err := NativeAssetsReading(rolloverLeague, b.records); err != nil || assets.Seeded {
		t.Fatalf("a rollover seeded a league: %+v %v", assets, err)
	}
	b.revert(act, now.Add(time.Hour))
	if season, err := SeasonAt(b.records, now.Add(2*time.Hour)); err != nil || season != 2026 {
		t.Fatalf("season after the revert: %d %v", season, err)
	}
}

func TestStartingSnapshotMustBeForTheCreationSeasonAndBeforeAnyRollover(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	seed := func(season int) RawCommand {
		native := ledgerSnapshot()
		native.Season = season
		return RawCommand{League: rolloverLeague, Command: NativeSeed, Args: RawArgs{Native: native},
			Request: "seed", Effective: "now", Visibility: "league", Expected: 1}
	}
	if err := CheckNativeAppend(seed(2025), b.records); err == nil ||
		!strings.Contains(err.Error(), "must be for season 2026") {
		t.Fatalf("a snapshot for another season was accepted: %v", err)
	}
	if err := CheckNativeAppend(seed(2026), b.records); err != nil {
		t.Fatalf("the creation season's snapshot was refused: %v", err)
	}
	b.set("calendar.boundary", `"start_of_day"`, "now", seasonMoment(3))
	b.rollover(2027, rolloverMoment())
	if err := CheckNativeAppend(seed(2026), b.records); err == nil ||
		!strings.Contains(err.Error(), "before moving the league to a new season") {
		t.Fatalf("a snapshot after a rollover was accepted: %v", err)
	}
}
