package leaguemigration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Review is the pre-switch comparison of the app's league against MFL, worded per franchise. Digest
// names exactly what this review said, so the cutover can require that its reader saw it.
type Review struct {
	League      string           `json:"league"`
	Summary     string           `json:"summary"`
	Differences int              `json:"differences"`
	LeagueWide  []string         `json:"leagueWide"`
	Franchises  []FranchiseCheck `json:"franchises"`
	Notes       []string         `json:"notes"`
	CutOver     bool             `json:"cutOver"`
	Digest      string           `json:"digest"`
}

// FranchiseCheck is one franchise's lines: "Roster matches", "2 contracts differ: ...".
type FranchiseCheck struct {
	Franchise string   `json:"franchise"`
	MFLID     string   `json:"mflId"`
	Name      string   `json:"name"`
	Lines     []string `json:"lines"`
	Matches   bool     `json:"matches"`
}

// Difference is one field that differs, named by where it lives. The round-trip proof fails on any.
type Difference struct {
	Franchise string
	Field     string
	MFL, App  string
}

// AppLeague is the app's league as the ledger reads it now.
type AppLeague struct {
	League    string
	Name      string
	Snapshot  commissioner.RawNativeSnapshot
	Directory commissioner.NativeDirectory
	Rules     leaguerules.RuleSet
	CutOver   bool
}

// ReadApp reads the app league back from its ledger, exactly as the app's own screens read it.
func ReadApp(league string, records []commissioner.Record, now time.Time) (AppLeague, error) {
	assets, err := commissioner.NativeAssetsReading(league, records, now)
	if err != nil {
		return AppLeague{}, fmt.Errorf("read the app league's assets: %w", err)
	}
	if assets.Snapshot == nil {
		return AppLeague{}, fmt.Errorf("the app league has no rosters recorded yet")
	}
	directory, err := commissioner.NativeDirectoryReading(league, records, now)
	if err != nil {
		return AppLeague{}, fmt.Errorf("read the app league's franchises: %w", err)
	}
	rules, err := commissioner.RulesNow(records, now)
	if err != nil {
		return AppLeague{}, fmt.Errorf("read the app league's rules: %w", err)
	}
	return AppLeague{League: league, Name: directory.Name, Snapshot: *assets.Snapshot, Directory: directory,
		Rules: rules, CutOver: commissioner.CutOver(records)}, nil
}

// Compare lists every field where the app league differs from the MFL league as a plan reads it.
func Compare(plan Plan, app AppLeague) []Difference {
	c := comparison{plan: plan, app: app}
	c.league()
	c.franchises()
	c.players()
	c.charges()
	c.picks()
	c.settings()
	return c.out
}

type comparison struct {
	plan Plan
	app  AppLeague
	out  []Difference
}

func (c *comparison) add(franchise, field string, mfl, app any) {
	c.out = append(c.out, Difference{Franchise: franchise, Field: field, MFL: fmt.Sprint(mfl), App: fmt.Sprint(app)})
}

func (c *comparison) league() {
	m, a := c.plan.Seed, c.app.Snapshot
	if c.plan.Create.Name != c.app.Name {
		c.add("", "league name", c.plan.Create.Name, c.app.Name)
	}
	if m.Season != a.Season {
		c.add("", "season", m.Season, a.Season)
	}
	if !maps.Equal(m.Conferences, a.Conferences) {
		c.add("", "conferences", m.Conferences, a.Conferences)
	}
	if !slices.Equal(m.Champions, a.Champions) {
		c.add("", "past champions", m.Champions, a.Champions)
	}
}

func (c *comparison) franchises() {
	seats := map[string]commissioner.NativeDirectorySeat{}
	for _, s := range c.app.Directory.Franchises {
		seats[s.Franchise] = s
	}
	held := map[string]commissioner.NativeFranchise{}
	for _, f := range c.app.Snapshot.Franchises {
		held[f.ID] = f
	}
	if len(c.app.Snapshot.Franchises) != len(c.plan.Seed.Franchises) {
		c.add("", "franchise count", len(c.plan.Seed.Franchises), len(c.app.Snapshot.Franchises))
	}
	for _, f := range c.plan.Seed.Franchises {
		seat, ok := seats[f.ID]
		if !ok || held[f.ID].HeldID != f.HeldID {
			c.add(f.ID, "franchise", f.HeldID, "missing")
			continue
		}
		if seat.Name != f.Name {
			c.add(f.ID, "name", f.Name, seat.Name)
		}
		if seat.Abbreviation != f.Abbreviation {
			c.add(f.ID, "abbreviation", f.Abbreviation, seat.Abbreviation)
		}
		if held[f.ID].Division != f.Division {
			c.add(f.ID, "division", f.Division, held[f.ID].Division)
		}
	}
}

