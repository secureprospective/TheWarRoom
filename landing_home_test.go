package main

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	commsapp "github.com/secureprospective/TheWarRoom/internal/app/comms"
	"github.com/secureprospective/TheWarRoom/internal/app/comms/commstest"
	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/store/state"

	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestLandingDropUsesLastHeldWeekNotMirror(t *testing.T) {
	t.Parallel()
	in := realComms(t)
	in.mirror.Players = nil
	id, err := playerid.New("16162")
	if err != nil {
		t.Fatal(err)
	}
	first, last, future := int64(100), int64(200), int64(400)
	in.rosterTimes = rosterWeekTimes(facts.NflGameColumns{Year: []int64{2026, 2026, 2026, 2026},
		Week: []int64{1, 2, 2, 3}, Kickoff: []*int64{&first, &first, &last, &future}})
	salary, old, later := 0.58, 0.4, 9.0
	end := "2028"
	in.rosters = facts.RosterWeekColumns{Year: []int64{2026, 2026, 2026, 2026, 2026},
		Week: []int64{1, 3, 2, 0, 2}, Fid: []string{"0001", "0001", "0001", "0001", "0002"},
		Pid:          []string{"16162", "16162", "16162", "16162", "16162"},
		Salary:       []*float64{&old, &later, &salary, &later, &later},
		ContractYear: []*string{&end, &end, &end, &end, &end}}
	tx := leaguefeed.Transaction{Kind: "FREE_AGENT", Franchise: "0001", Time: time.Unix(300, 0),
		AddsDrops: &leaguefeed.RosterChange{Out: []leaguefeed.Asset{{Player: &id}}}}
	got := in.transactionSentence(tx)
	if !strings.Contains(got, "dropped Jahmyr Gibbs") || !strings.Contains(got, "$580,000, contract through 2028") {
		t.Fatal(got)
	}
	tx.Franchise = "0003"
	if got := in.transactionSentence(tx); !strings.Contains(got, "salary not held") {
		t.Fatal(got)
	}
}

func TestLandingNewsTodayCacheNormalOldCacheOneNotice(t *testing.T) {
	t.Parallel()
	for _, days := range []int{0, 2} {
		t.Run(strconv.Itoa(days), func(t *testing.T) {
			in := realResearch(t)
			a := targetTestApp(t)
			a.core.Lookup = in.lookup
			a.core.FallbackLog = func(string) {}
			editResearch(a, func(value *researchHeld) { value.links = map[string]string{} })
			at := time.Now().AddDate(0, 0, -days).Truncate(time.Second)
			news := map[string]heldSeasonFeed[[]researchfeeds.News]{
				"16162": {FetchedAt: at, Value: in.held.news["16162"].Value},
				"13589": {FetchedAt: at, Value: []researchfeeds.News{}},
			}
			a.mergeResearchArchive(context.Background(), nil, in.held.depth, in.injuries, news)
			in.held.news, in.now = a.snapshotResearch().news, time.Now()
			out := buildResearch(in, RawResearchScope{Mine: "0001"})
			if days == 0 {
				if len(out.News.Summary) != 0 || out.News.Status != "ok" || a.snapshotResearch().news["16162"].RefreshError != "" {
					t.Fatal(out.News)
				}
				if !strings.Contains(out.News.AsOf, "held · fetched "+reportTime(at, time.UTC)) {
					t.Fatal(out.News.AsOf)
				}
			} else if len(out.News.Summary) != 1 || !strings.Contains(out.News.Summary[0].Cells[1], "archived copy") {
				t.Fatal(out.News)
			}
		})
	}
}

func TestLandingNewsRefreshFailureOneNotice(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	for _, id := range []string{"16162", "13589"} {
		h := in.held.news[id]
		h.FetchedAt, h.RefreshError = in.now, "fixture offline"
		in.held.news[id] = h
	}
	out := buildResearch(in, RawResearchScope{})
	if len(out.News.Summary) != 1 || !strings.Contains(out.News.Summary[0].Cells[1], "refresh failed") {
		t.Fatal(out.News)
	}
}

