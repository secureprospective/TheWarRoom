package commissioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestLineupLedgerByteEquivalent(t *testing.T) {
	b := seededLedger(t)
	env := LineupFacts{Facts: nativecheck.Facts{}, Locks: map[string]time.Time{"KC": seasonMoment(20)}, ScheduleHeld: true}
	for _, p := range b.records[1].Command.Args.Native.Players {
		env.Facts[p.ID] = nativecheck.PlayerFacts{Name: p.Name, Team: "KC"}
	}
	b.add(RawCommand{League: "native.test", Command: LineupCorrect, Reason: "A recorded correction",
		Args: RawArgs{Lineup: &RawLineupChange{Season: 2026, Week: 1, Franchise: "franchise.01", Starters: []string{"0099"}}}}, seasonMoment(3))
	for _, records := range [][]Record{b.records, WithReplayCache(b.records)} {
		ledger := NewLineupLedger(records)
		var wg sync.WaitGroup
		for _, franchise := range []string{"franchise.01", "franchise.02"} {
			for _, week := range []int{1, 2} {
				for _, at := range []time.Time{seasonMoment(10), seasonMoment(21)} {
					for _, principal := range []string{Replay(records).Creator, "stranger"} {
						wg.Go(func() {
							want, wantErr := legacyLineupReading("native.test", franchise, principal, week, records, at, env)
							got, err := ledger.Read(context.Background(), "native.test", franchise, principal, week, at, env)
							if fmt.Sprint(err) != fmt.Sprint(wantErr) {
								t.Errorf("errors: %v vs %v", err, wantErr)
								return
							}
							a, e := json.Marshal(want)
							if e != nil {
								t.Error(e)
								return
							}
							z, e := json.Marshal(got)
							if e != nil {
								t.Error(e)
								return
							}
							if !bytes.Equal(a, z) {
								t.Errorf("lineups differ: %s vs %s", a, z)
							}
						})
					}
				}
			}
		}
		wg.Wait()
		if len(ledger.bases) != 1 {
			t.Fatalf("same revision folded %d times", len(ledger.bases))
		}
		got, err := ledger.Read(t.Context(), "native.test", "franchise.01", "stranger", 1, seasonMoment(10), env)
		if err != nil {
			t.Fatal(err)
		}
		var snapshot *RawNativeSnapshot
		for _, base := range ledger.bases {
			snapshot = base.Assets.Snapshot
		}
		got.Starters[0].Name = "edited"
		got.Rules[0].Words = "edited"
		again, err := ledger.Read(t.Context(), "native.test", "franchise.01", "stranger", 1, seasonMoment(10), env)
		for _, base := range ledger.bases {
			if base.Assets.Snapshot != snapshot {
				t.Fatal("same revision rebuilt its assets")
			}
		}
		if err != nil || again.Starters[0].Name == "edited" || again.Rules[0].Words == "edited" {
			t.Fatal("caller changed shared inputs", err)
		}
	}
}

func TestLineupLedgerBoundariesAndNewRevision(t *testing.T) {
	b := seededLedger(t)
	records := WithReplayCache(b.records)
	ledger := NewLineupLedger(records)
	env := LineupFacts{}
	at := seasonMoment(10)
	principal := Replay(records).Creator
	before, err := ledger.Read(t.Context(), "native.test", "franchise.01", principal, 1, at, env)
	if err != nil {
		t.Fatal(err)
	}
	next := slices.Clone(records)
	next = append(next, lineupRecord("new", LineupCorrect, 1, "0099"))
	after, err := NewLineupLedger(next).Read(t.Context(), "native.test", "franchise.01", principal, 1, at, env)
	if err != nil || before.Set || !after.Set {
		t.Fatalf("new revision not observed: %v", err)
	}
	for _, week := range []int{0, 23} {
		if _, err := ledger.Read(t.Context(), "native.test", "franchise.01", principal, week, at, env); err == nil {
			t.Fatal("bad week accepted")
		}
	}
}

