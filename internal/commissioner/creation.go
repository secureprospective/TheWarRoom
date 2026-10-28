package commissioner

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	legacyTeams = 32
	// A head-to-head league needs two franchises to play a game.
	minFranchises = 2
	// maxFranchises is the product's cap: a league has at most 32 franchises, whatever its settings.
	maxFranchises = 32
	// firstNFLSeason is the NFL's first season; a league cannot start before it.
	firstNFLSeason = 1920
	lastSeason     = 9999
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

func presetSeed(preset string, count, season int) ([]SeedValue, error) {
	p, _ := presetByID(preset)
	if p.rules == nil {
		return nil, nil
	}
	changes := leaguerules.SeedChanges(p.rules(count), "preset", season, time.Unix(1, 0).UTC())
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
	p, _ := presetByID(a.Preset)
	if a.Source == "" {
		a.Source = cmp.Or(p.source, SourceNative)
	}
	if a.FranchiseCount == 0 {
		a.FranchiseCount = p.defaultTeams
	}
	if err := a.validatePreset(); err != nil {
		return r, err
	}
	if a.Season == 0 {
		return r, fmt.Errorf("choose the NFL season this league starts in")
	}
	ids := franchiseIDs(a.FranchiseCount)
	if p.imported {
		// The MFL league's own values: Go read them at import, and validateCreation decodes each one.
		if len(a.Franchises) != 0 && !slices.Equal(a.Franchises, ids) {
			return r, fmt.Errorf("the app lists a new league's franchises itself")
		}
		a.Franchises = ids
		r.Args = a
		return r, nil
	}
	seed, err := presetSeed(a.Preset, a.FranchiseCount, a.Season)
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
	if _, ok := presetByID(a.Preset); !ok {
		return unknownPresetError()
	}
	if a.FranchiseCount < minFranchises || a.FranchiseCount > maxFranchises {
		return fmt.Errorf("choose between %d and %d franchises", minFranchises, maxFranchises)
	}
	return nil
}

// validateCreationSeason accepts 0, a league recorded before seasons were numbered; Prepared refuses it
// for a new league.
func (a RawArgs) validateCreationSeason() error {
	if a.Season != 0 && (a.Season < firstNFLSeason || a.Season > lastSeason) {
		return fmt.Errorf("choose the NFL season this league starts in")
	}
	return nil
}

func (a RawArgs) validateCreationIdentity() error {
	if a.Source != "" && a.Source != SourceNative && a.Source != SourceMFLImport {
		return fmt.Errorf("choose a native league or an MFL import")
	}
	if strings.TrimSpace(a.Name) == "" || a.Entry != "" || a.People != nil {
		return fmt.Errorf("give the new league a name")
	}
	return nil
}

func (a RawArgs) validateCreation() error {
	if err := a.validateCreationIdentity(); err != nil {
		return err
	}
	if err := a.validateCreationSeason(); err != nil {
		return err
	}
	if err := a.validatePreset(); err != nil {
		return err
	}
	if !slices.Equal(a.Franchises, franchiseIDs(a.FranchiseCount)) {
		return fmt.Errorf("the new league's franchises do not match its size")
	}
	if p, _ := presetByID(a.Preset); p.imported && a.Source != SourceMFLImport {
		return fmt.Errorf("a league with its own MFL settings is imported from MFL")
	} else if !p.seeds() && len(a.Seed) != 0 {
		return fmt.Errorf("an empty league starts with every rule waiting for a ruling")
	} else if p.seeds() && len(a.Seed) == 0 {
		return fmt.Errorf("a preset league needs its starting values")
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

func seedChanges(r Record, season int) ([]leaguerules.Change, error) {
	changes := make([]leaguerules.Change, 0, len(r.Command.Args.Seed))
	for _, s := range r.Command.Args.Seed {
		setting, err := leaguerules.DecodeStored(leaguerules.FieldPath(s.Field), s.Value)
		if err != nil {
			return nil, fmt.Errorf("read preset value: %w", err)
		}
		changes = append(changes, leaguerules.Change{LeagueID: r.Command.League, Field: leaguerules.FieldPath(s.Field),
			New: setting, Effective: leaguerules.Effective{Kind: leaguerules.Now},
			RecordedAt: r.Receipt.RecordedAt, Season: season})
	}
	return leaguerules.UpgradeSeed(changes), nil
}

func validRestorePreset(preset string) bool {
	_, ok := presetByID(preset)
	return preset == "" || ok
}
