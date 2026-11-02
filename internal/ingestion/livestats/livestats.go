// Package livestats turns a held live NFL box score into the stat lines the league's own scoring
// engine prices, so a game being played can be scored under the league's rules. It converts only
// what the live source actually reports; everything else is named as pending, never recorded as a
// zero, so a provisional score says plainly what it is still waiting for.
package livestats

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Authority names the live source a provisional line came from. The score ladder reads it: a
// league counts the newest live revision until the final feed lands.
const Authority = "ESPN live"

// Pending is what a live box score never reports. Scoring keeps these absent rather than zero, so
// a provisional total is marked incomplete and says what is still to come.
func Pending() []string {
	return []string{"Live source evidence is provisional: unreported stat fields and play distances remain pending until final evidence arrives"}
}

// Line is one player's live stat line as the live source reported it.
type Line struct {
	// SourceID is the live source's own athlete id; the app's crosswalk turns it into a league player.
	SourceID string
	Name     string
	Team     string
	Stats    fantasyscore.StatLine
	// Words is the box-score line in the source's own labels, so a reader can check the conversion.
	Words string
}

// column is one box-score cell this package counts: the source's label, and the league stats it
// records. A cell the source writes as a pair ("3-22" sacks taken and the yards they cost)
// records both stats in order.
type column struct {
	label string
	stats []leaguerules.StatID
}

func columns(category string) []column {
	switch category {
	case "passing":
		return []column{{"C/ATT", []leaguerules.StatID{"pass_completion", "pass_attempt"}},
			{"YDS", []leaguerules.StatID{"passing_yards"}}, {"TD", []leaguerules.StatID{"passing_td"}},
			{"INT", []leaguerules.StatID{"interception_thrown"}},
			{"SACKS", []leaguerules.StatID{"times_sacked", "times_sacked_yards"}}}
	case "rushing":
		return []column{{"CAR", []leaguerules.StatID{"rushing_attempt"}},
			{"YDS", []leaguerules.StatID{"rushing_yards"}}, {"TD", []leaguerules.StatID{"rushing_td"}}}
	case "receiving":
		return []column{{"REC", []leaguerules.StatID{"reception"}},
			{"YDS", []leaguerules.StatID{"receiving_yards"}}, {"TD", []leaguerules.StatID{"receiving_td"}},
			{"TGTS", []leaguerules.StatID{"target"}}}
	case "defensive":
		// TOT is total tackles; AST, when supplied, records assists separately.
		return []column{{"TOT", []leaguerules.StatID{"tackle"}}, {"AST", []leaguerules.StatID{"assist"}}, {"SACKS", []leaguerules.StatID{"sack"}},
			{"TFL", []leaguerules.StatID{"tackle_for_loss"}}, {"PD", []leaguerules.StatID{"pass_defensed"}},
			{"QB HTS", []leaguerules.StatID{"qb_hit"}}, {"TD", []leaguerules.StatID{"def_tds"}}}
	case "fumbles":
		return []column{{"FUM", []leaguerules.StatID{"fumbles_total"}}, {"LOST", []leaguerules.StatID{"fumble_lost"}}}
	case "interceptions":
		return []column{{"INT", []leaguerules.StatID{"interception_caught"}}, {"YDS", []leaguerules.StatID{"interception_return_yards"}}}
	case "kickReturns":
		return []column{{"YDS", []leaguerules.StatID{"kickoff_return_yards"}}}
	case "puntReturns":
		return []column{{"YDS", []leaguerules.StatID{"punt_return_yards"}}}
	case "kicking":
		return []column{{"XP", []leaguerules.StatID{"pat_made", "pat_attempt"}}}
	default:
		return nil
	}
}

// decimal reads one box-score cell. An empty cell or the source's own dash is absent, not zero.
func decimal(raw string) (leaguerules.Decimal, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.Trim(raw, "-") == "" {
		return 0, false, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false, fmt.Errorf("read %q: %w", raw, err)
	}
	scaled := value * 10000
	if scaled > 1<<53 || scaled < -(1<<53) {
		return 0, false, fmt.Errorf("read %q: outside the recordable range", raw)
	}
	return leaguerules.Decimal(scaled), true, nil
}

// cell records one column's stats from one cell. A paired cell ("19/30", "3-22") is split on its
// separator; a cell with fewer parts than the column names records only the parts it has.
func cell(line *fantasyscore.StatLine, col column, raw string) error {
	parts := []string{raw}
	if len(col.stats) > 1 {
		parts = strings.FieldsFunc(raw, func(r rune) bool { return r == '/' || r == '-' })
	}
	for i, id := range col.stats {
		if i >= len(parts) {
			return nil
		}
		value, ok, err := decimal(parts[i])
		if err != nil {
			return fmt.Errorf("%s: %w", col.label, err)
		}
		if !ok {
			continue
		}
		line.Quantities[id] = value
		line.Labels[id] = string(id)
	}
	return nil
}

// Lines is every live stat line a held game reports, one per athlete the box score names. A game
// with no box score yet has none. Two categories naming the same athlete (a quarterback who ran)
// are merged into one line.
func Lines(game livegame.Game) ([]Line, error) {
	order := []string{}
	byID := map[string]*Line{}
	for _, box := range game.Box {
		cols := columns(box.Category)
		if len(cols) == 0 {
			continue
		}
		for _, row := range box.Rows {
			if row.ID == "" {
				continue
			}
			held, ok := byID[row.ID]
			if !ok {
				order = append(order, row.ID)
				held = &Line{SourceID: row.ID, Name: row.Name, Team: box.Team,
					Stats: fantasyscore.StatLine{Quantities: map[leaguerules.StatID]leaguerules.Decimal{},
						Labels: map[leaguerules.StatID]string{}}}
				byID[row.ID] = held
			}
			words, err := rowStats(held, box, cols, row)
			if err != nil {
				return nil, fmt.Errorf("live box score for %s: %w", row.Name, err)
			}
			held.Words = strings.TrimSpace(held.Words + " " + box.Category + " " + words)
		}
	}
	out := make([]Line, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out, nil
}

func rowStats(held *Line, box livegame.Box, cols []column, row livegame.BoxRow) (string, error) {
	words := []string{}
	for _, col := range cols {
		index := -1
		for i, label := range box.Labels {
			if label == col.label {
				index = i
			}
		}
		if index < 0 || index >= len(row.Stats) {
			continue
		}
		if err := cell(&held.Stats, col, row.Stats[index]); err != nil {
			return "", err
		}
		words = append(words, col.label+" "+row.Stats[index])
	}
	return strings.Join(words, ", "), nil
}
