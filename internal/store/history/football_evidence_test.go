package history

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func footballBody(t *testing.T) ([]byte, string) {
	t.Helper()
	raw := RawFootballEvidence{Version: 1, Season: 2026, Source: "commissioner reference import", AsOf: time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC),
		Players: map[string]RawFootballPlayer{"0099": {Identity: "reference.player.99", Facts: nativecheck.PlayerFacts{Name: "Player 99", Position: leaguerules.QB, Team: "DAL", DesignationHeld: true}}},
		Weeks:   []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC)}}}}}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(body)
	return body, hex.EncodeToString(hash[:])
}

func TestFootballImportPinsIdentityFactsAndScheduleAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	store, pools := statStore(t, path)
	body, digest := footballBody(t)
	if err := store.ImportFootball(t.Context(), body, digest); err != nil {
		t.Fatal(err)
	}
	evidence, err := store.LoadFootball(t.Context(), digest)
	if err != nil {
		t.Fatal(err)
	}
	want, err := evidence.LineupFacts(t.Context(), 2026, 1)
	if err != nil || !want.ScheduleHeld || want.Facts["0099"].Team != "DAL" {
		t.Fatalf("held facts: %+v %v", want, err)
	}
	if evidence.Identities()["0099"] != "reference.player.99" {
		t.Fatal("identity not pinned")
	}
	evidence.PlayerFacts()["0099"] = nativecheck.PlayerFacts{}
	evidence.Identities()["0099"] = "changed"
	weeks, err := evidence.ScheduleDates(t.Context(), 2026)
	if err != nil {
		t.Fatal(err)
	}
	weeks[0].Games[0].Teams[0] = "BAD"
	if err := pools.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, _ := statStore(t, path)
	restored, err := reopened.LoadFootball(t.Context(), digest)
	if err != nil {
		t.Fatal(err)
	}
	got, err := restored.LineupFacts(t.Context(), 2026, 1)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("reopen changed evidence: %+v %v", got, err)
	}
	for _, scope := range [][2]int{{2025, 1}, {2026, 2}} {
		if _, err := restored.LineupFacts(t.Context(), scope[0], scope[1]); err == nil {
			t.Fatal("missing schedule guessed")
		}
	}
	for _, bad := range []string{"", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"} {
		if _, err := reopened.LoadFootball(t.Context(), bad); err == nil {
			t.Fatal("missing digest accepted")
		}
	}
}

func TestInvalidFootballImportPreservesPriorState(t *testing.T) {
	store, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	body, digest := footballBody(t)
	if err := store.ImportFootball(t.Context(), body, digest); err != nil {
		t.Fatal(err)
	}
	before := footballRows(t, store)
	for _, change := range []func(*RawFootballEvidence){
		func(r *RawFootballEvidence) { r.Version = 2 },
		func(r *RawFootballEvidence) { r.Players["99"] = r.Players["0099"]; delete(r.Players, "0099") },
		func(r *RawFootballEvidence) { r.Players["0100"] = r.Players["0099"] },
		func(r *RawFootballEvidence) {
			p := r.Players["0099"]
			p.Facts.Position = "coach"
			r.Players["0099"] = p
		},
		func(r *RawFootballEvidence) { r.Weeks[0].Games[0].Teams[1] = "DAL" },
		func(r *RawFootballEvidence) { r.Weeks[0].Games[0].Kickoff = time.Time{} },
	} {
		var raw RawFootballEvidence
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatal(err)
		}
		change(&raw)
		bad, err := json.Marshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.ImportFootball(t.Context(), bad, footballDigest(bad)); err == nil {
			t.Fatal("invalid import accepted")
		}
	}
	for _, bad := range [][]byte{body, append(append([]byte(nil), body...), []byte(` {}`)...), []byte(`{"unknown":true}`)} {
		if err := store.ImportFootball(t.Context(), bad, "wrong"); err == nil {
			t.Fatal("invalid digest or JSON accepted")
		}
	}
	if footballRows(t, store) != before {
		t.Fatal("invalid import changed archived state")
	}
	if _, err := store.LoadFootball(t.Context(), digest); err != nil {
		t.Fatal(err)
	}
}

func TestFootballLoadRejectsBodyStoredUnderWrongDigest(t *testing.T) {
	store, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	body, _ := footballBody(t)
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	const wrong = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if _, err := store.pools.Write().ExecContext(t.Context(), `INSERT INTO raw_archive (sha256,size,body) VALUES (?,?,?)`, wrong, len(body), compressed.Bytes()); err != nil {
		t.Fatal(err)
	}
	before := footballRows(t, store)
	if _, err := store.LoadFootball(t.Context(), wrong); err == nil {
		t.Fatal("changed body accepted under a pinned digest")
	}
	if footballRows(t, store) != before {
		t.Fatal("failed load changed prior state")
	}
}

func footballRows(t *testing.T, s *Store) [2]int {
	t.Helper()
	var counts [2]int
	if err := s.pools.Read().QueryRowContext(t.Context(), `SELECT (SELECT COUNT(*) FROM raw_archive),(SELECT COUNT(*) FROM fetch_log)`).Scan(&counts[0], &counts[1]); err != nil {
		t.Fatal(err)
	}
	return counts
}

func TestPinnedBidEnvironmentUsesScheduleAndFailsClosedAfterSeason(t *testing.T) {
	kickoff := time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC)
	evidence := &PinnedFootball{raw: RawFootballEvidence{Season: 2026, Weeks: []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: kickoff}}}}}}
	before, err := evidence.BidEnvironment(t.Context(), "native.test", kickoff.Add(-time.Hour))
	if err != nil || !before.Season.WeekKnown || before.Season.Week != 0 {
		t.Fatalf("before kickoff: %+v %v", before, err)
	}
	during, err := evidence.BidEnvironment(t.Context(), "native.test", kickoff)
	if err != nil || !during.Season.WeekKnown || during.Season.Week != 1 {
		t.Fatalf("at kickoff: %+v %v", during, err)
	}
	evidence.raw.Weeks[0].Games[0].Final = true
	ended, err := evidence.BidEnvironment(t.Context(), "native.test", kickoff.Add(24*time.Hour))
	if err != nil || ended.Season.WeekKnown {
		t.Fatalf("finished schedule guessed a week: %+v %v", ended, err)
	}
}
