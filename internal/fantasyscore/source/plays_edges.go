package source

import (
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func edgeStatIDs() []leaguerules.StatID {
	return []leaguerules.StatID{"third_down_conversion", "fourth_down_conversion", "third_down_passing_conversion", "fourth_down_passing_conversion", "red_zone_carry", "red_zone_target", "red_zone_pass_attempt", "rushing_tfl"}
}

func attachEdges(weeks []PlayerWeek, index map[string]int, p RawPlay, season, week int) {
	if !p.EdgeAvailable {
		return
	}
	add := func(player string, id leaguerules.StatID) {
		i, ok := index[playKey(player, season, week)]
		if ok && weeks[i].GameID == p.GameID && weeks[i].SeasonType == p.SeasonType {
			weeks[i].Line.Quantities[id] += 10000
		}
	}
	for _, c := range []struct {
		flag           string
		player, passer leaguerules.StatID
	}{
		{p.ThirdConversion, "third_down_conversion", "third_down_passing_conversion"},
		{p.FourthConversion, "fourth_down_conversion", "fourth_down_passing_conversion"},
	} {
		if c.flag != "1" {
			continue
		}
		if p.RushAttempt == "1" {
			add(p.Rusher, c.player)
		}
		if p.CompletePass == "1" {
			add(p.Receiver, c.player)
			add(p.Passer, c.passer)
		}
	}
	if yard, err := strconv.Atoi(p.Yardline); err == nil && yard <= 20 {
		if p.RushAttempt == "1" {
			add(p.Rusher, "red_zone_carry")
		}
		if p.PassAttempt == "1" {
			add(p.Passer, "red_zone_pass_attempt")
			add(p.Receiver, "red_zone_target")
		}
	}
	if p.RushAttempt == "1" && p.TackledForLoss == "1" {
		add(p.Rusher, "rushing_tfl")
	}
}
