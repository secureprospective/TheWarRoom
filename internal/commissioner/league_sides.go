package commissioner

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	poolsField = "players.pools"
	sidesField = "league.sides"
	// sidesCounted is the citation on the sides of a league whose player pools are per side.
	sidesCounted = "Counted from the conferences: each conference is a side"
	// noSides is the citation on the sides of a league with one player pool.
	noSides = "One player pool: the league has no sides"
)

// errPoolsInPlay refuses a player pools change once the league has played or moved a player.
var errPoolsInPlay = errors.New("player pools can't change once the season has games or player moves")

// countSides fills the league's sides from its conferences when its player pools are per side, and reads
// any other league as having none. Sides are counted, never set, so they never wait for a ruling.
func countSides(rules *leaguerules.RuleSet) {
	if leaguerules.PoolsPerSide(*rules) {
		rules.League.Sides = leaguerules.Value[map[string]string]{State: leaguerules.IsKnown,
			Value: leaguerules.CountedSides(*rules), Citation: sidesCounted}
		return
	}
	rules.League.Sides = leaguerules.Value[map[string]string]{State: leaguerules.IsKnown,
		Value: map[string]string{}, Citation: noSides}
}

// LeagueSides is each franchise's side as the league stands at its last entry; nil for a league with one
// player pool. Sides are fixed once play begins, so every entry played reads the same sides.
func LeagueSides(records []Record) (map[string]string, error) {
	if len(records) == 0 {
		return nil, nil
	}
	rules, err := RulesNow(records, records[len(records)-1].Receipt.RecordedAt)
	if err != nil {
		return nil, err
	}
	if !leaguerules.PoolsPerSide(rules) {
		return nil, nil
	}
	return rules.League.Sides.Value, nil
}

// sideField says a change of this rule can change the league's sides.
func sideField(field string) bool {
	switch field {
	case poolsField, alignmentField, conferencesField, teamsField, divisionsField:
		return true
	}
	return false
}

// checkSidesChange checks a change that could move the sides: sides are counted, not set; player pools
// change now, in an app league, before play; and once play begins no change may move a team or rename a
// side. With a player pool per side every team needs a side and the league at least two.
func checkSidesChange(raw RawCommand, records []Record, at time.Time) error {
	if raw.Command != RulesSet {
		return nil
	}
	if raw.Args.Field == sidesField {
		return Refused(errors.New("rule change refused: sides are counted from the conferences, each " +
			"conference a side; change a division's conference to move a team"))
	}
	if !sideField(raw.Args.Field) {
		return nil
	}
	if raw.Args.Field == poolsField {
		if err := checkPoolsChange(raw, records); err != nil {
			return err
		}
	}
	before, err := RulesNow(records, at)
	if err != nil {
		return nil //nolint:nilerr // The resolve check after this one refuses rules that cannot be read.
	}
	trial := withProposed(raw, records, at)
	after, err := RulesNow(trial, at)
	if err != nil || !leaguerules.PoolsPerSide(after) {
		return nil //nolint:nilerr // As above: an unreadable result is refused by the resolve check.
	}
	if playBegun(records) && !sameSides(before, after) {
		return Refused(errors.New("rule change refused: sides are fixed once the season has games or " +
			"player moves; this change would move a team to another side or rename a side"))
	}
	if err := completeSides(after.League.Sides.Value, LeagueFranchises(trial)); err != nil {
		return Refused(fmt.Errorf("rule change refused: %w", err))
	}
	return nil
}

// checkPoolsChange holds a player pools change to an app league, now, before play.
func checkPoolsChange(raw RawCommand, records []Record) error {
	if LeagueSource(records) != SourceNative {
		return Refused(errors.New("rule change refused: an imported league keeps MFL's one player pool"))
	}
	if raw.Effective != "now" {
		return Refused(errors.New("rule change refused: a Player pools change takes effect now; choose now"))
	}
	if playBegun(records) {
		return Refused(fmt.Errorf("rule change refused: %w", errPoolsInPlay))
	}
	return nil
}

func sameSides(before, after leaguerules.RuleSet) bool {
	if !leaguerules.PoolsPerSide(before) {
		return false
	}
	return maps.Equal(before.League.Sides.Value, after.League.Sides.Value)
}

// completeSides holds a league with a player pool per side to a side for every team, and two sides at least.
func completeSides(sides map[string]string, franchises []string) error {
	named := map[string]bool{}
	for _, id := range franchises {
		if sides[id] == "" {
			return errors.New("with a player pool per side every team needs a side: give every team a " +
				"division and every division a conference")
		}
		named[sides[id]] = true
	}
	if len(named) < 2 {
		return errors.New("a player pool per side needs at least two sides: put the divisions in two conferences or more")
	}
	return nil
}

// sidesWords names each side and the teams on it: "AFC: Team 1, Team 2; NFC: Team 3, Team 4".
func sidesWords(records []Record, sides leaguerules.Value[map[string]string]) (string, error) {
	if sides.State != leaguerules.IsKnown {
		return leaguerules.DescribeField(sidesField, sides).Words, nil
	}
	if len(sides.Value) == 0 {
		return "None: the league has one player pool", nil
	}
	names, err := franchiseNames(records)
	if err != nil {
		return "", err
	}
	bySide := map[string][]string{}
	for _, id := range LeagueFranchises(records) {
		if side := sides.Value[id]; side != "" {
			bySide[side] = append(bySide[side], cmpName(names, id))
		}
	}
	parts := make([]string, 0, len(bySide))
	for _, side := range slices.Sorted(maps.Keys(bySide)) {
		parts = append(parts, side+": "+strings.Join(bySide[side], ", "))
	}
	return strings.Join(parts, "; "), nil
}

func cmpName(names map[string]string, id string) string {
	if name := names[id]; name != "" {
		return name
	}
	return id
}
