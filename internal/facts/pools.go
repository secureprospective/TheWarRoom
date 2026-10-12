package facts

import (
	"fmt"
	"strconv"
	"strings"
)

type PoolPick struct {
	Franchise     string
	Week          int64
	Matchup, Pick string
	Rank          int64
}
type PoolSeason struct {
	Picks    []PoolPick
	Survivor string
}

func CompilePools(inputs []ResolvedInput) (map[int64]PoolSeason, error) {
	out := map[int64]PoolSeason{}
	for _, input := range inputs {
		yearText, file, ok := strings.Cut(input.File, "/")
		if !ok || (file != "pool_POOLTYPENFL.json" && file != "survivorPool.json") {
			continue
		}
		if err := input.Validate(); err != nil {
			return nil, err
		}
		year, err := strconv.ParseInt(yearText, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("pool season: %w", err)
		}
		raw, err := decodeRaw(input.Body)
		if err != nil {
			return nil, err
		}
		if err := raw.Validate(); err != nil {
			return nil, err
		}
		season := out[year]
		if file == "survivorPool.json" {
			season.Survivor = value(raw.get("error", "$t").text())
			if season.Survivor == "" {
				season.Survivor = "Setup is recorded; no survivor report is implemented"
			}
		} else {
			picks, err := poolPicks(raw)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", input.File, err)
			}
			season.Picks = picks
		}
		out[year] = season
	}
	return out, nil
}

func poolPicks(raw RawValue) ([]PoolPick, error) {
	picks := []PoolPick{}
	for _, franchise := range raw.get("poolPicks", "franchise").rows() {
		id, err := requiredID(franchise, "id", "pool franchise")
		if err != nil {
			return nil, err
		}
		for _, week := range franchise.get("week").rows() {
			number := week.get("week").whole()
			if number == nil || *number < 1 || *number > 18 {
				return nil, fmt.Errorf("pool: invalid week")
			}
			for _, game := range week.get("game").rows() {
				pick, matchup := value(game.get("pick").text()), value(game.get("matchup").text())
				rank := game.get("rank").whole()
				teams := strings.Split(matchup, ",")
				if pick == "" || len(teams) != 2 || (pick != teams[0] && pick != teams[1]) || rank == nil || *rank < 1 || *rank > 16 {
					return nil, fmt.Errorf("pool: invalid pick, matchup or confidence weight")
				}
				picks = append(picks, PoolPick{Franchise: id, Week: *number, Matchup: matchup, Pick: pick, Rank: *rank})
			}
		}
	}
	return picks, nil
}
