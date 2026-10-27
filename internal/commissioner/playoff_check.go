package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// PlayoffStartWeek is the first playoff week the season's rules name, or false while they name none.
func PlayoffStartWeek(records []Record, season int, now time.Time) (int, bool) {
	rules, err := RulesInForce(records, season, ReadingInstant(records, now))
	if err != nil || rules.Playoffs.StartWeek.State != leaguerules.IsKnown || rules.Playoffs.StartWeek.Value < 1 {
		return 0, false
	}
	return rules.Playoffs.StartWeek.Value, true
}

// firstKickoff is the earliest NFL kickoff of the week the facts describe. It is not known while the
// week's kickoffs are not held.
func firstKickoff(facts LineupFacts) (time.Time, bool) {
	if !facts.ScheduleHeld || len(facts.Locks) == 0 {
		return time.Time{}, false
	}
	first := time.Time{}
	for _, at := range facts.Locks {
		if first.IsZero() || at.Before(first) {
			first = at
		}
	}
	return first, true
}

// places are the games and the places in them that a season's bracket has, for the conferences it is played in.
type places struct {
	games map[string]bool
	slots map[string]string
}

func playoffPlaces(r PlayoffRules, conferences []string) places {
	out := places{games: map[string]bool{}, slots: map[string]string{}}
	for _, n := range r.Shape {
		scopes := conferences
		if n.LeagueWide() {
			scopes = []string{""}
		}
		for _, scope := range scopes {
			id := competition.GameID(scope, n.ID)
			out.games[id] = true
			out.slots[id+"/home"], out.slots[id+"/away"] = scope, scope
		}
	}
	return out
}

// CheckPlayoffAppend holds a bracket command to the season's playoff rules and to the NFL calendar before it
// is recorded. facts describe the first playoff week; while its kickoffs are not held the command is taken
// as the commissioner states it. The title and history commands have their own checks.
func CheckPlayoffAppend(raw RawCommand, records []Record, now time.Time, facts LineupFacts) error {
	switch {
	case raw.Command == TitleSet:
		return checkTitleAppend(raw, records, now)
	case raw.Command == HistoryLink:
		return CheckHistoryLinkAppend(raw, records, now)
	case !isBracketCommand(raw.Command):
		return nil
	}
	change, err := DecodeBracketChange(raw.Args, raw.Command)
	if err != nil {
		return Refused(err)
	}
	if len(records) == 0 || !seeded(records) {
		return Refused(fmt.Errorf("record the league's starting snapshot before setting its playoffs"))
	}
	at := ReadingInstant(records, now)
	season, err := SeasonAt(records, at)
	if err != nil {
		return err
	}
	if change.Season != season {
		return Refused(fmt.Errorf("the playoffs being set are season %d's, but the league is in season %d", change.Season, season))
	}
	if err := checkBracketStarted(raw, change, facts, now); err != nil {
		return Refused(err)
	}
	rules, err := RulesInForce(records, season, at)
	if err != nil {
		return err
	}
	state := foldBracket(records, season, func(id string) string { return id })
	return Refused(checkBracketChange(raw.Command, change, records, rules, state))
}

// checkTitleAppend refuses a title for a season the league has not reached.
func checkTitleAppend(raw RawCommand, records []Record, now time.Time) error {
	title, err := DecodeTitleChange(raw.Args)
	if err != nil {
		return Refused(err)
	}
	if len(records) == 0 || !seeded(records) {
		return Refused(fmt.Errorf("record the league's starting snapshot before naming its champion"))
	}
	season, err := SeasonAt(records, ReadingInstant(records, now))
	if err != nil {
		return err
	}
	if title.Season > season {
		return Refused(fmt.Errorf("season %d has not been played, so it has no champion to name", title.Season))
	}
	return nil
}

// checkBracketStarted holds a change to whether the first playoff game has kicked off, the way a draft change
// is held to whether its clock runs.
func checkBracketStarted(raw RawCommand, change RawBracketChange, facts LineupFacts, now time.Time) error {
	first, known := firstKickoff(facts)
	if !known || !changesAfterKickoff(raw.Command) {
		return nil
	}
	started := !now.Before(first)
	switch {
	case started && !change.Started:
		return fmt.Errorf("the first playoff game has kicked off; say so and give a reason to change the bracket now")
	case !started && change.Started:
		return fmt.Errorf("the first playoff game has not kicked off; this is a regular bracket change")
	}
	return nil
}

