package history

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// perSidePoint is a 16-Team Mirror league's restore point: player 1000 on both sides with different contracts.
func perSidePoint() commissioner.RawRosterRestorePoint {
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	snapshot := commissioner.RawNativeSnapshot{Season: 2026, RosterAt: at, DraftAt: at, FutureAt: at,
		Sides: leaguerules.CountedSides(leaguerules.Mirror16()),
		Players: []commissioner.NativePlayer{
			{ID: "1000", Franchise: "franchise.01", Status: domain.RosterActive, Salary: 100, ExpirationYear: 2027},
			{ID: "1000", Franchise: "franchise.09", Status: domain.RosterIR, Salary: 900, ExpirationYear: 2029},
			{ID: "2000", Franchise: "franchise.02", Status: domain.RosterActive, Salary: 200, ExpirationYear: 2027},
		}}
	divisions := []string{"AFC East", "AFC West", "NFC East", "NFC West"}
	for i := 1; i <= 16; i++ {
		id := fmt.Sprintf("franchise.%02d", i)
		snapshot.Franchises = append(snapshot.Franchises, commissioner.NativeFranchise{ID: id, HeldID: id,
			Name: fmt.Sprintf("Franchise %d", i), Abbreviation: fmt.Sprintf("F%d", i), Division: divisions[(i-1)/4]})
	}
	return commissioner.RawRosterRestorePoint{Version: 1, League: "native.mirror", Revision: 2, CapturedAt: at,
		Source: commissioner.SourceNative, Preset: commissioner.PresetMirror16, Snapshot: snapshot}
}

// Both sides are saved in one transaction and restored together: the stand-up saves both sides' ownership,
// and the point it saves restores into a second fresh store with both copies exact.
func TestPerSideRostersSaveAndRestoreBothSides(t *testing.T) {
	first, _, _ := newStore(t)
	if _, err := first.StandUpRosters(t.Context(), perSidePoint()); err != nil {
		t.Fatal(err)
	}
	saved, err := first.RosterRestorePoint(t.Context(), "native.mirror")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.Snapshot.Players, perSidePoint().Snapshot.Players) {
		t.Fatalf("saved copies: %+v", saved.Snapshot.Players)
	}
	rows, err := first.pools.Read().QueryContext(t.Context(),
		`SELECT side,franchise FROM player_ownership o JOIN player_external_id e ON e.player=o.player
		 WHERE o.league='native.mirror' AND e.provider='mfl' AND e.value='1000' ORDER BY side`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var owned []string
	for rows.Next() {
		var side, franchise string
		if err := rows.Scan(&side, &franchise); err != nil {
			t.Fatal(err)
		}
		owned = append(owned, side+"="+franchise)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(owned, []string{"AFC=franchise.01", "NFC=franchise.09"}) {
		t.Fatalf("ownership: %v", owned)
	}
	if len(saved.Identities) != 2 {
		t.Fatalf("each player's identity is exported once: %+v", saved.Identities)
	}
	second, _, _ := newStore(t)
	if _, err := second.StandUpRosters(t.Context(), saved); err != nil {
		t.Fatal(err)
	}
	restored, err := second.RosterRestorePoint(t.Context(), "native.mirror")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.Snapshot, saved.Snapshot) || !reflect.DeepEqual(restored.Identities, saved.Identities) {
		t.Fatalf("restore changed the rosters: %+v", restored.Snapshot.Players)
	}
	twice := perSidePoint()
	twice.Snapshot.Players = append(twice.Snapshot.Players, commissioner.NativePlayer{ID: "1000",
		Franchise: "franchise.02", Status: domain.RosterActive, ExpirationYear: 2027})
	if err := twice.Validate(); err == nil {
		t.Fatal("a restore point with two copies on one side was accepted")
	}
}
