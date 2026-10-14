package commissioner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	PresetLegacy  = "legacy_nfl"
	PresetEmpty   = "empty"
	legacyTeams   = 32
	minFranchises = 2
	maxFranchises = 32
	// ledgerSeason is the one season the ledger holds until the rollover Act numbers seasons.
	ledgerSeason = 1
)

// SeedValue is one preset value, stored whole (state and citation) so replay never depends on later code.
type SeedValue struct {
	Field string          `json:"field"`
	Value json.RawMessage `json:"value"`
}

func franchiseIDs(count int) []string {
	ids := make([]string, count)
	for i := range ids {
		ids[i] = fmt.Sprintf("franchise.%02d", i+1)
	}
	return ids
}

func presetSeed(preset string) ([]SeedValue, error) {
	if preset != PresetLegacy {
		return nil, nil
	}
	changes := leaguerules.SeedChanges(leaguerules.LegacyNFL(), "preset", ledgerSeason, time.Unix(1, 0).UTC())
	seed := make([]SeedValue, len(changes))
	for i, c := range changes {
		body, err := leaguerules.EncodeStored(c.New)
		if err != nil {
			return nil, fmt.Errorf("preset %s: %w", c.Field, err)
		}
		seed[i] = SeedValue{Field: string(c.Field), Value: body}
	}
	return seed, nil
}

func sameSeed(a, b []SeedValue) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		var left, right bytes.Buffer
		if a[i].Field != b[i].Field || json.Compact(&left, a[i].Value) != nil ||
			json.Compact(&right, b[i].Value) != nil || !bytes.Equal(left.Bytes(), right.Bytes()) {
			return false
		}
	}
	return true
}

// Prepared fills what Go owns on a new league: its franchise list and its preset values.
// A caller may not supply either; a retry that repeats Go's own expansion is accepted.
func (r RawCommand) Prepared() (RawCommand, error) {
	if r.Command != Create {
		return r, nil
	}
	a := r.Args
	if a.Preset == PresetLegacy && a.FranchiseCount == 0 {
		a.FranchiseCount = legacyTeams
	}
	if err := a.validatePreset(); err != nil {
		return r, err
	}
	ids := franchiseIDs(a.FranchiseCount)
	seed, err := presetSeed(a.Preset)
	if err != nil {
		return r, err
	}
	if (len(a.Franchises) != 0 && !slices.Equal(a.Franchises, ids)) || (len(a.Seed) != 0 && !sameSeed(a.Seed, seed)) {
		return r, fmt.Errorf("the app lists a new league's franchises and preset values itself")
	}
	a.Franchises, a.Seed = ids, seed
	r.Args = a
	return r, nil
}

func (a RawArgs) validatePreset() error {
	switch a.Preset {
	case PresetLegacy:
		if a.FranchiseCount != legacyTeams {
			return fmt.Errorf("the Legacy NFL preset is for 32 franchises; start an empty league for another size")
		}
	case PresetEmpty:
		if a.FranchiseCount < minFranchises || a.FranchiseCount > maxFranchises {
			return fmt.Errorf("choose between 2 and 32 franchises")
		}
	default:
		return fmt.Errorf("start from the Legacy NFL preset or an empty league")
	}
	return nil
}

func (a RawArgs) validateCreation() error {
	if strings.TrimSpace(a.Name) == "" || a.Entry != "" || a.People != nil {
		return fmt.Errorf("give the new league a name")
	}
	if err := a.validatePreset(); err != nil {
		return err
	}
	if !slices.Equal(a.Franchises, franchiseIDs(a.FranchiseCount)) {
		return fmt.Errorf("the new league's franchises do not match its size")
	}
	if a.Preset == PresetEmpty && len(a.Seed) != 0 {
		return fmt.Errorf("an empty league starts with every rule waiting for a ruling")
	}
	if a.Preset == PresetLegacy && len(a.Seed) == 0 {
		return fmt.Errorf("the Legacy NFL preset needs its starting values")
	}
	seen := make(map[string]bool, len(a.Seed))
	for _, s := range a.Seed {
		if seen[s.Field] {
			return fmt.Errorf("a preset value was listed twice")
		}
		seen[s.Field] = true
		if _, err := leaguerules.DecodeStored(leaguerules.FieldPath(s.Field), s.Value); err != nil {
			return fmt.Errorf("preset value: %w", err)
		}
	}
	return nil
}

// creationEvents opens the league for its creator and names each franchise until CM-018 renames it.
func creationEvents(r Record) []governance.Event {
	ids := make([]governance.FranchiseID, len(r.Command.Args.Franchises))
	events := make([]governance.Event, 0, len(ids)+1)
	for i, id := range r.Command.Args.Franchises {
		ids[i] = governance.FranchiseID(id)
	}
	events = append(events, governance.LeagueCreated{Creator: governance.Principal(r.Principal), Franchises: ids})
	for i, id := range ids {
		events = append(events, governance.FranchiseIdentitySet{Franchise: id,
			Name: fmt.Sprintf("Franchise %d", i+1), Abbreviation: fmt.Sprintf("F%d", i+1)})
	}
	return events
}

func seedChanges(r Record) ([]leaguerules.Change, error) {
	changes := make([]leaguerules.Change, 0, len(r.Command.Args.Seed))
	for _, s := range r.Command.Args.Seed {
		setting, err := leaguerules.DecodeStored(leaguerules.FieldPath(s.Field), s.Value)
		if err != nil {
			return nil, fmt.Errorf("read preset value: %w", err)
		}
		changes = append(changes, leaguerules.Change{LeagueID: r.Command.League, Field: leaguerules.FieldPath(s.Field),
			New: setting, Effective: leaguerules.Effective{Kind: leaguerules.Now},
			RecordedAt: r.Receipt.RecordedAt, Season: ledgerSeason})
	}
	return changes, nil
}
