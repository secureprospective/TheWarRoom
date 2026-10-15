package commissioner

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Where the bracket stands.
const (
	bracketUnavailable = "Unavailable"
	bracketNeeds       = "Needs a ruling"
	bracketWaiting     = "Waiting for the regular season"
	bracketReady       = "Ready to build"
	bracketBuilt       = "Built"
	bracketArchived    = "Archived"
)

// PlayoffRulesView is the playoff rules a season runs under, in plain values.
type PlayoffRulesView struct {
	Words           string              `json:"words"`
	PerConference   int                 `json:"perConference"`
	DivisionWinners int                 `json:"divisionWinners"`
	WildCards       int                 `json:"wildCards"`
	ByeSeeds        int                 `json:"byeSeeds"`
	Seeding         string              `json:"seeding"`
	Reseed          bool                `json:"reseed"`
	StartWeek       int                 `json:"startWeek"`
	FinalWeek       int                 `json:"finalWeek"`
	HomeBonus       leaguerules.Decimal `json:"homeBonus"`
	Shape           []BracketNodeSpec   `json:"shape"`
}

// BracketChampion is the season's champion and the title it carries. Source says where the champion comes from.
type BracketChampion struct {
	Franchise string `json:"franchise"`
	Name      string `json:"name"`
	Label     string `json:"label"`
	Source    string `json:"source"`
	Record    string `json:"record"`
}

// HistoryRecord is a franchise's history link, read with the franchise's name.
type HistoryRecord struct {
	Record         string `json:"record"`
	Franchise      string `json:"franchise"`
	Name           string `json:"name"`
	ByCommissioner bool   `json:"byCommissioner"`
	Reason         string `json:"reason,omitempty"`
}

// NativeBracket is a season's playoffs: the rules, each conference's seeds, every round's games, who was set
// by hand, and the champion once the final is decided. It never guesses: Words says why a bracket is missing.
type NativeBracket struct {
	League     string            `json:"league"`
	Name       string            `json:"name"`
	Season     int               `json:"season"`
	Revision   int64             `json:"revision"`
	Status     string            `json:"status"`
	Words      string            `json:"words"`
	Preset     string            `json:"preset"`
	Presets    []string          `json:"presets"`
	Rules      PlayoffRulesView  `json:"rules"`
	Started    bool              `json:"started"`
	StartKnown bool              `json:"startKnown"`
	Fields     []ConferenceField `json:"fields"`
	Rounds     []BracketRound    `json:"rounds"`
	Champion   *BracketChampion  `json:"champion,omitempty"`
	Title      BracketTitle      `json:"title"`
	History    []HistoryRecord   `json:"history"`
	Changes    []ScoreChange     `json:"changes"`
}

func rulesView(r PlayoffRules) PlayoffRulesView {
	view := PlayoffRulesView{Words: r.Words, PerConference: r.PerConference, DivisionWinners: r.DivisionWinners,
		WildCards: r.WildCards, ByeSeeds: r.ByeSeeds, Seeding: "Division winners first, then wild cards", Reseed: r.Reseed,
		StartWeek: r.StartWeek, FinalWeek: r.FinalWeek, HomeBonus: r.HomeBonus, Shape: []BracketNodeSpec{}}
	for _, n := range r.Shape {
		view.Shape = append(view.Shape, BracketNodeSpec{ID: n.ID, Week: n.Week, HomeSeed: n.Home.Seed, AwaySeed: n.Away.Seed,
			HomeFrom: n.Home.From, AwayFrom: n.Away.From})
	}
	return view
}

// NativeBracketReading builds a season's playoffs from its final standings. facts describe the first playoff
// week's kickoffs and may be empty, in which case it is not known whether the playoffs have begun.
func NativeBracketReading(ctx context.Context, league string, records []Record, src ResultSources, season int,
	now time.Time, facts LineupFacts) (NativeBracket, error) {
	if season < 1920 || season > 9999 {
		return NativeBracket{}, fmt.Errorf("choose a season")
	}
	out := NativeBracket{League: league, Season: season, Revision: int64(len(records)), Status: bracketUnavailable,
		Presets: []string{}, Fields: []ConferenceField{}, Rounds: []BracketRound{}, History: []HistoryRecord{},
		Changes: []ScoreChange{}, Rules: PlayoffRulesView{Shape: []BracketNodeSpec{}}}
	if len(records) == 0 {
		return out, fmt.Errorf("this app league has not been created")
	}
	out.Name = Replay(records).Name
	if !seeded(records) {
		out.Words = "Record the starting snapshot first"
		return out, nil
	}
	b := bracketReader{league: league, records: records, src: src, season: season, now: now, out: &out}
	if err := b.prepare(facts); err != nil {
		return out, err
	}
	if out.Status != bracketUnavailable {
		return out, nil
	}
	return out, b.build(ctx)
}

