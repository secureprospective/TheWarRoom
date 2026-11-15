package history

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
)

// NativeLineupsReading (one ledger replay per week, used by the export replay) gives every franchise
// exactly what NativeLineupReading gives it alone, on the Legacy fixture, for several weeks, in any
// franchise order, and leaves the ledger untouched: the shared base is not mutated between franchises.
func TestNativeLineupsReadingEqualsPerFranchiseReading(t *testing.T) {
	t.Parallel()
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	svc, store, _ := migrationService(t, export)
	if _, err := svc.Import(t.Context(), export.LeagueID, "Import", "creator"); err != nil {
		t.Fatal(err)
	}
	league := leaguemigration.AppLeagueID(export.LeagueID, export.Season)
	records, err := store.nativeRecords(t.Context(), league)
	if err != nil {
		t.Fatal(err)
	}
	assets, err := commissioner.NativeAssetsReading(league, records)
	if err != nil {
		t.Fatal(err)
	}
	// Set a lineup for some franchises in some weeks so readings differ between franchises and weeks.
	env := &lineupEnvFake{}
	store.SetLineupEnvironment(env)
	var changes []commissioner.RawLineupChange
	seen := map[string]bool{}
	for _, p := range assets.Snapshot.Players {
		if seen[p.Franchise] || len(seen) == 5 {
			continue
		}
		seen[p.Franchise] = true
		changes = append(changes, commissioner.RawLineupChange{Season: export.Season, Week: 1 + len(seen)%3, Franchise: p.Franchise, Starters: []string{p.ID}})
	}
	if _, err := leaguemigration.SyncLineups(t.Context(), migrationLedger{store}, league, "Import starters", "creator", export.Season, changes); err != nil {
		t.Fatal(err)
	}
	if records, err = store.nativeRecords(t.Context(), league); err != nil {
		t.Fatal(err)
	}
	at := migrationtest.ReadAt.Add(2 * time.Hour)
	directory, err := commissioner.NativeDirectoryReading(league, records, at)
	if err != nil {
		t.Fatal(err)
	}
	franchises := make([]string, 0, len(directory.Franchises))
	for _, seat := range directory.Franchises {
		franchises = append(franchises, seat.Franchise)
	}
	if len(franchises) != 32 {
		t.Fatalf("franchises = %d, want 32", len(franchises))
	}
	reversed := slices.Clone(franchises)
	slices.Reverse(reversed)
	before, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	principal := commissioner.PrincipalFor(records, commissioner.Replay(records).Creator)
	setSeen := 0
	for _, week := range []int{1, 2, 3, 13} {
		facts, err := env.LineupFacts(t.Context(), export.Season, week)
		if err != nil {
			t.Fatal(err)
		}
		forward, err := commissioner.NativeLineupsReading(league, principal, franchises, week, records, at, facts)
		if err != nil {
			t.Fatal(err)
		}
		backward, err := commissioner.NativeLineupsReading(league, principal, reversed, week, records, at, facts)
		if err != nil {
			t.Fatal(err)
		}
		if len(forward) != len(franchises) || len(backward) != len(franchises) {
			t.Fatalf("week %d: %d/%d lineups for %d franchises", week, len(forward), len(backward), len(franchises))
		}
		for i, franchise := range franchises {
			alone, err := commissioner.NativeLineupReading(league, franchise, principal, week, records, at, facts)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(forward[i], alone) {
				t.Fatalf("week %d %s: batch reading differs from the single reading", week, franchise)
			}
			if !reflect.DeepEqual(backward[len(franchises)-1-i], alone) {
				t.Fatalf("week %d %s: reading depends on franchise order (shared base mutated)", week, franchise)
			}
			if alone.Set {
				setSeen++
			}
		}
	}
	if setSeen == 0 {
		t.Fatal("no franchise had a set lineup: the fixture does not distinguish franchises")
	}
	after, err := json.Marshal(records)
	if err != nil || string(before) != string(after) {
		t.Fatal("reading mutated the ledger", err)
	}
}
