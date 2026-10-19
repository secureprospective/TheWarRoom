package commissioner

import (
	"encoding/csv"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type PreviewTarget struct {
	League  string `json:"league"`
	Name    string `json:"name"`
	Version int64  `json:"version"`
}

type PreviewField struct {
	Label        string `json:"label"`
	Before       string `json:"before"`
	BeforeSource string `json:"beforeSource"`
	After        string `json:"after"`
	AfterSource  string `json:"afterSource"`
	When         string `json:"when"`
}

// Count is the number of catalogue readings, not players, teams or changed records.
// A nil count means no catalogue reading is known to use this rule.
type PreviewAffects struct {
	Readings []string `json:"readings"`
	Count    *int     `json:"count"`
	Words    string   `json:"words"`
}

type Preview struct {
	Question  string         `json:"question"`
	Target    PreviewTarget  `json:"target"`
	Fields    []PreviewField `json:"fields"`
	Affects   PreviewAffects `json:"affects"`
	Blockers  []string       `json:"blockers"`
	Safeguard string         `json:"safeguard"`
}

// PreviewCatalogue is derived from the same registry the landing uses.
// IDs stay inside Go; the preview carries only the reading names.
type PreviewCatalogue struct {
	questions map[string]string
	readings  map[string][]string
}

func ReadPreviewCatalogue(body string) (PreviewCatalogue, error) {
	rows, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	if err != nil {
		return PreviewCatalogue{}, fmt.Errorf("read rule catalogue: %w", err)
	}
	if len(rows) == 0 {
		return PreviewCatalogue{}, fmt.Errorf("rule catalogue is empty")
	}
	id, name := slices.Index(rows[0], "id"), slices.Index(rows[0], "name")
	question, governs := slices.Index(rows[0], "gm_question"), slices.Index(rows[0], "governs")
	if id < 0 || name < 0 || question < 0 || governs < 0 {
		return PreviewCatalogue{}, fmt.Errorf("rule catalogue is missing its reading columns")
	}
	names := map[string]string{}
	for _, row := range rows[1:] {
		names[row[id]] = previewReadingName(row[name])
	}
	out := PreviewCatalogue{questions: map[string]string{}, readings: map[string][]string{}}
	for _, row := range rows[1:] {
		if !strings.HasPrefix(row[id], "CM-") {
			continue
		}
		out.questions[row[id]] = row[question]
		out.readings[row[id]] = []string{}
		for _, reading := range strings.Fields(row[governs]) {
			if reading == "ledger-wide" {
				continue
			}
			if names[reading] == "" {
				return PreviewCatalogue{}, fmt.Errorf("rule catalogue names an unknown reading %s", reading)
			}
			out.readings[row[id]] = append(out.readings[row[id]], names[reading])
		}
	}
	return out, nil
}

func previewReadingName(name string) string {
	for _, marker := range []string{" O=", " R="} {
		name, _, _ = strings.Cut(name, marker)
	}
	name = strings.TrimSpace(strings.TrimRight(name, " ⛔◆"))
	return strings.ReplaceAll(name, " handler", " plans")
}

func (c PreviewCatalogue) affects(control string) PreviewAffects {
	readings := append([]string{}, c.readings[control]...)
	if len(readings) == 0 {
		return PreviewAffects{Readings: readings, Words: "Not known yet"}
	}
	count := len(readings)
	words := fmt.Sprintf("%d owner readings use this rule", count)
	if count == 1 {
		words = "1 owner reading uses this rule"
	}
	return PreviewAffects{Readings: readings, Count: &count, Words: words}
}

// RulePreview describes the proposed value even when the append checks refuse it.
// No trial value is resolved as already in force, including next-rollover changes.
func RulePreview(raw RawCommand, records []Record, catalogue PreviewCatalogue, now time.Time) (Preview, error) {
	if raw.Command != RulesSet {
		return Preview{}, fmt.Errorf("a preview is available for rule changes only")
	}
	owner, err := ownerFor(leaguerules.FieldPath(raw.Args.Field))
	if err != nil {
		return Preview{}, err
	}
	reading, err := RulesReading(raw.League, records, now)
	if err != nil {
		return Preview{}, err
	}
	setting, err := leaguerules.DecodeSetting(leaguerules.FieldPath(raw.Args.Field), raw.Args.Value)
	if err != nil {
		return Preview{}, fmt.Errorf("describe proposed rule: %w", err)
	}
	out := Preview{Question: catalogue.questions[owner.control],
		Target: PreviewTarget{League: raw.League, Name: reading.Name, Version: raw.Expected},
		Fields: []PreviewField{}, Affects: catalogue.affects(owner.control), Blockers: []string{}}
	for _, section := range reading.Sections {
		for _, row := range section.Rows {
			if row.Field != raw.Args.Field {
				continue
			}
			out.Fields = append(out.Fields, PreviewField{Label: row.Label, Before: row.Value,
				BeforeSource: row.Citation, After: leaguerules.DescribeField(
					leaguerules.FieldPath(raw.Args.Field), setting).Words,
				AfterSource: "Your proposed change", When: timingWords(raw.Effective)})
			out.Safeguard = row.Safeguards.NextRollover
			if raw.Effective == "now" {
				out.Safeguard = row.Safeguards.Now
			}
		}
	}
	return out, nil
}
