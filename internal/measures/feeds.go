package measures

import (
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"
)

// SeasonToken is replaced by the season in a per-season feed's URL.
const SeasonToken = "{season}"

// Feed is one file the table-driven loader reads: where it lives, which column holds the player
// id and of what type, which columns hold the period, and which rows to keep. Its fields are the
// source_fields rows named "<feed>.<column>". A feed with a FirstSeason is one file per season
// from FirstSeason on; without one it is a single file.
type Feed struct {
	Name         string
	Source       string
	URL          string
	FirstSeason  int
	IDColumn     string
	IDType       string
	SeasonColumn string
	WeekColumn   string
	Filter       []Condition
}

// Grain is the grain of every measure the feed fills, set by its period columns.
func (f Feed) Grain() Grain {
	switch {
	case f.WeekColumn != "":
		return GrainWeek
	case f.SeasonColumn != "":
		return GrainSeason
	default:
		return GrainPlayer
	}
}

// URLFor returns the feed's URL for a season. A single-file feed ignores the season.
func (f Feed) URLFor(season int) string {
	return strings.ReplaceAll(f.URL, SeasonToken, strconv.Itoa(season))
}

// Condition keeps a row when Column's value is one of Values, or, when Exclude is set, when it
// is none of them.
type Condition struct {
	Column  string
	Exclude bool
	Values  []string
}

// Keep reports whether a row passes every condition. value reads a column of the row.
func (f Feed) Keep(value func(column string) string) bool {
	for _, c := range f.Filter {
		if slices.Contains(c.Values, value(c.Column)) == c.Exclude {
			return false
		}
	}
	return true
}

// Columns returns every column the feed reads: the id, the period, the filter and each field's.
func (r *Registry) Columns(f Feed) []string {
	cols := []string{f.IDColumn}
	for _, c := range []string{f.SeasonColumn, f.WeekColumn} {
		if c != "" {
			cols = append(cols, c)
		}
	}
	for _, c := range f.Filter {
		cols = append(cols, c.Column)
	}
	for _, sf := range r.FeedFields(f) {
		cols = append(cols, FeedColumn(sf))
	}
	slices.Sort(cols)
	return slices.Compact(cols)
}

// FeedFields returns the source_fields rows a feed fills.
func (r *Registry) FeedFields(f Feed) []SourceField {
	var out []SourceField
	for _, sf := range r.Fields {
		if feed, _, ok := strings.Cut(sf.Field, "."); ok && sf.Source == f.Source && feed == f.Name {
			out = append(out, sf)
		}
	}
	return out
}

// FeedColumn is the column a feed field reads: the part of its name after "<feed>.".
func FeedColumn(sf SourceField) string {
	_, col, _ := strings.Cut(sf.Field, ".")
	return col
}

// Feed returns the named feed.
func (r *Registry) Feed(name string) (Feed, bool) {
	for _, f := range r.Feeds {
		if f.Name == name {
			return f, true
		}
	}
	return Feed{}, false
}

func (r *Registry) loadFeeds(fsys fs.FS) []error {
	rows, err := readTable(fsys, "feeds.csv",
		"feed", "source", "url", "first_season", "id_column", "id_type", "season_column", "week_column", "filter")
	if err != nil {
		return []error{err}
	}
	var errs []error
	for i, row := range rows {
		f, err := r.feedRow(row)
		if err != nil {
			errs = append(errs, fmt.Errorf("feeds.csv line %d: feed %q: %w", i+2, row[0], err))
		}
		r.Feeds = append(r.Feeds, f)
	}
	for _, f := range r.Feeds {
		fields := r.FeedFields(f)
		if len(fields) == 0 {
			errs = append(errs, fmt.Errorf("feeds.csv: feed %q has no source_fields rows", f.Name))
		}
		for _, sf := range fields {
			if m := r.measures[sf.Measure]; m.Grain != f.Grain() {
				errs = append(errs, fmt.Errorf("source_fields.csv: %s fills %s-grain %s from a %s-grain feed",
					sf.Field, m.Grain, m.Name, f.Grain()))
			}
		}
	}
	return errs
}

// feedRow reads and checks one feeds.csv row.
func (r *Registry) feedRow(row []string) (Feed, error) {
	f := Feed{Name: row[0], Source: row[1], URL: row[2], IDColumn: row[4], IDType: row[5],
		SeasonColumn: row[6], WeekColumn: row[7]}
	if row[3] != "" {
		season, err := strconv.Atoi(row[3])
		if err != nil || season < minSeason {
			return f, fmt.Errorf("first_season %q is not a season", row[3])
		}
		f.FirstSeason = season
	}
	filter, err := parseFilter(row[8])
	if err != nil {
		return f, err
	}
	f.Filter = filter
	src, known := r.Source(f.Source)
	switch {
	case !identifier.MatchString(f.Name):
		return f, fmt.Errorf("name is not lower snake case")
	case slices.ContainsFunc(r.Feeds, func(o Feed) bool { return o.Name == f.Name }):
		return f, fmt.Errorf("listed twice")
	case !known:
		return f, fmt.Errorf("unknown source %q", f.Source)
	case !src.MatchesURL(f.URL):
		return f, fmt.Errorf("URL is not under source %s", f.Source)
	case (f.FirstSeason != 0) != strings.Contains(f.URL, SeasonToken):
		return f, fmt.Errorf("needs both a first_season and %s in its URL, or neither", SeasonToken)
	case f.IDColumn == "" || !identifier.MatchString(f.IDType) || f.IDType == IDTypeMFL:
		return f, fmt.Errorf("needs an id column and a non-MFL id type")
	case f.WeekColumn != "" && f.SeasonColumn == "":
		return f, fmt.Errorf("has a week column but no season column")
	}
	return f, nil
}

// parseFilter reads "col=a|b&col2!=c": conditions joined by &, values by |.
func parseFilter(s string) ([]Condition, error) {
	if s == "" {
		return nil, nil
	}
	var out []Condition
	for _, part := range strings.Split(s, "&") {
		col, vals, ok := strings.Cut(part, "=")
		if !ok || vals == "" {
			return nil, fmt.Errorf("filter %q: %q is not column=values or column!=values", s, part)
		}
		c := Condition{Column: col, Values: strings.Split(vals, "|")}
		if c.Column, c.Exclude = strings.CutSuffix(col, "!"); c.Column == "" {
			return nil, fmt.Errorf("filter %q: %q names no column", s, part)
		}
		out = append(out, c)
	}
	return out, nil
}
