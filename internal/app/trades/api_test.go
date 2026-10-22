package trades

import (
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type fakeHost struct {
	calls    []string
	ready    error
	ownErr   error
	state    error
	snap     snapshot.Snapshot
	held     snapshot.Sourced[[]leaguefeed.PendingTrade]
	accepted struct {
		offer leaguefeed.PendingTrade
		id    string
	}
	acceptErr error
	builder   func(RawTradeBuilder, TradeBuilderView) (TradeBuilderView, error)
}

func (h *fakeHost) Ready() error { h.calls = append(h.calls, "ready"); return h.ready }
func (h *fakeHost) TradeState(string) (snapshot.Snapshot, snapshot.Sourced[[]leaguefeed.PendingTrade], error) {
	h.calls = append(h.calls, "state")
	return h.snap, h.held, h.state
}
func (h *fakeHost) DraftTradeAccept(_ string, offer leaguefeed.PendingTrade, _ snapshot.Snapshot,
	_ snapshot.Sourced[[]leaguefeed.PendingTrade], id string) (envelope.Receipt, error) {
	h.calls = append(h.calls, "accept")
	h.accepted.offer, h.accepted.id = offer, id
	return envelope.Receipt{CorrelationID: id}, h.acceptErr
}
func (h *fakeHost) TradeBuilder(r RawTradeBuilder, out TradeBuilderView) (TradeBuilderView, error) {
	h.calls = append(h.calls, "builder")
	return h.builder(r, out)
}
func (h *fakeHost) TradesBuildStamp() string { return "dev stamp" }

func tradeHost(t *testing.T) *fakeHost {
	t.Helper()
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	h := &fakeHost{}
	h.snap.Players.Value = []snapshot.Player{{ID: id, Name: "Player", Position: domain.PosWR}}
	h.snap.Franchises.Value = []snapshot.Franchise{{ID: "0001", Name: "One"}, {ID: "0002", Name: "Two"}}
	h.held = snapshot.Sourced[[]leaguefeed.PendingTrade]{Value: []leaguefeed.PendingTrade{
		{ID: "77", Offering: "0002", OfferedTo: "0001", Gives: []leaguefeed.Asset{{Player: &id}},
			Expires: time.Date(2026, 10, 7, 16, 0, 0, 0, time.UTC)},
	}, Provenance: snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: "2026-10-07T15:00:00Z"}}}
	return h
}

func TestTargetTradesWrapsStateAndDetachesOutput(t *testing.T) {
	t.Parallel()
	h := tradeHost(t)
	api := New(h)
	r, err := api.TargetTrades("0001")
	if err != nil || len(r.Offers) != 1 || r.Offers[0].OtherName != "Two" || r.Offers[0].Get[0].Name != "Player" {
		t.Fatal(r, err)
	}
	r.Offers[0].Get[0].Name = "mutated"
	if again, _ := api.TargetTrades("0001"); again.Offers[0].Get[0].Name != "Player" || h.held.Value[0].Gives[0].Player.String() != "0099" {
		t.Fatal("reading aliased the held offer", again)
	}
	h.state = errors.New("trade state: lineup selection: unknown franchise \"9\"")
	if _, err := api.TargetTrades("9"); err == nil || err.Error() != `target trades: trade state: lineup selection: unknown franchise "9"` {
		t.Fatal(err)
	}
}

func TestTargetDraftTradeAcceptRefusesBeforeTheHost(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]string{
		"unavailable": "target draft trade accept: pending trades unavailable: offline",
		"gone":        "target draft trade accept: offer 77 is no longer pending",
		"offerer":     "you made this offer; accept is the other owner's",
		"state":       "target draft trade accept: no snapshot",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h, franchise := tradeHost(t), "0001"
			switch name {
			case "unavailable":
				h.held.Value = nil
				h.held.Provenance.Freshness = domain.Freshness{State: domain.FreshFail, Note: "offline"}
			case "gone":
				h.held.Value = nil
			case "offerer":
				franchise = "0002"
			case "state":
				h.state = errors.New("no snapshot")
			}
			_, err := New(h).TargetDraftTradeAccept(franchise, "77")
			if err == nil || err.Error() != want || !slices.Equal(h.calls, []string{"state"}) {
				t.Fatalf("%v %v", err, h.calls)
			}
		})
	}
}

func TestTargetDraftTradeAcceptHandsTheOfferToTheHost(t *testing.T) {
	t.Parallel()
	h := tradeHost(t)
	r, err := New(h).TargetDraftTradeAccept("0001", "77")
	if err != nil || !slices.Equal(h.calls, []string{"state", "accept"}) || h.accepted.offer.ID != "77" ||
		!regexp.MustCompile(`^trade-[0-9a-f]{32}$`).MatchString(h.accepted.id) || r.CorrelationID != h.accepted.id {
		t.Fatal(r, err, h.calls, h.accepted)
	}
	h.acceptErr = errors.New("target draft trade accept: save: locked")
	if _, err := New(h).TargetDraftTradeAccept("0001", "77"); err == nil || err.Error() != "target draft trade accept: save: locked" {
		t.Fatal(err)
	}
}

