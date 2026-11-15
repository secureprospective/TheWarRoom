package moves

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

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

func TestConditionalDropVersionsAreAppendOnlyAndFreezeWithBid(t *testing.T) {
	s, pools := testStore(t)
	ctx := context.Background()
	e, f := testBid(t, "conditional-1")
	if err := s.SaveBid(ctx, e, f); err != nil {
		t.Fatal(err)
	}
	stale, err := e.Invalidate(bidAt().Add(time.Minute), "condition changed")
	if err != nil {
		t.Fatal(err)
	}
	draft, err := stale.Rebase(bidAt().Add(2*time.Minute), "condition changed")
	if err != nil {
		t.Fatal(err)
	}
	ready, err := draft.Check(bidAt().Add(3*time.Minute), snapshot.Snapshot{}, envelope.BidCheck{})
	if err != nil {
		t.Fatal(err)
	}
	f.ConditionalDrop, err = playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	f.ConditionalName = "Rostered, Rick (QB, BUF)"
	if err := s.SaveBid(ctx, ready, f); err != nil {
		t.Fatal(err)
	}
	got, err := s.BidFacts(ctx, ready.ID())
	if err != nil || got.ConditionalDrop != f.ConditionalDrop || got.ConditionalName != f.ConditionalName {
		t.Fatalf("stored conditional = %+v, %v", got, err)
	}
	var versions int
	if err := pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM bid_conditions WHERE correlation_id = ?`, ready.ID()).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if versions != 2 {
		t.Fatalf("got %d versions, want 2", versions)
	}
	var old string
	if err := pools.Read().QueryRowContext(ctx, `SELECT player FROM bid_conditions
 WHERE correlation_id = ? ORDER BY sequence LIMIT 1`, ready.ID()).Scan(&old); err != nil {
		t.Fatal(err)
	}
	if old != "" {
		t.Fatalf("earlier plan condition overwritten: %q", old)
	}
	handed, err := ready.HandOffManual(bidAt().Add(4*time.Minute), "copied")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(ctx, handed); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`UPDATE bid_conditions SET player = '0777' WHERE correlation_id = ?`,
		`DELETE FROM bid_conditions WHERE correlation_id = ?`,
		`INSERT INTO bid_conditions VALUES (?, 99, '0777', 'Changed')`,
	} {
		if _, err := pools.Write().ExecContext(ctx, query, ready.ID()); err == nil {
			t.Fatalf("accepted %s", query)
		}
	}
}

func TestAppendOnlyTriggers(t *testing.T) {
	store, pools := testStore(t)
	ctx := context.Background()
	e := testMove(t, "guarded", time.Now().UTC(), envelope.ChecksPass)
	if err := store.Save(ctx, e); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`UPDATE move_envelopes SET spec = spec WHERE correlation_id = ?`,
		`DELETE FROM move_envelopes WHERE correlation_id = ?`,
		`UPDATE move_audit SET note = 'changed' WHERE correlation_id = ?`,
		`DELETE FROM move_audit WHERE correlation_id = ?`,
	} {
		if _, err := pools.Write().ExecContext(ctx, query, e.ID()); err == nil ||
			!strings.Contains(err.Error(), "append-only") {
			t.Fatalf("trigger failed for %s: %v", query, err)
		}
	}
	got, err := store.Get(ctx, e.ID())
	if err != nil || !reflect.DeepEqual(got.Receipt(), e.Receipt()) {
		t.Fatalf("triggers: %+v, %v", got, err)
	}
}
