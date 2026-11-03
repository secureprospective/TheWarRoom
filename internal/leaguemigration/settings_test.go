package leaguemigration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestDefaultPresetMatchesTeamCount(t *testing.T) {
	t.Parallel()
	for count, want := range map[int]string{16: "mirror16", 32: "legacy_nfl", 12: "default32", 20: "default32"} {
		if got := DefaultPreset(count); got != want {
			t.Errorf("%d teams: %s, want %s", count, got, want)
		}
	}
	if _, err := Base("invented"); err == nil {
		t.Fatal("accepted an unknown settings bundle")
	}
}

type heldWithoutReferenceLedger struct {
	Ledger
	records []commissioner.Record
	writes  int
}

func (l *heldWithoutReferenceLedger) Records(context.Context, string) ([]commissioner.Record, error) {
	return l.records, nil
}

func (l *heldWithoutReferenceLedger) Execute(context.Context, commissioner.RawCommand, string) (commissioner.Receipt, error) {
	l.writes++
	return commissioner.Receipt{}, errors.New("a held start must not invent a reference league")
}

func TestHeldStartingAssetsDoNotRequireAnMFLReference(t *testing.T) {
	create, err := (commissioner.RawCommand{League: "native.held-assets", Command: commissioner.Create,
		Args: commissioner.RawArgs{Name: "Held assets", Preset: commissioner.PresetEmpty, FranchiseCount: 4, Season: 2026}}).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	ledger := &heldWithoutReferenceLedger{records: []commissioner.Record{
		{Command: create, Receipt: commissioner.Receipt{Entry: "create", RecordedAt: at}},
		{Command: commissioner.RawCommand{League: create.League, Command: commissioner.RulesSet,
			Args: commissioner.RawArgs{Field: "league.startingAssets", Value: json.RawMessage(`"held"`)}, Effective: "now"},
			Receipt: commissioner.Receipt{Entry: "assets", RecordedAt: at.Add(time.Second)}},
	}}
	if err := BackfillFollowing(t.Context(), ledger, create.League); err != nil || ledger.writes != 0 {
		t.Fatalf("native held start required an MFL identity: writes=%d error=%v", ledger.writes, err)
	}
}
