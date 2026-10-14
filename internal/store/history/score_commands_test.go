package history

import (
	"encoding/json"
	"errors"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func scoreCommand(t *testing.T, command string, change commissioner.RawScoreChange,
	expected int64, request string) commissioner.RawCommand {
	t.Helper()
	body, err := json.Marshal(change)
	if err != nil {
		t.Fatal(err)
	}
	return commissioner.RawCommand{League: "native.test", Command: command,
		Args: commissioner.RawArgs{Value: body}, Reason: "Correct the held evidence",
		Expected: expected, Request: request, Effective: "now", Visibility: "league"}
}

func pointsDelta(points leaguerules.Decimal) *leaguerules.Decimal { return &points }

func scoreRecords(t *testing.T, s *Store) []commissioner.Record {
	t.Helper()
	events, err := s.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	records, err := commissionerRecords(events)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func runScoreCommand(t *testing.T, s *Store, raw commissioner.RawCommand, principal string) commissioner.Receipt {
	t.Helper()
	receipt, err := s.ExecuteCommissioner(t.Context(), raw, principal, nil)
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func scoreChangesFor(t *testing.T, s *Store) commissioner.ScoreChanges {
	t.Helper()
	changes, err := commissioner.PlayerScoreChanges(scoreRecords(t, s), "0099", 2025, 1)
	if err != nil {
		t.Fatal(err)
	}
	return changes
}

func setupScoreLeague(t *testing.T, s *Store) {
	t.Helper()
	runScoreCommand(t, s, createCommand(), "creator")
	role := commissioner.RawCommand{League: "native.test", Command: "commish.role.grant",
		Args:   commissioner.RawArgs{People: &commissioner.RawPeople{Principal: "cc", Role: governance.CoCommissioner}},
		Reason: "Appoint the second commissioner", Expected: 1, Request: "cc", Effective: "now", Visibility: "private"}
	runScoreCommand(t, s, role, "creator")
}

func TestNativePositionCommandsCompensationAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "position.db")
	s, pools := statStore(t, path)
	setupScoreLeague(t, s)
	set := scoreCommand(t, commissioner.PlayerPosition,
		commissioner.RawScoreChange{Player: "0099", Position: leaguerules.DT}, 2, "position")
	receipt := runScoreCommand(t, s, set, "cc")
	if retry := runScoreCommand(t, s, set, "cc"); retry != receipt {
		t.Fatal("retry appended a second override")
	}
	assertPositionPoints(t, s, leaguerules.DT, 25000)
	clearCommand := scoreCommand(t, commissioner.PlayerPositionClear,
		commissioner.RawScoreChange{Player: "0099"}, 3, "clear")
	cleared := runScoreCommand(t, s, clearCommand, "cc")
	assertPositionPoints(t, s, "", 15000)
	runScoreCommand(t, s, reverseCommand(cleared.Entry, 4, "undo-clear"), "cc")
	assertPositionPoints(t, s, leaguerules.DT, 25000)
	runScoreCommand(t, s, reverseCommand(receipt.Entry, 5, "undo-position"), "cc")
	assertPositionPoints(t, s, "", 15000)
	before := scoreChangesFor(t, s)
	if len(before.Changes) != 2 || before.Changes[0].Entry != receipt.Entry || before.Changes[0].Status != "Reversed" {
		t.Fatalf("override provenance: %+v", before)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, _ := statStore(t, path)
	if after := scoreChangesFor(t, reopened); !reflect.DeepEqual(before, after) {
		t.Fatalf("reopen: %+v", after)
	}
	set.Expected, set.Request = 6, "position-after-reopen"
	again := runScoreCommand(t, reopened, set, "cc")
	assertPositionPoints(t, reopened, leaguerules.DT, 25000)
	runScoreCommand(t, reopened, reverseCommand(again.Entry, 7, "undo-after-reopen"), "cc")
	assertPositionPoints(t, reopened, "", 15000)
	if _, err := reopened.ExecuteCommissioner(t.Context(), reverseCommand("missing", 8, "missing"), "creator", nil); err == nil {
		t.Fatal("missing compensation target accepted")
	}
}

func assertPositionPoints(t *testing.T, s *Store, want leaguerules.Position, total leaguerules.Decimal) {
	t.Helper()
	changes := scoreChangesFor(t, s)
	if changes.Position != want {
		t.Fatalf("position = %s, want %s", changes.Position, want)
	}
	position, why := changes.PositionUsed(leaguerules.LB)
	if why == "" {
		t.Fatal("position source has no plain explanation")
	}
	points, _, err := fantasyscore.Score(leaguerules.LegacyNFL().Scoring, position,
		fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{"tackle": 10000}})
	if err != nil || points.State != leaguerules.IsKnown || points.Value != total {
		t.Fatalf("position score = %+v, %v; want %d", points, err, total)
	}
}

func TestNativeDeltaCommandsReplayOnceAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "delta.db")
	s, pools := statStore(t, path)
	setupScoreLeague(t, s)
	raw := scoreCommand(t, commissioner.ScorePlayer, commissioner.RawScoreChange{
		Player: "0099", Season: 2025, Week: 1, Delta: pointsDelta(12500)}, 2, "delta")
	receipt := runScoreCommand(t, s, raw, "cc")
	if retry := runScoreCommand(t, s, raw, "cc"); retry != receipt {
		t.Fatal("retry appended twice")
	}
	if change := scoreChangesFor(t, s); change.Delta != 12500 || len(change.Changes) != 1 ||
		change.Changes[0].Entry != receipt.Entry {
		t.Fatalf("first delta: %+v", change)
	}
	inverse := scoreCommand(t, commissioner.ScorePlayer, commissioner.RawScoreChange{
		Player: "0099", Season: 2025, Week: 1, Delta: pointsDelta(-12500)}, 3, "inverse")
	inv := runScoreCommand(t, s, inverse, "cc")
	if change := scoreChangesFor(t, s); change.Delta != 0 {
		t.Fatalf("inverse: %+v", change)
	}
	runScoreCommand(t, s, reverseCommand(inv.Entry, 4, "undo-inverse"), "cc")
	if change := scoreChangesFor(t, s); change.Delta != 12500 {
		t.Fatalf("compensated inverse: %+v", change)
	}
	before := scoreChangesFor(t, s)
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, _ := statStore(t, path)
	for range 3 {
		if after := scoreChangesFor(t, reopened); !reflect.DeepEqual(before, after) {
			t.Fatalf("replay multiplied: %+v", after)
		}
	}
	runScoreCommand(t, reopened, reverseCommand(receipt.Entry, 5, "undo-delta"), "cc")
	if change := scoreChangesFor(t, reopened); change.Delta != 0 {
		t.Fatalf("revert after reopen: %+v", change)
	}
	for _, scope := range []struct {
		player       string
		season, week int
	}{{"0100", 2025, 1}, {"0099", 2026, 1}, {"0099", 2025, 2}} {
		change, err := commissioner.PlayerScoreChanges(scoreRecords(t, reopened), scope.player, scope.season, scope.week)
		if err != nil || change.Delta != 0 || len(change.Changes) != 0 {
			t.Fatalf("scope leaked: %+v %v", change, err)
		}
	}
}