func checkBracketChange(command string, change RawBracketChange, records []Record, rules leaguerules.RuleSet, state bracketState) error {
	switch command {
	case BracketPresetSave:
		return checkPresetSave(change, state)
	case BracketBuild:
		return checkBuild(change, state, rules)
	}
	graph, words := state.graph(rules)
	if words != "" {
		return fmt.Errorf("%s", words)
	}
	pr := ResolvePlayoffRules(rules, graph)
	if pr.Words != "" {
		return fmt.Errorf("%s", pr.Words)
	}
	if !leagueFranchise(records, change.Team) {
		return fmt.Errorf("that franchise is not in this app league")
	}
	layout := conferencesFor(records, rules, records[0].Command.Args.Franchises)
	if layout.Words != "" {
		return fmt.Errorf("%s", layout.Words)
	}
	return checkPlace(command, change, state, pr, layout)
}

func checkPlace(command string, change RawBracketChange, state bracketState, pr PlayoffRules,
	layout conferenceLayout) error {
	conference := layout.Of[change.Team]
	if command == BracketSeed {
		if change.Slot > pr.PerConference {
			return fmt.Errorf("a conference sends %d teams to the playoffs, so seed %d does not exist", pr.PerConference, change.Slot)
		}
		return nil
	}
	if !state.Built {
		return fmt.Errorf("build the playoff bracket before changing it")
	}
	known := playoffPlaces(pr, layout.Conferences)
	if command == BracketAdvance {
		if !known.games[change.Game] {
			return fmt.Errorf("%s is not a game in this bracket", change.Game)
		}
		return nil
	}
	scope, ok := known.slots[change.Node]
	switch {
	case !ok:
		return fmt.Errorf("%s is not a place in this bracket", change.Node)
	case scope != "" && scope != conference:
		return fmt.Errorf("that franchise plays in the %s conference, but this game is in the %s conference", conference, scope)
	}
	return nil
}

func checkBuild(change RawBracketChange, state bracketState, rules leaguerules.RuleSet) error {
	trial := state
	trial.Preset = change.Preset
	graph, words := trial.graph(rules)
	if words != "" {
		return fmt.Errorf("%s", words)
	}
	if pr := ResolvePlayoffRules(rules, graph); pr.Words != "" {
		return fmt.Errorf("%s", pr.Words)
	}
	if state.Built && state.Preset == change.Preset {
		return fmt.Errorf("the bracket is already built from %s. Nothing would change", presetWords(change.Preset))
	}
	return nil
}

func checkPresetSave(change RawBracketChange, state bracketState) error {
	if _, words := competition.ShapeFrom(nodesOf(change.Shape)); words != "" {
		return fmt.Errorf("%s", words)
	}
	ids := make([]string, len(change.Shape))
	for i, n := range change.Shape {
		ids[i] = n.ID
	}
	slices.Sort(ids)
	if len(slices.Compact(slices.Clone(ids))) != len(ids) {
		return fmt.Errorf("a bracket shape cannot name a game twice")
	}
	if held, ok := state.Presets[change.Name]; ok && slices.Equal(held, change.Shape) {
		return fmt.Errorf("that bracket shape is already saved under this name. Nothing would change")
	}
	return nil
}

// playoffCommand is any of the commands that set up and decide the playoffs and the season's title.
func playoffCommand(command string) bool {
	return isBracketCommand(command) || command == TitleSet || command == HistoryLink
}

func (a RawArgs) validatePlayoff(command string) error {
	var err error
	switch {
	case isBracketCommand(command):
		_, err = DecodeBracketChange(a, command)
	case command == TitleSet:
		_, err = DecodeTitleChange(a)
	default:
		_, err = DecodeHistoryLink(a)
	}
	return err
}

// playoffLabel describes a playoff entry for the ledger; franchises are named by their ids there.
func playoffLabel(r Record) string {
	identity := func(id string) string { return id }
	switch c := r.Command.Command; {
	case isBracketCommand(c):
		return bracketLabel(c, *r.Command.Args.Bracket, identity)
	case c == TitleSet:
		return titleLabel(*r.Command.Args.Title)
	}
	return linkLabel(*r.Command.Args.Link, identity)
}

// playoffPlacement says why a playoff command is recorded now for the league; it is empty for any other command.
func playoffPlacement(command string) string {
	if playoffCommand(command) {
		return "playoff changes, the champion's title and history links take effect now and are shown to every owner"
	}
	return ""
}
