package moves

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func bidAt() time.Time { return time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC) }

func testBid(t *testing.T, id string) (envelope.Envelope, envelope.BidFacts) {
	t.Helper()
	player, err := playerid.New("0531")
	if err != nil {
		t.Fatal(err)
	}
	beat := bid.Points(6500)
	facts := envelope.BidFacts{Player: player, Salary: 500_000_000, Years: 3, Points: 7000,
		Snipe: true, Beat: &beat, Evidence: "https://example.invalid/thread#12", Closes: "2026-05-10T04:00:00Z"}
	check := envelope.BidCheck{Input: bid.Input{Salary: facts.Salary, Years: facts.Years,
		Minimum: bid.MinimumSalary(3), Experience: bid.Experience{Years: 3, Known: true}, WeekKnown: true}}
	e, err := envelope.DraftBid(bidAt(), envelope.BidRequest{LeagueID: "1", FranchiseID: "0001", Facts: facts},
		check, func() string { return id })
	if err != nil {
		t.Fatal(err)
	}
	return e, facts
}

func TestBidFactsRoundTripWithAndWithoutBidToBeat(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	e, facts := testBid(t, "bid-1")
	if err := s.SaveBid(ctx, e, facts); err != nil {
		t.Fatal(err)
	}
	got, err := s.BidFacts(ctx, "bid-1")
	if err != nil || got.Beat == nil || *got.Beat != 6500 || got.Player != facts.Player || got.Salary != facts.Salary ||
		got.Years != 3 || got.Points != 7000 || !got.Snipe || got.Evidence != facts.Evidence || got.Closes != facts.Closes {
		t.Fatalf("facts = %+v, %v", got, err)
	}
	e2, facts2 := testBid(t, "bid-2")
	facts2.Beat, facts2.Snipe = nil, false
	if err := s.SaveBid(ctx, e2, facts2); err != nil {
		t.Fatal(err)
	}
	all, err := s.AllBidFacts(ctx)
	if err != nil || len(all) != 2 || all["bid-2"].Beat != nil || all["bid-2"].Snipe {
		t.Fatalf("all facts = %+v, %v", all, err)
	}
	if _, err := s.BidFacts(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing facts: %v", err)
	}
}

func TestSaveBidRefusesFactsThatAreNotThePlans(t *testing.T) {
	s, _ := testStore(t)
	e, facts := testBid(t, "bid-1")
	other, err := playerid.New("0777")
	if err != nil {
		t.Fatal(err)
	}
	facts.Player = other
	if err := s.SaveBid(context.Background(), e, facts); err == nil {
		t.Error("facts for another player were saved")
	}
	ir := testMove(t, "ir-1", bidAt(), envelope.ChecksPass)
	_, good := testBid(t, "x")
	if err := s.SaveBid(context.Background(), ir, good); err == nil {
		t.Error("facts were saved on a roster move")
	}
}

func TestBidFactsEditUntilHandedOffThenFrozen(t *testing.T) {
	s, pools := testStore(t)
	ctx := context.Background()
	e, facts := testBid(t, "bid-1")
	if err := s.SaveBid(ctx, e, facts); err != nil {
		t.Fatal(err)
	}
	facts.Salary, facts.Points = 600_000_000, 8400
	if err := s.SaveBid(ctx, e, facts); err != nil {
		t.Fatalf("an unposted bid could not be revised: %v", err)
	}
	handed, err := e.HandOffManual(bidAt().Add(time.Minute), "copied")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, handed); err != nil {
		t.Fatalf("saving the hand-off: %v", err)
	}
	facts.Salary = 700_000_000
	if err := s.SaveBid(ctx, handed, facts); err == nil || !strings.Contains(err.Error(), "frozen") {
		t.Errorf("facts changed after hand-off: %v", err)
	}
	got, err := s.BidFacts(ctx, "bid-1")
	if err != nil || got.Salary != 600_000_000 {
		t.Errorf("facts after refused edit = %+v, %v", got, err)
	}
	if _, err := pools.Write().ExecContext(ctx, `DELETE FROM bid_facts`); err == nil {
		t.Error("bid facts were deleted")
	}
}

