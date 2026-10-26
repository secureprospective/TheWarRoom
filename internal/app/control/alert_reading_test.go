package control

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestLineupAlertDecisions(t *testing.T) {
	t.Parallel()
	now := time.Unix(1800000000, 0)
	week := leagueweek.Week{Number: 6, Games: []leagueweek.Game{{Kickoff: now.Add(time.Hour)}}}
	for _, tc := range []struct {
		name                string
		input               LineupAlertInput
		week                leagueweek.Week
		failed, unavailable string
		alert               bool
	}{
		{name: "unknown lock", week: leagueweek.Week{}, unavailable: "lineup lock unknown"},
		{name: "failed week", week: week, failed: "offline", unavailable: "lineup lock unknown; offline"},
		{name: "last lock passed", week: leagueweek.Week{Games: []leagueweek.Game{{Kickoff: now}}}},
		{name: "feed fails first", week: week, input: LineupAlertInput{Provenance: snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshFail, Note: "feed unavailable"}}}, unavailable: "feed unavailable"},
		{name: "wrong week", week: week, input: LineupAlertInput{Week: 5}, unavailable: "saved lineup is week 5, current week is 6"},
		{name: "rules unavailable", week: week, input: LineupAlertInput{Week: 6, RulesSource: snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshFail, Note: "rules missing"}}}, unavailable: "rules missing"},
		{name: "full and legal", week: week, input: LineupAlertInput{Week: 6, Check: lineup.Result{Full: true, Legal: true}}},
		{name: "partial", week: week, input: LineupAlertInput{Week: 6, Check: lineup.Result{Legal: true, Problems: []lineup.Problem{{Message: "Need a kicker. "}, {Message: "Need a quarterback; "}}}}, alert: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := AlertReading{}
			LineupAlert(&r, tc.input, tc.week, tc.failed, now)
			if tc.unavailable != "" {
				if len(r.Unavailable) != 1 || r.Unavailable[0].Kind != "lineup" || r.Unavailable[0].Note != tc.unavailable {
					t.Fatalf("unavailable: %+v", r)
				}
			} else if len(r.Unavailable) != 0 {
				t.Fatalf("unexpected unavailable: %+v", r)
			}
			if tc.alert {
				if len(r.Alerts) != 1 || r.Alerts[0].Detail != "Need a kicker; Need a quarterback" || r.Alerts[0].At != now.Add(time.Hour) || r.Alerts[0].Urgency != AlertUrgency(now, now.Add(time.Hour)) || r.Alerts[0].Provenance != tc.input.Provenance {
					t.Fatalf("alert: %+v", r)
				}
			} else if len(r.Alerts) != 0 {
				t.Fatalf("unexpected alert: %+v", r)
			}
		})
	}
}

func TestTradeAlertDecisions(t *testing.T) {
	t.Parallel()
	now := time.Unix(1800000000, 0)
	for _, tc := range []struct {
		name        string
		input       TradeAlertInput
		title       string
		at          time.Time
		unavailable bool
	}{
		{name: "none"},
		{name: "only outgoing or expired", input: TradeAlertInput{Offers: []TradeAlertOffer{{Direction: "from_you", Expires: now.Add(time.Hour)}, {Direction: "to_you", Expires: now}, {Direction: "to_you", Expires: now.Add(-time.Hour)}}}},
		{name: "undated", input: TradeAlertInput{Offers: []TradeAlertOffer{{Direction: "to_you"}}}, title: "1 offer to you"},
		{name: "earliest known expiry", input: TradeAlertInput{Offers: []TradeAlertOffer{{Direction: "to_you", Expires: now.Add(4 * time.Hour)}, {Direction: "to_you"}, {Direction: "to_you", Expires: now.Add(2 * time.Hour)}}}, title: "3 offers to you", at: now.Add(2 * time.Hour)},
		{name: "failed evidence", input: TradeAlertInput{Provenance: snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshFail, Note: "offline"}}, Offers: []TradeAlertOffer{{Direction: "to_you"}}}, unavailable: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := AlertReading{}
			TradeAlert(&r, tc.input, now)
			if tc.unavailable {
				if len(r.Unavailable) != 1 || r.Unavailable[0].Kind != "trades" || r.Unavailable[0].Note != "offline" {
					t.Fatalf("unavailable: %+v", r)
				}
			} else if len(r.Unavailable) != 0 {
				t.Fatalf("unexpected unavailable: %+v", r)
			}
			if tc.title != "" {
				if len(r.Alerts) != 1 || r.Alerts[0].Title != tc.title || r.Alerts[0].At != tc.at || r.Alerts[0].Provenance != tc.input.Provenance {
					t.Fatalf("alert: %+v", r)
				}
			} else if len(r.Alerts) != 0 {
				t.Fatalf("unexpected alerts: %+v", r)
			}
		})
	}
}

