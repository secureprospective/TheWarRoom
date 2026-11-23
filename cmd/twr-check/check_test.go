package main

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

func d(p float64) leaguerules.Decimal { return leaguerules.Decimal(p * 10000) }
func dp(p float64) *leaguerules.Decimal {
	v := d(p)
	return &v
}

func kickKC() time.Time  { return time.Date(2026, 12, 6, 18, 0, 0, 0, time.UTC) }
func kickBUF() time.Time { return time.Date(2026, 12, 6, 21, 25, 0, 0, time.UTC) }

// league is a two-team league's recorded-shape readings; each test bends one thing.
type league struct {
	week1, week2 commissioner.NativeResults
	standings    commissioner.NativeStandings
	games        leagueservice.NFLGames
	lineups      func(franchise string, at time.Time) ([]commissioner.LineupPlayer, bool) // false: token cannot read
}

func starter(id, name, team string, pts float64) commissioner.StarterScore {
	return commissioner.StarterScore{Player: id, Name: name, Team: team, Points: dp(pts)}
}

func goodLeague() *league {
	a := commissioner.FranchiseWeekScore{Franchise: "A", Name: "Aces", Points: dp(80), Starters: []commissioner.StarterScore{
		starter("p1", "One", "KC", 30), starter("p2", "Two", "BUF", 50)}}
	b := commissioner.FranchiseWeekScore{Franchise: "B", Name: "Bears", Points: dp(95), Adjustment: d(5), Starters: []commissioner.StarterScore{
		starter("p3", "Three", "KC", 40), starter("p4", "Four", "BUF", 50)}}
	ev := &commissioner.ScoreEvidence{Live: "final", Source: "nflverse"}
	l := &league{}
	l.week2 = commissioner.NativeResults{Season: 2026, Week: 2, Evidence: ev, Scores: []commissioner.FranchiseWeekScore{a, b},
		Games: []commissioner.NativeGame{{Week: 2, Home: "A", Away: "B", Status: commissioner.GameFinal, HomePoints: dp(80), AwayPoints: dp(95), Winner: "B"}}}
	l.week1 = commissioner.NativeResults{Season: 2026, Week: 1, Evidence: ev,
		Games: []commissioner.NativeGame{{Week: 1, Home: "A", Away: "B", Status: commissioner.GameFinal, HomePoints: dp(100), AwayPoints: dp(90), Winner: "A"}}}
	l.standings = commissioner.NativeStandings{Season: 2026, Through: 2, Computed: []commissioner.StandingRow{
		{Franchise: "A", Name: "Aces", Wins: 1, Losses: 1, PointsFor: d(180)},
		{Franchise: "B", Name: "Bears", Wins: 1, Losses: 1, PointsFor: d(185)}}}
	game := func(id, home, away, status string, k time.Time) livegame.Game {
		return livegame.Game{ID: id, Season: 2026, Week: 2, Kickoff: k, Status: status,
			Home: livegame.Team{Abbreviation: home}, Away: livegame.Team{Abbreviation: away}}
	}
	l.games = leagueservice.NFLGames{Board: livegame.Held{Games: []livegame.Game{
		game("1", "KC", "DEN", "post", kickKC()), game("2", "BUF", "MIA", "post", kickBUF())}}}
	l.lineups = func(f string, _ time.Time) ([]commissioner.LineupPlayer, bool) {
		for _, s := range l.week2.Scores {
			if s.Franchise == f {
				var out []commissioner.LineupPlayer
				for _, st := range s.Starters {
					out = append(out, commissioner.LineupPlayer{Player: st.Player, Name: st.Name, Team: st.Team})
				}
				return out, true
			}
		}
		return nil, true
	}
	return l
}

// serve answers the workflow wire like the league server does, read mode only.
func (l *league) serve(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req serverhttp.RawNativeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Workflow == nil || req.Workflow.Mode != "read" {
			t.Errorf("twr-check must only send read-mode workflows: %+v", req)
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		var scope serverhttp.RawWorkflowScope
		_ = json.Unmarshal(req.Workflow.Arguments, &scope)
		var data any
		switch req.Workflow.Command {
		case "results.read":
			data = l.week2
			if scope.Week == 1 {
				data = l.week1
			}
		case "standings.read":
			data = l.standings
		case "nfl.games.read":
			data = l.games
		case "lineup.read":
			at, _ := time.Parse(time.RFC3339, scope.At)
			players, ok := l.lineups(scope.Franchise, at)
			if !ok {
				http.Error(w, "no", http.StatusConflict)
				return
			}
			data = commissioner.NativeLineup{Franchise: scope.Franchise, Set: true, Starters: players}
		default:
			http.Error(w, "unknown", http.StatusBadRequest)
			return
		}
		raw, _ := json.Marshal(data)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(serverhttp.WorkflowResponse{Label: "Test", AppRelease: 1, Data: raw})
	}))
	t.Cleanup(srv.Close)
	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte("tok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return srv, file
}

