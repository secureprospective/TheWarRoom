// Package measures is the registry behind the measure dictionary: what each measure means,
// which sources exist, which source field feeds which measure, and which files the table-driven
// loader reads. The registry is four checked-in CSV files embedded in the binary. The history store loads them into tables, and
// docs/data-layer/Measure_Dictionary.md is rendered from them. A source's own vocabulary
// appears only in source_fields.csv.
package measures

import (
	"embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

//go:embed measures.csv sources.csv source_fields.csv feeds.csv
var embedded embed.FS

// Family groups measures by what they describe.
type Family string

const (
	FamilyExposure     Family = "exposure"     // time on the field: snaps, routes
	FamilyOpportunity  Family = "opportunity"  // chances: targets, carries, pass-rush snaps
	FamilyOutcome      Family = "outcome"      // results: yards, tackles, fantasy points
	FamilyPrior        Family = "prior"        // before the NFL: draft, combine, college
	FamilyAvailability Family = "availability" // games played, injury status
	FamilyContext      Family = "context"      // team and scheme facts
)

// Families is the fixed order the dictionary renders in.
func Families() []Family {
	return []Family{FamilyExposure, FamilyOpportunity, FamilyOutcome, FamilyPrior, FamilyAvailability, FamilyContext}
}

// Grain is the period a measure is recorded at. Season-grain values sit at week 0. Player-grain
// values are facts that belong to no period (a birth date, a draft slot) and sit at season 0,
// week 0.
type Grain string

const (
	GrainPlayer Grain = "player"
	GrainSeason Grain = "season"
	GrainWeek   Grain = "week"
)

// minSeason is the earliest season a period-grain value may carry.
const minSeason = 1900

// UnitText marks a measure whose value is a label (an injury status), not a number.
const UnitText = "text"

// Measure is one meaning. Positions is nil when the measure applies to every position.
type Measure struct {
	Name      string
	Family    Family
	Grain     Grain
	Unit      string
	Positions []domain.Position
	Meaning   string
}

// IsText reports whether values are labels rather than numbers.
func (m Measure) IsText() bool { return m.Unit == UnitText }

// ValidPeriod reports whether a season and week fit the grain: 0 and 0 for player, a season
// and week 0 for season, a season and week 1 or more for week.
func (m Measure) ValidPeriod(season, week int) bool {
	switch m.Grain {
	case GrainPlayer:
		return season == 0 && week == 0
	case GrainSeason:
		return season >= minSeason && week == 0
	case GrainWeek:
		return season >= minSeason && week >= 1
	default:
		return false
	}
}

// SourceStatus is the declared state of a source. Lost is observed from loads, never declared.
type SourceStatus string

const (
	SourceActive  SourceStatus = "active"
	SourceRetired SourceStatus = "retired"
)

// Source is one place data comes from. A fetch belongs to it when the URL's host is Host or a
// subdomain of it and the path starts with PathPrefix. MaxAge is how long the source may go
// without a successful load, after a failure, before it counts as lost.
type Source struct {
	ID         string
	Name       string
	Status     SourceStatus
	MaxAge     time.Duration
	Host       string
	PathPrefix string
}

// MatchesURL is Matches for a URL written as text.
func (s Source) MatchesURL(raw string) bool {
	u, err := url.Parse(strings.ReplaceAll(raw, SeasonToken, "0"))
	return err == nil && s.Matches(u)
}

// Matches reports whether a fetched URL belongs to this source.
func (s Source) Matches(u *url.URL) bool {
	h := strings.ToLower(u.Hostname())
	if h != s.Host && !strings.HasSuffix(h, "."+s.Host) {
		return false
	}
	return strings.HasPrefix(u.Path, s.PathPrefix)
}

// SourceField maps one field of one source onto a measure. When several sources feed a
// measure, the lowest Priority wins in the features layer.
type SourceField struct {
	Source   string
	Field    string
	Measure  string
	Priority int
}

// Registry is the validated content of the four files.
type Registry struct {
	Measures []Measure
	Sources  []Source
	Fields   []SourceField
	Feeds    []Feed

	measures map[string]Measure
	fields   map[[2]string]SourceField // (source, field)
}

// Source returns the source with id.
func (r *Registry) Source(id string) (Source, bool) {
	for _, s := range r.Sources {
		if s.ID == id {
			return s, true
		}
	}
	return Source{}, false
}

// Measure returns the named measure.
func (r *Registry) Measure(name string) (Measure, bool) {
	m, ok := r.measures[name]
	return m, ok
}

// Field returns the mapping for a source's field.
func (r *Registry) Field(source, field string) (SourceField, bool) {
	f, ok := r.fields[[2]string{source, field}]
	return f, ok
}

// SourceFor returns the source a fetched URL belongs to.
func (r *Registry) SourceFor(u *url.URL) (string, bool) {
	for _, s := range r.Sources {
		if s.Matches(u) {
			return s.ID, true
		}
	}
	return "", false
}

// FieldsFeeding returns the mappings for a measure in priority order.
func (r *Registry) FieldsFeeding(measure string) []SourceField {
	var out []SourceField
	for _, f := range r.Fields {
		if f.Measure == measure {
			out = append(out, f)
		}
	}
	slices.SortFunc(out, func(a, b SourceField) int { return a.Priority - b.Priority })
	return out
}

// Embedded loads the registry shipped in the binary.
func Embedded() (*Registry, error) { return Load(embedded) }

// Load reads and validates measures.csv, sources.csv, source_fields.csv and feeds.csv from fsys.
// Every problem is reported, not only the first.
func Load(fsys fs.FS) (*Registry, error) {
	r := &Registry{measures: map[string]Measure{}, fields: map[[2]string]SourceField{}}
	var errs []error
	errs = append(errs, r.loadMeasures(fsys)...)
	errs = append(errs, r.loadSources(fsys)...)
	if len(errs) == 0 {
		errs = append(errs, r.loadFields(fsys)...)
	}
	if len(errs) == 0 {
		errs = append(errs, r.loadFeeds(fsys)...)
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("measures: invalid registry: %w", err)
	}
	slices.SortFunc(r.Measures, func(a, b Measure) int { return strings.Compare(a.Name, b.Name) })
	return r, nil
}

var (
	measureName = regexp.MustCompile(`^([a-z]+)\.[a-z0-9_]+$`)
	identifier  = regexp.MustCompile(`^[a-z0-9_]+$`)
)

func (r *Registry) loadMeasures(fsys fs.FS) []error {
	rows, err := readTable(fsys, "measures.csv", "measure", "grain", "unit", "positions", "meaning")
	if err != nil {
		return []error{err}
	}
	var errs []error
	for i, row := range rows {
		at := fmt.Sprintf("measures.csv line %d", i+2)
		m := Measure{Name: row[0], Grain: Grain(row[1]), Unit: row[2], Meaning: strings.TrimSpace(row[4])}
		match := measureName.FindStringSubmatch(m.Name)
		switch {
		case match == nil:
			errs = append(errs, fmt.Errorf("%s: measure %q is not family.name in lower snake case", at, m.Name))
			continue
		case !slices.Contains(Families(), Family(match[1])):
			errs = append(errs, fmt.Errorf("%s: %q has unknown family %q", at, m.Name, match[1]))
		case !slices.Contains([]Grain{GrainPlayer, GrainSeason, GrainWeek}, m.Grain):
			errs = append(errs, fmt.Errorf("%s: %q has grain %q, want player, season or week", at, m.Name, m.Grain))
		case !identifier.MatchString(m.Unit):
			errs = append(errs, fmt.Errorf("%s: %q has unit %q, want one lower-case word", at, m.Name, m.Unit))
		case m.Meaning == "":
			errs = append(errs, fmt.Errorf("%s: %q has no meaning", at, m.Name))
		case r.measures[m.Name].Name != "":
			errs = append(errs, fmt.Errorf("%s: %q is listed twice", at, m.Name))
		}
		m.Family = Family(match[1])
		positions, perr := parsePositions(row[3])
		if perr != nil {
			errs = append(errs, fmt.Errorf("%s: %q: %w", at, m.Name, perr))
		}
		m.Positions = positions
		r.Measures = append(r.Measures, m)
		r.measures[m.Name] = m
	}
	return errs
}

// parsePositions reads "all" or a space-separated list of engine positions.
func parsePositions(s string) ([]domain.Position, error) {
	if s == "all" {
		return nil, nil
	}
	known := []domain.Position{domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK,
		domain.PosDE, domain.PosDT, domain.PosLB, domain.PosCB, domain.PosS}
	var out []domain.Position
	for _, f := range strings.Fields(s) {
		p := domain.Position(f)
		if !slices.Contains(known, p) || slices.Contains(out, p) {
			return nil, fmt.Errorf("positions %q: %q is unknown or repeated", s, f)
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("positions is empty, want \"all\" or a list")
	}
	return out, nil
}

func (r *Registry) loadSources(fsys fs.FS) []error {
	rows, err := readTable(fsys, "sources.csv", "source", "name", "status", "max_age_days", "host", "path_prefix")
	if err != nil {
		return []error{err}
	}
	var errs []error
	seen := map[string]bool{}
	for i, row := range rows {
		at := fmt.Sprintf("sources.csv line %d", i+2)
		s := Source{ID: row[0], Name: row[1], Status: SourceStatus(row[2]), Host: row[4], PathPrefix: row[5]}
		days, derr := strconv.Atoi(row[3])
		s.MaxAge = time.Duration(days) * 24 * time.Hour
		switch {
		case !identifier.MatchString(s.ID):
			errs = append(errs, fmt.Errorf("%s: source id %q is not lower snake case", at, s.ID))
		case seen[s.ID]:
			errs = append(errs, fmt.Errorf("%s: source %q is listed twice", at, s.ID))
		case s.Name == "":
			errs = append(errs, fmt.Errorf("%s: source %q has no name", at, s.ID))
		case s.Status != SourceActive && s.Status != SourceRetired:
			errs = append(errs, fmt.Errorf("%s: source %q has status %q, want active or retired", at, s.ID, s.Status))
		case derr != nil || days < 1:
			errs = append(errs, fmt.Errorf("%s: source %q max_age_days %q is not a positive whole number", at, s.ID, row[3]))
		case s.Host == "" || s.Host != strings.ToLower(s.Host):
			errs = append(errs, fmt.Errorf("%s: source %q host %q must be a lower-case host name", at, s.ID, s.Host))
		case s.PathPrefix != "" && !strings.HasPrefix(s.PathPrefix, "/"):
			errs = append(errs, fmt.Errorf("%s: source %q path_prefix %q must start with /", at, s.ID, s.PathPrefix))
		}
		seen[s.ID] = true
		r.Sources = append(r.Sources, s)
	}
	return errs
}

func (r *Registry) loadFields(fsys fs.FS) []error {
	rows, err := readTable(fsys, "source_fields.csv", "source", "field", "measure", "priority")
	if err != nil {
		return []error{err}
	}
	sources := map[string]bool{}
	for _, s := range r.Sources {
		sources[s.ID] = true
	}
	var errs []error
	feeds := map[[2]string]bool{}  // (source, measure)
	priority := map[string][]int{} // measure → priorities taken
	for i, row := range rows {
		at := fmt.Sprintf("source_fields.csv line %d", i+2)
		f := SourceField{Source: row[0], Field: row[1], Measure: row[2]}
		p, perr := strconv.Atoi(row[3])
		f.Priority = p
		_, measureKnown := r.measures[f.Measure]
		switch {
		case !sources[f.Source]:
			errs = append(errs, fmt.Errorf("%s: unknown source %q", at, f.Source))
		case !measureKnown:
			errs = append(errs, fmt.Errorf("%s: unknown measure %q", at, f.Measure))
		case f.Field == "":
			errs = append(errs, fmt.Errorf("%s: %s has an empty field name", at, f.Source))
		case perr != nil || p < 1:
			errs = append(errs, fmt.Errorf("%s: priority %q is not a whole number from 1", at, row[3]))
		case r.fields[[2]string{f.Source, f.Field}].Source != "":
			errs = append(errs, fmt.Errorf("%s: %s.%s is mapped twice", at, f.Source, f.Field))
		case feeds[[2]string{f.Source, f.Measure}]:
			errs = append(errs, fmt.Errorf("%s: %s feeds %s from two fields; one source, one field per measure", at, f.Source, f.Measure))
		case slices.Contains(priority[f.Measure], p):
			errs = append(errs, fmt.Errorf("%s: two sources share priority %d for %s", at, p, f.Measure))
		}
		feeds[[2]string{f.Source, f.Measure}] = true
		priority[f.Measure] = append(priority[f.Measure], p)
		r.Fields = append(r.Fields, f)
		r.fields[[2]string{f.Source, f.Field}] = f
	}
	return errs
}

// readTable reads a CSV whose first row must be exactly header, and returns the data rows.
func readTable(fsys fs.FS, name string, header ...string) ([][]string, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	defer func() { _ = f.Close() }()
	cr := csv.NewReader(f)
	cr.FieldsPerRecord = len(header)
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	if len(rows) == 0 || !slices.Equal(rows[0], header) {
		return nil, fmt.Errorf("%s: header must be %s", name, strings.Join(header, ","))
	}
	return rows[1:], nil
}

// Fact is one value as a source reports it, before mapping: the source's id for the player,
// the period, the source's field name and the raw text. IDType "mfl" means ID is the MFL id.
type Fact struct {
	IDType string
	ID     string
	Season int
	Week   int
	Field  string
	Raw    string
}

// IDTypeMFL is the id type that resolves as the player id itself.
const IDTypeMFL = "mfl"

// Batch is one load: the facts parsed from one fetched body. BodySHA256 links the load to its
// raw_archive row; it is empty when the body is not known.
type Batch struct {
	Source     string
	BodySHA256 string
	Facts      []Fact
	Scope      *Scope
}

// Scope, on a batch, says the batch is its source's whole report for these week-grain measures
// in these seasons. A week value held from an earlier load and missing from this one is now zero.
type Scope struct {
	Seasons  []int
	Measures []string
}