func TestBidFactsHoldNoOtherFranchise(t *testing.T) {
	_, pools := testStore(t)
	rows, err := pools.Read().QueryContext(context.Background(), `SELECT name FROM pragma_table_info('bid_facts')`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	want := map[string]bool{"correlation_id": true, "player": true, "salary_cents": true, "years": true,
		"points_milli": true, "snipe": true, "beat_milli": true, "evidence": true, "closes": true}
	banned := regexp.MustCompile(`franchise|team|owner|bidder|opponent`)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		if !want[name] || banned.MatchString(name) {
			t.Errorf("unexpected bid_facts column %q", name)
		}
		delete(want, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(want) != 0 {
		t.Errorf("missing bid_facts columns %v", want)
	}
}

// legacyDDL is the move store as it was before bid_facts: the migration must add the new table
// without touching a stored envelope.
const legacyDDL = `
CREATE TABLE move_envelopes (correlation_id TEXT PRIMARY KEY, created TEXT NOT NULL, spec TEXT NOT NULL);
CREATE TABLE move_audit (
 correlation_id TEXT NOT NULL REFERENCES move_envelopes(correlation_id),
 sequence INTEGER NOT NULL CHECK (sequence >= 0), at TEXT NOT NULL, from_state TEXT NOT NULL,
 event TEXT NOT NULL, to_state TEXT NOT NULL, note TEXT NOT NULL, PRIMARY KEY (correlation_id, sequence));
CREATE TRIGGER move_envelopes_no_update BEFORE UPDATE ON move_envelopes
BEGIN SELECT RAISE(ABORT, 'move_envelopes is append-only'); END;`

func TestMigrationUpFromADatabaseWithoutBidFacts(t *testing.T) {
	ctx := context.Background()
	pools, err := db.Open(ctx, filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	if _, err := pools.Write().ExecContext(ctx, legacyDDL); err != nil {
		t.Fatal(err)
	}
	s := New(pools)
	old := testMove(t, "ir-old", bidAt(), envelope.ChecksPass, envelope.HandOff)
	if err := s.Save(ctx, old); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.Initialize(ctx); err != nil {
			t.Fatalf("upgrade: %v", err)
		}
	}
	got, err := s.Get(ctx, "ir-old")
	if err != nil || got.State() != envelope.HandedOff {
		t.Fatalf("legacy envelope after upgrade: %v, %v", got.State(), err)
	}
	e, facts := testBid(t, "bid-new")
	if err := s.SaveBid(ctx, e, facts); err != nil {
		t.Fatalf("bid after upgrade: %v", err)
	}
	if all, err := s.AllBidFacts(ctx); err != nil || len(all) != 1 {
		t.Errorf("bid facts after upgrade: %v, %v", all, err)
	}
	if list, err := s.List(ctx, "1", ""); err != nil || len(list) != 2 {
		t.Errorf("list after upgrade: %d, %v", len(list), err)
	}
}

func TestBidFactsStayFrozenWhenAHandedOffPlanIsWithdrawn(t *testing.T) {
	s, _ := testStore(t)
	ctx := context.Background()
	e, facts := testBid(t, "bid-1")
	if err := s.SaveBid(ctx, e, facts); err != nil {
		t.Fatal(err)
	}
	handed, err := e.HandOffManual(bidAt().Add(time.Minute), "copied")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, handed); err != nil {
		t.Fatal(err)
	}
	gone, err := handed.WithdrawBid(bidAt().Add(2*time.Minute), "did not post")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, gone); err != nil {
		t.Fatalf("saving the withdrawal: %v", err)
	}
	if got, err := s.Get(ctx, "bid-1"); err != nil || got.State() != envelope.Withdrawn || len(got.Receipt().Audit) != 3 {
		t.Fatalf("stored plan = %v, %v", got.Receipt(), err)
	}
	changed := facts
	changed.Salary, changed.Years = 900_000_000, 2
	if err := s.SaveBid(ctx, gone, changed); err == nil || !strings.Contains(err.Error(), "frozen") {
		t.Errorf("facts changed after the withdrawal: %v", err)
	}
	got, err := s.BidFacts(ctx, "bid-1")
	if err != nil || got.Salary != facts.Salary || got.Years != facts.Years || got.Points != facts.Points {
		t.Errorf("facts after withdrawal = %+v, %v", got, err)
	}
	if list, err := s.Awaiting(ctx); err != nil || len(list) != 0 {
		t.Errorf("the watcher still waits on a withdrawn plan: %d, %v", len(list), err)
	}
}
