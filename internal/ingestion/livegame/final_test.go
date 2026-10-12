package livegame

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type memFinals struct {
	kept  map[string][]byte
	reads int
}

func (m *memFinals) Keep(_ context.Context, kind, id string, body []byte, _ time.Time) error {
	m.kept[kind+":"+id] = body
	return nil
}
func (m *memFinals) Bodies(_ context.Context, kind, id string, each func(string, []byte, time.Time) bool) error {
	m.reads++
	for key, body := range m.kept {
		if k, i, _ := cutKey(key); k == kind && (id == "" || i == id) && !each(i, body, time.Unix(0, 0)) {
			return nil
		}
	}
	return nil
}
func cutKey(key string) (string, string, bool) {
	for i := range key {
		if key[i] == ':' {
			return key[:i], key[i+1:], true
		}
	}
	return key, "", false
}

// seeded is the strip as the app seeds it from MFL's schedule: MFL team codes, "AWAY-HOME" ids.
func seeded(t *testing.T) Held {
	games, err := ParseScoreboard(fixture(t, "espn-scoreboard"))
	if err != nil {
		t.Fatal(err)
	}
	mfl := map[string]string{"JAX": "JAC", "GB": "GBP", "NE": "NEP", "NO": "NOS", "LV": "LVR", "WSH": "WAS", "SF": "SFO", "TB": "TBB"}
	for i := range games {
		for _, team := range []*Team{&games[i].Home, &games[i].Away} {
			if code, ok := mfl[team.Abbreviation]; ok {
				team.Abbreviation = code
			}
			team.ID = ""
		}
		games[i].ID = games[i].Away.Abbreviation + "-" + games[i].Home.Abbreviation
		games[i].Plays, games[i].Box = []Play{}, []Box{}
	}
	return Held{Games: games, Source: "Held MFL NFL schedule"}
}

func finalAdapter(t *testing.T, finals *memFinals, calls *[]string, fail map[string]bool) Adapter {
	return Adapter{Finals: finals, Board: seeded(t), Fetch: func(_ context.Context, kind, id string) ([]byte, error) {
		*calls = append(*calls, kind+":"+id)
		if fail[kind] {
			return nil, errors.New("captured source unavailable")
		}
		switch kind {
		case "espn":
			return fixture(t, "espn-scoreboard"), nil
		case "summary":
			return fixture(t, "espn-summary-"+id), nil
		case "plays":
			return fixture(t, "espn-core-plays"), nil
		}
		t.Fatalf("unexpected %s %s", kind, id)
		return nil, nil
	}}
}

func TestFinalIsReadOncePerFollowAndHeldAcrossRestart(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	finals := &memFinals{kept: map[string][]byte{}}
	calls := []string{}
	a := finalAdapter(t, finals, &calls, nil)
	a.Final(ctx, at, "BUF-LAR") // not started: nothing to read
	if len(calls) != 0 {
		t.Fatal("a game that has not started was fetched", calls)
	}
	a.Final(ctx, at, "DET-ARI")
	if !reflect.DeepEqual(calls, []string{"espn:", "summary:401872991", "plays:401872991"}) {
		t.Fatal("first follow of a final", calls)
	}
	d := a.Details["DET-ARI"]
	if !d.Final || d.Game.Status != "post" || len(d.Game.Plays) == 0 || len(d.Game.Box) == 0 || d.Failure != "" {
		t.Fatalf("final not held: %+v", d)
	}
	if len(finals.kept) != 2 {
		t.Fatal("final bodies not kept", len(finals.kept))
	}
	reads := finals.reads
	a.Final(ctx, at.Add(time.Minute), "DET-ARI")
	a.Final(ctx, at.Add(2*time.Minute), "")
	a.Final(ctx, at.Add(3*time.Minute), "DET-ARI")
	if len(calls) != 3 || finals.reads != reads {
		t.Fatal("a held final was read again", calls, finals.reads-reads)
	}
	restarted := []string{}
	b := finalAdapter(t, finals, &restarted, nil)
	b.Final(ctx, at.Add(time.Hour), "DET-ARI")
	if len(restarted) != 0 || !b.Details["DET-ARI"].Final || len(b.Details["DET-ARI"].Game.Plays) == 0 {
		t.Fatal("restart did not read the kept final", restarted, b.Details["DET-ARI"])
	}
}

func TestFinalFailureShowsAndRetriesOnlyOnTheNextFollow(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	calls := []string{}
	fail := map[string]bool{"summary": true}
	a := finalAdapter(t, &memFinals{kept: map[string][]byte{}}, &calls, fail)
	a.Final(ctx, at, "DET-ARI")
	if d := a.Details["DET-ARI"]; d.Final || d.Failure == "" {
		t.Fatalf("failure not shown: %+v", d)
	}
	for i := range 5 {
		a.Final(ctx, at.Add(time.Duration(i)*time.Minute), "DET-ARI")
	}
	if len(calls) != 2 {
		t.Fatal("a failed final looped", calls)
	}
	fail["summary"] = false
	a.Final(ctx, at, "")
	a.Final(ctx, at, "DET-ARI")
	if len(calls) != 5 || !a.Details["DET-ARI"].Final {
		t.Fatal("following again did not retry once", calls)
	}
}

func TestTeamCodesMatchAcrossMFLAndESPN(t *testing.T) {
	a := Adapter{Board: seeded(t)}
	if g, ok := a.Followed("PHI-JAX"); !ok || g.ID != "PHI-JAC" {
		t.Fatal("ESPN codes do not find the MFL-seeded game")
	}
	espn, err := ParseScoreboard(fixture(t, "espn-scoreboard"))
	if err != nil {
		t.Fatal(err)
	}
	a.Board = Held{Games: espn, Source: "ESPN"}
	if g, ok := a.Followed("PHI-JAC"); !ok || g.ID != "401872981" {
		t.Fatal("an MFL follow id lost its game when the board became ESPN's")
	}
	w := Window{Teams: [2]string{"JAC", "PHI"}}
	if !w.plays(espn[2]) {
		t.Fatal("window in MFL codes does not match ESPN's game")
	}
}

func TestScoringPlaysIgnoreTheScoreWobbleAroundATry(t *testing.T) {
	plays := []Play{{Sequence: 1}, {Sequence: 2, HomeScore: 7}, {Sequence: 3, HomeScore: 7}, {Sequence: 4, HomeScore: 7, AwayScore: 3}}
	got := ScoringPlays(plays)
	if len(got) != 2 || got[0].Sequence != 2 || got[1].Sequence != 4 {
		t.Fatal(got)
	}
	// The captured DET at ARI game: touchdown to 45-21, then a penalty read 45-20, then the kickoff 45-21.
	all, err := ParseCore(fixture(t, "espn-core-plays"))
	if err != nil {
		t.Fatal(err)
	}
	scoring := ScoringPlays(all)
	for _, p := range scoring {
		if strings.HasPrefix(p.Text, "PENALTY") || strings.Contains(p.Text, " kicks ") {
			t.Fatal("a non-scoring play listed:", p.Text)
		}
	}
	if last := scoring[len(scoring)-1]; last.HomeScore != 21 || last.AwayScore != 45 {
		t.Fatal("final scoring play", last.Text, last.AwayScore, last.HomeScore)
	}
}
