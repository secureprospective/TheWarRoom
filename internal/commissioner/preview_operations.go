package commissioner

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

// operationPreview projects a private trial ledger through the same native readings owners use.
// Preparation and refusals belong to the store; no trial entry is ever recorded here.
func operationPreview(raw RawCommand, records []Record, catalogue PreviewCatalogue, now time.Time) (Preview, error) {
	if len(records) == 0 {
		return Preview{}, fmt.Errorf("create this app league before changing it")
	}
	safeguard, err := safeguardOf(raw)
	if err != nil {
		return Preview{}, err
	}
	trial := append(slices.Clone(records), Record{Command: raw,
		Principal: previewPrincipal(raw, records), Receipt: Receipt{Entry: "proposed." + raw.Request,
			Revision: raw.Expected + 1, RecordedAt: now}})
	if raw.Command == ScheduleGenerate {
		weeks, err := FreezeSchedule(raw, records)
		if err != nil {
			return Preview{}, err
		}
		trial[len(trial)-1].GeneratedSchedule = weeks
	}
	label, err := entryLabel(trial, len(trial)-1)
	if err != nil {
		return Preview{}, err
	}
	before, err := operationValues(raw, records, now)
	if err != nil {
		return Preview{}, err
	}
	after, err := operationValues(raw, trial, now)
	if err != nil {
		return Preview{}, err
	}
	fields := ChangedPreviewFields(before, after)
	if len(fields) == 0 {
		fields = UnchangedPreviewFields(raw, before, after)
	}
	fields = append(fields, PreviewField{Label: "League ledger", Before: "This change is not recorded",
		BeforeSource: "Recorded league ledger", After: label, AfterSource: proposed, When: timingWords(raw.Effective)})
	PreviewFieldSources(raw, fields)
	control := controlOfCommand(raw.Command)
	if d, ok := peopleDefinitionFor(raw.Command); ok {
		control = d.control
	}
	if err := NamePreviewFields(fields, records, now); err != nil {
		return Preview{}, err
	}
	readings := operationReadings(control)
	count := len(readings)
	affects := PreviewAffects{Readings: readings, Count: &count}

	return Preview{Question: catalogue.questions[control], Safeguard: safeguard,
		Target: PreviewTarget{League: raw.League, Name: Replay(records).Name, Version: raw.Expected},
		Fields: fields, Blockers: []string{}, Affects: affects}, nil
}

func previewPrincipal(raw RawCommand, records []Record) string {
	if raw.Args.People != nil && raw.Args.People.Principal != "" {
		return raw.Args.People.Principal
	}
	return Replay(records).Creator
}

func ChangedPreviewFields(before, after map[string]string) []PreviewField {
	keys := slices.Sorted(maps.Keys(before))
	for key := range after {
		if _, ok := before[key]; !ok {
			keys = append(keys, key)
		}
	}
	slices.SortFunc(keys, labelOrder)
	fields := []PreviewField{}
	for _, key := range keys {
		was, became := before[key], after[key]
		if was == became {
			continue
		}
		if was == "" {
			was = "None recorded"
		}
		if became == "" {
			became = "None recorded"
		}
		fields = append(fields, PreviewField{Label: key, Before: was, BeforeSource: "Current app league readings",
			After: became, AfterSource: proposed, When: "now"})
	}
	return fields
}

// labelOrder sorts preview labels as a person reads them: each run of digits compares as a number, so
// "week 2" comes before "week 10"; labels that read the same fall back to plain text order.
func labelOrder(a, b string) int {
	x, y := a, b
	for x != "" && y != "" {
		dx, dy := digitRun(x), digitRun(y)
		if dx > 0 && dy > 0 {
			nx, ny := strings.TrimLeft(x[:dx], "0"), strings.TrimLeft(y[:dy], "0")
			if c := cmp.Or(cmp.Compare(len(nx), len(ny)), strings.Compare(nx, ny)); c != 0 {
				return c
			}
			x, y = x[dx:], y[dy:]
			continue
		}
		if x[0] != y[0] {
			return cmp.Compare(x[0], y[0])
		}
		x, y = x[1:], y[1:]
	}
	return cmp.Or(cmp.Compare(len(x), len(y)), strings.Compare(a, b))
}

func digitRun(s string) int {
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	return n
}

func operationReadings(control string) []string {
	switch control {
	case "CM-018", "CM-019", "CM-020", "CM-021", "CM-022":
		return []string{"People", "Ledger"}
	case "CM-023", "CM-026":
		return []string{"Calendar", "Ledger"}
	case "CM-029", "CM-030", "CM-031", "CM-032":
		return []string{"Lineups", "Results", "Standings", "Ledger"}
	case "CM-033":
		return []string{"Standings", "Draft", "Ledger"}
	case "CM-035":
		return []string{"Waivers", "Rosters", "Cap", "Ledger"}
	case "CM-036":
		return []string{"Bids", "Rosters", "Cap", "Ledger"}
	case "CM-037":
		return []string{"Trades", "Rosters", "Cap", "Draft", "Ledger"}
	case "CM-040", "CM-041", "CM-042", "CM-043", "CM-044", "CM-045":
		return []string{"Draft", "Rosters", "Cap", "Ledger"}
	default:
		return []string{"Rosters", "Cap", "Draft", "Ledger"}
	}
}

