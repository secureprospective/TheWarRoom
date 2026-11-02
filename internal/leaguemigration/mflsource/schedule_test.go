package mflsource

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
)

func TestMFLShapedSchedule(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body string
		bad        bool
		games      int
	}{
		{"past week odd count and idle franchises", `{"schedule":{"weeklySchedule":[{"week":"1","matchup":[{"franchise":[{"id":"0001","isHome":"0","score":"105.25"},{"id":"0002","isHome":"1","score":"99"}]}]},{"week":"2"}]}}`, false, 1},
		{"empty playoff week", `{"schedule":{"weeklySchedule":{"week":"1"}}}`, false, 0},
		{"unknown franchise", `{"schedule":{"weeklySchedule":{"week":"1","matchup":{"franchise":[{"id":"9999","isHome":"0"},{"id":"0002","isHome":"1"}]}}}}`, true, 0},
		{"invalid week", `{"schedule":{"weeklySchedule":{"week":"bad"}}}`, true, 0},
		{"duplicate week", `{"schedule":{"weeklySchedule":[{"week":"1"},{"week":"1"}]}}`, true, 0},
		{"missing week", `{"schedule":{"weeklySchedule":{"week":"2"}}}`, true, 0},
		{"same team", `{"schedule":{"weeklySchedule":{"week":"1","matchup":{"franchise":[{"id":"0001","isHome":"0"},{"id":"0001","isHome":"1"}]}}}}`, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := leagueschedule.Parse([]byte(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			weeks, err := Schedule(raw, map[string]string{"0001": "franchise.01", "0002": "franchise.02", "0003": "franchise.03", "0004": "franchise.04"})
			if (err != nil) != tc.bad {
				t.Fatalf("schedule %+v error %v", weeks, err)
			}
			if tc.bad {
				return
			}
			if len(weeks[0].Games) != tc.games {
				t.Fatalf("lost games: %+v", weeks)
			}
			if tc.games > 0 && (weeks[0].Games[0].Home != "franchise.02" || weeks[0].Games[0].Away != "franchise.01") {
				t.Fatal("home/away swapped", weeks)
			}
		})
	}
}

func TestFollowingLeagueReadsSettingsNotIdentity(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	records := []commissioner.Record{{Command: commissioner.RawCommand{League: "native.any", Command: commissioner.Create,
		Args: commissioner.RawArgs{Source: commissioner.SourceNative, Season: 2026}}, Principal: "creator",
		Receipt: commissioner.Receipt{Entry: "create", RecordedAt: at}}}
	set := commissioner.Record{Command: commissioner.RawCommand{League: "native.any", Command: commissioner.RulesSet,
		Args: commissioner.RawArgs{Field: "schedule.follow", Value: json.RawMessage(`{"Source":"mfl","League":"00123","Season":2026}`)}, Effective: "now"},
		Receipt: commissioner.Receipt{Entry: "follow", RecordedAt: at.Add(time.Second)}}
	records = append(records, set)
	id, season, who, ok := FollowingLeague("not-an-import-id", records)
	if !ok || id != "00123" || season != 2026 || who != "creator" {
		t.Fatalf("%s %d %s %t", id, season, who, ok)
	}
	set.Command.Args.Value = json.RawMessage(`{"Source":"","League":"","Season":0}`)
	set.Receipt.Entry = "clear"
	set.Receipt.RecordedAt = at.Add(2 * time.Second)
	records = append(records, set)
	if _, _, _, ok := FollowingLeague("native.mfl-00123-2026", records); ok {
		t.Fatal("cleared setting still follows")
	}
	if _, _, _, ok := FollowingLeague("native.any", nil); ok {
		t.Fatal("empty ledger follows")
	}
}