func TestNativeFranchiseDeltaRequiresG3(t *testing.T) {
	path := filepath.Join(t.TempDir(), "franchise.db")
	s, pools := statStore(t, path)
	setupScoreLeague(t, s)
	raw := scoreCommand(t, commissioner.ScoreFranchise, commissioner.RawScoreChange{
		Franchise: "franchise.01", Season: 2025, Week: 1, Delta: pointsDelta(25000)}, 2, "team")
	raw.Gravity = "G2"
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "cc", nil); err == nil {
		t.Fatal("franchise delta accepted at G2")
	}
	raw.Gravity = "G3"
	raw.Reason = " "
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "cc", nil); err == nil {
		t.Fatal("franchise delta lacks reason")
	}
	raw.Reason = "Correct franchise points"
	receipt := runScoreCommand(t, s, raw, "cc")
	change, err := commissioner.FranchiseScoreChanges(scoreRecords(t, s), "franchise.01", 2025, 1)
	if err != nil || change.Delta != 25000 || len(change.Changes) != 1 || change.Changes[0].Entry != receipt.Entry {
		t.Fatalf("franchise points: %+v %v", change, err)
	}
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	s, _ = statStore(t, path)
	change, err = commissioner.FranchiseScoreChanges(scoreRecords(t, s), "franchise.01", 2025, 1)
	if err != nil || change.Delta != 25000 {
		t.Fatalf("franchise reopen: %+v %v", change, err)
	}
	runScoreCommand(t, s, reverseCommand(receipt.Entry, 3, "undo-team"), "cc")
	change, err = commissioner.FranchiseScoreChanges(scoreRecords(t, s), "franchise.01", 2025, 1)
	if err != nil || change.Delta != 0 {
		t.Fatalf("franchise revert: %+v %v", change, err)
	}
	bad := scoreCommand(t, commissioner.ScoreFranchise, commissioner.RawScoreChange{
		Franchise: "franchise.99", Season: 2025, Week: 1, Delta: pointsDelta(10000)}, 4, "bad-team")
	if _, err := s.ExecuteCommissioner(t.Context(), bad, "cc", nil); err == nil {
		t.Fatal("foreign franchise accepted")
	}
}

