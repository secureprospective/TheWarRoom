package market

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// RawContractDecisionsScope deliberately has no franchise field; the service derives the owner's seat.
type RawContractDecisionsScope struct {
	League     string `json:"league"`
	ViewerZone string `json:"viewerZone"`
}

func (r RawContractDecisionsScope) Validate() error {
	if err := (commissioner.RawNativeCapScope{League: r.League, ViewerZone: r.ViewerZone}).Validate(); err != nil {
		return fmt.Errorf("contract decisions scope: %w", err)
	}
	return nil
}

type ContractDecisionChoice struct {
	Years int           `json:"years"`
	Level int           `json:"level"`
	Quote ContractQuote `json:"quote"`
}

type ContractDecisionOption struct {
	Action   string                   `json:"action"`
	Deadline time.Time                `json:"deadline"`
	OpensAt  *time.Time               `json:"opensAt,omitempty"`
	Open     bool                     `json:"open"`
	Past     bool                     `json:"pastDeadline"`
	Choices  []ContractDecisionChoice `json:"choices"`
}

type RecordedContractDecision struct {
	Action string    `json:"action"`
	Years  int       `json:"years"`
	Level  int       `json:"level"`
	At     time.Time `json:"at"`
}

type ContractDecisionPlayer struct {
	Player     string                    `json:"player"`
	Name       string                    `json:"name"`
	Expiration int                       `json:"expiration"`
	Decision   *RecordedContractDecision `json:"decision,omitempty"`
	Options    []ContractDecisionOption  `json:"options"`
}

type ContractDecisions struct {
	League    string                   `json:"league"`
	Franchise string                   `json:"franchise"`
	Revision  int64                    `json:"revision"`
	Season    int                      `json:"season"`
	Words     string                   `json:"words"`
	Players   []ContractDecisionPlayer `json:"players"`
}

func OwnerContractDecisionActions() []nativecheck.ContractAction {
	return []nativecheck.ContractAction{nativecheck.ActionExtension, nativecheck.ActionTag,
		nativecheck.ActionTender, nativecheck.ActionResign}
}

// ContractDecisionsReading quotes each rule-defined action on this owner's current-season expiring players.
func ContractDecisionsReading(scope RawContractDecisionsScope, franchise string, records []commissioner.Record,
	now time.Time, local *time.Location, facts nativecheck.Facts) (ContractDecisions, error) {
	if err := scope.Validate(); err != nil {
		return ContractDecisions{}, fmt.Errorf("contract decisions input: %w", err)
	}
	base, err := commissioner.ReadNativeBase(scope.League, records, now)
	if err != nil {
		return ContractDecisions{}, fmt.Errorf("read contract decision rules: %w", err)
	}
	if !base.Assets.Seeded || base.Assets.Snapshot == nil {
		return ContractDecisions{}, fmt.Errorf("contract decisions: %w",
			commissioner.Refused(errors.New("record starting rosters before reading contract decisions")))
	}
	if local == nil {
		return ContractDecisions{}, fmt.Errorf("choose the owner's time zone")
	}
	out := ContractDecisions{League: scope.League, Franchise: franchise, Revision: int64(len(records)),
		Season: base.Rules.Season, Players: []ContractDecisionPlayer{},
		Words: "Only rule-enabled, priceable options with a dated league deadline are listed. No walk action is defined; an untouched expiring contract follows the existing rollover pending process."}
	if gap := contractGapWords(base.Assets.Snapshot, franchise); gap != "" {
		out.Words = gap
		return out, nil
	}
	reader := decisionReader{scope: scope, franchise: franchise, records: records, now: now, local: local,
		facts: facts, rules: base.Rules}
	for _, held := range base.Assets.Snapshot.Players {
		if held.Franchise != franchise {
			continue
		}
		row, ok, err := reader.row(held)
		if err != nil {
			return ContractDecisions{}, err
		}
		if ok {
			out.Players = append(out.Players, row)
		}
	}
	slices.SortFunc(out.Players, func(a, b ContractDecisionPlayer) int {
		return cmpText(a.Name, b.Name)
	})
	if len(out.Players) == 0 {
		out.Words = "No priceable owner contract decisions with a dated deadline are due under the current rules and calendar. " +
			"No walk action is defined; an untouched expiring contract follows the existing rollover pending process."
	}
	return out, nil
}

