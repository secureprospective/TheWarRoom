package control

import (
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/sitenews"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

// fakeHost is an offline Host: every read is a field, every call is counted.
type fakeHost struct {
	ready      error
	readyCalls atomic.Int32
	names      map[string]string
	namesErr   error
	fetches    atomic.Int32
	onFetch    func()
	news       []sitenews.RawItem
	newsErr    error
	alerts     AlertReading
	alertsErr  error
	week       WeekSnapshot
	now        time.Time
	health     error
	windows    []string
	windowErr  error
	lineupWeek int
	txs        report.HeldFeed[[]leaguefeed.Transaction]
	cfg        league.RawConfig
}

func (f *fakeHost) Ready() error                    { f.readyCalls.Add(1); return f.ready }
func (f *fakeHost) ScoringConfig() league.RawConfig { return f.cfg }
func (f *fakeHost) FetchRuleNames() (map[string]string, domain.Freshness, error) {
	f.fetches.Add(1)
	if f.onFetch != nil {
		f.onFetch()
	}
	return f.names, report.LiveFreshness(time.Now()), f.namesErr
}
func (f *fakeHost) FetchSiteNews() ([]sitenews.RawItem, domain.Freshness, error) {
	f.fetches.Add(1)
	if f.onFetch != nil {
		f.onFetch()
	}
	return f.news, report.LiveFreshness(time.Now()), f.newsErr
}
func (f *fakeHost) Zone() *time.Location                            { return time.UTC }
func (f *fakeHost) OpenMFLAccounting() error                        { return nil }
func (f *fakeHost) RulebookMarkdown() []byte                        { return nil }
func (f *fakeHost) Params() ParamsResult                            { return ParamsResult{} }
func (f *fakeHost) SetParam(string, string, float64) SetParamResult { return SetParamResult{} }
func (f *fakeHost) ResetParam(string, string) SetParamResult        { return SetParamResult{} }
func (f *fakeHost) KeyStatus() (MFLKeyStatus, error)                { return MFLKeyStatus{}, nil }
func (f *fakeHost) SetKey(string) (MFLKeyStatus, error)             { return MFLKeyStatus{}, nil }
func (f *fakeHost) DeleteKey() (MFLKeyStatus, error)                { return MFLKeyStatus{}, nil }
func (f *fakeHost) KeyUnavailable(string) MFLKeyStatus              { return MFLKeyStatus{} }
func (f *fakeHost) SeasonFeeds() (report.HeldFeed[[]leaguefeed.Transaction],
	report.HeldFeed[leaguefeed.Lineups], int, report.HeldFeed[[]leaguefeed.PendingTrade]) {
	var lineups report.HeldFeed[leaguefeed.Lineups]
	return f.txs, lineups, f.lineupWeek, report.HeldFeed[[]leaguefeed.PendingTrade]{}
}
func (f *fakeHost) Alerts(string) (AlertReading, error) { return f.alerts, f.alertsErr }
func (f *fakeHost) SourceHealth() (string, string, error) {
	return "live", "All sources live", f.health
}
func (f *fakeHost) Week() WeekSnapshot { return f.week }
func (f *fakeHost) Now() time.Time     { return f.now }
func (f *fakeHost) SetWindow(kind, at string) error {
	f.windows = append(f.windows, kind+"="+at)
	return f.windowErr
}

func TestLeagueSettingsHoldsNamesAndReadsDuringALoad(t *testing.T) {
	t.Parallel()
	cfg, err := league.Parse(readTestdata(t, "internal/ingestion/league/testdata/league-2026.json"),
		readTestdata(t, "internal/ingestion/league/testdata/rules-2026.json"))
	if err != nil {
		t.Fatal(err)
	}
	host := &fakeHost{ready: errors.New("no db"), cfg: cfg}
	held := &Held{}
	api := New(host, held)
	if _, err := api.TargetLeagueSettings(); err == nil || err.Error() != "league settings: startup: no db" ||
		host.fetches.Load() != 0 {
		t.Fatalf("not ready: %v, %d fetches", err, host.fetches.Load())
	}
	host.ready, host.namesErr = nil, errors.New("mfl down")
	r, err := api.TargetLeagueSettings()
	if err != nil || !strings.Contains(r.Note, "codes are shown: mfl down") {
		t.Fatalf("failed names: %+v %v", r.Note, err)
	}
	// While one read is in its fetch, the holder answers another reader: no lock spans the Host call.
	host.namesErr, host.names = nil, map[string]string{"#P": "Passing TD"}
	host.onFetch = func() {
		done := make(chan struct{})
		go func() { held.SnapshotRuleNames(); held.SnapshotSiteNews(); close(done) }()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("holder locked across the fetch")
		}
	}
	if _, err := api.TargetLeagueSettings(); err != nil {
		t.Fatal(err)
	}
	before := host.fetches.Load()
	if _, err := api.TargetLeagueSettings(); err != nil || host.fetches.Load() != before {
		t.Fatalf("held names fetched again: %v", err)
	}
}