func TestLandingNewsOrderUsesFetchTimeRegardlessOfOwner(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	in.held.news["16162"] = heldSeasonFeed[[]researchfeeds.News]{FetchedAt: in.now.AddDate(0, 0, -2),
		Value: []researchfeeds.News{{Article: "older", Headline: "older", Published: "1 hour"}}}
	in.held.news["13589"] = heldSeasonFeed[[]researchfeeds.News]{FetchedAt: in.now,
		Value: []researchfeeds.News{{Article: "newer", Headline: "newer", Published: "1 day"}}}
	out := buildResearch(in, RawResearchScope{})
	if out.Articles[0].Article != "newer" {
		t.Fatal(out.Articles)
	}
	out = buildResearch(in, RawResearchScope{Mine: "0001"})
	if out.Articles[0].Article != "newer" || out.News.Rows[0].Mine {
		t.Fatal(out.Articles, out.News.Rows)
	}
}

type landingNewsTransport struct{ asked []string }

func (f *landingNewsTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	ids := strings.Split(r.URL.Query().Get("P"), ",")
	f.asked = append(f.asked, ids...)
	return seasonResponse(`{"playerProfiles":{"playerProfile":{"id":"16162","name":"Fixture Player","news":{}}}}`), nil
}

func TestLandingTeamPositionsNeverEnterNewsBatch(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	positions := []string{"RB", "Def", "ST", "Off", "TMDL", "TMQB"}
	ids := []string{"16162", "0501", "0532", "17400", "17401", "17402"}
	raw := []players.RawPlayer{}
	for i, id := range ids {
		raw = append(raw, players.RawPlayer{ID: id, Name: "Fixture", Position: positions[i]})
	}
	lookup, err := normalize.NewLookup(raw)
	if err != nil {
		t.Fatal(err)
	}
	a.core.Lookup = lookup
	mirror, _, _ := a.core.League.ContractsView()
	mirror.Players = slices.Clone(mirror.Players[:1])
	mirror.Players[0].MFLID = ids[0]
	for _, id := range ids[1:] {
		p := mirror.Players[0]
		p.MFLID = id
		mirror.Players = append(mirror.Players, p)
	}
	if _, err := a.core.League.Replace(a.core.Ctx, mirror); err != nil {
		t.Fatal(err)
	}
	transport := &landingNewsTransport{}
	a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}
	a.core.SeasonPause = func(context.Context, time.Duration) error { return nil }
	a.core.FallbackLog = func(string) {}
	if err := researchAPI(a).TargetResearchNewsRefresh(""); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(transport.asked, ids[:1]) {
		t.Fatal(transport.asked)
	}
}