func alertRoster(t *testing.T) snapshot.Snapshot {
	t.Helper()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	return snapshot.Snapshot{Rosters: snapshot.Sourced[[]snapshot.Roster]{Provenance: snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshLive}}, Value: []snapshot.Roster{{FranchiseID: "0001", Players: []snapshot.RosterPlayer{{ID: id, RosterStatus: domain.RosterActive}}}}}, Players: snapshot.Sourced[[]snapshot.Player]{Value: []snapshot.Player{{ID: id, Name: "Kicker", Position: domain.PosK, Team: "DET"}}}}
}

func TestIRAlertsOnlyKnownIneligibleOwnPlayers(t *testing.T) {
	t.Parallel()
	snap := alertRoster(t)
	p := snap.Rosters.Value[0].Players[0]
	p.RosterStatus = domain.RosterIR
	snap.Rosters.Value[0].Players = []snapshot.RosterPlayer{p, p}
	snap.Rosters.Value = append(snap.Rosters.Value, snapshot.Roster{FranchiseID: "other", Players: []snapshot.RosterPlayer{p}})
	injuries := snapshot.Injuries{Status: map[string]string{"0042": ""}, Provenance: snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshLive}}}
	for _, tc := range []struct {
		name                            string
		loaded                          bool
		rosterFresh, injuryFresh        string
		status                          string
		missingName, alert, unavailable bool
	}{
		{name: "not loaded", rosterFresh: domain.FreshLive, injuryFresh: domain.FreshLive, unavailable: true},
		{name: "stale roster", loaded: true, rosterFresh: domain.FreshStale, injuryFresh: domain.FreshLive, unavailable: true},
		{name: "stale injuries", loaded: true, rosterFresh: domain.FreshLive, injuryFresh: domain.FreshStale, unavailable: true},
		{name: "eligible", loaded: true, rosterFresh: domain.FreshLive, injuryFresh: domain.FreshLive, status: "IR"},
		{name: "ineligible", loaded: true, rosterFresh: domain.FreshLive, injuryFresh: domain.FreshLive, status: "Q", alert: true},
		{name: "name fallback", loaded: true, rosterFresh: domain.FreshLive, injuryFresh: domain.FreshLive, status: "Q", missingName: true, alert: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Each case owns the locally constructed publication; the production function only reads it.
			held := snap
			held.Rosters.Provenance.Freshness.State = tc.rosterFresh
			if tc.missingName {
				held.Players.Value = nil
			}
			injury := injuries
			injury.Provenance.Freshness.State = tc.injuryFresh
			injury.Status = map[string]string{"0042": tc.status}
			calls := 0
			r := AlertReading{}
			IRAlerts(&r, "0001", held, tc.loaded, injury, func() string { calls++; return "held-target" })
			if tc.alert {
				want := "Possible IR violation: Kicker"
				if tc.missingName {
					want = "Possible IR violation: Player name unavailable"
				}
				if len(r.Alerts) != 1 || r.Alerts[0].Title != want || r.Alerts[0].MFLURL != "held-target" || r.Alerts[0].PlayerID != "0042" || r.Alerts[0].Provenance != injury.Provenance || calls != 1 {
					t.Fatalf("alert: %+v calls %d", r, calls)
				}
			} else if len(r.Alerts) != 0 || calls != 0 {
				t.Fatalf("unexpected alert or target read: %+v %d", r, calls)
			}
			if (len(r.Unavailable) != 0) != tc.unavailable {
				t.Fatalf("unavailable: %+v", r)
			}
		})
	}
}

