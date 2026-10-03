package measures

import (
	"fmt"
	"strings"
)

// DictionaryPath is where the rendered dictionary lives, relative to the repository root.
const DictionaryPath = "docs/data-layer/Measure_Dictionary.md"

// Dictionary renders the registry as the Measure Dictionary document. The output depends only
// on the registry, so a test can prove the checked-in document is current.
func (r *Registry) Dictionary() string {
	var b strings.Builder
	b.WriteString("# Measure Dictionary\n\n")
	b.WriteString("Generated from `internal/measures/*.csv` by `make measure-dictionary`. ")
	b.WriteString("Do not edit it by hand: change the CSV files and regenerate.\n\n")
	b.WriteString("Every number in `history.db` is stored as player · season · week · measure. ")
	b.WriteString("Week 0 holds season-level values. A measure is named for what it means, never for ")
	b.WriteString("the source it came from. When several sources feed a measure, the first listed wins.\n")

	for _, fam := range Families() {
		var rows []string
		for _, m := range r.Measures {
			if m.Family != fam {
				continue
			}
			rows = append(rows, fmt.Sprintf("| `%s` | %s | %s | %s | %s | %s |",
				m.Name, m.Grain, m.Unit, positionsLabel(m), cell(m.Meaning), r.feedLabel(m.Name)))
		}
		if len(rows) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n\n", fam)
		b.WriteString("| Measure | Grain | Unit | Positions | Meaning | Sources |\n")
		b.WriteString("|---|---|---|---|---|---|\n")
		b.WriteString(strings.Join(rows, "\n") + "\n")
	}

	b.WriteString("\n## Sources\n\n")
	b.WriteString("A source is lost when its latest load failed and it has had no successful load ")
	b.WriteString("within its window.\n\n")
	b.WriteString("| Source | Name | Status | Lost after | Fetched from |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, s := range r.Sources {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %d days | `%s%s` |\n",
			s.ID, cell(s.Name), s.Status, int(s.MaxAge.Hours()/24), s.Host, s.PathPrefix)
	}
	return b.String()
}

// feedLabel lists the source fields feeding a measure, in priority order.
func (r *Registry) feedLabel(measure string) string {
	feeds := r.FieldsFeeding(measure)
	if len(feeds) == 0 {
		return "none yet"
	}
	parts := make([]string, len(feeds))
	for i, f := range feeds {
		parts[i] = fmt.Sprintf("`%s` %s", f.Source, f.Field)
	}
	return strings.Join(parts, ", then ")
}

func positionsLabel(m Measure) string {
	if m.Positions == nil {
		return "all"
	}
	parts := make([]string, len(m.Positions))
	for i, p := range m.Positions {
		parts[i] = string(p)
	}
	return strings.Join(parts, " ")
}

// cell escapes a table cell's pipe characters.
func cell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }
