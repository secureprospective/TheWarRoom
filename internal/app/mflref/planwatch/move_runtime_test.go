package planwatch

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type memoryMoves struct {
	value   envelope.Envelope
	saveErr error
	saves   int
}

func (m *memoryMoves) Get(context.Context, string) (envelope.Envelope, error) { return m.value, nil }
func (m *memoryMoves) Save(_ context.Context, e envelope.Envelope) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.value = e
	m.saves++
	return nil
}
func (m *memoryMoves) Awaiting(context.Context) ([]envelope.Envelope, error) {
	return []envelope.Envelope{m.value}, nil
}
func (m *memoryMoves) BidFacts(context.Context, string) (envelope.BidFacts, error) {
	return envelope.BidFacts{}, errors.New("missing bid facts")
}
func (m *memoryMoves) ContractFacts(context.Context, string) (envelope.ContractFacts, error) {
	return envelope.ContractFacts{}, errors.New("missing contract facts")
}

func readyMove(t *testing.T, at time.Time) envelope.Envelope {
	t.Helper()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	spec := envelope.Spec{Intent: "roster.ir", LeagueID: "1", FranchiseID: "0001", Gravity: envelope.G2, Undo: envelope.Reversible,
		Subject: envelope.Subject{Players: []playerid.PlayerID{id}}, Expected: envelope.ExpectedChange{Player: id, RosterStatus: domain.RosterIR},
		Target: envelope.Target{Kind: envelope.Mapped, URL: "https://myfantasyleague.com/options"}}
	e, err := envelope.Restore("move", at, spec, []envelope.AuditEntry{{At: at, From: envelope.Draft, Event: envelope.ChecksPass, To: envelope.Ready, Note: "ready"}})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestMoveActionsSaveBeforeEffectsAndRefuseOwnership(t *testing.T) {
	at := time.Now()
	store := &memoryMoves{value: readyMove(t, at)}
	effects := []string{}
	ownErr := error(nil)
	s := MoveActions{Store: store, Own: func(string) error { return ownErr }, Now: func() time.Time { return at },
		Open: func(string) {
			if store.value.State() != envelope.HandedOff {
				t.Fatal("opened before save")
			}
			effects = append(effects, "open")
		},
		Audit: func(envelope.Envelope) { effects = append(effects, "audit") }, Changed: func() { effects = append(effects, "changed") }, Wake: func() { effects = append(effects, "wake") }}
	store.saveErr = errors.New("disk full")
	if _, err := s.HandOff(context.Background(), "move"); err == nil || store.value.State() != envelope.Ready || len(effects) != 0 {
		t.Fatal(err, store.value.State(), effects)
	}
	store.saveErr = nil
	ownErr = errors.New("not yours")
	if _, err := s.HandOff(context.Background(), "move"); !errors.Is(err, ownErr) || store.saves != 0 || len(effects) != 0 {
		t.Fatal(err, effects)
	}
	ownErr = nil
	if _, err := s.HandOff(context.Background(), "move"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(effects, []string{"open", "audit", "changed", "wake"}) {
		t.Fatal(effects)
	}
	store.value = readyMove(t, at)
	effects = nil
	if _, err := s.Withdraw(context.Background(), "move"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(effects, []string{"audit", "changed"}) {
		t.Fatal(effects)
	}
}

func TestMoveRefreshOrderAndSourceRecency(t *testing.T) {
	calls := []envelope.Source{}
	pauses := 0
	s := MoveRefresh{Pause: func() func(context.Context, time.Duration) error {
		return func(context.Context, time.Duration) error { pauses++; return nil }
	},
		Awaiting: func(context.Context) ([]envelope.Envelope, error) { return nil, nil }, Recent: func(envelope.Source, time.Time) bool { return false },
		Refresh: func(_ context.Context, source envelope.Source) error { calls = append(calls, source); return nil }}
	sources := map[envelope.Source]bool{envelope.Rosters: true, envelope.Contracts: true, envelope.Transactions: true, envelope.PendingTrades: true}
	if err := s.Run(context.Background(), sources); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []envelope.Source{envelope.Rosters, envelope.Transactions, envelope.PendingTrades}) || pauses != 2 {
		t.Fatal(calls, pauses)
	}
	calls = nil
	s.Pause = func() func(context.Context, time.Duration) error {
		return func(context.Context, time.Duration) error { return context.Canceled }
	}
	if err := s.Run(context.Background(), sources); !errors.Is(err, context.Canceled) || len(calls) != 1 {
		t.Fatal(err, calls)
	}
	now := time.Now()
	feeds := MoveSourceFeeds{Transactions: report.Feed{FetchedAt: now.Add(-time.Second)}}
	if !feeds.Recent(envelope.Transactions, now.Add(-time.Minute), func() time.Time { return now }) {
		t.Fatal("fresh post-handoff evidence ignored")
	}
	if feeds.Recent(envelope.Rosters, time.Time{}, func() time.Time { t.Fatal("sampled unsupported feed clock"); return now }) {
		t.Fatal("rosters considered daily feed")
	}
	if feeds.Recent(envelope.Transactions, now, func() time.Time { return now }) {
		t.Fatal("pre-handoff evidence accepted")
	}
}

func TestMoveWatchFactsAndUnknownOnce(t *testing.T) {
	store := &memoryMoves{}
	logs := 0
	s := MoveWatch{Store: store, Unknown: map[string]bool{}, Log: func(string) { logs++ }}
	for _, predicate := range []envelope.Predicate{envelope.BidPredicate{}, envelope.ContractPredicate{}} {
		if _, err := s.WithBidFacts(context.Background(), "id", predicate); err == nil {
			t.Fatal("missing facts accepted")
		}
	}
	spec := readyMove(t, time.Now()).Receipt().Spec
	spec.Intent = "unknown"
	unknown, err := envelope.New(time.Now(), spec, func() string { return "unknown" })
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Sources([]envelope.Envelope{unknown, unknown})) != 0 || logs != 1 {
		t.Fatal(logs)
	}
}

func TestRosterPlanReviewPreservesUnknownAndUsesEnvelopeRoute(t *testing.T) {
	at := time.Now()
	store := &memoryMoves{value: readyMove(t, at)}
	s := RosterPlanReview{Store: store, Now: func() time.Time { return at }, Log: func(string) {}, Audit: func(envelope.Envelope) {}}
	if s.ReturnToDraft(context.Background(), "move", eligibility.Decision{Verdict: eligibility.Ineligible}, false) || store.saves != 0 {
		t.Fatal("unknown verdict changed plan")
	}
	if !s.ReturnToDraft(context.Background(), "move", eligibility.Decision{Verdict: eligibility.Ineligible, Reason: "lost eligibility"}, true) || store.value.State() != envelope.Draft {
		t.Fatal(store.value.State())
	}
	audit := store.value.Receipt().Audit
	if len(audit) != 3 || audit[1].To != envelope.Stale || audit[2].To != envelope.Draft {
		t.Fatal(audit)
	}
}

func TestMovePollingFastUntilAndReadyWake(t *testing.T) {
	at := time.Now()
	e, err := readyMove(t, at).HandOff(at)
	if err != nil {
		t.Fatal(err)
	}
	s := MovePolling{Lock: func() {}, Unlock: func() {}, Awaiting: func(context.Context) ([]envelope.Envelope, error) { return []envelope.Envelope{e}, nil }}
	if got := s.FastUntil(context.Background()); !got.Equal(at.Add(30 * time.Minute)) {
		t.Fatal(got)
	}
	if err := RequestMovesPass(func() error { return errors.New("startup failed") }, func() { t.Fatal("woke before ready") }); err == nil {
		t.Fatal("startup error lost")
	}
}

func TestMoveObservationConfirmationAndFailures(t *testing.T) {
	at := time.Now().UTC().Truncate(time.Second)
	held := snapshot.Snapshot{Rosters: snapshot.Sourced[[]snapshot.Roster]{Provenance: snapshot.Provenance{Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: at.Add(-time.Hour).Format(time.RFC3339)}}}}
	s := MoveObservation{Snapshot: func(context.Context) (snapshot.Snapshot, error) { return held, nil }, Season: func() (envelope.Observation, error) { return envelope.Observation{}, nil }, Checked: func() time.Time { return at }, Contracts: func() []envelope.ContractEvidence { return nil }, League: "1"}
	observation, err := s.Read(context.Background())
	if err != nil || observation.Rosters.Provenance.Freshness.FetchedAt != at.Format(time.RFC3339) || observation.Contracts.Provenance != observation.Rosters.Provenance {
		t.Fatal(observation, err)
	}
	held.Rosters.Provenance.Freshness.State = domain.FreshFail
	observation, err = s.Read(context.Background())
	if err != nil || observation.Rosters.Provenance.Freshness.FetchedAt == at.Format(time.RFC3339) {
		t.Fatal("failed roster relabeled confirmed", observation, err)
	}
	failure := errors.New("build failed")
	s.Snapshot = func(context.Context) (snapshot.Snapshot, error) { return snapshot.Snapshot{}, failure }
	s.Season = func() (envelope.Observation, error) {
		t.Fatal("read season after failed roster snapshot")
		return envelope.Observation{}, nil
	}
	if _, err := s.Read(context.Background()); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

func TestRosterRevalidationOrderingAndCandidateAdmission(t *testing.T) {
	calls := []string{}
	s := RosterRevalidation{Release: func(context.Context) { calls = append(calls, "release") }, ContractsAvailable: true, Contracts: func(context.Context) error { calls = append(calls, "contracts"); return nil }, Ready: func() error { return nil },
		List: func(context.Context) ([]envelope.Receipt, error) {
			return []envelope.Receipt{{CorrelationID: "ready", State: envelope.Ready, Spec: envelope.Spec{Intent: "roster.ir"}}, {CorrelationID: "blocked", State: envelope.Blocked, Spec: envelope.Spec{Intent: "roster.ir"}}, {CorrelationID: "trade", State: envelope.Ready, Spec: envelope.Spec{Intent: "mfl.trade.accept"}}}, nil
		},
		Snapshot: func(context.Context) (snapshot.Snapshot, error) {
			calls = append(calls, "snapshot")
			return snapshot.Snapshot{}, nil
		},
		Lock: func() { calls = append(calls, "lock") }, Unlock: func() { calls = append(calls, "unlock") },
		Decision: func(_ context.Context, _ snapshot.Snapshot, id string) (eligibility.Decision, bool) {
			if id != "ready" {
				t.Fatal(id)
			}
			calls = append(calls, "decision")
			return eligibility.Decision{}, true
		},
		Return: func(context.Context, string, eligibility.Decision, bool) bool {
			calls = append(calls, "return")
			return true
		}, Changed: func(context.Context) { calls = append(calls, "changed") }, Log: func(string) {}}
	s.Run(context.Background())
	want := []string{"release", "contracts", "snapshot", "lock", "decision", "return", "changed", "unlock"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatal(calls)
	}
	calls = nil
	s.List = func(context.Context) ([]envelope.Receipt, error) { return nil, nil }
	s.Run(context.Background())
	if !reflect.DeepEqual(calls, []string{"release", "contracts"}) {
		t.Fatal("built snapshot for no candidates", calls)
	}
	RevalidateRelease(context.Background(), false, func(context.Context) error { t.Fatal("validated missing stores"); return nil }, nil)
}