func (c *comparison) players() {
	app := map[string]commissioner.NativePlayer{}
	for _, p := range c.app.Snapshot.Players {
		app[p.ID] = p
	}
	for _, m := range c.plan.Seed.Players {
		a, ok := app[m.ID]
		delete(app, m.ID)
		if !ok {
			c.add(m.Franchise, "player "+m.ID+" rostered", "on the roster", "missing")
			continue
		}
		field := "player " + m.ID + " "
		for _, d := range []struct {
			name     string
			mfl, app any
		}{{"franchise", m.Franchise, a.Franchise}, {"roster status", m.Status, a.Status},
			{"salary", m.Salary, a.Salary}, {"contract year", m.ExpirationYear, a.ExpirationYear},
			{"contract status", m.ContractStatus, a.ContractStatus}, {"contract", m.ContractInfo, a.ContractInfo},
			{"MFL contract status", m.HeldContractStatus, a.HeldContractStatus}} {
			if d.mfl != d.app {
				c.add(m.Franchise, field+d.name, d.mfl, d.app)
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(app)) {
		c.add(app[id].Franchise, "player "+id+" rostered", "not on MFL", "on the roster")
	}
}

func (c *comparison) charges() {
	app := map[string]commissioner.NativeCharge{}
	for _, a := range c.app.Snapshot.Adjustments {
		app[a.ID] = a
	}
	for _, m := range c.plan.Seed.Adjustments {
		a, ok := app[m.ID]
		delete(app, m.ID)
		switch {
		case !ok:
			c.add(m.Franchise, "salary adjustment "+m.ID, m.Amount, "missing")
		case a.Franchise != m.Franchise || a.Amount != m.Amount || a.Description != m.Description:
			c.add(m.Franchise, "salary adjustment "+m.ID, m.Amount, a.Amount)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(app)) {
		c.add(app[id].Franchise, "salary adjustment "+id, "not on MFL", app[id].Amount)
	}
}

func pickKey(p commissioner.NativePick) string {
	if p.Slot > 0 {
		return fmt.Sprintf("%d round %d pick %d", p.Season, p.Round, p.Slot)
	}
	return fmt.Sprintf("%d round %d pick of %s", p.Season, p.Round, p.Original)
}

func (c *comparison) picks() {
	app := map[string]commissioner.NativePick{}
	for _, p := range c.app.Snapshot.Picks {
		app[pickKey(p)] = p
	}
	for _, m := range c.plan.Seed.Picks {
		key := pickKey(m)
		a, ok := app[key]
		delete(app, key)
		switch {
		case !ok:
			c.add(m.Owner, "the "+key, "owned", "missing")
		case a.Owner != m.Owner:
			c.add(m.Owner, "the "+key+" owner", m.Owner, a.Owner)
		case a.Player != m.Player || a.Skipped != m.Skipped:
			c.add(m.Owner, "the "+key+" selection", m.Player, a.Player)
		}
	}
	for _, key := range slices.Sorted(maps.Keys(app)) {
		c.add(app[key].Owner, "the "+key, "not on MFL", "owned")
	}
}

// settings compares every setting the import carries; the counted ones are read from the franchises.
func (c *comparison) settings() {
	skip := carried()
	app := map[leaguerules.FieldPath]leaguerules.Setting{}
	for _, ch := range leaguerules.SeedChanges(c.app.Rules, "", 0, epoch) {
		app[ch.Field] = ch.New
	}
	for _, ch := range leaguerules.SeedChanges(c.plan.Rules, "", 0, epoch) {
		if skip[ch.Field] {
			continue
		}
		if stateAndValue(ch.New) != stateAndValue(app[ch.Field]) {
			label := string(ch.Field)
			if f, ok := leaguerules.Field(ch.Field); ok {
				label = f.Label
			}
			c.add("", "setting "+label, leaguerules.DescribeField(ch.Field, ch.New).Words,
				leaguerules.DescribeField(ch.Field, app[ch.Field]).Words)
		}
	}
}

// stateAndValue is a setting's state and value with every citation blanked: the review compares what a
// setting is, not where its words say it came from.
func stateAndValue(s leaguerules.Setting) string {
	body, err := json.Marshal(s)
	if err != nil {
		return "unreadable"
	}
	var tree any
	if err := json.Unmarshal(body, &tree); err != nil {
		return "unreadable"
	}
	out, _ := json.Marshal(blankCitations(tree)) //nolint:errchkjson // decoded JSON always encodes
	return string(out)
}

func blankCitations(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if k == "Citation" {
				t[k] = ""
				continue
			}
			t[k] = blankCitations(child)
		}
	case []any:
		for i := range t {
			t[i] = blankCitations(t[i])
		}
	}
	return v
}

// Words turns the differences and the plan into the review a commissioner reads.
func Words(league string, plan Plan, app AppLeague, diffs []Difference) Review {
	r := Review{League: league, Differences: len(diffs), Notes: slices.Clone(plan.Notes), CutOver: app.CutOver,
		LeagueWide: []string{}, Franchises: []FranchiseCheck{}}
	byFranchise := map[string][]Difference{}
	for _, d := range diffs {
		if d.Franchise == "" {
			r.LeagueWide = append(r.LeagueWide, fmt.Sprintf("The %s differs: %s on MFL, %s in the app", d.Field, d.MFL, d.App))
			continue
		}
		byFranchise[d.Franchise] = append(byFranchise[d.Franchise], d)
	}
	for _, f := range plan.Seed.Franchises {
		r.Franchises = append(r.Franchises, franchiseCheck(f, plan.Seed, byFranchise[f.ID]))
	}
	switch {
	case app.CutOver:
		r.Summary = "This league has been moved over from MFL; the app runs it now"
	case len(diffs) == 0:
		r.Summary = "Everything matches MFL: rosters, contracts, picks, franchises, divisions and settings"
	default:
		r.Summary = fmt.Sprintf("%d things differ from MFL; each is listed under its franchise or the league", len(diffs))
	}
	body, _ := json.Marshal(r) //nolint:errchkjson // plain strings, ints and bools always encode
	sum := sha256.Sum256(body)
	r.Digest = hex.EncodeToString(sum[:8])
	return r
}

func franchiseCheck(f commissioner.NativeFranchise, seed commissioner.RawNativeSnapshot,
	diffs []Difference) FranchiseCheck {
	check := FranchiseCheck{Franchise: f.ID, MFLID: f.HeldID, Name: f.Name, Lines: []string{}, Matches: len(diffs) == 0}
	groups := map[string][]string{}
	for _, d := range diffs {
		kind := group(d.Field)
		groups[kind] = append(groups[kind], fmt.Sprintf("%s (%s on MFL, %s in the app)", describe(d.Field, seed), d.MFL, d.App))
	}
	for _, kind := range []string{"roster", "contract", "pick", "salary adjustment", "franchise"} {
		items := groups[kind]
		if len(items) == 0 {
			check.Lines = append(check.Lines, matchWords[kind])
			continue
		}
		noun := kind + "s"
		if len(items) == 1 {
			noun = kind
		}
		verb := "differ"
		if len(items) == 1 {
			verb = "differs"
		}
		check.Lines = append(check.Lines, fmt.Sprintf("%d %s %s: %s", len(items), noun, verb, strings.Join(items, "; ")))
	}
	return check
}

//nolint:gochecknoglobals // fixed words, read only
var matchWords = map[string]string{"roster": "Roster matches", "contract": "Contracts match",
	"pick": "Picks match", "salary adjustment": "Salary adjustments match", "franchise": "Name and division match"}

func group(field string) string {
	switch {
	case strings.HasSuffix(field, " rostered") || strings.HasSuffix(field, " franchise") ||
		strings.HasSuffix(field, " roster status"):
		return "roster"
	case strings.HasPrefix(field, "player "):
		return "contract"
	case strings.HasPrefix(field, "the "):
		return "pick"
	case strings.HasPrefix(field, "salary adjustment"):
		return "salary adjustment"
	}
	return "franchise"
}

// describe names a player by name where MFL's list holds it.
func describe(field string, seed commissioner.RawNativeSnapshot) string {
	rest, ok := strings.CutPrefix(field, "player ")
	if !ok {
		return field
	}
	id, what, _ := strings.Cut(rest, " ")
	for _, p := range seed.Players {
		if p.ID == id && p.Name != "" {
			return p.Name + " " + what
		}
	}
	return field
}