// bracketReader carries one reading's inputs through its steps.
type bracketReader struct {
	league    string
	records   []Record
	src       ResultSources
	season    int
	now       time.Time
	out       *NativeBracket
	rules     leaguerules.RuleSet
	directory NativeDirectory
	state     bracketState
	pr        PlayoffRules
	layout    conferenceLayout
}

func (b *bracketReader) stop(status, words string) { b.out.Status, b.out.Words = status, words }

// prepare reads everything a bracket needs except the standings, and stops with plain words when it cannot go on.
func (b *bracketReader) prepare(facts LineupFacts) error {
	at := ReadingInstant(b.records, b.now)
	var err error
	if b.directory, err = NativeDirectoryReading(b.league, b.records, b.now); err != nil {
		return err
	}
	names := b.directory.seats()
	b.state = foldBracket(b.records, b.season, func(id string) string { return names.of(id).Name })
	b.out.Changes = b.state.Changes
	b.out.Presets = append(b.out.Presets, slices.Sorted(maps.Keys(b.state.Presets))...)
	b.out.Preset = cmpPreset(b.state.Preset)
	b.out.Title = TitleReading(b.records, b.season)
	if b.out.History, err = historyRecords(b.league, b.records, names); err != nil {
		return err
	}
	if first, known := firstKickoff(facts); known {
		b.out.StartKnown, b.out.Started = true, !b.now.Before(first)
	}
	first, err := firstSeason(b.records)
	if err != nil {
		return err
	}
	current, err := SeasonAt(b.records, at)
	if err != nil {
		return err
	}
	switch {
	case b.season < first:
		b.out.Champion = b.linkedChampion(nil)
		b.stop(bracketArchived, fmt.Sprintf("Season %d was played before this league moved into the app, so only its champion is kept", b.season))
		return nil
	case b.season > current:
		b.stop(bracketWaiting, fmt.Sprintf("Season %d has not been played yet", b.season))
		return nil
	}
	return b.readRules(at)
}

func cmpPreset(id string) string {
	if id == "" {
		return presetLeagueRules
	}
	return id
}

func (b *bracketReader) readRules(at time.Time) error {
	var err error
	if b.rules, err = RulesInForce(b.records, b.season, at); err != nil {
		return err
	}
	graph, words := b.state.graph(b.rules)
	if words != "" {
		b.stop(bracketNeeds, words)
		return nil
	}
	b.pr = ResolvePlayoffRules(b.rules, graph)
	b.out.Rules = rulesView(b.pr)
	if b.pr.Words != "" {
		b.stop(bracketNeeds, b.pr.Words)
		return nil
	}
	b.layout = conferencesFor(b.records, b.rules, directoryIDs(b.directory))
	if b.layout.Words != "" {
		b.stop(bracketNeeds, b.layout.Words)
	}
	return nil
}

// historyRecords lists every history link, read through the franchise names.
func historyRecords(league string, records []Record, names seatIndex) ([]HistoryRecord, error) {
	links, err := HistoryLinks(league, records)
	if err != nil {
		return nil, err
	}
	out := []HistoryRecord{}
	for _, record := range slices.Sorted(maps.Keys(links)) {
		l := links[record]
		out = append(out, HistoryRecord{Record: record, Franchise: l.Franchise, Name: names.of(l.Franchise).Name,
			ByCommissioner: l.ByCommissioner, Reason: l.Reason})
	}
	return out, nil
}

// linkedChampion is the champion a history link names for the season, or the one the final decided.
func (b *bracketReader) linkedChampion(won *BracketSide) *BracketChampion {
	record := championRecord(b.season)
	var linked *HistoryRecord
	for i := range b.out.History {
		if b.out.History[i].Record == record {
			linked = &b.out.History[i]
		}
	}
	switch {
	case linked != nil && (linked.ByCommissioner || won == nil):
		source := "Kept in the league's history"
		if linked.ByCommissioner {
			source = "Set by the commissioner's history link"
		}
		return &BracketChampion{Franchise: linked.Franchise, Name: linked.Name, Label: b.out.Title.Label, Source: source,
			Record: record}
	case won != nil:
		return &BracketChampion{Franchise: won.Franchise, Name: won.Name, Label: b.out.Title.Label, Source: "Won the final",
			Record: record}
	}
	return nil
}
