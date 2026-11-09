package commissioner

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestLineupLockSettingFlowsToAdmissionAndReading(t *testing.T) {
	for _, policy := range []string{"each_player_game", "first_kickoff"} {
		for _, moment := range []struct {
			name                     string
			at                       time.Time
			playerLocked, weekLocked bool
		}{
			{"before", seasonMoment(4), false, false},
			{"first kickoff", seasonMoment(5), false, true},
			{"player kickoff", seasonMoment(7), true, true},
		} {
			t.Run(policy+"/"+moment.name, func(t *testing.T) {
				b := seededLedger(t)
				b.set("lineup.lock", `"`+policy+`"`, "now", seasonMoment(3))
				b.set("lineup.positions", `{"QB":{"min":0,"max":1}}`, "now", seasonMoment(3))
				b.set("lineup.groups", `{}`, "now", seasonMoment(3))
				player := NativePlayer{ID: "0099", Franchise: "franchise.01", Status: domain.RosterActive}
				b.records[1].Command.Args.Native.Players = []NativePlayer{player}
				b.records[1].Command.Args.Native.RosterAt = seasonMoment(2)
				env := LineupFacts{Facts: nativecheck.Facts{"0099": {Name: "Later player", Position: leaguerules.QB, Team: "KC"}},
					Locks: map[string]time.Time{"BUF": seasonMoment(5), "KC": seasonMoment(7)}, ScheduleHeld: true}
				raw := RawCommand{League: "native.test", Command: LineupSet, Args: RawArgs{Lineup: &RawLineupChange{
					Season: 2026, Week: 1, Franchise: "franchise.01", Starters: []string{"0099"}}}}
				wantLocked := moment.playerLocked
				if policy == "first_kickoff" {
					wantLocked = moment.weekLocked
				}
				before, err := json.Marshal(b.records)
				if err != nil {
					t.Fatal(err)
				}
				err = CheckLineupAppend(raw, b.records, moment.at, env)
				if (err != nil) != wantLocked {
					t.Fatalf("locked=%t admission=%v", wantLocked, err)
				}
				after, err := json.Marshal(b.records)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatal("admission mutated ledger", err)
				}
				reading, err := NativeLineupReading("native.test", "franchise.01", "creator", 1, b.records, moment.at, env)
				if err != nil || len(reading.Bench) != 1 || reading.Bench[0].Locked != wantLocked {
					t.Fatalf("reading=%+v error=%v", reading, err)
				}
				raw.Command = LineupCorrect
				if err := CheckLineupAppend(raw, b.records, moment.at, env); err != nil {
					t.Fatal("correction must bypass lock", err)
				}
			})
		}
	}
}

func TestStartingAssetsSettingNotOriginControlsSeedAndCutover(t *testing.T) {
	for _, source := range []string{SourceNative, SourceMFLImport} {
		for _, assets := range []string{"empty", "held"} {
			t.Run(source+"/"+assets, func(t *testing.T) {
				b := seededLedger(t)
				b.records[0].Command.Args.Source = source
				b.records = b.records[:1]
				b.set("league.startingAssets", `"`+assets+`"`, "now", seasonMoment(2))
				held := *seededLedger(t).records[1].Command.Args.Native
				held.RosterAt = seasonMoment(3)
				raw := RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &held}}
				if err := CheckNativeAppend(raw, b.records); (err == nil) != (assets == "held") {
					t.Fatalf("held seed: %v", err)
				}
				empty, err := NativeStart("native.test", b.records, seasonMoment(3))
				if (err == nil) != (assets == "empty") {
					t.Fatalf("empty start: %v", err)
				}
				if assets == "empty" {
					raw.Args.Native = &empty
					if err := CheckNativeAppend(raw, b.records); err != nil {
						t.Fatal(err)
					}
				}
				b.add(raw, seasonMoment(3))
				cut := RawCommand{Command: Cutover}
				if err := CheckCutoverAppend(cut, b.records); (err == nil) != (assets == "held") {
					t.Fatalf("cutover: %v", err)
				}
			})
		}
	}
}

func TestStartingSettingsDefaultsAndRevert(t *testing.T) {
	b := newLedger(t, PresetEmpty)
	rules, err := RulesNow(b.records, seasonMoment(2))
	if err != nil || rules.Lineup.Lock.Value != leaguerules.EachPlayerGame || rules.League.StartingAssets.Value != leaguerules.EmptyAssets {
		t.Fatalf("native defaults: %+v %v", rules, err)
	}
	entry := b.set("lineup.lock", `"first_kickoff"`, "now", seasonMoment(3))
	rules, err = RulesNow(b.records, seasonMoment(3))
	if err != nil || rules.Lineup.Lock.Value != leaguerules.FirstKickoff {
		t.Fatal("setting not in force", err)
	}
	b.revert(entry, seasonMoment(4))
	rules, err = RulesNow(b.records, seasonMoment(4))
	if err != nil || rules.Lineup.Lock.Value != leaguerules.EachPlayerGame {
		t.Fatal("revert did not restore setting", err)
	}
}