// contractGapWords is the snapshot's own words when this franchise's contracts cannot be priced yet.
func contractGapWords(snapshot *commissioner.RawNativeSnapshot, franchise string) string {
	if err := snapshot.CheckContracts(franchise); err != nil {
		return err.Error()
	}
	return ""
}

type decisionReader struct {
	scope     RawContractDecisionsScope
	franchise string
	records   []commissioner.Record
	now       time.Time
	local     *time.Location
	facts     nativecheck.Facts
	rules     leaguerules.RuleSet
}

// row is one held player's decisions, priced against the contract before any owner choice this season.
func (d decisionReader) row(held commissioner.NativePlayer) (ContractDecisionPlayer, bool, error) {
	season := d.rules.Season
	player := held
	if before, found := commissioner.ContractDecisionBaseline(d.records, d.franchise, held.ID, season); found {
		player = before
	}
	if player.ExpirationYear != season || player.Pending {
		return ContractDecisionPlayer{}, false, nil
	}
	row := ContractDecisionPlayer{Player: player.ID, Name: decisionPlayerName(player, d.facts),
		Expiration: player.ExpirationYear, Options: []ContractDecisionOption{}}
	decision, hasDecision := commissioner.LatestContractDecision(d.records, d.franchise, held.ID, season)
	if hasDecision {
		choice := decision.Command.Args.ContractDecision
		row.Decision = &RecordedContractDecision{Action: choice.Action, Years: choice.Years,
			Level: choice.Level, At: decision.Receipt.RecordedAt}
	}
	for _, action := range OwnerContractDecisionActions() {
		option, ok, err := contractDecisionOption(d.scope, d.franchise, player, action, d.records, d.now, d.local,
			d.facts, d.rules)
		if err != nil {
			return ContractDecisionPlayer{}, false, err
		}
		if ok {
			row.Options = append(row.Options, option)
		}
	}
	return row, len(row.Options) > 0 || hasDecision, nil
}

func contractDecisionOption(scope RawContractDecisionsScope, franchise string, player commissioner.NativePlayer,
	action nativecheck.ContractAction, records []commissioner.Record, now time.Time, local *time.Location,
	facts nativecheck.Facts, rules leaguerules.RuleSet) (ContractDecisionOption, bool, error) {
	window, ok := decisionWindow(records, scope.League, franchise, action, rules.Season, now, local)
	if !ok {
		return ContractDecisionOption{}, false, nil
	}
	quotes := decisionQuoteInputs(action, rules.Contracts)
	option := ContractDecisionOption{Action: string(action), Deadline: window.deadline,
		OpensAt: window.opensAt, Open: window.open, Past: !now.Before(window.deadline), Choices: []ContractDecisionChoice{}}
	for _, input := range quotes {
		quote, err := QuoteContract(RawContractQuote{League: scope.League, Franchise: franchise, Player: player.ID,
			Action: string(action), ViewerZone: scope.ViewerZone, Years: input.years, Level: input.level},
			records, now, local, facts)
		if err != nil {
			return ContractDecisionOption{}, false, fmt.Errorf("quote %s for %s: %w", action, player.ID, err)
		}
		if !quote.Priceable {
			continue
		}
		option.Choices = append(option.Choices, ContractDecisionChoice{Years: input.years, Level: input.level, Quote: quote})
	}
	return option, len(option.Choices) > 0, nil
}

type decisionWindowDetails struct {
	deadline time.Time
	opensAt  *time.Time
	open     bool
}

