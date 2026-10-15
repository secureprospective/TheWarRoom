package commissioner

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// StatWeek is the held evidence for one league player's regular-season week.
type StatWeek struct {
	Player   string
	Week     int
	Evidence source.NativeEvidence
}

// StatFeed supplies every league player's held weeks for a season, and the names to call the players by.
type StatFeed interface {
	Weeks(ctx context.Context) ([]StatWeek, error)
	Name(player string) string
}

// StatChange is one thing the NFL changed in a player's week, in the words a GM reads.
type StatChange struct {
	Stat   string `json:"stat"`
	Label  string `json:"label"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// StatQueueItem is a change the NFL made that the league has not yet decided on.
type StatQueueItem struct {
	ID      string       `json:"id"`
	Player  string       `json:"player"`
	Name    string       `json:"name"`
	Season  int          `json:"season"`
	Week    int          `json:"week"`
	Changes []StatChange `json:"changes"`
	// Points is the change in the player's points if the change is counted; absent when it cannot be known yet.
	Points    *leaguerules.Decimal `json:"points,omitempty"`
	Flip      string               `json:"flip"`
	FlipWords string               `json:"flipWords"`
	Words     string               `json:"words"`
	Arrived   time.Time            `json:"arrived"`
	SHA256    string               `json:"sha256"`
	// flip is the full verdict the preview of approving the item shows.
	flip ResultFlip
}

// StatDecision is a decision already made about a player's statistics, with who made it and when.
type StatDecision struct {
	Entry  string    `json:"entry"`
	Kind   string    `json:"kind"`
	Status string    `json:"status"`
	Player string    `json:"player"`
	Name   string    `json:"name"`
	Season int       `json:"season"`
	Week   int       `json:"week"`
	Words  string    `json:"words"`
	By     string    `json:"by"`
	At     time.Time `json:"at"`
	Reason string    `json:"reason"`
}

// StatQueue is the statistics the NFL changed after the league counted them, waiting for a decision.
type StatQueue struct {
	League   string          `json:"league"`
	Season   int             `json:"season"`
	Revision int64           `json:"revision"`
	Words    string          `json:"words"`
	Open     []StatQueueItem `json:"open"`
	Decided  []StatDecision  `json:"decided"`
}

// StatQueueReading compares the newest version of every player's week with the version the league counts.
// Only a week whose lines differ makes an item, and a turned-down item stays out until the feed changes again.
func StatQueueReading(ctx context.Context, league string, records []Record, feed StatFeed, src ResultSources,
	season int, now time.Time) (StatQueue, error) {
	out := StatQueue{League: league, Season: season, Revision: int64(len(records)), Open: []StatQueueItem{},
		Decided: []StatDecision{}}
	if len(records) == 0 {
		return out, fmt.Errorf("this app league has not been created")
	}
	if season < 1920 || season > 9999 {
		return out, fmt.Errorf("choose a season")
	}
	if !seeded(records) {
		out.Words = "Record the starting snapshot first"
		return out, nil
	}
	rules, err := RulesInForce(records, season, ReadingInstant(records, now))
	if err != nil {
		return out, err
	}
	effects, err := foldStatEffects(records)
	if err != nil {
		return out, err
	}
	weeks, err := feed.Weeks(ctx)
	if err != nil {
		return out, fmt.Errorf("read the held weekly statistics: %w", err)
	}
	builder := queueBuilder{league: league, season: season, records: records, feed: feed, src: src, rules: rules,
		effects: effects, now: now}
	for _, week := range weeks {
		item, waiting, err := builder.item(ctx, week)
		if err != nil {
			return out, err
		}
		if waiting {
			out.Open = append(out.Open, item)
		}
	}
	slices.SortFunc(out.Open, compareQueueItems)
	out.Decided, err = builder.decisions(weeks)
	if err != nil {
		return out, err
	}
	out.Words = queueWords(len(out.Open))
	return out, nil
}

func queueWords(open int) string {
	switch open {
	case 0:
		return "No statistics are waiting for a decision"
	case 1:
		return "1 statistic change is waiting for a decision"
	}
	return fmt.Sprintf("%d statistic changes are waiting for a decision", open)
}

// compareQueueItems puts the most recent arrival first and breaks ties the same way every time.
func compareQueueItems(a, b StatQueueItem) int {
	switch {
	case a.Arrived.After(b.Arrived):
		return -1
	case b.Arrived.After(a.Arrived):
		return 1
	case a.Player != b.Player:
		return strings.Compare(a.Player, b.Player)
	}
	return a.Week - b.Week
}

type queueBuilder struct {
	league  string
	season  int
	records []Record
	feed    StatFeed
	src     ResultSources
	rules   leaguerules.RuleSet
	effects statEffects
	now     time.Time
}

// item makes the queue entry for one week, or reports that nothing is waiting.
func (b queueBuilder) item(ctx context.Context, week StatWeek) (StatQueueItem, bool, error) {
	item, waiting, err := b.candidate(week)
	if err != nil || !waiting {
		return StatQueueItem{}, false, err
	}
	if err := b.price(ctx, &item, week); err != nil {
		return StatQueueItem{}, false, err
	}
	return item, true, nil
}

// candidate is the change the NFL has made to a week the league counted, before it is priced. It is not
// waiting when the newest version counts the same, or when the league already turned that version down.
func (b queueBuilder) candidate(week StatWeek) (StatQueueItem, bool, error) {
	versions := week.Evidence.Versions()
	key := statKey{week.Player, b.season, week.Week}
	counted, err := countedVersion(versions, b.effects.baselines[key])
	if err != nil || counted == nil {
		return StatQueueItem{}, false, err
	}
	newest := versions[len(versions)-1]
	id := StatItemID{Season: b.season, Week: week.Week, Player: week.Player, Seq: newest.Seq, Digest: newest.Digest[:statDigestLen]}
	if newest.Digest == counted.Digest || b.effects.rejected[id] {
		return StatQueueItem{}, false, nil
	}
	return StatQueueItem{ID: id.String(), Player: week.Player, Name: b.name(week.Player), Season: b.season,
		Week: week.Week, Changes: b.differences(*counted.Held.Week, *newest.Held.Week), Arrived: newest.Held.FetchedAt,
		SHA256: newest.Held.SHA256}, true, nil
}

func (b queueBuilder) name(player string) string {
	if name := b.feed.Name(player); name != "" {
		return name
	}
	return "player " + player
}

// price fills in the points the change is worth and whether it turns over a result, each read from the
// same readings that would follow the approval.
func (b queueBuilder) price(ctx context.Context, item *StatQueueItem, week StatWeek) error {
	raw := RawCommand{League: b.league, Command: StatApprove, Request: "queue." + item.ID,
		Args: RawArgs{Stat: &RawStatChange{Item: item.ID}}}
	scope := source.RawNativeScope{Player: week.Player, Season: b.season, Week: week.Week}
	before, err := NativePlayerScoreReading(b.league, scope, b.records, week.Evidence, b.now)
	if err != nil {
		return err
	}
	after, err := NativePlayerScoreReading(b.league, scope, scoreTrial(raw, b.records), week.Evidence, b.now)
	if err != nil {
		return err
	}
	item.Points = pointsChange(before, after)
	flip, err := ResultFlipCheck(ctx, b.league, b.records, b.src, raw, b.season, week.Week, b.now)
	if err != nil {
		return err
	}
	item.Flip, item.FlipWords, item.flip = flip.Status, flip.Words, flip
	item.Words = fmt.Sprintf("%s, week %d: %s, %s, %s", item.Name, week.Week, changesWords(item.Changes),
		signedPointsWords(item.Points), flipSummary(flip.Status))
	return nil
}

// total is the points a reading counts, final or so far; absent while the reading has no number.
func (r NativePlayerScore) total() *leaguerules.Decimal {
	if r.Points != nil {
		return r.Points
	}
	return r.PartialPoints
}

func pointsChange(before, after NativePlayerScore) *leaguerules.Decimal {
	was, now := before.total(), after.total()
	if was == nil || now == nil {
		return nil
	}
	change := *now - *was
	return &change
}

func signedPointsWords(change *leaguerules.Decimal) string {
	switch {
	case change == nil:
		return "the change in points is not known yet"
	case *change == 0:
		return "no change in points"
	case *change == 10000 || *change == -10000:
		return signed(*change) + " point"
	}
	return signed(*change) + " points"
}

func signed(points leaguerules.Decimal) string {
	if points > 0 {
		return "+" + scorePointsWords(points)
	}
	return scorePointsWords(points)
}

func flipSummary(status string) string {
	switch status {
	case FlipNo:
		return "no result changes"
	case FlipYes:
		return "a result changes"
	}
	return "we can't yet tell whether a result changes"
}

func changesWords(changes []StatChange) string {
	parts := make([]string, len(changes))
	for i, c := range changes {
		parts[i] = fmt.Sprintf("%s %s → %s", c.Label, c.Before, c.After)
	}
	return strings.Join(parts, "; ")
}

// differences lists, in a fixed order, what changed between two versions of a week.
func (b queueBuilder) differences(before, after source.PlayerWeek) []StatChange {
	out := []StatChange{}
	if before.Position != after.Position {
		out = append(out, StatChange{Stat: "position", Label: "position", Before: orNone(before.Position),
			After: orNone(after.Position)})
	}
	ids := map[leaguerules.StatID]bool{}
	for id := range before.Line.Quantities {
		ids[id] = true
	}
	for id := range after.Line.Quantities {
		ids[id] = true
	}
	sorted := make([]leaguerules.StatID, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	slices.Sort(sorted)
	for _, id := range sorted {
		was, hadBefore := before.Line.Quantities[id]
		now, hadAfter := after.Line.Quantities[id]
		if hadBefore == hadAfter && was == now {
			continue
		}
		out = append(out, StatChange{Stat: string(id), Label: b.statLabel(id, after, before),
			Before: quantityWords(was, hadBefore), After: quantityWords(now, hadAfter)})
	}
	return out
}

func orNone(position string) string {
	if position == "" {
		return "none"
	}
	return position
}

func quantityWords(quantity leaguerules.Decimal, recorded bool) string {
	if !recorded {
		return "not recorded"
	}
	return scorePointsWords(quantity)
}

// statLabel names a stat the way the league's scoring does, in lower case, falling back to the feed's label.
func (b queueBuilder) statLabel(id leaguerules.StatID, weeks ...source.PlayerWeek) string {
	if b.rules.Scoring.Catalogue.State == leaguerules.IsKnown {
		if stat, ok := b.rules.Scoring.Catalogue.Value[id]; ok && stat.Label.State == leaguerules.IsKnown {
			return lowerFirst(stat.Label.Value)
		}
	}
	for _, week := range weeks {
		if label := week.Line.Labels[id]; label != "" {
			return lowerFirst(label)
		}
	}
	return strings.ReplaceAll(string(id), "_", " ")
}

// lowerFirst makes a label read inside a sentence, leaving acronyms such as "QB hits" alone.
func lowerFirst(label string) string {
	runes := []rune(label)
	if len(runes) == 0 || (len(runes) > 1 && unicode.IsUpper(runes[1])) {
		return label
	}
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
