package leaguearchive_test

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguearchive"
)

type sink struct{ fetches []archive.Fetch }

func (s *sink) Record(_ context.Context, f archive.Fetch) error {
	s.fetches = append(s.fetches, f)
	return nil
}

func league(id, host string) string {
	return `{"version":"1.0","league":{"id":"` + id + `","baseURL":"https://` + host + `"},"encoding":"utf-8"}`
}

const loginError = `{"version":"1.0","error":{"$t":"API requires logged in user in league ID 51719"},"encoding":"utf-8"}`

// Fixture is three seasons under three league ids, a body saved twice and two kinds of MFL error.
func Fixture() fstest.MapFS {
	at := time.Date(2026, 10, 5, 2, 43, 11, 0, time.UTC)
	f := func(body string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(body), ModTime: at} }
	return fstest.MapFS{
		"2013/league.json":                f(league("51719", "www48.myfantasyleague.com")),
		"2013/polls.json":                 f(loginError),
		"2013/weekly/rosters_W01.json":    f(`{"rosters":{"franchise":[]}}`),
		"2014/league.json":                f(league("47710", "www46.myfantasyleague.com")),
		"2014/survivorPool.json":          f(`{"error":"League Does Not Use Survivor Pool"}`),
		"2014/players_DETAILS1.json":      f(`{"players":{"player":[]}}`),
		"2026/league.json":                f(league("14432", "www47.myfantasyleague.com")),
		"2026/adp.json":                   f(`{"adp":{"player":[]}}`),
		"2026/weekly/rosters_W01.json":    f(`{"rosters":{"franchise":[]}}`),
		"2026/playoffBracket_4.json":      f(`{"playoffBracket":{}}`),
		"2026/messageBoardThread/77.json": f(`{"thread":{}}`),
		"allRules.json":                   f(`{"allRules":{}}`),
	}
}

func TestImportRecordsEachFileAsTheRequestItAnswered(t *testing.T) {
	var s sink
	rep, err := leaguearchive.Import(context.Background(), Fixture(), "league_archive", nil, &s)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Files != 12 || rep.Recorded != 12 || rep.Held != 0 {
		t.Fatalf("report = %+v, want 12 files all recorded", rep)
	}
	urls := map[string]archive.Fetch{}
	for _, f := range s.fetches {
		if f.Source != "league_archive" || f.Status != 200 || len(f.Gzip) == 0 || f.SHA256 == "" {
			t.Errorf("fetch %s = %+v, want a 200 body named for league_archive", f.URL, f)
		}
		urls[f.URL] = f
	}
	want := []string{
		"https://www48.myfantasyleague.com/2013/export?JSON=1&L=51719&TYPE=polls",
		"https://www48.myfantasyleague.com/2013/export?JSON=1&L=51719&TYPE=rosters&W=1",
		"https://www46.myfantasyleague.com/2014/export?DETAILS=1&JSON=1&L=47710&TYPE=players",
		"https://api.myfantasyleague.com/2026/export?JSON=1&TYPE=adp",
		"https://www47.myfantasyleague.com/2026/export?BRACKET_ID=4&JSON=1&L=14432&TYPE=playoffBracket",
		"https://www47.myfantasyleague.com/2026/export?JSON=1&L=14432&THREAD=77&TYPE=messageBoardThread",
		"https://api.myfantasyleague.com/2026/export?JSON=1&TYPE=allRules",
	}
	for _, u := range want {
		if _, ok := urls[u]; !ok {
			t.Errorf("no fetch for %s", u)
		}
	}
	w1 := urls["https://www48.myfantasyleague.com/2013/export?JSON=1&L=51719&TYPE=rosters&W=1"]
	w26 := urls["https://www47.myfantasyleague.com/2026/export?JSON=1&L=14432&TYPE=rosters&W=1"]
	if w1.SHA256 != w26.SHA256 {
		t.Error("one body saved twice must carry one sha256, so the store keeps it once")
	}
	if !w1.FetchedAt.Equal(time.Date(2026, 10, 5, 2, 43, 11, 0, time.UTC)) {
		t.Errorf("captured at %v, want the file's modification time", w1.FetchedAt)
	}
	polls := urls["https://www48.myfantasyleague.com/2013/export?JSON=1&L=51719&TYPE=polls"]
	if polls.Err != "MFL error: API requires logged in user in league ID 51719" {
		t.Errorf("error body recorded with Err %q", polls.Err)
	}
	if len(rep.Errors) != 2 || rep.Errors[0].Export != "polls" || rep.Errors[0].Season != 2013 ||
		rep.Errors[1].Export != "survivorPool" || rep.Errors[1].Text != "MFL error: League Does Not Use Survivor Pool" {
		t.Errorf("error bodies = %+v, want polls 2013 and survivorPool 2014", rep.Errors)
	}
}