func TestSiteNewsRefreshesOnlyWhenFailedOrOld(t *testing.T) {
	t.Parallel()
	host := &fakeHost{newsErr: errors.New("mfl down")}
	api := New(host, &Held{})
	r, err := api.TargetSiteNews()
	if err != nil || r.Note == "" || host.fetches.Load() != 1 {
		t.Fatalf("failed news: %+v %v", r, err)
	}
	host.newsErr = nil
	if _, err := api.TargetSiteNews(); err != nil || host.fetches.Load() != 2 {
		t.Fatalf("a failed read must refresh on the next open: %v", err)
	}
	if _, err := api.TargetSiteNews(); err != nil || host.fetches.Load() != 2 {
		t.Fatalf("an hour-fresh read must be served held: %v", err)
	}
}

func TestSeasonReadingSerializesEmptyFeedsAsLists(t *testing.T) {
	t.Parallel()
	host := &fakeHost{ready: errors.New("no db")}
	api := New(host, &Held{})
	if _, err := api.TargetSeason(); err == nil || err.Error() != "target season: startup: no db" {
		t.Fatal(err)
	}
	host.ready, host.lineupWeek = nil, 6
	r, err := api.TargetSeason()
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(r)
	for _, want := range []string{`"transactions":{"value":[]`, `"matchups":[]`, `"franchises":[]`,
		`"pendingTrades":{"value":[]`, `"source":"liveScoring W=6"`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("season reading lacks %s: %s", want, body)
		}
	}
	host.txs.Value = []leaguefeed.Transaction{{Kind: "TRADE"}}
	r, _ = api.TargetSeason()
	r.Transactions.Value[0].Kind = "changed"
	if host.txs.Value[0].Kind != "TRADE" {
		t.Fatal("season reading aliases the host's feed")
	}
}

func TestShellStatusAndWindowRefusalOrder(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	host := &fakeHost{ready: errors.New("no db"), now: now, week: WeekSnapshot{Week: leagueweek.Week{Number: 5,
		Games: []leagueweek.Game{{Kickoff: now.Add(-time.Hour)}, {Kickoff: now.Add(-4 * time.Hour), Final: true}}}}}
	api := New(host, &Held{})
	if _, err := api.TargetShellStatus(); err == nil || err.Error() != "shell status: startup: no db" {
		t.Fatal(err)
	}
	// The request is checked before startup, so a bad date never waits on it.
	err := api.TargetSetWindow("nope", "")
	if err == nil || !strings.Contains(err.Error(), "unknown league window") || host.readyCalls.Load() != 1 {
		t.Fatalf("bad kind: %v after %d ready calls", err, host.readyCalls.Load())
	}
	if err := api.TargetSetWindow("draft", "tomorrow"); err == nil ||
		!strings.HasPrefix(err.Error(), "set a date") {
		t.Fatal(err)
	}
	if err := api.TargetSetWindow("draft", ""); err == nil || err.Error() != "set league date: startup: no db" ||
		len(host.windows) != 0 {
		t.Fatal(err, host.windows)
	}
	host.ready = nil
	if err := api.TargetSetWindow("cut_day", "2026-11-01T18:00:00Z"); err != nil ||
		len(host.windows) != 1 || host.windows[0] != "cut_day=2026-11-01T18:00:00Z" {
		t.Fatal(err, host.windows)
	}
	host.windowErr = errors.New("this window already follows the rulebook")
	err = api.TargetSetWindow("cut_day", "")
	if !errors.Is(err, host.windowErr) || err.Error() != host.windowErr.Error() {
		t.Fatal("the host's refusal must reach the GM unwrapped:", err)
	}
	s, err := api.TargetShellStatus()
	if err != nil || s.State != "live" || s.Gameday != "Week 5 · 1 NFL game in progress · 1 final" {
		t.Fatalf("status: %+v %v", s, err)
	}
	host.health = errors.New("history unreadable")
	if _, err := api.TargetShellStatus(); err == nil || err.Error() != "shell status: history unreadable" {
		t.Fatal(err)
	}
}

func TestAlertsAreSortedAndRefusalsPrefixed(t *testing.T) {
	t.Parallel()
	host := &fakeHost{alertsErr: errors.New("lineup: not ready")}
	api := New(host, &Held{})
	if _, err := api.TargetAlerts("0001"); err == nil || err.Error() != "target alerts: lineup: not ready" {
		t.Fatal(err)
	}
	host.alertsErr = nil
	host.alerts = AlertReading{Alerts: []Alert{{Kind: "trades", Urgency: "U1"}, {Kind: "bye", Urgency: "U2"},
		{Kind: "ir", Urgency: "U1", PlayerID: "0042"}}, Unavailable: []Unavailable{}}
	r, err := api.TargetAlerts("0001")
	if err != nil || r.Alerts[0].Kind != "bye" || r.Alerts[1].Kind != "ir" || r.Alerts[2].Kind != "trades" {
		t.Fatalf("alerts out of order: %+v %v", r.Alerts, err)
	}
}
