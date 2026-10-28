package history

import (
	"cmp"
	"strconv"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// assertAppHoldsSource checks the native league against the raw MFL export directly, never through the
// import's own translation, so a field the import itself dropped fails here by name.
func assertAppHoldsSource(t *testing.T, export leaguemigration.Export, app leaguemigration.AppLeague) {
	t.Helper()
	held := map[string]string{}
	for _, f := range app.Snapshot.Franchises {
		held[f.HeldID] = f.ID
	}
	seats := map[string][3]string{}
	for _, s := range app.Directory.Franchises {
		seats[s.Franchise] = [3]string{s.Name, s.Abbreviation, ""}
	}
	for _, f := range app.Snapshot.Franchises {
		seat := seats[f.ID]
		seat[2] = f.Division
		seats[f.ID] = seat
	}
	for _, f := range export.Config.Franchises {
		if got := seats[held[f.ID]]; got != [3]string{f.Name, f.Abbrev, f.Division} {
			t.Errorf("franchise %s name/abbreviation/division: MFL %q %q %q, app %q", f.ID, f.Name, f.Abbrev, f.Division, got)
		}
	}
	players := map[string]int{}
	for i, p := range app.Snapshot.Players {
		players[p.ID] = i
	}
	if len(app.Snapshot.Players) != len(export.Rosters) {
		t.Errorf("rostered players: MFL %d, app %d", len(export.Rosters), len(app.Snapshot.Players))
	}
	for _, r := range export.Rosters {
		i, ok := players[r.Player]
		if !ok {
			t.Errorf("player %s is not on the app roster", r.Player)
			continue
		}
		p := app.Snapshot.Players[i]
		salary, _ := domain.ParseMoneyMillions(r.Salary)
		year, _ := strconv.Atoi(r.ContractYear)
		status := map[string]domain.RosterStatus{"ROSTER": domain.RosterActive, "INJURED_RESERVE": domain.RosterIR,
			"IR": domain.RosterIR, "TAXI_SQUAD": domain.RosterTaxi}[r.Status]
		for field, same := range map[string]bool{
			"franchise": p.Franchise == held[r.Franchise], "status": p.Status == status, "salary": p.Salary == salary,
			"contract year": p.ExpirationYear == year, "contract info": p.ContractInfo == r.ContractInfo,
			"contract status": cmp.Or(p.HeldContractStatus, string(p.ContractStatus)) == r.ContractStatus,
			"name":            p.Name == export.Players[r.Player],
		} {
			if !same {
				t.Errorf("player %s %s lost: MFL %+v, app %+v", r.Player, field, r, p)
			}
		}
	}
	assertPicksAndChargesHoldSource(t, export, app, held)
	assertSettingsHoldSource(t, export, app.Rules)
}

func assertPicksAndChargesHoldSource(t *testing.T, export leaguemigration.Export, app leaguemigration.AppLeague,
	held map[string]string) {
	t.Helper()
	picks := map[string]bool{}
	for _, p := range app.Snapshot.Picks {
		picks[strings.Join([]string{strconv.Itoa(p.Season), strconv.Itoa(p.Round), strconv.Itoa(p.Slot), p.Original,
			p.Owner, p.Player, strconv.FormatBool(p.Skipped)}, "/")] = true
	}
	for _, d := range export.Draft {
		player := ""
		if d.Player != nil {
			player = d.Player.String()
		}
		if !picks[strings.Join([]string{strconv.Itoa(export.Season), strconv.Itoa(d.Round), strconv.Itoa(d.Slot), "",
			held[d.Franchise], player, strconv.FormatBool(d.Skipped)}, "/")] {
			t.Errorf("draft pick %d.%02d lost", d.Round, d.Slot)
		}
	}
	for _, f := range export.Future {
		if !picks[strings.Join([]string{strconv.Itoa(f.Year), strconv.Itoa(f.Round), "0", held[f.Original],
			held[f.Owner], "", "false"}, "/")] {
			t.Errorf("the %d round %d pick of %s (owner, original, year, round) lost", f.Year, f.Round, f.Original)
		}
	}
	if len(app.Snapshot.Picks) != len(export.Draft)+len(export.Future) {
		t.Errorf("picks: MFL %d, app %d", len(export.Draft)+len(export.Future), len(app.Snapshot.Picks))
	}
	charges := map[string]bool{}
	for _, c := range app.Snapshot.Adjustments {
		charges[c.ID+"/"+c.Franchise+"/"+strconv.FormatInt(c.Amount.Cents(), 10)+"/"+c.Description] = true
	}
	for _, a := range export.Adjustments {
		amount, _ := domain.ParseSignedMoneyMillions(a.Amount)
		if !charges[a.ID+"/"+held[a.Franchise]+"/"+strconv.FormatInt(amount.Cents(), 10)+"/"+a.Description] {
			t.Errorf("salary adjustment %s lost", a.ID)
		}
	}
	if len(app.Snapshot.Champions) != len(export.Champions) {
		t.Errorf("history: MFL %d champions, app %d", len(export.Champions), len(app.Snapshot.Champions))
	}
	for i, c := range export.Champions {
		if i < len(app.Snapshot.Champions) && (app.Snapshot.Champions[i].Season != c.Season ||
			app.Snapshot.Champions[i].Franchise != held[c.Franchise]) {
			t.Errorf("the %d champion lost", c.Season)
		}
	}
}

func assertSettingsHoldSource(t *testing.T, export leaguemigration.Export, r leaguerules.RuleSet) {
	t.Helper()
	c := export.Config
	number := func(raw string) int { n, _ := strconv.Atoi(raw); return n }
	checks := map[string]bool{
		"IR spots":          c.InjuredReserve == "" || r.IR.Slots.Value == number(c.InjuredReserve),
		"taxi spots":        c.TaxiSquad == "" || r.Taxi.Slots.Value == number(c.TaxiSquad),
		"roster size":       c.RosterSize == "" || r.Roster.Limits.Value.Max.Value == number(c.RosterSize),
		"starters":          c.Starters.Count == "" || r.Lineup.Groups.Value[leaguerules.Total].Max.Value == number(c.Starters.Count),
		"regular weeks":     r.Schedule.RegularWeeks.Value == number(c.LastRegularSeasonWeek)-number(c.StartWeek)+1,
		"starting position": len(r.Lineup.Positions.Value) == len(c.Starters.Positions),
	}
	if c.UsesSalaries == "1" {
		amount, _ := domain.ParseMoneyMillions(c.SalaryCapAmount)
		checks["salary cap"] = int64(r.Cap.Amount.Value) == amount.Cents()
		checks["IR salary share"] = int(r.Cap.IRShare.Value) == number(c.IncludeIRWithSalary)*100
	}
	for _, p := range c.Starters.Positions {
		b := r.Lineup.Positions.Value[leaguerules.Position(p.Name)]
		low, high, _ := strings.Cut(p.Limit, "-")
		if high == "" {
			high = low
		}
		checks["starting "+p.Name] = b.Min.Value == number(low) && b.Max.Value == number(high)
	}
	raw := 0
	for _, b := range c.ScoringRules {
		raw += len(b.Rules)
		for _, rule := range b.Rules {
			id := leaguerules.StatID("mfl_" + rule.Event)
			if def, ok := leaguerules.StatByMFL(rule.Event); ok {
				id = def.ID
			}
			found := false
			for _, kept := range r.Scoring.Rules.Value[id] {
				found = found || strings.Join(positions(kept.Positions), "|") == b.Positions
			}
			checks["scoring "+rule.Event+" for "+b.Positions] = found
		}
	}
	kept := 0
	for _, rules := range r.Scoring.Rules.Value {
		kept += len(rules)
	}
	checks["every scoring rule"] = kept == raw
	for name, ok := range checks {
		if !ok {
			t.Errorf("setting %s lost", name)
		}
	}
}

func positions(in []leaguerules.Position) []string {
	out := make([]string, len(in))
	for i, p := range in {
		out[i] = string(p)
	}
	return out
}
