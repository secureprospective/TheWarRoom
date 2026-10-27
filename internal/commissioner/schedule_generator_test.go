package commissioner

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestScheduleEveryTeamCount(t *testing.T) {
	for count := 2; count <= 32; count++ {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			teams := make([]string, count)
			divisions := make(map[string]string)
			for i := range teams {
				teams[i] = fmt.Sprintf("team.%02d", i)
				divisions[teams[i]] = fmt.Sprint(i / 4)
			}
			original := append([]string{}, teams...)
			rules := leaguerules.ScheduleRules{Pattern: leaguerules.Known(leaguerules.DivisionWeighted)}
			weeks, err := competition.GenerateSchedule(teams, divisions, rules)
			if err != nil || len(weeks) != 13 {
				t.Fatal(weeks, err)
			}
			byeCounts := make(map[string]int)
			for i, week := range weeks {
				seen := make(map[string]bool)
				if week.Week != i+1 || len(week.Games) != count/2 {
					t.Fatal(week)
				}
				for _, game := range week.Games {
					if game.Home == game.Away || seen[game.Home] || seen[game.Away] {
						t.Fatal("franchise plays twice", week)
					}
					seen[game.Home], seen[game.Away] = true, true
				}
				if count%2 == 1 {
					if week.Bye == "" || seen[week.Bye] {
						t.Fatal("missing or duplicate bye", week)
					}
					seen[week.Bye] = true
					byeCounts[week.Bye]++
				} else if week.Bye != "" {
					t.Fatal("even league has a bye", week)
				}
				for _, team := range teams {
					if !seen[team] {
						t.Fatal("franchise left idle without a bye", team, week)
					}
				}
			}
			if !reflect.DeepEqual(original, teams) {
				t.Fatal("generator mutated caller teams")
			}
			again, err := competition.GenerateSchedule(teams, divisions, rules)
			if err != nil || !reflect.DeepEqual(weeks, again) {
				t.Fatal("non-deterministic schedule", err)
			}
			if count%2 == 1 {
				for _, a := range teams {
					for _, b := range teams {
						if byeCounts[a]-byeCounts[b] > 1 {
							t.Fatal("unbalanced byes", byeCounts)
						}
					}
				}
			}
		})
	}
}

func TestScheduleDivisionWeightAndPatternRefusal(t *testing.T) {
	teams := []string{"a", "b", "c", "d"}
	divisions := map[string]string{"a": "one", "b": "one", "c": "two", "d": "two"}
	rules := leaguerules.ScheduleRules{RegularWeeks: leaguerules.Known(52),
		Pattern: leaguerules.Known(leaguerules.DivisionWeighted)}
	weeks, err := competition.GenerateSchedule(teams, divisions, rules)
	if err != nil {
		t.Fatal(err)
	}
	meetings := make(map[string]int)
	for _, week := range weeks {
		for _, game := range week.Games {
			if game.Home == "a" {
				meetings[game.Away]++
			}
			if game.Away == "a" {
				meetings[game.Home]++
			}
		}
	}
	if meetings["b"] != 26 || meetings["c"] != 13 || meetings["d"] != 13 {
		t.Fatal("division opponents must meet twice as often", meetings)
	}
	rules.Pattern = leaguerules.Needs[leaguerules.SchedulePattern]("choose")
	if _, err := competition.GenerateSchedule(teams, divisions, rules); err == nil ||
		err.Error() != "Choose how often teams meet in this smaller league." {
		t.Fatal(err)
	}
}

func TestScheduleRejectsInvalidInputs(t *testing.T) {
	rules := leaguerules.ScheduleRules{Pattern: leaguerules.Known(leaguerules.DivisionWeighted)}
	for _, teams := range [][]string{{"a"}, make([]string, 33), {"a", "a"}, {"a", ""}, {"a", "b"}} {
		if _, err := competition.GenerateSchedule(teams, map[string]string{"a": "one"}, rules); err == nil {
			t.Fatal("accepted invalid franchises or alignment", teams)
		}
	}
	for _, weeks := range []int{0, -1, 53} {
		rules.RegularWeeks = leaguerules.Known(weeks)
		if _, err := competition.GenerateSchedule([]string{"a", "b"}, map[string]string{"a": "one", "b": "one"},
			rules); err == nil {
			t.Fatal("accepted invalid week count", weeks)
		}
	}
	rules.RegularWeeks = leaguerules.Known(13)
	rules.Pattern = leaguerules.Known(leaguerules.SchedulePattern("unknown"))
	if _, err := competition.GenerateSchedule([]string{"a", "b"}, map[string]string{"a": "one", "b": "one"},
		rules); err == nil {
		t.Fatal("accepted unknown pattern")
	}
}

func TestLegacyScheduleBalanceAndMeetings(t *testing.T) {
	teams := make([]string, 32)
	divisions := make(map[string]string)
	for i := range teams {
		teams[i] = fmt.Sprintf("franchise.%02d", i+1)
		divisions[teams[i]] = fmt.Sprint(i / 4)
	}
	weeks, err := competition.GenerateSchedule(teams, divisions, leaguerules.ScheduleRules{
		RegularWeeks: leaguerules.Known(13), Pattern: leaguerules.Known(leaguerules.DivisionWeighted),
	})
	if err != nil {
		t.Fatal(err)
	}
	homes, aways := make(map[string]int), make(map[string]int)
	meetings := make(map[[2]string]int)
	for _, week := range weeks {
		for _, game := range week.Games {
			homes[game.Home]++
			aways[game.Away]++
			meetings[[2]string{game.Home, game.Away}]++
		}
	}
	for _, team := range teams {
		if homes[team]+aways[team] != 13 || homes[team] < 6 || homes[team] > 7 {
			t.Fatal("Legacy requires six or seven home games", team, homes, aways)
		}
		cross := 0
		for _, other := range teams {
			if team == other {
				continue
			}
			home, away := meetings[[2]string{team, other}], meetings[[2]string{other, team}]
			if divisions[team] == divisions[other] {
				if home != 1 || away != 1 {
					t.Fatal("division opponents require reciprocal meetings", team, other, home, away)
				}
			} else {
				if home+away > 1 {
					t.Fatal("cross-division opponent repeated", team, other, home, away)
				}
				cross += home + away
			}
		}
		if cross != 7 {
			t.Fatal("expected seven cross-division games", team, cross)
		}
	}
}

func TestScheduleHomesBalancedAcrossSizesAndLengths(t *testing.T) {
	for count := 2; count <= 32; count++ {
		teams := make([]string, count)
		divisions := make(map[string]string)
		for i := range teams {
			teams[i] = fmt.Sprint(i)
			divisions[teams[i]] = fmt.Sprint(i / 4)
		}
		for length := 1; length <= 52; length++ {
			weeks, err := competition.GenerateSchedule(teams, divisions, leaguerules.ScheduleRules{
				RegularWeeks: leaguerules.Known(length), Pattern: leaguerules.Known(leaguerules.DivisionWeighted),
			})
			if err != nil {
				t.Fatal(count, length, err)
			}
			balance := make(map[string]int)
			for _, week := range weeks {
				for _, game := range week.Games {
					balance[game.Home]++
					balance[game.Away]--
				}
			}
			for team, difference := range balance {
				if difference < -1 || difference > 1 {
					t.Fatal(count, length, team, difference)
				}
			}
		}
	}
}