type outcome struct {
	code   int
	out    string
	detail string
}

func (l *league) run(t *testing.T) outcome {
	t.Helper()
	srv, tokenFile := l.serve(t)
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	detail := filepath.Join(t.TempDir(), "detail.md")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-server", srv.URL, "-token-file", tokenFile, "-league", "native.test", "-season", "2026", "-week", "2", "-out", detail}, &stdout, &stderr, roots)
	md, _ := os.ReadFile(detail)
	return outcome{code, stdout.String() + stderr.String(), string(md)}
}

func (o outcome) level(t *testing.T, check string) Level {
	t.Helper()
	for _, line := range strings.Split(o.out, "\n") {
		if strings.Contains(line, check+":") {
			return Level(strings.Fields(line)[0])
		}
	}
	t.Fatalf("no line for %q in:\n%s", check, o.out)
	return ""
}

func TestAllGoodWeekIsGreen(t *testing.T) {
	o := goodLeague().run(t)
	if o.code != 0 {
		t.Fatalf("exit %d\n%s", o.code, o.out)
	}
	for _, c := range []string{"Scores add up", "Official stats final", "Standings match games", "Lineups locked at kickoff"} {
		if got := o.level(t, c); got != Green {
			t.Errorf("%s = %s\n%s", c, got, o.out)
		}
	}
	if !strings.Contains(o.detail, "Aces") {
		t.Errorf("detail file missing: %q", o.detail)
	}
}

func TestWrongTotalIsRed(t *testing.T) {
	l := goodLeague()
	l.week2.Scores[0].Points = dp(81.5)
	l.week2.Games[0].HomePoints = dp(81.5) // the game agrees; only the starters-plus-adjustments sum is off
	o := l.run(t)
	if o.code != 1 || o.level(t, "Scores add up") != Red {
		t.Fatalf("exit %d\n%s", o.code, o.out)
	}
}

func TestNonFinalGameIsAmberNotRed(t *testing.T) {
	l := goodLeague()
	l.games.Board.Games[1].Status = "in"
	o := l.run(t)
	if o.code != 0 || o.level(t, "Official stats final") != Amber {
		t.Fatalf("exit %d\n%s", o.code, o.out)
	}
	l = goodLeague()
	l.week2.Evidence.Provisional = true
	if o = l.run(t); o.level(t, "Official stats final") != Amber {
		t.Fatalf("provisional evidence should be amber\n%s", o.out)
	}
}

func TestStandingsMismatchIsRed(t *testing.T) {
	l := goodLeague()
	l.standings.Computed[0].Wins, l.standings.Computed[0].Losses = 2, 0
	o := l.run(t)
	if o.code != 1 || o.level(t, "Standings match games") != Red {
		t.Fatalf("exit %d\n%s", o.code, o.out)
	}
	l = goodLeague()
	l.standings.Computed[1].PointsFor = d(190)
	if o = l.run(t); o.level(t, "Standings match games") != Red {
		t.Fatalf("points-for mismatch should be red\n%s", o.out)
	}
}

func TestStarterAddedAfterKickoffIsRed(t *testing.T) {
	l := goodLeague()
	// Aces' lineup at kickoff held a different KC player; "One" was swapped in after the game began.
	l.lineups = func(f string, at time.Time) ([]commissioner.LineupPlayer, bool) {
		players, ok := goodLeague().lineups(f, at)
		if f == "A" {
			players[0] = commissioner.LineupPlayer{Player: "p9", Name: "Nine", Team: "KC"}
		}
		return players, ok
	}
	o := l.run(t)
	if o.code != 1 || o.level(t, "Lineups locked at kickoff") != Red || !strings.Contains(o.detail, "One was scored as a starter") {
		t.Fatalf("exit %d\n%s\n%s", o.code, o.out, o.detail)
	}
}

func TestUnreadableLineupIsUnknownNotFailure(t *testing.T) {
	l := goodLeague()
	l.lineups = func(f string, at time.Time) ([]commissioner.LineupPlayer, bool) {
		if f == "B" {
			return nil, false
		}
		return goodLeague().lineups(f, at)
	}
	o := l.run(t)
	if o.code != 0 || o.level(t, "Lineups locked at kickoff") != Unknown || !strings.Contains(o.detail, "token cannot read franchise B") {
		t.Fatalf("exit %d\n%s\n%s", o.code, o.out, o.detail)
	}
}

func TestBadTokenExitsTwo(t *testing.T) {
	l := goodLeague()
	srv, _ := l.serve(t)
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	file := filepath.Join(t.TempDir(), "token")
	_ = os.WriteFile(file, []byte("wrong"), 0o600)
	var out, errs bytes.Buffer
	code := run([]string{"-server", srv.URL, "-token-file", file, "-league", "native.test", "-season", "2026", "-week", "2"}, &out, &errs, roots)
	if code != 2 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
}
