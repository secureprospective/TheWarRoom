package mflsource

import (
	"testing"

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

func TestFollowingLeagueUsesRecordedImportIdentity(t *testing.T) {
	t.Parallel()
	records := []commissioner.Record{{Command: commissioner.RawCommand{Command: commissioner.Create,
		Args: commissioner.RawArgs{Source: commissioner.SourceMFLImport, Season: 2026}}, Principal: "creator"}}
	for _, tc := range []struct {
		league string
		source string
		want   bool
	}{
		{"native.mfl-00123-2026", commissioner.SourceMFLImport, true},
		{"native.mfl-00123-2026", commissioner.SourceNative, false},
		{"native.mfl-00123-2025", commissioner.SourceMFLImport, false},
		{"native.mfl-bad-2026", commissioner.SourceMFLImport, false},
		{"native.other", commissioner.SourceMFLImport, false},
	} {
		records[0].Command.Args.Source = tc.source
		id, season, principal, ok := FollowingLeague(tc.league, records)
		if ok != tc.want || (ok && (id != "00123" || season != 2026 || principal != "creator")) {
			t.Fatalf("%+v: %s %d %s %t", tc, id, season, principal, ok)
		}
	}
	if _, _, _, ok := FollowingLeague("native.mfl-00123-2026", nil); ok {
		t.Fatal("missing ledger followed")
	}
}
