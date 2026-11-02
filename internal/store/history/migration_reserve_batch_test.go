package history

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestPreparedReserveBatch(t *testing.T) {
	for _, tc := range []struct {
		name, principal                string
		missing, badSecond, fault, bad bool
	}{
		{name: "success and retry", principal: "creator"},
		{name: "unauthorized", principal: "outsider", bad: true},
		{name: "missing injury facts", principal: "creator", missing: true, bad: true},
		{name: "second step fails", principal: "creator", badSecond: true, bad: true},
		{name: "write fails", principal: "creator", fault: true, bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := lineupTest(t)
			runScoreCommand(t, f.s, ruleCommand("calendar.boundary", `"start_of_day"`, "now", "Set the day boundary", revision(t, f.s)), "creator")
			raw := commissioner.RawCommand{League: "native.test", Command: commissioner.WindowSet, Expected: revision(t, f.s), Request: "reserve-window", Reason: "Open reserve moves", Effective: "now", Visibility: "league", Args: commissioner.RawArgs{Window: &commissioner.RawWindowArgs{ID: "reserve_test", Start: &commissioner.RawDate{Month: 1, Day: 1}, Opens: []leaguerules.Action{"ir"}}}}
			runScoreCommand(t, f.s, raw, "creator")
			f.at = f.at.Add(time.Minute)
			id := f.roster[f.a][leaguerules.RB][2]
			fact := f.env.facts[id]
			fact.Designation, fact.DesignationHeld = leaguerules.InjuredReserve, true
			f.env.facts[id] = fact
			facts := f.env.facts
			if tc.missing {
				facts = nativecheck.Facts{}
			}
			batch := []commissioner.RawCommand{}
			for i, bucket := range []string{"ir", "active"} {
				player := id
				if tc.badSecond && i == 1 {
					player = "999999"
				}
				batch = append(batch, commissioner.RawCommand{League: "native.test", Command: commissioner.RosterReserve, Expected: revision(t, f.s), Request: fmt.Sprintf("transactions.sync.reserve.%d", i), Reason: "Record observed reserve move", Effective: f.at.Add(-time.Second).Format(time.RFC3339Nano), Visibility: "league", Args: commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: player, Franchise: f.a, Bucket: bucket}}})
			}
			before := fingerprint(t, f.s)
			version := revision(t, f.s)
			if tc.fault {
				f.s.batchFault = func(int) error { return errors.New("injected reserve write failure") }
			}
			receipts, err := f.s.ExecuteCommissionerBatchWithFacts(t.Context(), batch, tc.principal, nil, facts, time.UTC)
			if (err != nil) != tc.bad {
				t.Fatalf("want bad=%t: receipts %d, %v", tc.bad, len(receipts), err)
			}
			if tc.bad {
				if before != fingerprint(t, f.s) {
					t.Fatal("refused reserve batch left a partial move")
				}
				return
			}
			if len(receipts) != 2 || revision(t, f.s) != version+2 {
				t.Fatal("reserve batch did not append both steps")
			}
			records, err := f.s.nativeRecords(t.Context(), "native.test")
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range records[len(records)-2:] {
				if r.Command.Args.WaiverOutcome == nil || r.Command.Args.WaiverOutcome.Player == nil {
					t.Fatal("reserve batch appended without ordinary prepared outcome")
				}
			}
			stable := fingerprint(t, f.s)
			if _, err := f.s.ExecuteCommissionerBatchWithFacts(t.Context(), batch, tc.principal, nil, facts, time.UTC); err != nil || stable != fingerprint(t, f.s) {
				t.Fatal("retry appended reserve moves", err)
			}
			// A retry contributes no tail; the next new command must still see both prior moves.
			next := batch[0]
			next.Request = "transactions.sync.reserve.next"
			batch = append(batch, next)
			got, err := f.s.ExecuteCommissionerBatchWithFacts(t.Context(), batch, tc.principal, nil, facts, time.UTC)
			if err != nil || len(got) != 3 || revision(t, f.s) != version+3 {
				t.Fatal("mixed retry did not append exactly one move", err)
			}
			if got[0].Entry != receipts[0].Entry || got[1].Entry != receipts[1].Entry {
				t.Fatal("mixed retry changed prior receipts")
			}
			assertBatchTail(t, f.s, version+2, got[2].Entry)
		})
	}
}

func assertBatchTail(t *testing.T, s *Store, after int64, entry string) {
	t.Helper()
	tx, err := s.pools.BeginWrite(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	tail, err := s.eventsAfterInTransaction(t.Context(), tx, "native.test", after)
	if err != nil || len(tail) != 1 || tail[0].Receipt.Entry != entry {
		t.Fatalf("read only appended tail: %v, %v", tail, err)
	}
	full, err := s.eventsInTransaction(t.Context(), tx, "native.test")
	if err != nil || len(full) != int(after)+1 || full[len(full)-1].Receipt.Entry != entry {
		t.Fatal("tail differs from authoritative full replay", err)
	}
	tail, err = s.eventsAfterInTransaction(t.Context(), tx, "native.test", after+1)
	if err != nil || len(tail) != 0 {
		t.Fatal("retry head has a nonempty tail", err)
	}
}
