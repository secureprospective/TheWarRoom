package livegame

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFailoverAndPolitePolling(t *testing.T) {
	at := time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC)
	windows := []Window{{Teams: [2]string{"DET", "ARI"}, Kickoff: at.Add(-time.Hour)}}
	calls := []string{}
	fail := map[string]bool{}
	a := Adapter{Fetch: func(_ context.Context, kind, id string) ([]byte, error) {
		calls = append(calls, kind)
		if fail[kind] {
			return nil, errors.New("captured source unavailable")
		}
		switch kind {
		case "espn":
			return fixture(t, "espn-scoreboard"), nil
		case "yahoo":
			return fixture(t, "yahoo-scoreboard"), nil
		default:
			t.Fatalf("unexpected %s %s", kind, id)
			return nil, nil
		}
	}}
	ctx := context.Background()
	a.Refresh(ctx, at, false, windows, "")
	if len(calls) != 0 {
		t.Fatal("closed Now fetched")
	}
	a.Refresh(ctx, at, true, windows, "")
	if !reflect.DeepEqual(calls, []string{"espn"}) || a.Board.Source != "ESPN" || !a.Board.At.Equal(at) {
		t.Fatalf("primary %v %+v", calls, a.Board)
	}
	a.Refresh(ctx, at.Add(29*time.Second), true, windows, "")
	if len(calls) != 1 {
		t.Fatal("early poll")
	}
	fail["espn"] = true
	a.Refresh(ctx, at.Add(30*time.Second), true, windows, "")
	if !reflect.DeepEqual(calls, []string{"espn", "espn", "yahoo"}) || a.Board.Source != "Yahoo failover" || a.Board.Failure == "" {
		t.Fatalf("failover %v %+v", calls, a.Board)
	}
	kept := a.Board.At
	keptID := a.Board.Games[0].ID
	fail["yahoo"] = true
	a.Refresh(ctx, at.Add(89*time.Second), true, windows, "")
	if len(calls) != 3 {
		t.Fatal("error backoff absent")
	}
	a.Refresh(ctx, at.Add(90*time.Second), true, windows, "")
	if len(calls) != 5 || !a.Board.At.Equal(kept) || a.Board.Games[0].ID != keptID || a.Board.Failure == "" {
		t.Fatal("held copy was lost or retimed")
	}
}
func TestWindowBoundaries(t *testing.T) {
	at := time.Date(2026, 10, 13, 0, 15, 0, 0, time.UTC)
	for _, tc := range []struct {
		name              string
		now               time.Time
		final, open, want bool
	}{
		{"before kickoff", at.Add(-time.Nanosecond), false, true, false},
		{"kickoff", at, false, true, true},
		{"after final", at.Add(time.Hour), true, true, false},
		{"never marked final, last minute", at.Add(MaxGame - time.Nanosecond), false, true, true},
		{"never marked final, six hours on", at.Add(MaxGame), false, true, false},
		{"Now closed", at, false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			a := Adapter{Fetch: func(context.Context, string, string) ([]byte, error) {
				calls++
				return fixture(t, "espn-scoreboard"), nil
			}}
			w := []Window{{Teams: [2]string{"BUF", "LAR"}, Kickoff: at, Final: tc.final}}
			a.Refresh(context.Background(), tc.now, tc.open, w, "")
			if (calls > 0) != tc.want {
				t.Fatalf("calls %d", calls)
			}
		})
	}
}
func TestFollowedDetailsAndHeldTime(t *testing.T) {
	calls := []string{}
	broken := false
	a := Adapter{Fetch: func(_ context.Context, kind, _ string) ([]byte, error) {
		calls = append(calls, kind)
		if broken {
			return nil, errors.New("summary down")
		}
		switch kind {
		case "summary":
			return fixture(t, "espn-summary-401872991"), nil
		case "plays":
			return fixture(t, "espn-core-plays"), nil
		default:
			t.Fatal(kind)
			return nil, nil
		}
	}}
	at := time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC)
	g := Game{ID: "401872991", Status: "in"}
	a.Board.Source = "ESPN"
	a.refreshDetail(context.Background(), at, g)
	if !reflect.DeepEqual(calls, []string{"summary", "plays"}) || len(a.Details[g.ID].Game.Plays) != 191 {
		t.Fatalf("details %v", calls)
	}
	broken = true
	a.refreshDetail(context.Background(), at.Add(time.Minute), g)
	d := a.Details[g.ID]
	if !d.At.Equal(at) || len(d.Game.Plays) != 191 || d.Failure == "" {
		t.Fatal("lost held detail")
	}
	a.Board.Source = "Yahoo failover"
	a.refreshDetail(context.Background(), at.Add(2*time.Minute), g)
	if len(calls) != 3 || !a.Details[g.ID].At.Equal(at) || a.Details[g.ID].Failure == "" {
		t.Fatal("Yahoo queried ESPN or lost held detail")
	}
}

