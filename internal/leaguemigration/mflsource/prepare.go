package mflsource

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Prepare checks the read and builds the plan. Anything missing or unreadable is listed in Problems;
// nothing is guessed, and a league with problems gets no plan.
func Prepare(export Export) (leaguemigration.Plan, error) {
	p := preparation{export: export}
	plan := leaguemigration.Plan{}
	rules, problems, notes := Settings(export.Config)
	p.problems = append(p.problems, problems...)
	plan.Rules, plan.Notes = rules, notes
	plan.Seed, plan.Mapping = p.snapshot()
	plan.Owners = p.owners(plan.Mapping)
	plan.Create = p.create(rules)
	if len(export.Rosters) == 0 {
		p.problem("MFL holds no rostered players for this league, so there is nothing to move; " +
			"start a new app league instead")
	}
	if len(p.problems) > 0 {
		return leaguemigration.Plan{}, &leaguemigration.Problems{List: p.problems}
	}
	if err := plan.Seed.Validate(); err != nil {
		return leaguemigration.Plan{}, &leaguemigration.Problems{List: []string{"the league's rosters and picks do not hold together: " + err.Error()}}
	}
	create, err := (commissioner.RawCommand{League: "native.check", Command: commissioner.Create, Args: plan.Create,
		Request: "check", Reason: "check", Effective: "now", Visibility: "league"}).Prepared()
	if err == nil {
		err = create.Validate()
	}
	if err != nil {
		return leaguemigration.Plan{}, &leaguemigration.Problems{List: []string{"the league's settings do not hold together: " + err.Error()}}
	}
	plan.Create = create.Args
	return plan, nil
}

type preparation struct {
	export   Export
	problems []string
}

func (p *preparation) problem(format string, args ...any) {
	p.problems = append(p.problems, fmt.Sprintf(format, args...))
}

func (p *preparation) create(rules leaguerules.RuleSet) commissioner.RawArgs {
	changes := leaguerules.SeedChanges(rules, "", p.export.Season, time.Unix(1, 0).UTC())
	seed := make([]commissioner.SeedValue, 0, len(changes))
	for _, c := range changes {
		body, err := leaguerules.EncodeStored(c.New)
		if err != nil {
			p.problem("the setting %s could not be kept: %v", c.Field, err)
			continue
		}
		seed = append(seed, commissioner.SeedValue{Field: string(c.Field), Value: body})
	}
	name := strings.TrimSpace(p.export.Name)
	if name == "" {
		p.problem("MFL lists the league with no name")
	}
	return commissioner.RawArgs{Name: name, Preset: commissioner.PresetMFLLeague, Source: commissioner.SourceMFLImport,
		Season: p.export.Season, FranchiseCount: len(p.export.Config.Franchises), Seed: seed}
}

// snapshot is the starting assets: franchises in MFL id order, then rosters, adjustments and picks.
func (p *preparation) snapshot() (commissioner.RawNativeSnapshot, map[string]string) {
	e := p.export
	if e.Season < 1920 || e.Season > 9999 {
		p.problem("MFL names no season for the league")
	}
	at := e.ReadAt
	seed := commissioner.RawNativeSnapshot{Season: e.Season, RosterAt: at, DraftAt: at, FutureAt: at,
		Players: []commissioner.NativePlayer{}, Picks: []commissioner.NativePick{}, Adjustments: []commissioner.NativeCharge{}}
	seed.Franchises, seed.Conferences = p.franchises(), conferences(e.Config)
	mapping := map[string]string{}
	for _, f := range seed.Franchises {
		mapping[f.HeldID] = f.ID
	}
	p.players(&seed, mapping)
	p.adjustments(&seed, mapping)
	p.picks(&seed, mapping)
	p.champions(&seed, mapping)
	return seed, mapping
}