func TestNativeScoreCommandRefusalsDoNotAppend(t *testing.T) {
	s, _, _ := newStore(t)
	setupScoreLeague(t, s)
	raw := scoreCommand(t, commissioner.PlayerPosition,
		commissioner.RawScoreChange{Player: "0099", Position: leaguerules.DT}, 2, "position")
	for _, principal := range []string{"owner", "dot", "unknown"} {
		if _, err := s.ExecuteCommissioner(t.Context(), raw, principal, nil); err == nil {
			t.Fatalf("%s authorized", principal)
		}
	}
	for _, mutate := range []func(*commissioner.RawCommand){
		func(r *commissioner.RawCommand) { r.Reason = " " },
		func(r *commissioner.RawCommand) { r.Effective = "next_rollover" },
		func(r *commissioner.RawCommand) { r.Visibility = "private" },
		func(r *commissioner.RawCommand) { r.Args.Name = "extra" },
		func(r *commissioner.RawCommand) { r.Args.Value = json.RawMessage(`{"player":"99","position":"DT"}`) },
		func(r *commissioner.RawCommand) { r.Args.Value = json.RawMessage(`{"player":"0099","position":"NT"}`) },
		func(r *commissioner.RawCommand) {
			r.Args.Value = json.RawMessage(`{"player":"0099","position":"DT","total":5}`)
		},
	} {
		bad := raw
		mutate(&bad)
		if _, err := s.ExecuteCommissioner(t.Context(), bad, "cc", nil); err == nil {
			t.Fatalf("invalid payload accepted: %+v", bad)
		}
	}
	raw.Expected = 1
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "cc", nil); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("version: %v", err)
	}
	if records := scoreRecords(t, s); len(records) != 2 {
		t.Fatalf("refusal appended: %d", len(records))
	}
	raw.Expected = 2
	receipt := runScoreCommand(t, s, raw, "cc")
	raw.Reason = "Changed retry"
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "cc", nil); err == nil {
		t.Fatal("request key reused")
	}
	ledger, err := s.CommissionerLedger(t.Context(), raw.League, "creator", nil)
	if err != nil || ledger.Entries[0].ID != receipt.Entry || !strings.Contains(ledger.Entries[0].What, "league scoring") ||
		ledger.Entries[0].Undo != "league" {
		t.Fatalf("ledger: %+v %v", ledger, err)
	}
}

func TestNativeScoreDeltaValidationAndSeam(t *testing.T) {
	for _, change := range []commissioner.RawScoreChange{
		{Player: "0099", Season: 2025, Week: 1},
		{Player: "0099", Season: 2025, Week: 1, Delta: pointsDelta(0)},
		{Player: "0099", Season: 2025, Week: 23, Delta: pointsDelta(10000)},
		{Player: "0099", Season: 1919, Week: 1, Delta: pointsDelta(10000)},
		{Player: "0099", Franchise: "franchise.01", Season: 2025, Week: 1, Delta: pointsDelta(10000)},
	} {
		if err := change.Validate(commissioner.ScorePlayer); err == nil {
			t.Fatalf("invalid delta: %+v", change)
		}
	}
	seam := commissioner.ResultFlipCheck(commissioner.RawScoreChange{Player: "0099", Season: 2025, Week: 1, Delta: pointsDelta(math.MaxInt64)})
	if seam.State != leaguerules.NeedsRuling || !strings.Contains(seam.Citation, "not known yet") {
		t.Fatalf("result flip asserted without results: %+v", seam)
	}
}

func TestNativeDeltaOverflowRefusesAppendAndCompensation(t *testing.T) {
	s, _, _ := newStore(t)
	setupScoreLeague(t, s)
	maximum := scoreCommand(t, commissioner.ScorePlayer, commissioner.RawScoreChange{
		Player: "0099", Season: 2025, Week: 1, Delta: pointsDelta(math.MaxInt64)}, 2, "maximum")
	runScoreCommand(t, s, maximum, "cc")
	overflow := scoreCommand(t, commissioner.ScorePlayer, commissioner.RawScoreChange{
		Player: "0099", Season: 2025, Week: 1, Delta: pointsDelta(1)}, 3, "overflow")
	if _, err := s.ExecuteCommissioner(t.Context(), overflow, "cc", nil); err == nil {
		t.Fatal("overflow appended")
	}
	inverse := scoreCommand(t, commissioner.ScorePlayer, commissioner.RawScoreChange{
		Player: "0099", Season: 2025, Week: 1, Delta: pointsDelta(-math.MaxInt64)}, 3, "balance")
	receipt := runScoreCommand(t, s, inverse, "cc")
	overflow.Expected = 4
	runScoreCommand(t, s, overflow, "cc")
	if _, err := s.ExecuteCommissioner(t.Context(), reverseCommand(receipt.Entry, 5, "overflow-revert"), "cc", nil); err == nil {
		t.Fatal("compensation restored an unrepresentable points total")
	}
	if records := scoreRecords(t, s); len(records) != 5 {
		t.Fatalf("overflow changed ledger: %d", len(records))
	}
}
