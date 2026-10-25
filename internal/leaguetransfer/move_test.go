package leaguetransfer

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"

	"github.com/google/uuid"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func moveFixture(t *testing.T) (*history.Store, *history.Store, RawExport) {
	t.Helper()
	source, _ := importTarget(t, filepath.Join(t.TempDir(), "source.db"))
	at := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	source.SetClock(func() time.Time { return at })
	command := commissioner.RawCommand{League: "native.test", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Moving league", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Request: "create", Reason: "Create the league", Effective: "now", Visibility: "league"}
	runTransferCommand(t, source, command, "creator")
	evidence := history.RawFootballEvidence{Version: 1, Season: 2026, Source: "commissioner reference import", AsOf: at,
		Players: map[string]history.RawFootballPlayer{"0099": {Identity: "reference.99", Facts: nativecheck.PlayerFacts{Name: "QB", Position: leaguerules.QB, Team: "DAL"}}},
		Weeks:   []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(time.Hour)}}}}}
	body, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	football := digest(body)
	if err := source.ImportFootball(t.Context(), body, football); err != nil {
		t.Fatal(err)
	}
	raw, err := Capture(t.Context(), source, "native.test", "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", football, at, nil)
	if err != nil {
		t.Fatal(err)
	}
	target, _ := importTarget(t, filepath.Join(t.TempDir(), "target.db"))
	target.SetClock(func() time.Time { return at })
	if _, err := Import(t.Context(), target, bytes.NewReader(encryptedImport(t, raw)), [32]byte{7}); err != nil {
		t.Fatal(err)
	}
	return source, target, raw
}
func moveAction(t *testing.T, source, target *history.Store, raw RawExport, action string) {
	t.Helper()
	if err := Move(t.Context(), source, target, raw, action, "creator", true); err != nil {
		t.Fatalf("%s: %v", action, err)
	}
}
func assertWritable(t *testing.T, s *history.Store, league string, want bool) {
	t.Helper()
	status, err := s.TransferStatus(t.Context(), league)
	if err != nil || status.Writable != want || status.Label == "" {
		t.Fatal(status, err)
	}
	head, err := s.CommittedHead(t.Context(), league, commissioner.Stream)
	if err != nil {
		t.Fatal(err)
	}
	raw := commissioner.RawCommand{League: league, Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: "cap.amount", Value: json.RawMessage(strconv.FormatInt(100000000000+head*1000, 10))}, Expected: head, Request: uuid.NewString(), Reason: "Check current authority", Effective: "now", Visibility: "league"}
	_, err = s.ExecuteCommissioner(t.Context(), raw, "creator", nil)
	after, readErr := s.CommittedHead(t.Context(), league, commissioner.Stream)
	if readErr != nil || (!want && after != head) {
		t.Fatal("rejected write changed history", after, head, readErr)
	}
	if (err == nil) != want {
		t.Fatalf("write allowed=%v, want=%v: %v", err == nil, want, err)
	}
}
func TestMoveSealAbortAndActivation(t *testing.T) {
	source, target, raw := moveFixture(t)
	for _, action := range []string{"seal", "activate", "abort"} {
		if err := Move(t.Context(), source, target, raw, action, "owner.0", true); err == nil {
			t.Fatal("owner moved league")
		}
		if err := Move(t.Context(), source, target, raw, action, "creator", false); err == nil {
			t.Fatal("unconfirmed move")
		}
	}
	moveAction(t, source, target, raw, "seal")
	assertWritable(t, source, raw.League, false)
	assertWritable(t, target, raw.League, false)
	moveAction(t, source, target, raw, "abort")
	moveAction(t, source, target, raw, "abort")
	status, err := source.TransferStatus(t.Context(), raw.League)
	if err != nil || !status.Writable {
		t.Fatal(status, err)
	}
	moveAction(t, source, target, raw, "seal")
	moveAction(t, source, target, raw, "activate")
	moveAction(t, source, target, raw, "activate")
	if err := Move(t.Context(), source, target, raw, "abort", "creator", true); err == nil {
		t.Fatal("post activation abort unsealed source")
	}
	assertWritable(t, source, raw.League, false)
	assertWritable(t, target, raw.League, true)
	leagues, err := target.NativeLeagues(t.Context())
	if err != nil || len(leagues) != 1 {
		t.Fatal(leagues, err)
	}
}
func TestMoveRejectsStaleExportAndWrongDestination(t *testing.T) {
	source, target, raw := moveFixture(t)
	other, _ := importTarget(t, filepath.Join(t.TempDir(), "other.db"))
	if err := Move(t.Context(), source, other, raw, "seal", "creator", true); err == nil {
		t.Fatal("unprepared destination paused source")
	}
	status, err := source.TransferStatus(t.Context(), raw.League)
	if err != nil || !status.Writable {
		t.Fatal(status, err)
	}
	moveAction(t, source, target, raw, "seal")
	if err := Move(t.Context(), source, other, raw, "activate", "creator", true); err == nil {
		t.Fatal("alternate destination activated")
	}
	moveAction(t, source, target, raw, "abort")
	assertWritable(t, source, raw.League, true)
	if err := Move(t.Context(), source, target, raw, "seal", "creator", true); err == nil {
		t.Fatal("stale export sealed")
	}
}
func TestReverseMoveIncludesEveryServerEvent(t *testing.T) {
	source, target, raw := moveFixture(t)
	// Populate both league-list caches before authority changes.
	if _, err := source.NativeLeagues(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := target.NativeLeagues(t.Context()); err != nil {
		t.Fatal(err)
	}
	moveAction(t, source, target, raw, "seal")
	moveAction(t, source, target, raw, "activate")
	assertWritable(t, target, raw.League, true)
	fence, err := target.TransferFence(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := Capture(t.Context(), target, raw.League, fence.Epoch, raw.Football, raw.FrozenAt, raw.Aliases)
	if err != nil {
		t.Fatal(err)
	}
	if len(reverse.Events) <= len(raw.Events) {
		t.Fatal("no server event in reverse")
	}
	moveAction(t, target, source, reverse, "seal")
	assertWritable(t, target, raw.League, false)
	assertWritable(t, source, raw.League, false)
	moveAction(t, target, source, reverse, "abort")
	moveAction(t, target, source, reverse, "seal")
	moveAction(t, target, source, reverse, "activate")
	events, _, _, err := source.CaptureTransfer(t.Context(), raw.League, []string{raw.Football})
	if err != nil || !reflect.DeepEqual(events, reverse.Events) {
		t.Fatal("reverse lost or rewrote server events", err)
	}
	leagues, err := source.NativeLeagues(t.Context())
	if err != nil || len(leagues) != 1 {
		t.Fatal(leagues, err)
	}
	leagues, err = target.NativeLeagues(t.Context())
	if err != nil || len(leagues) != 0 {
		t.Fatal(leagues, err)
	}
	assertWritable(t, source, raw.League, true)
	assertWritable(t, target, raw.League, false)
}
func TestMoveActivationRacesAbort(t *testing.T) {
	source, target, raw := moveFixture(t)
	moveAction(t, source, target, raw, "seal")
	var wg sync.WaitGroup
	var errs [2]error
	for i, action := range []string{"activate", "abort"} {
		wg.Go(func() { errs[i] = Move(t.Context(), source, target, raw, action, "creator", true) })
	}
	wg.Wait()
	if (errs[0] == nil) == (errs[1] == nil) {
		t.Fatal("expected one decision", errs)
	}
	a, err := source.TransferStatus(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	b, err := target.TransferStatus(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	if a.Writable == b.Writable {
		t.Fatal("expected exactly one authority", a, b)
	}
}