func (p *preparation) franchises() []commissioner.NativeFranchise {
	held := slices.Clone(p.export.Config.Franchises)
	slices.SortFunc(held, func(a, b league.Franchise) int { return strings.Compare(a.ID, b.ID) })
	if len(held) < 2 || len(held) > 32 {
		p.problem("MFL lists %d franchises; a league needs between 2 and 32", len(held))
	}
	divisions := map[string]bool{}
	for _, d := range p.export.Config.Divisions {
		divisions[d.ID] = true
	}
	out := make([]commissioner.NativeFranchise, 0, len(held))
	seen := map[string]bool{}
	for i, f := range held {
		switch {
		case strings.TrimSpace(f.ID) == "" || seen[f.ID]:
			p.problem("MFL lists franchise %q more than once or without an id", f.ID)
		case strings.TrimSpace(f.Name) == "":
			p.problem("franchise %s has no name on MFL; name it there, then import again", f.ID)
		case f.Division != "" && !divisions[f.Division]:
			p.problem("franchise %s sits in division %q, which MFL does not list", f.ID, f.Division)
		}
		seen[f.ID] = true
		out = append(out, commissioner.NativeFranchise{ID: fmt.Sprintf("franchise.%02d", i+1), HeldID: f.ID,
			Name: f.Name, Abbreviation: f.Abbrev, Division: f.Division})
	}
	return out
}

// conferences names the conference of each division, by the conference's own name.
func conferences(config league.RawConfig) map[string]string {
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
	if len(out) == 0 {
		return nil
	}
	return out
}

func (p *preparation) franchise(mapping map[string]string, held, what string) (string, bool) {
	id, ok := mapping[held]
	if !ok {
		p.problem("%s names franchise %q, which MFL does not list", what, held)
	}
	return id, ok
}

// playerID keeps an MFL player id exactly, leading zeros included.
func (p *preparation) playerID(raw, what string) (string, bool) {
	id, err := playerid.New(raw)
	if err != nil || id.String() != raw {
		p.problem("%s has the player id %q, which is not an MFL player id", what, raw)
		return "", false
	}
	if len(p.export.Players) > 0 {
		if _, ok := p.export.Players[raw]; !ok {
			p.problem("%s has player %s, who is not in MFL's player list", what, raw)
			return "", false
		}
	}
	return raw, true
}

func (p *preparation) players(seed *commissioner.RawNativeSnapshot, mapping map[string]string) {
	seen := map[string]string{}
	for _, r := range p.export.Rosters {
		what := "the roster of franchise " + r.Franchise
		franchise, ok := p.franchise(mapping, r.Franchise, what)
		id, ok2 := p.playerID(r.Player, what)
		if !ok || !ok2 {
			continue
		}
		if prior, dup := seen[id]; dup {
			p.problem("player %s is listed twice, on franchises %s and %s", id, prior, r.Franchise)
			continue
		}
		seen[id] = r.Franchise
		status, ok := rosterStatus(r.Status)
		if !ok {
			p.problem("player %s on franchise %s has the roster status %q, which the app does not know",
				id, r.Franchise, r.Status)
		}
		salary, ok := p.salary(r)
		year, ok2 := p.contractYear(r)
		if !ok || !ok2 || status == "" {
			continue
		}
		// The contract status is read exactly as the app's own MFL mirror reads it.
		record, err := normalize.Contract(rosters.RawRoster{FranchiseID: r.Franchise, PlayerID: id, Salary: r.Salary,
			ContractYear: r.ContractYear, ContractStatus: r.ContractStatus, ContractInfo: r.ContractInfo, Status: r.Status})
		if err != nil {
			p.problem("player %s on franchise %s could not be read: %v", id, r.Franchise, err)
			continue
		}
		player := commissioner.NativePlayer{Name: p.export.Players[id], ID: id,
			Franchise: franchise, Status: status, Salary: salary, ExpirationYear: year,
			ContractStatus: record.ContractStatus, ContractInfo: r.ContractInfo}
		switch {
		case strings.TrimSpace(r.ContractStatus) == "":
			player.ContractStatus = "" // MFL holds no contract status; none is invented
		case r.ContractStatus != string(record.ContractStatus):
			player.HeldContractStatus = r.ContractStatus
		}
		seed.Players = append(seed.Players, player)
	}
}

