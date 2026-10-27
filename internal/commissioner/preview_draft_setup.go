package commissioner

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

func draftSetupValues(raw RawCommand, records []Record, now time.Time) (map[string]string, error) {
	d := raw.Args.Draft
	rules, err := draftRulesFor(records, d.Season, now)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	out["Later-round draft pattern"] = FieldWords(rules, "draft.laterPattern")
	out["Draft order tie rule"] = FieldWords(rules, "draft.tieRule")
	if d.Season == 0 {
		return out, nil
	}
	prefix := fmt.Sprintf("Season %d rookie draft", d.Season)
	out[prefix+" order source"] = FieldWords(rules, "draft.orderSource")
	rounds := slices.SortedFunc(maps.Keys(rules.Draft.Order.Value), func(a, b int) int { return a - b })
	for _, round := range rounds {
		out[fmt.Sprintf("%s, round %d order", prefix, round)] = strings.Join(rules.Draft.Order.Value[round], ", ")
	}
	start, found := draftStartFor(records, d.Season)
	out[prefix+" start"] = "No start recorded"
	if found {
		out[prefix+" start"] = start.Format("Jan 2, 2006 at 3:04 PM MST")
	}
	return out, nil
}

// DraftSetupFields compares the actual order, including the final standings used for automatic selection.
func DraftSetupFields(raw RawCommand, records []Record, now time.Time, standings DraftStandings) ([]PreviewField, error) {
	if raw.Args.Draft == nil || raw.Args.Draft.Season == 0 || DraftTrayCommand(raw.Command) {
		return nil, nil
	}
	before, err := draftSetupReading(raw.Args.Draft.Season, records, now, standings)
	if err != nil {
		return nil, err
	}
	trial := append(slices.Clone(records), Record{Command: raw,
		Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: now}})
	after, err := draftSetupReading(raw.Args.Draft.Season, trial, now, standings)
	if err != nil {
		return nil, err
	}
	return ChangedPreviewFields(before, after), nil
}

func draftSetupReading(season int, records []Record, now time.Time, standings DraftStandings) (map[string]string, error) {
	basis, err := readDraft(records, season, now, standings)
	if err != nil {
		return nil, err
	}
	out := map[string]string{"Rookie draft": basis.words}
	for _, round := range basis.order {
		picks := []string{}
		for _, pick := range round.Picks {
			picks = append(picks, pick.Owner)
		}
		out[fmt.Sprintf("Season %d rookie draft, round %d order", season, round.Round)] = strings.Join(picks, ", ")
	}
	return out, nil
}