func legacyLineupReading(league, franchise, principal string, week int, records []Record, at time.Time,
	env LineupFacts) (NativeLineup, error) {
	if week < 1 || week > 22 {
		return NativeLineup{}, fmt.Errorf("choose a week")
	}
	base, err := ReadNativeBase(league, records, at)
	if err != nil {
		return NativeLineup{}, err
	}
	if !base.Assets.Seeded {
		return NativeLineup{}, fmt.Errorf("record the league's starting snapshot before reading a lineup")
	}
	seat, ok := base.Directory.seat(franchise)
	if !ok {
		return NativeLineup{}, fmt.Errorf("choose one of this league's franchises")
	}
	instant := ReadingInstant(records, at)
	season, err := SeasonAt(records, instant)
	if err != nil {
		return NativeLineup{}, err
	}
	rules, err := RulesInForce(records, season, instant)
	if err != nil {
		return NativeLineup{}, err
	}
	env.lockPolicy = rules.Lineup.Lock.Value
	held, set, err := RecordedLineupFor(records, franchise, season, week)
	if err != nil {
		return NativeLineup{}, err
	}
	if err := authorizeLineupReading(league, principal, seat.Owner, held.Corrected, records); err != nil {
		return NativeLineup{}, err
	}
	out := NativeLineup{League: league, Name: Replay(records).Name, Revision: int64(len(records)), Season: season,
		Week: week, Franchise: franchise, FranchiseName: seat.Name, Zone: env.zone().String(), Set: set,
		Corrected: held.Corrected, Reason: held.Reason, Starters: []LineupPlayer{}, Bench: []LineupPlayer{},
		Rules: RuleLines(base.Rules, "lineup.positions", "lineup.groups", "lineup.invalidPenalty", "lineup.missedFlag", "lineup.lock")}
	starting := map[string]bool{}
	for _, id := range held.Starters {
		starting[id] = true
	}
	for _, id := range held.Starters {
		out.Starters = append(out.Starters, lineupPlayer(id, env, instant))
	}
	for _, p := range base.playersOf()[franchise] {
		if p.Status == domain.RosterActive && !starting[p.ID] {
			out.Bench = append(out.Bench, lineupPlayer(p.ID, env, instant))
		}
	}
	sortLineupPlayers(out.Bench)
	out.Verdict, _ = judgeStarters(rules.Lineup, set, held.Starters, env.Facts)
	out.Words = lineupWords(out, seat.Name)
	return out, nil
}

func TestLineupLedgerProfileFixtureEquivalence(t *testing.T) {
	path := os.Getenv("TWR_LINEUP_RECORDS")
	if path == "" {
		t.Skip("set TWR_LINEUP_RECORDS to a disposable profile's exported ledger")
	}
	body, err := os.ReadFile(path) //nolint:gosec // Opt-in test fixture path chosen by the test operator, not an application request.
	if err != nil {
		t.Fatal(err)
	}
	var records []Record
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatal(err)
	}
	for i, r := range records {
		if err := r.Validate(); err != nil {
			t.Fatalf("fixture record %d: %v", i, err)
		}
	}
	if len(records) == 0 {
		t.Fatal("empty fixture")
	}
	league := records[0].Command.League
	at := ReadingInstant(records, time.Time{})
	principal := Replay(records).Creator
	ledger := NewLineupLedger(WithReplayCache(records))
	for _, franchise := range []string{"franchise.01", "franchise.02", "franchise.05", "franchise.32"} {
		for _, week := range []int{1, 8} {
			want, wantErr := legacyLineupReading(league, franchise, principal, week, records, at, LineupFacts{})
			got, err := ledger.Read(t.Context(), league, franchise, principal, week, at, LineupFacts{})
			if fmt.Sprint(err) != fmt.Sprint(wantErr) {
				t.Fatalf("errors %v vs %v", err, wantErr)
			}
			a, err := json.Marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			b, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(a, b) {
				t.Fatalf("fixture lineup differs: %s week %d", franchise, week)
			}
		}
	}
	t.Logf("%d records; eight byte-identical franchise/week readings", len(records))
}

func TestLineupLedgerCrossesRolloverWithoutNewEntries(t *testing.T) {
	b := seededLedger(t)
	boundary := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	value, err := json.Marshal(boundary.Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}
	b.add(RawCommand{League: "native.test", Command: RulesSet, Effective: "now", Args: RawArgs{
		Field: rolloverField, Value: value}}, seasonMoment(3))
	ledger := NewLineupLedger(b.records)
	principal := Replay(b.records).Creator
	for _, at := range []time.Time{boundary.Add(-time.Nanosecond), boundary, boundary.Add(time.Hour)} {
		want, wantErr := legacyLineupReading("native.test", "franchise.01", principal, 1, b.records, at, LineupFacts{})
		got, err := ledger.Read(t.Context(), "native.test", "franchise.01", principal, 1, at, LineupFacts{})
		if err != nil || wantErr != nil || got.Season != want.Season || fmt.Sprint(got.Rules) != fmt.Sprint(want.Rules) {
			t.Fatalf("rollover reading changed: %+v %v vs %+v %v", got, err, want, wantErr)
		}
	}
	if len(ledger.bases) != 2 {
		t.Fatalf("rollover must separate inputs: %d", len(ledger.bases))
	}
}