func decisionWindow(records []commissioner.Record, league, franchise string, action nativecheck.ContractAction,
	season int, now time.Time, local *time.Location) (decisionWindowDetails, bool) {
	calendar, err := commissioner.CalendarReading(league, records, now, season, leaguerules.Action(action), franchise, local)
	if err != nil {
		return decisionWindowDetails{}, false
	}
	pick := windowPick{now: now}
	for _, window := range calendar.Clock.Windows {
		if slices.Contains(window.Opens, leaguerules.Action(action)) && window.Close != nil {
			pick.consider(leagueWindow{start: window.Start, close: *window.Close})
		}
	}
	for _, exception := range calendar.Clock.Exceptions {
		if exception.Action == leaguerules.Action(action) && exception.Franchise == franchise && !exception.Permanent {
			pick.consider(leagueWindow{start: exception.Start, close: exception.Until})
		}
	}
	selected := pick.selected()
	if selected == nil {
		return decisionWindowDetails{}, false
	}
	var opensAt *time.Time
	if now.Before(selected.start) {
		start := selected.start
		opensAt = &start
	}
	return decisionWindowDetails{deadline: selected.close, opensAt: opensAt,
		open: calendar.Decision.Open}, true
}

type leagueWindow struct{ start, close time.Time }

// windowPick keeps the soonest-closing live window, else the next to open, else the latest closed one.
type windowPick struct {
	now                  time.Time
	live, upcoming, past *leagueWindow
}

func (p *windowPick) consider(candidate leagueWindow) {
	switch {
	case !p.now.Before(candidate.start) && p.now.Before(candidate.close):
		if p.live == nil || candidate.close.Before(p.live.close) {
			p.live = &candidate
		}
	case p.now.Before(candidate.start):
		if p.upcoming == nil || candidate.start.Before(p.upcoming.start) {
			p.upcoming = &candidate
		}
	case !candidate.close.IsZero():
		if p.past == nil || candidate.close.After(p.past.close) {
			p.past = &candidate
		}
	}
}

func (p *windowPick) selected() *leagueWindow {
	switch {
	case p.live != nil:
		return p.live
	case p.upcoming != nil:
		return p.upcoming
	default:
		return p.past
	}
}

type decisionTerms struct{ years, level int }

// decisionQuoteInputs lists the terms the league's contract rules offer for each owner decision.
func decisionQuoteInputs(action nativecheck.ContractAction, rules leaguerules.ContractsRules) []decisionTerms {
	terms := map[nativecheck.ContractAction]func(leaguerules.ContractsRules) []decisionTerms{
		nativecheck.ActionTag:       func(leaguerules.ContractsRules) []decisionTerms { return []decisionTerms{{}} },
		nativecheck.ActionExtension: extensionTerms,
		nativecheck.ActionTender:    tenderTerms,
		nativecheck.ActionResign:    resignTerms,
	}[action]
	if terms == nil {
		return []decisionTerms{}
	}
	return terms(rules)
}

func extensionTerms(rules leaguerules.ContractsRules) []decisionTerms {
	terms := []decisionTerms{}
	if rules.Extension.State != leaguerules.IsKnown || rules.Extension.Value.AddedYears.State != leaguerules.IsKnown {
		return terms
	}
	for years := 1; years <= rules.Extension.Value.AddedYears.Value; years++ {
		terms = append(terms, decisionTerms{years: years})
	}
	return terms
}

func tenderTerms(rules leaguerules.ContractsRules) []decisionTerms {
	terms := []decisionTerms{}
	if rules.Tender.State != leaguerules.IsKnown || rules.Tender.Value.Prices.State != leaguerules.IsKnown {
		return terms
	}
	for level := range rules.Tender.Value.Prices.Value {
		terms = append(terms, decisionTerms{level: level + 1})
	}
	return terms
}

func resignTerms(rules leaguerules.ContractsRules) []decisionTerms {
	levels := tenderTerms(rules)
	if rules.ReSign.State != leaguerules.IsKnown || rules.ReSign.Value.Premiums.State != leaguerules.IsKnown {
		return []decisionTerms{}
	}
	years := make([]int, 0, len(rules.ReSign.Value.Premiums.Value))
	for term := range rules.ReSign.Value.Premiums.Value {
		years = append(years, term)
	}
	sort.Ints(years)
	terms := []decisionTerms{}
	for _, term := range years {
		for _, level := range levels {
			terms = append(terms, decisionTerms{years: term, level: level.level})
		}
	}
	return terms
}

func decisionPlayerName(player commissioner.NativePlayer, facts nativecheck.Facts) string {
	if name := facts[player.ID].Name; name != "" {
		return name
	}
	if player.Name != "" {
		return player.Name
	}
	return "Player " + player.ID
}

func cmpText(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
