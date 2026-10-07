package moves

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func testStore(t *testing.T) (*Store, *db.Pools) {
	t.Helper()
	pools, err := db.Open(context.Background(), filepath.Join(t.TempDir(), "moves.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pools.Close(); err != nil {
			t.Error(err)
		}
	})
	store := New(pools)
	for range 2 {
		if err := store.Initialize(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return store, pools
}

func testMove(t *testing.T, id string, created time.Time, events ...envelope.Event) envelope.Envelope {
	t.Helper()
	player, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	deadline := created.Add(time.Hour)
	spec := envelope.Spec{
		Intent: "roster.ir", LeagueID: "1", FranchiseID: "0001",
		Subject:  envelope.Subject{Players: []playerid.PlayerID{player}, Picks: []string{"2027:1"}},
		Expected: envelope.ExpectedChange{Player: player, RosterStatus: domain.RosterIR},
		Gravity:  envelope.G2, Undo: envelope.Reversible,
		Target:   envelope.Target{Kind: envelope.Mapped, URL: "https://fixture.invalid/move"},
		Deadline: &deadline,
	}
	audit := []envelope.AuditEntry{}
	from := envelope.Draft
	for i, event := range events {
		var to envelope.State
		switch event {
		case envelope.Rebase:
			to = envelope.Draft
		case envelope.ChecksPass:
			to = envelope.Ready
		case envelope.ChecksBlock:
			to = envelope.Blocked
		case envelope.HandOff:
			to = envelope.HandedOff
		case envelope.NoChange:
			to = envelope.NotYetDone
		case envelope.Partial:
			to = envelope.NotVerified
		case envelope.Match:
			to = envelope.Landed
		case envelope.Contradiction:
			to = envelope.Failed
		case envelope.Invalidate:
			to = envelope.Stale
		case envelope.AwaitDOT:
			spec.Intent = "trade.propose"
			to = envelope.DOTReview
		case envelope.AwaitBid:
			spec.Intent = "bid.submit"
			to = envelope.BidPending
		case envelope.AwaitWaiver:
			spec.Intent = "waiver.claim"
			to = envelope.WaiverPending
		}
		audit = append(audit, envelope.AuditEntry{
			At: created.Add(time.Duration(i) * time.Second), From: from, Event: event, To: to, Note: "test",
		})
		from = to
	}
	e, err := envelope.Restore(id, created, spec, audit)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestSaveRoundTripIdempotenceAndPrefix(t *testing.T) {
	store, pools := testStore(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 7, 12, 0, 0, 123, time.UTC)
	draft := testMove(t, "roundtrip", at)
	ready := testMove(t, "roundtrip", at, envelope.ChecksPass)
	for _, e := range []envelope.Envelope{draft, draft, ready, ready} {
		if err := store.Save(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	restored, err := store.Get(ctx, ready.ID())
	if err != nil || !reflect.DeepEqual(restored.Receipt(), ready.Receipt()) ||
		!restored.Created().Equal(at) {
		t.Fatalf("round trip: %+v, %v", restored, err)
	}
	before, err := json.Marshal(ready.Receipt().Spec)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(restored.Receipt().Spec)
	if err != nil || string(before) != string(after) {
		t.Fatalf("spec JSON: %s != %s: %v", before, after, err)
	}
	var envelopes, entries int
	if err := pools.Read().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM move_envelopes`).Scan(&envelopes); err != nil {
		t.Fatal(err)
	}
	if err := pools.Read().QueryRowContext(ctx, `SELECT COUNT(*) FROM move_audit`).Scan(&entries); err != nil {
		t.Fatal(err)
	}
	if envelopes != 1 || entries != 1 {
		t.Fatalf("duplicate save: %d, %d", envelopes, entries)
	}
	diverged := testMove(t, "roundtrip", at, envelope.ChecksBlock)
	for _, conflicting := range []envelope.Envelope{
		diverged, draft, testMove(t, "roundtrip", at.Add(time.Second)),
	} {
		if err := store.Save(ctx, conflicting); err == nil || !strings.Contains(err.Error(), ready.ID()) {
			t.Fatalf("conflict accepted: %v", err)
		}
	}
	unchanged, err := store.Get(ctx, ready.ID())
	if err != nil || !reflect.DeepEqual(unchanged.Receipt(), ready.Receipt()) {
		t.Fatalf("conflict overwrote history: %+v, %v", unchanged, err)
	}
	if _, err := store.Get(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if err := store.Save(ctx, envelope.Envelope{}); err == nil {
		t.Fatal("saved zero envelope")
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

func TestListLatestOrderingAndScope(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	for _, e := range []envelope.Envelope{
		testMove(t, "a", at), testMove(t, "b", at), testMove(t, "c", at.Add(time.Nanosecond)),
		testMove(t, "a", at, envelope.ChecksPass),
	} {
		if err := store.Save(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	scoped := testMove(t, "other", at.Add(time.Hour)).Receipt()
	scoped.Spec.FranchiseID = "0002"
	scoped.Spec.LeagueID = "2"
	other, err := envelope.Restore(scoped.CorrelationID, at.Add(time.Hour), scoped.Spec, scoped.Audit)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, other); err != nil {
		t.Fatal(err)
	}
	listed, err := store.List(ctx, "1", "0001")
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, r := range listed {
		ids = append(ids, r.CorrelationID)
	}
	if !reflect.DeepEqual(ids, []string{"c", "b", "a"}) || listed[2].State != envelope.Ready {
		t.Fatalf("list latest/order: %+v", listed)
	}
	for _, scope := range [][2]string{{"1", "0002"}, {"2", "0001"}, {"absent", ""}} {
		empty, err := store.List(ctx, scope[0], scope[1])
		if err != nil || empty == nil || len(empty) != 0 {
			t.Fatalf("empty scope: %+v, %v", empty, err)
		}
	}
}

func TestAwaitingExactlyWaitingStates(t *testing.T) {
	store, _ := testStore(t)
	ctx := context.Background()
	at := time.Now().UTC()
	cases := []struct {
		state   envelope.State
		events  []envelope.Event
		waiting bool
	}{
		{envelope.Draft, nil, false},
		{envelope.Blocked, []envelope.Event{envelope.ChecksBlock}, false},
		{envelope.Ready, []envelope.Event{envelope.ChecksPass}, false},
		{envelope.HandedOff, nil, true},
		{envelope.NotYetDone, []envelope.Event{envelope.NoChange}, true},
		{envelope.NotVerified, []envelope.Event{envelope.Partial}, true},
		{envelope.DOTReview, []envelope.Event{envelope.AwaitDOT}, true},
		{envelope.BidPending, []envelope.Event{envelope.AwaitBid}, true},
		{envelope.WaiverPending, []envelope.Event{envelope.AwaitWaiver}, true},
		{envelope.Landed, []envelope.Event{envelope.Match}, false},
		{envelope.Failed, []envelope.Event{envelope.Contradiction}, false},
		{envelope.Stale, []envelope.Event{envelope.Invalidate}, false},
	}
	empty, err := store.Awaiting(ctx)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty awaiting: %+v, %v", empty, err)
	}
	want := map[string]envelope.State{}
	for _, tc := range cases {
		events := tc.events
		if tc.state != envelope.Draft && tc.state != envelope.Blocked && tc.state != envelope.Ready {
			events = append([]envelope.Event{envelope.ChecksPass, envelope.HandOff}, events...)
		}
		e := testMove(t, string(tc.state), at, events...)
		if e.State() != tc.state {
			t.Fatal(e.State(), tc.state)
		}
		if err := store.Save(ctx, e); err != nil {
			t.Fatal(err)
		}
		if tc.waiting {
			want[e.ID()] = tc.state
		}
	}
	found, err := store.Awaiting(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]envelope.State{}
	for _, e := range found {
		got[e.ID()] = e.State()
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("awaiting: %v != %v", got, want)
	}
}

func TestCorruptHistoryNamesID(t *testing.T) {
	store, pools := testStore(t)
	ctx := context.Background()
	at := time.Now().UTC()
	for _, id := range []string{"wrong-transition", "sequence-gap", "bad-time"} {
		e := testMove(t, id, at)
		if err := store.Save(ctx, e); err != nil {
			t.Fatal(err)
		}
		seq, stamp, to := 0, at.Format(timestampLayout), envelope.Landed
		if id == "sequence-gap" {
			seq = 1
		}
		if id == "bad-time" {
			stamp = "bad"
		}
		if _, err := pools.Write().ExecContext(ctx, `
INSERT INTO move_audit (correlation_id, sequence, at, from_state, event, to_state, note)
VALUES (?, ?, ?, ?, ?, ?, '')`, id, seq, stamp, envelope.Draft, envelope.ChecksPass, to); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Get(ctx, id); err == nil || !strings.Contains(err.Error(), id) {
			t.Fatalf("corruption not identified: %v", err)
		}
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO move_envelopes (correlation_id, created, spec) VALUES (?, ?, ?)`,
		"invalid-json", at.Format(timestampLayout), "{"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "invalid-json"); err == nil || !strings.Contains(err.Error(), "invalid-json") {
		t.Fatalf("bad JSON not identified: %v", err)
	}
}

func TestInvalidStoredSpecNamesID(t *testing.T) {
	store, pools := testStore(t)
	ctx := context.Background()
	at := time.Now().UTC()
	spec := testMove(t, "invalid-spec", at).Receipt().Spec
	spec.Intent = ""
	body, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pools.Write().ExecContext(ctx,
		`INSERT INTO move_envelopes (correlation_id, created, spec) VALUES (?, ?, ?)`,
		"invalid-spec", at.Format(timestampLayout), string(body)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "invalid-spec"); err == nil || !strings.Contains(err.Error(), "invalid-spec") {
		t.Fatalf("invalid spec not identified: %v", err)
	}
}

func TestSaveRollsBackEnvelopeAndAuditTogether(t *testing.T) {
	store, pools := testStore(t)
	ctx := context.Background()
	if _, err := pools.Write().ExecContext(ctx, `
CREATE TRIGGER reject_second_entry BEFORE INSERT ON move_audit WHEN NEW.sequence = 1
BEGIN SELECT RAISE(ABORT, 'test insertion failure'); END;`); err != nil {
		t.Fatal(err)
	}
	e := testMove(t, "atomic", time.Now().UTC(), envelope.ChecksPass, envelope.HandOff)
	if err := store.Save(ctx, e); err == nil || !strings.Contains(err.Error(), "test insertion failure") {
		t.Fatalf("expected insertion failure: %v", err)
	}
	if _, err := store.Get(ctx, e.ID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("envelope survived rollback: %v", err)
	}
	var count int
	if err := pools.Read().QueryRowContext(ctx, `SELECT COUNT(*) FROM move_audit`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit survived rollback: %d", count)
	}
}