func rosterStatus(raw string) (domain.RosterStatus, bool) {
	switch strings.TrimSpace(raw) {
	case "ROSTER":
		return domain.RosterActive, true
	case "INJURED_RESERVE", "IR": // MFL's live export says INJURED_RESERVE; older exports said IR
		return domain.RosterIR, true
	case "TAXI_SQUAD":
		return domain.RosterTaxi, true
	}
	return "", false
}

func (p *preparation) salary(r Rostered) (domain.Money, bool) {
	if strings.TrimSpace(r.Salary) == "" {
		return 0, true
	}
	salary, err := domain.ParseMoneyMillions(r.Salary)
	if err != nil {
		p.problem("player %s on franchise %s has the salary %q, which is not an amount", r.Player, r.Franchise, r.Salary)
		return 0, false
	}
	return salary, true
}

func (p *preparation) contractYear(r Rostered) (int, bool) {
	if strings.TrimSpace(r.ContractYear) == "" {
		return 0, true
	}
	year, err := strconv.Atoi(strings.TrimSpace(r.ContractYear))
	if err != nil || year < 0 || year > 9999 {
		p.problem("player %s on franchise %s has the contract year %q, which is not a year",
			r.Player, r.Franchise, r.ContractYear)
		return 0, false
	}
	return year, true
}

func (p *preparation) adjustments(seed *commissioner.RawNativeSnapshot, mapping map[string]string) {
	seen := map[string]bool{}
	for _, a := range p.export.Adjustments {
		franchise, ok := p.franchise(mapping, a.Franchise, "a salary adjustment")
		amount, err := domain.ParseSignedMoneyMillions(a.Amount)
		switch {
		case !ok:
			continue
		case err != nil:
			p.problem("salary adjustment %s reads %q, which is not an amount", a.ID, a.Amount)
			continue
		case strings.TrimSpace(a.ID) == "" || seen[a.ID]:
			p.problem("a salary adjustment for franchise %s has no id of its own", a.Franchise)
			continue
		}
		seen[a.ID] = true
		seed.Adjustments = append(seed.Adjustments, commissioner.NativeCharge{ID: a.ID, Franchise: franchise,
			Amount: amount, Description: a.Description})
	}
}

func (p *preparation) picks(seed *commissioner.RawNativeSnapshot, mapping map[string]string) {
	for _, d := range p.export.Draft {
		owner, ok := p.franchise(mapping, d.Franchise, fmt.Sprintf("draft pick %d.%02d", d.Round, d.Slot))
		if !ok {
			continue
		}
		pick := commissioner.NativePick{Season: seed.Season, Round: d.Round, Slot: d.Slot, Owner: owner, Skipped: d.Skipped}
		if d.Player != nil {
			pick.Player = d.Player.String()
		}
		seed.Picks = append(seed.Picks, pick)
	}
	for _, f := range p.export.Future {
		what := fmt.Sprintf("the %d round %d pick", f.Year, f.Round)
		owner, ok := p.franchise(mapping, f.Owner, what)
		original, ok2 := p.franchise(mapping, f.Original, what)
		if !ok || !ok2 {
			continue
		}
		seed.Picks = append(seed.Picks, commissioner.NativePick{Season: f.Year, Round: f.Round, Original: original, Owner: owner})
	}
}

func (p *preparation) champions(seed *commissioner.RawNativeSnapshot, mapping map[string]string) {
	for _, c := range p.export.Champions {
		franchise, ok := p.franchise(mapping, c.Franchise, fmt.Sprintf("the %d champion", c.Season))
		if ok {
			seed.Champions = append(seed.Champions, commissioner.NativeChampion{Season: c.Season, Franchise: franchise})
		}
	}
}

// owners are MFL's owners under their app franchises, in franchise order.
func (p *preparation) owners(mapping map[string]string) []leaguemigration.Owner {
	out := make([]leaguemigration.Owner, 0, len(p.export.Owners))
	for _, o := range p.export.Owners {
		if id, ok := mapping[o.Franchise]; ok {
			out = append(out, leaguemigration.Owner{Franchise: id, Name: strings.TrimSpace(o.Name), Email: strings.TrimSpace(o.Email)})
		}
	}
	slices.SortFunc(out, func(a, b leaguemigration.Owner) int { return strings.Compare(a.Franchise, b.Franchise) })
	return out
}
