package commissioner

import (
	"fmt"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// RuleLine is one league rule in force, in the commissioner's own words.
type RuleLine struct {
	Label string `json:"label"`
	Words string `json:"words"`
}

// nativeBase is what every native legality reading starts from: the recorded starting rosters, the
// native directory and the rules in force now, all read from the one ledger.
type nativeBase struct {
	assets    NativeAssets
	directory NativeDirectory
	rules     leaguerules.RuleSet
}

func readNativeBase(league string, records []Record, now time.Time) (nativeBase, error) {
	state, err := GovernanceState(records)
	if err != nil {
		return nativeBase{}, err
	}
	directory, err := nativeDirectory(league, records, state, now)
	if err != nil {
		return nativeBase{}, err
	}
	assets, err := NativeAssetsReading(league, records)
	if err != nil {
		return nativeBase{}, err
	}
	rules, err := RulesNow(records, now)
	if err != nil {
		return nativeBase{}, err
	}
	return nativeBase{assets: assets, directory: directory, rules: rules}, nil
}

// playersOf groups the recorded players by franchise.
func (b nativeBase) playersOf() map[string][]nativecheck.RosterPlayer {
	byFranchise := map[string][]nativecheck.RosterPlayer{}
	if !b.assets.Seeded {
		return byFranchise
	}
	for _, p := range b.assets.Snapshot.Players {
		byFranchise[p.Franchise] = append(byFranchise[p.Franchise], nativecheck.RosterPlayer{ID: p.ID, Status: p.Status})
	}
	return byFranchise
}

func (b nativeBase) source() string {
	if !b.assets.Seeded {
		return "App league; no starting snapshot recorded, so there are no rosters to judge"
	}
	return "App league; judged from the recorded rosters and this league's own rules, never from MFL"
}

// ruleLines reads the named catalogue fields from the rules in force.
func ruleLines(rules leaguerules.RuleSet, paths ...string) []RuleLine {
	fields, values := leaguerules.Catalogue(), leaguerules.ReadSettings(rules)
	lines := make([]RuleLine, 0, len(paths))
	for _, path := range paths {
		for i, f := range fields {
			if string(f.Path) == path {
				lines = append(lines, RuleLine{Label: f.Label, Words: values[i].Words})
			}
		}
	}
	return lines
}

// NativeRosterFranchise is one franchise's roster verdict beside its name.
type NativeRosterFranchise struct {
	nativecheck.RosterVerdict
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
}

// NativeRosterCheck judges every native franchise's roster against the league's size and position limits.
type NativeRosterCheck struct {
	League       string                  `json:"league"`
	Name         string                  `json:"name"`
	Revision     int64                   `json:"revision"`
	RosterSeason int                     `json:"rosterSeason"`
	RulesSeason  int                     `json:"rulesSeason"`
	Seeded       bool                    `json:"seeded"`
	Source       string                  `json:"source"`
	Rules        []RuleLine              `json:"rules"`
	Franchises   []NativeRosterFranchise `json:"franchises"`
}

// NativeRosterCheckReading is public: every owner may read how each roster stands.
func NativeRosterCheckReading(league string, records []Record, now time.Time, facts nativecheck.Facts) (NativeRosterCheck, error) {
	base, err := readNativeBase(league, records, now)
	if err != nil {
		return NativeRosterCheck{}, err
	}
	out := NativeRosterCheck{League: league, Name: Replay(records).Name, Revision: int64(len(records)),
		RulesSeason: base.rules.Season, Seeded: base.assets.Seeded, Source: base.source(),
		Rules:      ruleLines(base.rules, "roster.limits", "roster.reserveExcluded", "roster.positionBounds"),
		Franchises: []NativeRosterFranchise{}}
	if !base.assets.Seeded {
		return out, nil
	}
	out.RosterSeason = base.assets.Snapshot.Season
	players := base.playersOf()
	for _, seat := range base.directory.Franchises {
		out.Franchises = append(out.Franchises, NativeRosterFranchise{
			RosterVerdict: nativecheck.CheckRoster(base.rules.Roster, seat.Franchise, players[seat.Franchise], facts),
			Name:          seat.Name, Abbreviation: seat.Abbreviation})
	}
	return out, nil
}

// HouseRule is one piece of text the league has written down.
type HouseRule struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// NativeHouseRules is the league's own written rules, read-only text beside the settings.
type NativeHouseRules struct {
	League   string      `json:"league"`
	Name     string      `json:"name"`
	Revision int64       `json:"revision"`
	Season   int         `json:"season"`
	Source   string      `json:"source"`
	Words    string      `json:"words"`
	Rules    []HouseRule `json:"rules"`
}

// NativeHouseRulesReading is public: every owner may read what else the league has ruled.
func NativeHouseRulesReading(league string, records []Record, now time.Time) (NativeHouseRules, error) {
	if len(records) == 0 {
		return NativeHouseRules{}, fmt.Errorf("this app league has not been created")
	}
	rules, err := RulesNow(records, now)
	if err != nil {
		return NativeHouseRules{}, err
	}
	out := NativeHouseRules{League: league, Name: Replay(records).Name, Revision: int64(len(records)),
		Season: rules.Season, Source: "App league; written by this league's commissioner, never read from MFL",
		Rules: []HouseRule{}}
	texts := rules.House.Texts
	switch {
	case texts.State != leaguerules.IsKnown:
		out.Words = "Needs a ruling: house rules"
	case len(texts.Value) == 0:
		out.Words = "No house rules are written"
	case len(texts.Value) == 1:
		out.Words = "1 house rule"
	default:
		out.Words = fmt.Sprintf("%d house rules", len(texts.Value))
	}
	for id, text := range texts.Value {
		out.Rules = append(out.Rules, HouseRule{ID: id, Text: text})
	}
	sort.Slice(out.Rules, func(i, j int) bool { return out.Rules[i].ID < out.Rules[j].ID })
	return out, nil
}
