package history

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestClosedBooksPersistAndFallback(t *testing.T) {
	r := newRolloverDB(t)
	r.do(t, r.rollover(t, 2027, false, "close"))
	r.reopenFile(t)
	events, err := r.s.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	full, err := commissionerRecords(events)
	if err != nil {
		t.Fatal(err)
	}
	folded, err := r.s.nativeRecords(t.Context(), "native.test")
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(folded, full) {
		t.Fatal("restart did not load persisted summary")
	}
	assertStoredBooksReadings(t, r.s, "native.test", folded, full)
	var body []byte
	if err := r.s.pools.Read().QueryRow(`SELECT payload FROM league_season_summary`).Scan(&body); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"missing", "checksum", "json", "source", "version", "metadata"} {
		t.Run(mode, func(t *testing.T) { testStoredBooksFallback(t, r.s, "native.test", body, mode, full) })
	}
}

func testStoredBooksFallback(t *testing.T, s *Store, league string, original []byte, mode string, full []commissioner.Record) {
	t.Helper()
	tx, err := s.pools.Write().BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if mode == "missing" {
		if _, err := tx.ExecContext(t.Context(), `DELETE FROM league_season_summary`); err != nil {
			t.Fatal(err)
		}
	} else {
		body := original
		var raw commissioner.RawClosedBooks
		if err := json.Unmarshal(original, &raw); err != nil {
			t.Fatal(err)
		}
		switch mode {
		case "json":
			body = []byte("{")
		case "source":
			raw.SourceHash = strings.Repeat("0", 64)
		case "version":
			raw.Version++
		case "metadata":
			raw.Season++
		}
		if mode == "source" || mode == "version" || mode == "metadata" {
			body, err = json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
		}
		hash := sha256.Sum256(body)
		expected := hex.EncodeToString(hash[:])
		if mode == "checksum" {
			expected = strings.Repeat("0", 64)
		}
		if _, err := tx.ExecContext(t.Context(), `UPDATE league_season_summary SET payload=?,payload_hash=?`, body, expected); err != nil {
			t.Fatal(err)
		}
	}
	records, err := loadClosedBooks(t.Context(), tx, league, full)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(records, full) {
		t.Fatal("invalid summary did not fall back to full replay")
	}
	assertStoredBooksReadings(t, s, league, records, full)
}

func assertStoredBooksReadings(t *testing.T, s *Store, league string, folded, full []commissioner.Record) {
	t.Helper()
	got, gotErr := commissioner.PeopleReading(league, "creator", folded)
	want, wantErr := commissioner.PeopleReading(league, "creator", full)
	if !reflect.DeepEqual(got, want) || (gotErr != nil) != (wantErr != nil) {
		t.Fatalf("people differs: %v / %v", gotErr, wantErr)
	}
	gotAssets, gotErr := commissioner.NativeAssetsReading(league, folded, s.now())
	wantAssets, wantErr := commissioner.NativeAssetsReading(league, full, s.now())
	if !reflect.DeepEqual(gotAssets, wantAssets) || (gotErr != nil) != (wantErr != nil) {
		t.Fatalf("assets differs: %v / %v", gotErr, wantErr)
	}
}

func TestClosedBooksFailureRollsBackRollover(t *testing.T) {
	r := newRolloverDB(t)
	before := r.version(t)
	if _, err := r.s.pools.Write().ExecContext(t.Context(), `CREATE TRIGGER reject_summary BEFORE INSERT ON league_season_summary BEGIN SELECT RAISE(ABORT,'summary unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	_, err := r.s.ExecuteCommissioner(t.Context(), r.rollover(t, 2027, false, "fail-close"), "creator", nil)
	if err == nil || !strings.Contains(err.Error(), "summary unavailable") {
		t.Fatalf("expected injected summary failure: %v", err)
	}
	if got := r.version(t); got != before {
		t.Fatalf("failed summary left rollover revision %d, want %d", got, before)
	}
	var count int
	if err := r.s.pools.Read().QueryRow(`SELECT count(*) FROM league_season_summary`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed transaction left a summary")
	}
}