func TestImportSkipsFilesAlreadyHeld(t *testing.T) {
	var first sink
	if _, err := leaguearchive.Import(context.Background(), Fixture(), "league_archive", nil, &first); err != nil {
		t.Fatal(err)
	}
	held := map[string]bool{}
	for _, f := range first.fetches {
		held[leaguearchive.HeldKey(f.URL, f.SHA256)] = true
	}
	fsys := Fixture()
	fsys["2026/adp.json"] = &fstest.MapFile{Data: []byte(`{"adp":{"player":[{"id":"1"}]}}`)}
	var again sink
	rep, err := leaguearchive.Import(context.Background(), fsys, "league_archive", held, &again)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Recorded != 1 || rep.Held != 11 || again.fetches[0].URL != "https://api.myfantasyleague.com/2026/export?JSON=1&TYPE=adp" {
		t.Fatalf("report = %+v, fetches %v; want only the changed adp recorded", rep, again.fetches)
	}
}

func TestImportRefusesAMalformedArchiveWholly(t *testing.T) {
	cases := map[string]struct {
		file, body, want string
	}{
		"null":            {"2014/rules.json", `null`, "An archive file is incomplete or is not a valid MFL download."},
		"array":           {"2014/rules.json", `[]`, "An archive file is incomplete or is not a valid MFL download."},
		"not JSON":        {"2014/rules.json", `{"rules":`, "An archive file is incomplete or is not a valid MFL download."},
		"unknown suffix":  {"2014/rules_X1.json", `{}`, "An archive file has an unexpected name or folder."},
		"week outside":    {"2014/rosters_W03.json", `{}`, "An archive file has an unexpected name or folder."},
		"no league":       {"2015/rules.json", `{}`, "The archive could not be read."},
		"bad league id":   {"2015/league.json", league("x", "www45.myfantasyleague.com"), `An archive file is incomplete or is not a valid MFL download.`},
		"unknown folder":  {"2014/extra/rules.json", `{}`, "An archive file has an unexpected name or folder."},
		"not json suffix": {"2014/notes.txt", `x`, "An archive file has an unexpected name or folder."},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			fsys := Fixture()
			fsys[c.file] = &fstest.MapFile{Data: []byte(c.body)}
			var s sink
			_, err := leaguearchive.Import(context.Background(), fsys, "league_archive", nil, &s)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want it to contain %q", err, c.want)
			}
			if len(s.fetches) != 0 {
				t.Errorf("%d fetches recorded from a malformed archive, want none", len(s.fetches))
			}
		})
	}
}

func TestParseURLReadsBackTheExport(t *testing.T) {
	for _, rel := range []string{"2014/players_DETAILS1.json", "2013/weekly/rosters_W01.json", "2026/playoffBracket_4.json",
		"2026/transactions.json", "2026/playerScores_WYTD.json", "2026/pool_POOLTYPENFL.json"} {
		e, err := leaguearchive.ParsePath(rel, 2026)
		if err != nil {
			t.Fatal(err)
		}
		got, id, err := leaguearchive.ParseURL(e.URL(leaguearchive.League{ID: "14432", Host: "www47.myfantasyleague.com"}))
		if err != nil || got.Key() != e.Key() || id != "14432" {
			t.Errorf("%s: round trip = %s %q %v, want %s", rel, got.Key(), id, err, e.Key())
		}
	}
	if e, _ := leaguearchive.ParsePath("2026/transactions.json", 2026); e.Params["TRANS_TYPE"] != "*" {
		t.Errorf("transactions params = %v, want TRANS_TYPE=*", e.Params)
	}
	if leaguearchive.ScopeOf("messageBoard") != leaguearchive.ScopeOwner || leaguearchive.ScopeOf("adp") != leaguearchive.ScopeSite ||
		leaguearchive.ScopeOf("rosters") != leaguearchive.ScopeLeague {
		t.Error("scopes: messageBoard owner, adp site, rosters league")
	}
}