func TestTargetTradeBuilderOrderAndReading(t *testing.T) {
	t.Parallel()
	h := tradeHost(t)
	h.builder = func(r RawTradeBuilder, out TradeBuilderView) (TradeBuilderView, error) {
		if out.Teams == nil || out.Plans == nil || len(out.Teams)+len(out.Plans) != 0 || out.Preview != nil {
			t.Error("builder start", out)
		}
		out.Teams = []TradeBuilderTeam{{ID: r.Mine}}
		out.Plans = []TradeBuilderPlan{{ID: "a"}, {ID: "b"}}
		return out, nil
	}
	api := New(h)
	if got := api.TargetTradeBuilder(RawTradeBuilder{Mine: "invalid", Action: "read"}); got.Status != report.ReadingFailed ||
		*got.AbsenceReason != `trade builder scope: contracts scope: invalid franchise "invalid"` || len(h.calls) != 0 {
		t.Fatal(got, h.calls)
	}
	h.ready = errors.New("startup failed")
	if got := api.TargetTradeBuilder(RawTradeBuilder{Mine: "0001", Action: "read"}); *got.AbsenceReason != "startup failed" ||
		!slices.Equal(h.calls, []string{"ready"}) || got.Value.Teams == nil || got.Value.Plans == nil {
		t.Fatal(got, h.calls)
	}
	h.ready, h.calls = nil, nil
	got := api.TargetTradeBuilder(RawTradeBuilder{Mine: "0001", Action: "read"})
	if got.Status != report.ReadingOK || got.Count != 2 || got.BuildStamp != "dev stamp" || got.Value.Teams[0].ID != "0001" ||
		got.Provenance.Source != "whatif.db; held MFL rosters and futureDraftPicks" || !slices.Equal(h.calls, []string{"ready", "builder"}) {
		t.Fatal(got, h.calls)
	}
	h.builder = func(_ RawTradeBuilder, out TradeBuilderView) (TradeBuilderView, error) {
		out.Teams, out.Plans = []TradeBuilderTeam{{ID: "kept"}}, nil
		return out, errors.New("saved trade plans: closed")
	}
	got = api.TargetTradeBuilder(RawTradeBuilder{Mine: "0001", Action: "read"})
	body, err := json.Marshal(got.Value)
	if err != nil || got.Status != report.ReadingFailed || *got.AbsenceReason != "saved trade plans: closed" ||
		got.Provenance.Freshness.State != domain.FreshFail || !strings.Contains(string(body), `"plans":null`) || got.Value.Teams[0].ID != "kept" {
		t.Fatal(got, string(body), err)
	}
}

// The catalog lists exactly the API's exported methods: Wails binds every one of them.
func TestCommandsMatchTheBoundSurface(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeFor[*API]()
	var methods, commands []string
	for i := range typ.NumMethod() {
		methods = append(methods, typ.Method(i).Name)
	}
	for _, c := range Commands() {
		commands = append(commands, c.Method)
	}
	if !slices.Equal(methods, commands) {
		t.Fatalf("methods %v, catalog %v", methods, commands)
	}
}

func TestTradesWireTwins(t *testing.T) {
	t.Parallel()
	wire := func(v any) string {
		t.Helper()
		body, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	cell := Cell{Text: "value", Sub: "qualifier", Detail: "detail", Held: true}
	asset := Asset{Team: "DAL", Contract: "$1.25 · 3 years", Now: &cell, Dynasty: &cell,
		Token: "0099", Name: "Player", Position: domain.PosWR}
	team := Team{ID: "0001", Name: "Team", Assets: []Asset{asset}}
	projection := Projection{Team: "Team", Roster: []Asset{asset},
		Figures: []Figure{{Label: "Cap", Before: "$1", After: "$2", Change: "+$1"}}}
	offer := Offer{TradeID: "77", Direction: "to_you", OtherID: "0001", OtherName: "Team",
		Give: []Asset{asset}, Get: []Asset{asset}, Expires: time.Date(2026, 10, 7, 16, 0, 0, 0, time.UTC), Comments: "Comment"}
	reading := Reading{Offers: []Offer{offer}, Provenance: snapshot.Provenance{
		Source: "pendingTrades", Kind: "live", Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: "2026-10-07T16:00:00Z"}}}
	for _, r := range []Reading{reading, {}, {Offers: []Offer{}}, {Offers: []Offer{{Give: []Asset{}}}}} {
		if wire(r) != wire(WireReading(r)) {
			t.Fatal("trade reading twin drift")
		}
	}
	for _, p := range []Projection{projection, {}, {Figures: []Figure{}, Roster: []Asset{}}} {
		if wire(p) != wire(WireProjection(p)) {
			t.Fatal("projection twin drift")
		}
	}
	for _, teams := range [][]Team{{team}, nil, {}, {{Assets: []Asset{}}}} {
		converted := WireTeams(teams)
		if wire(teams) != wire(converted) || wire(teams) != wire(ServiceTeams(converted)) {
			t.Fatal("team twin drift")
		}
	}
	raw := RawTradeBuilder{Mine: "0001", Other: "0025", Action: "save", ID: "id", Give: []string{"0099"}, Get: []string{"12345"}, Rationale: "why"}
	if wire(raw) != wire(Input(raw)) {
		t.Fatal("request twin drift")
	}
	// The converters detach cells: a wire value never shares a pointer with the service value.
	converted := WireTeams([]Team{team})
	converted[0].Assets[0].Now.Text = "mutated"
	if cell.Text != "value" || ServiceTeams(converted)[0].Assets[0].Now == converted[0].Assets[0].Now {
		t.Fatal("aliased cell")
	}
}

func (h *fakeHost) RequireOwnFranchise(string) error { return h.ownErr }