func TestObservedFinalClosesHeldWindow(t *testing.T) {
	g, err := ParseSummary(fixture(t, "espn-summary-401872991"))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	a := Adapter{Board: Held{Games: []Game{g}, Source: "ESPN"}, Fetch: func(context.Context, string, string) ([]byte, error) {
		calls++
		return nil, errors.New("must not fetch")
	}}
	windows := []Window{{Teams: [2]string{g.Home.Abbreviation, g.Away.Abbreviation}, Kickoff: g.Kickoff}}
	a.Refresh(context.Background(), g.Kickoff.Add(time.Hour), true, windows, g.ID)
	if calls != 0 || Live(g.Kickoff.Add(time.Hour), a.EffectiveWindows(windows)) {
		t.Fatal("observed final kept polling")
	}
}
func TestFollowedCadenceAndErrorBackoff(t *testing.T) {
	g, err := ParseSummary(fixture(t, "espn-summary-401872991"))
	if err != nil {
		t.Fatal(err)
	}
	g.Status = "in"
	at := g.Kickoff.Add(time.Hour)
	calls := 0
	fail := false
	a := Adapter{Board: Held{Games: []Game{g}, Source: "ESPN"}, NextBoard: at.Add(time.Hour), Fetch: func(_ context.Context, kind, _ string) ([]byte, error) {
		calls++
		if fail {
			return nil, errors.New("details down")
		}
		if kind == "summary" {
			raw := string(fixture(t, "espn-summary-401872991"))
			raw = strings.ReplaceAll(raw, `"state":"post","completed":true`, `"state":"in","completed":false`)
			// Summary status has no clock: replay the captured final at a live instant.
			raw = strings.ReplaceAll(raw, `"status":{"type":`, `"status":{"displayClock":"5:06","period":1,"type":`)
			return []byte(raw), nil
		}
		return fixture(t, "espn-core-plays"), nil
	}}
	windows := []Window{{Teams: [2]string{g.Home.Abbreviation, g.Away.Abbreviation}, Kickoff: g.Kickoff}}
	a.Refresh(context.Background(), at, true, windows, g.ID)
	if calls != 2 {
		t.Fatalf("details initial calls %d failure %s", calls, a.Details[g.ID].Failure)
	}
	a.Refresh(context.Background(), at.Add(14*time.Second), true, windows, g.ID)
	if calls != 2 {
		t.Fatal("summary polled before 15s")
	}
	fail = true
	a.Refresh(context.Background(), at.Add(15*time.Second), true, windows, g.ID)
	if calls != 3 {
		t.Fatal("summary not polled at 15s")
	}
	a.Refresh(context.Background(), at.Add(44*time.Second), true, windows, g.ID)
	if calls != 3 {
		t.Fatal("summary failure did not back off")
	}
	a.Refresh(context.Background(), at.Add(45*time.Second), true, windows, g.ID)
	if calls != 4 {
		t.Fatal("summary retry missing")
	}
}
