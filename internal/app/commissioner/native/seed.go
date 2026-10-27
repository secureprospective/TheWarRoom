package native

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Conferences names the conference of each held division, by the conference's own name.
func Conferences(config league.RawConfig) map[string]string {
	names := map[string]string{}
	for _, c := range config.Conferences {
		names[c.ID] = cmp.Or(strings.TrimSpace(c.Name), c.ID)
	}
	out := map[string]string{}
	for _, d := range config.Divisions {
		if conference := cmp.Or(names[d.Conference], d.Conference); conference != "" {
			out[d.ID] = conference
		}
	}
	return out
}

// Franchises numbers the held franchises in id order and maps each held id to its native one.
func Franchises(held []league.Franchise) ([]commissioner.NativeFranchise, map[string]string) {
	held = slices.Clone(held)
	slices.SortFunc(held, func(a, b league.Franchise) int { return strings.Compare(a.ID, b.ID) })
	out := make([]commissioner.NativeFranchise, 0, len(held))
	mapping := map[string]string{}
	for i, f := range held {
		id := fmt.Sprintf("franchise.%02d", i+1)
		mapping[f.ID] = id
		out = append(out, commissioner.NativeFranchise{ID: id, HeldID: f.ID, Name: f.Name,
			Abbreviation: f.Abbrev, Division: f.Division})
	}
	return out, mapping
}

// SeedPlayers keeps each rostered player's name from the held player directory, so a plan can name them.
func SeedPlayers(seed *commissioner.RawNativeSnapshot, snap state.MirrorSnapshot, mapping map[string]string,
	directory []snapshot.Player) {
	names := make(map[string]string, len(directory))
	for _, p := range directory {
		names[p.ID.String()] = p.Name
	}
	for _, p := range snap.Players {
		seed.Players = append(seed.Players, commissioner.NativePlayer{Name: names[p.MFLID], ID: p.MFLID,
			Franchise: mapping[p.FranchiseID],
			Status:    p.RosterStatus, Salary: p.Salary, ExpirationYear: p.ContractYear,
			ContractStatus: p.ContractStatus, ContractInfo: p.ContractInfo})
	}
	for _, c := range snap.Adjustments {
		seed.Adjustments = append(seed.Adjustments, commissioner.NativeCharge{ID: c.ID, Franchise: mapping[c.FranchiseID],
			Amount: c.Amount, Description: c.Description})
	}
}

// SeedPicks records the held draft results and future picks under their native owners.
func SeedPicks(seed *commissioner.RawNativeSnapshot, results []draftreports.Pick,
	future []draftreports.FuturePick, mapping map[string]string) {
	for _, p := range results {
		pick := commissioner.NativePick{Season: seed.Season, Round: p.Round, Slot: p.Slot,
			Owner: mapping[p.Franchise], Skipped: p.Skipped}
		if p.Player != nil {
			pick.Player = p.Player.String()
		}
		seed.Picks = append(seed.Picks, pick)
	}
	for _, p := range future {
		seed.Picks = append(seed.Picks, commissioner.NativePick{Season: p.Year, Round: p.Round,
			Original: mapping[p.Original], Owner: mapping[p.Owner]})
	}
}

type SeedSource struct {
	Available func() bool
	Roster    func() (state.MirrorSnapshot, string)
	Season    func() int
	Draft     func() (season.Feed[[]draftreports.Pick], season.Feed[[]draftreports.FuturePick])
	Config    func() league.RawConfig
	Directory func() []snapshot.Player
}

func CaptureSeed(source SeedSource) (commissioner.RawNativeSnapshot, error) {
	if !source.Available() {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("hold the league rosters and settings before seeding")
	}
	snap, at := source.Roster()
	if snap.Season != source.Season() {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("the held rosters and draft reports must be from the same season")
	}
	rosterAt, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("read held roster date: %w", err)
	}
	results, future := source.Draft()
	seed := commissioner.RawNativeSnapshot{Season: snap.Season, RosterAt: rosterAt, DraftAt: results.FetchedAt,
		FutureAt: future.FetchedAt, Players: []commissioner.NativePlayer{}, Picks: []commissioner.NativePick{}, Adjustments: []commissioner.NativeCharge{}}
	config := source.Config()
	franchises, mapping := Franchises(config.Franchises)
	seed.Franchises, seed.Conferences = franchises, Conferences(config)
	directory := source.Directory()
	SeedPlayers(&seed, snap, mapping, directory)
	SeedPicks(&seed, results.Value, future.Value, mapping)
	if err := seed.Validate(); err != nil {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("held starting snapshot: %w", err)
	}
	return seed, nil
}
