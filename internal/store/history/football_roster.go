package history

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// nflRoster is nflverse's player list keyed by plain name: each name's recent players and their groups.
type nflRoster map[string][]rosterPlayer

type rosterPlayer struct{ gsis, group string }

// rosterGroups is the nflverse position groups an MFL position can be listed under.
var rosterGroups = map[leaguerules.Position][]string{ //nolint:gochecknoglobals // fixed lookup
	leaguerules.QB: {"QB"}, leaguerules.RB: {"RB"}, leaguerules.WR: {"WR"}, leaguerules.TE: {"TE"},
	leaguerules.PK: {"SPEC"}, leaguerules.DT: {"DL"}, leaguerules.DE: {"DL", "LB"}, leaguerules.LB: {"LB", "DL"},
	leaguerules.CB: {"DB"}, leaguerules.S: {"DB"},
}

// readRoster keeps the players of players.csv on an NFL roster in season or the one before it.
func readRoster(input io.Reader, season int) (nflRoster, error) {
	if input == nil {
		return nil, nil
	}
	reader := csv.NewReader(input)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read nflverse players: %w", err)
	}
	at := map[string]int{}
	for i, name := range header {
		at[name] = i
	}
	for _, name := range []string{"gsis_id", "display_name", "position_group", "last_season"} {
		if _, ok := at[name]; !ok {
			return nil, fmt.Errorf("nflverse players has no %s column", name)
		}
	}
	out := nflRoster{}
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read nflverse players: %w", err)
		}
		last, err := strconv.Atoi(row[at["last_season"]])
		if err != nil || last < season-1 || !source.IsGSIS(row[at["gsis_id"]]) {
			continue
		}
		key := plainName(row[at["display_name"]])
		out[key] = append(out[key], rosterPlayer{gsis: row[at["gsis_id"]], group: row[at["position_group"]]})
	}
}

// match is the one recent player with name ("Last, First" as MFL writes it) in a group position allows.
func (r nflRoster) match(name string, position leaguerules.Position) string {
	if last, first, ok := strings.Cut(name, ", "); ok {
		name = first + " " + last
	}
	found := ""
	for _, p := range r[plainName(name)] {
		if !slices.Contains(rosterGroups[position], p.group) {
			continue
		}
		if found != "" {
			return ""
		}
		found = p.gsis
	}
	return found
}

// plainName is a name in lower-case letters and single spaces: "James Thompson Jr." is "james thompson jr".
func plainName(name string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() != 0 {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
			space = false
		case r != '.' && r != '\'':
			space = true
		}
	}
	return b.String()
}