func operationValues(raw RawCommand, records []Record, now time.Time) (map[string]string, error) {
	out := map[string]string{}
	switch {
	case raw.Args.Bracket != nil:
		return bracketPreviewValues(raw, records, now)
	case raw.Args.People != nil:
		return peoplePreviewValues(raw, records)
	case raw.Args.Standings != nil:
		return standingsPreviewValues(raw, records)
	case raw.Args.Score != nil:
		return scorePreviewValues(raw, records)
	case raw.Args.Lineup != nil:
		return lineupPreviewValues(raw, records)

	case raw.Args.Draft != nil:
		return draftSetupValues(raw, records, now)
	case raw.Args.Schedule != nil:
		return schedulePreviewValues(records)
	case raw.Args.Window != nil:
		rules, err := RulesReading(raw.League, records, now)
		if err != nil {
			return nil, err
		}
		for _, section := range rules.Sections {
			for _, row := range section.Rows {
				out[row.Label] = row.Value
			}
		}
	case raw.Args.Exception != nil:
		return calendarPreviewValues(raw, records, now)

	}
	if nativePreviewCommand(raw) {
		if err := nativePreviewValues(raw, records, now, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// RolloverPreviewFields carries the actual dry-run plan, including every pending contract and waiting rule.
func RolloverPreviewFields(before *RolloverBefore, plan *RolloverPlan) []PreviewField {
	fields := []PreviewField{{Label: "League season", Before: fmt.Sprintf("Season %d", plan.From),
		BeforeSource: "Current app league season", After: fmt.Sprintf("Season %d", plan.To),
		AfterSource: "Season rollover plan", When: "now"}}
	for _, line := range plan.Lines {
		fields = append(fields, PreviewField{Label: "Season rollover", Before: fmt.Sprintf("Season %d", before.Rules.Season),
			BeforeSource: "Current app league readings", After: line, AfterSource: "Season rollover plan", When: "now"})
	}
	return fields
}

func nativePreviewCommand(raw RawCommand) bool {
	return raw.Command == OLineAuctionClose || raw.Args.Native != nil || raw.Args.Override != nil || raw.Args.Correction != nil ||
		raw.Args.Waiver != nil || raw.Args.Bid != nil || raw.Args.Trade != nil
}

func lineupPreviewValues(raw RawCommand, records []Record) (map[string]string, error) {
	c := raw.Args.Lineup
	held, set, err := RecordedLineupFor(records, c.Franchise, c.Season, c.Week)
	if err != nil {
		return nil, err
	}
	words := "No lineup recorded"
	if set {
		names, err := lineupPlayerNames(records, held.Starters)
		if err != nil {
			return nil, err
		}
		words = "Starting players: " + strings.Join(names, ", ")
	}
	return map[string]string{fmt.Sprintf("%s, season %d, week %d lineup", c.Franchise, c.Season, c.Week): words}, nil
}

// lineupPlayerNames shows starters by name; an owner never reads a player id.
func lineupPlayerNames(records []Record, ids []string) ([]string, error) {
	assets, err := NativeAssetsReading(records[0].Command.League, records)
	if err != nil {
		return nil, err
	}
	known := map[string]string{}
	if assets.Seeded {
		for _, p := range assets.Snapshot.Players {
			known[p.ID] = p.Name
		}
	}
	names := make([]string, len(ids))
	for i, id := range ids {
		names[i] = cmp.Or(known[id], "A player no longer on the roster")
	}
	return names, nil
}

func UnchangedPreviewFields(raw RawCommand, before, after map[string]string) []PreviewField {
	fields := []PreviewField{}
	for _, key := range slices.Sorted(maps.Keys(after)) {
		if before[key] != after[key] || !previewContextKey(raw, key) {
			continue
		}
		words := after[key]
		if words == "" {
			words = "None recorded"
		}
		fields = append(fields, PreviewField{Label: key, Before: words, BeforeSource: "Current app league readings",
			After: words, AfterSource: proposed + " (unchanged)", When: "now"})
	}
	return fields
}

func previewContextKey(raw RawCommand, key string) bool {
	switch {
	case raw.Args.People != nil:
		return strings.HasPrefix(key, "Last activity")
	case raw.Args.Waiver != nil:
		return strings.HasPrefix(key, "Waiver")
	case raw.Args.Bid != nil:
		return strings.HasPrefix(key, "Bids for player")
	case raw.Args.Trade != nil:
		return strings.HasPrefix(key, "This trade")
	default:
		return true
	}
}

// PreviewSeasonRecorded distinguishes a missing season from a failure to read a season the league holds.
func PreviewSeasonRecorded(records []Record, season int, now time.Time) (bool, error) {
	first, err := firstSeason(records)
	if err != nil {
		return false, err
	}
	current, err := SeasonAt(records, ReadingInstant(records, now))
	if err != nil {
		return false, err
	}
	return season >= first && season <= current, nil
}

func standingsPreviewValues(raw RawCommand, records []Record) (map[string]string, error) {
	view, err := StandingsAdjustmentsReading(records, raw.Args.Standings.Season)
	if err != nil {
		return nil, err
	}
	out := map[string]string{"Standings order source": "Computed league standings"}
	if view.Order != nil {
		out["Standings order source"] = "Commissioner order: " + strings.Join(view.Order.Order, ", ")
	}
	for id, r := range view.Records {
		out["Record set by hand for "+id] =
			countWords(r.Wins, "win", "wins") + ", " + countWords(r.Losses, "loss", "losses") + ", " +
				countWords(r.Ties, "tie", "ties")
	}
	return out, nil
}