func TestLandingRejectedNewsIDCannotLoseGoodPlayers(t *testing.T) {
	t.Parallel()
	ids := []string{"16162", "13589", "17400"}
	calls, pauses := 0, 0
	fetch := func(_ context.Context, ids []string) (map[string][]researchfeeds.News, error) {
		calls++
		if slices.Contains(ids, "17400") {
			return nil, errors.New("mfl: no valid player ids")
		}
		rows := map[string][]researchfeeds.News{}
		for _, id := range ids {
			rows[id] = []researchfeeds.News{{Headline: "held"}}
		}
		return rows, nil
	}
	held := map[string]heldSeasonFeed[[]researchfeeds.News]{}
	pause := func(context.Context, time.Duration) error { pauses++; return nil }
	if err := readResearchNewsBatches(context.Background(), held, 2026, ids, time.Now(), noNewsArchive, pause, fetch, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if len(held["16162"].Value) != 1 || len(held["13589"].Value) != 1 || calls != pauses {
		t.Fatal(held, calls, pauses)
	}
	// MFL holds no profile for 17400: that is its answer (no news), held for the day, not a lost read.
	if h := held["17400"]; h.RefreshError != "" || h.Value == nil || len(h.Value) != 0 || h.FetchedAt.IsZero() {
		t.Fatalf("player with no MFL profile: %+v", h)
	}
	before := calls
	if err := readResearchNewsBatches(context.Background(), held, 2026, ids, time.Now(), noNewsArchive, pause, fetch, func(string) {}); err != nil {
		t.Fatal(err)
	}
	if calls != before {
		t.Fatalf("asked MFL again the same day: %d calls", calls-before)
	}
}

func TestLandingShellFeedsDailyLiveOrArchive(t *testing.T) {
	t.Parallel()
	for _, offline := range []bool{false, true} {
		a := targetTestApp(t)
		transport := &seasonTransport{bodies: map[string]string{
			"calendar":  `{"calendar":{"event":{"id":"live","type":"TRADE","title":"Live event"}}}`,
			"tradeBait": `{"tradeBaits":{"tradeBait":{"franchise_id":"0025","inExchangeFor":"Live bait"}}}`,
		}}
		var err error
		// MFL serves the calendar only to a keyed request; trade bait stays unkeyed.
		a.core.MFLClient, err = mfl.New("api", 10000, mfl.WithTransport(transport), mfl.WithKeySource(
			func(context.Context) (mflkey.Key, error) { return "owner-key", nil }))
		if err != nil {
			t.Fatal(err)
		}
		a.core.SeasonPause = func(context.Context, time.Duration) error { return nil }
		a.core.FallbackLog = func(string) {}
		at := time.Now().AddDate(0, 0, -2).Truncate(time.Second)
		recordSeasonArchive(t, a, "calendar", "", `{"calendar":{"event":{"id":"old","type":"TRADE","title":"Held event"}}}`, at)
		recordSeasonArchive(t, a, "tradeBait", "", `{"tradeBaits":{"tradeBait":{"franchise_id":"0025","inExchangeFor":"Held bait"}}}`, at)
		if offline {
			transport.bodies["calendar"], transport.bodies["tradeBait"] = `{"error":"offline"}`, `{"error":"offline"}`
		}
		a.refreshShellFeeds(a.core.Ctx)
		a.refreshShellFeeds(a.core.Ctx)
		if !slices.Equal(transport.requests, []string{"calendar", "tradeBait"}) ||
			!slices.Equal(transport.keyed, []string{"calendar"}) {
			t.Fatal(transport.requests, transport.keyed)
		}
		wantEvent, wantBait, kind := "Live event", "Live bait", "live · fetched "
		if offline {
			wantEvent, wantBait, kind = "Held event", "Held bait", "archived copy · fetched "
		}
		calendar := readShellFeed(a.core.Ctx, a, &a.core.SeasonCalendar, "calendar", ingestion.ParseCalendar)
		bait := readShellFeed(a.core.Ctx, a, &a.core.SeasonTradeBait, "tradeBait", ingestion.ParseTradeBait)
		if len(calendar.Value) != 1 || calendar.Value[0].Title != wantEvent || len(bait.Value) != 1 || bait.Value[0].Want != wantBait {
			t.Fatal(calendar, bait)
		}
		for _, source := range []string{shellFeedTime(calendar, time.Local), shellFeedTime(bait, time.Local)} {
			if !strings.HasPrefix(source, kind) || offline && !strings.Contains(source, reportTime(at, time.Local)) {
				t.Fatal(source)
			}
		}
	}
}

func TestLandingAlertOrderUrgencyGravityID(t *testing.T) {
	t.Parallel()
	alerts := []controlapp.Alert{
		{Kind: "lineup", Urgency: "U2", At: time.Unix(10, 0)},
		{Kind: "ir", Urgency: "U2", PlayerID: "16162", At: time.Unix(1, 0)},
		{Kind: "ir", Urgency: "U2", PlayerID: "13589", At: time.Unix(100, 0)},
		{Kind: "trades", Urgency: "U3"},
	}
	slices.SortFunc(alerts, controlapp.CompareAlerts)
	if alerts[0].Kind != "trades" || alerts[1].PlayerID != "13589" ||
		alerts[2].PlayerID != "16162" || alerts[3].Kind != "lineup" {
		t.Fatal(alerts)
	}
}

// The Now landing scenario retains its fixture facade without owning Comms construction.
type commsLandingInput struct {
	commsapp.Input
	mirror      state.MirrorSnapshot
	rosters     facts.RosterWeekColumns
	rosterTimes map[[2]int64]int64
}

func realComms(t *testing.T) commsLandingInput {
	t.Helper()
	in := commstest.Real(t)
	return commsLandingInput{Input: in, mirror: in.Mirror, rosters: in.Rosters, rosterTimes: in.RosterTimes}
}

func rosterWeekTimes(games facts.NflGameColumns) map[[2]int64]int64 {
	return commsapp.RosterWeekTimes(games)
}

func (in commsLandingInput) transactionSentence(tx leaguefeed.Transaction) string {
	in.Mirror, in.Rosters, in.RosterTimes = in.mirror, in.rosters, in.rosterTimes
	in.Transactions.Value = []leaguefeed.Transaction{tx}
	board, err := commsapp.Build(in.Input, commsapp.RawCommsScope{})
	if err != nil {
		return err.Error()
	}
	return board.Events[0].Sentence
}