func TestByeInputsAndRoster(t *testing.T) {
	t.Parallel()
	snap := alertRoster(t)
	at := time.Unix(1800000000, 0)
	byes := report.HeldFeed[[]nflschedule.RawBye]{Value: []nflschedule.RawBye{{Team: "DET", Week: "6"}}, FetchedAt: at}
	week := leagueweek.Week{Number: 6}
	for _, tc := range []struct {
		name string
		snap snapshot.Snapshot
		mine string
		byes report.HeldFeed[[]nflschedule.RawBye]
		week leagueweek.Week
		want string
	}{
		{"held", snap, "0001", byes, week, ""},
		{"no feed", snap, "0001", report.HeldFeed[[]nflschedule.RawBye]{}, week, "Bye-week check unavailable: NFL bye weeks not held."},
		{"no week", snap, "0001", byes, leagueweek.Week{}, "Bye-week check unavailable: current NFL week not held."},
		{"other roster", snap, "0002", byes, week, "Bye-week check unavailable: your roster not held or out of date."},
	} {
		if got := ByeInputsNote(tc.snap, tc.mine, tc.byes, tc.week); got != tc.want {
			t.Errorf("%s: %q", tc.name, got)
		}
	}
	players, names, byeMap := ByeRoster(snap, "0001", byes.Value)
	if !reflect.DeepEqual(players, []lineup.ByePlayer{{Position: domain.PosK, Team: "DET", Active: true}}) || !reflect.DeepEqual(names, []string{"Kicker"}) || !reflect.DeepEqual(byeMap, map[string]int{"DET": 6}) {
		t.Fatalf("join: %v %v %v", players, names, byeMap)
	}
}

func TestByeAlertsHorizonAndFailurePrecedence(t *testing.T) {
	t.Parallel()
	now := time.Unix(1800000000, 0)
	in := ByeAlertInput{Snapshot: alertRoster(t), Franchise: "0001", Byes: report.HeldFeed[[]nflschedule.RawBye]{Value: []nflschedule.RawBye{{Team: "DET", Week: "6"}}, FetchedAt: now}, Starters: league.Starters{Count: "1", IOPStarters: "1", IDPStarters: "0", Positions: []league.PositionLimit{{Name: "PK", Limit: "1"}}}, RulesFreshness: func() domain.Freshness { return domain.Freshness{State: domain.FreshLive} }, Now: now}
	for _, week := range []int{4, 5, 6, 7} {
		in.Week.Number = week
		r := AlertReading{}
		var locks []int
		ByeAlerts(&r, in, func(w int) time.Time { locks = append(locks, w); return time.Time{} })
		if week == 5 || week == 6 {
			if len(r.Alerts) != 1 || r.Alerts[0].Title != "Week 6: no kicker available" || !reflect.DeepEqual(locks, []int{6}) || len(r.Unavailable) != 1 {
				t.Fatalf("week %d: %+v locks %v", week, r, locks)
			}
		} else if len(r.Alerts) != 0 || len(r.Unavailable) != 0 || len(locks) != 0 {
			t.Fatalf("outside horizon: %+v %v", r, locks)
		}
	}
	in.Byes = report.HeldFeed[[]nflschedule.RawBye]{}
	in.Starters = league.Starters{}
	in.RulesFreshness = func() domain.Freshness { t.Fatal("invalid rules must not read freshness"); return domain.Freshness{} }
	in.WeekFailure = "offline"
	r := AlertReading{}
	ByeAlerts(&r, in, func(int) time.Time { t.Fatal("read lock with failed inputs"); return time.Time{} })
	if len(r.Unavailable) != 1 || r.Unavailable[0].Note != "Bye-week check unavailable: current NFL week could not be checked." {
		t.Fatalf("precedence: %+v", r)
	}
}

func TestByeWeekLockReadsOnlyOtherWeek(t *testing.T) {
	t.Parallel()
	now := time.Unix(1800000000, 0)
	lock := now.Add(time.Hour)
	current := leagueweek.Week{Number: 6, Games: []leagueweek.Game{{Kickoff: lock}}}
	if got := ByeWeekLock(current, 6, now, func(int) ([]nflschedule.RawWeek, bool, error) {
		t.Fatal("current week read archive")
		return nil, false, nil
	}); got != lock {
		t.Fatal(got)
	}
	raw := nflschedule.RawWeek{Week: "7", Matchups: []nflschedule.RawMatchup{{Teams: [2]string{"DET", "NYJ"}, Kickoff: strconv.FormatInt(lock.Unix(), 10)}}}
	for _, tc := range []struct {
		name  string
		rows  []nflschedule.RawWeek
		found bool
		err   error
		want  time.Time
	}{
		{name: "held", rows: []nflschedule.RawWeek{raw}, found: true, want: lock},
		{name: "absent"}, {name: "empty", found: true}, {name: "bad", rows: []nflschedule.RawWeek{{Week: "bad"}}, found: true}, {name: "read failed", rows: []nflschedule.RawWeek{raw}, found: true, err: errors.New("read failed")},
	} {
		if got := ByeWeekLock(current, 7, now, func(w int) ([]nflschedule.RawWeek, bool, error) {
			if w != 7 {
				t.Fatal(w)
			}
			return tc.rows, tc.found, tc.err
		}); !got.Equal(tc.want) {
			t.Errorf("%s: %v", tc.name, got)
		}
	}
}
