package commissioner

import (
	"reflect"
	"testing"
)

func TestChosenFixtures(t *testing.T) {
	t.Parallel()
	weeks := []ScheduleWeek{{Week: 1, Games: []ScheduleGame{{ID: "w1", Home: "a", Away: "b"}}}, {Week: 2}}
	for _, tc := range []struct {
		name   string
		mutate func([]ScheduleWeek)
		bad    bool
	}{
		{"odd game count and resting teams", func([]ScheduleWeek) {}, false},
		{"unknown team", func(w []ScheduleWeek) { w[0].Games[0].Away = "unknown" }, true},
		{"self game", func(w []ScheduleWeek) { w[0].Games[0].Away = "a" }, true},
		{"duplicate team", func(w []ScheduleWeek) {
			w[0].Games = append(w[0].Games, ScheduleGame{ID: "extra", Home: "a", Away: "c"})
		}, true},
		{"duplicate game", func(w []ScheduleWeek) { w[1].Games = append(w[1].Games, w[0].Games[0]) }, true},
		{"invalid week", func(w []ScheduleWeek) { w[1].Week = 3 }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := cloneSchedule(weeks)
			tc.mutate(input)
			got, err := checkedFixtures(input, 2, []string{"a", "b", "c", "d"})
			if (err != nil) != tc.bad {
				t.Fatalf("result %v: %v", got, err)
			}
			if !tc.bad && !reflect.DeepEqual(got, input) {
				t.Fatal("changed chosen fixtures")
			}
		})
	}
	bye, err := franchiseScheduleGame(weeks[0], "c")
	if err != nil || bye.Venue != "Bye" {
		t.Fatalf("resting team: %+v %v", bye, err)
	}
}
