package history

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func eventFixture(id string) LeagueEvent {
	cents := int64(123456789)
	return LeagueEvent{ID: id, League: "legacy", Stream: "contracts", EffectiveAt: t0(), Actor: "commissioner", Scope: "league", Kind: "test", Payload: EventPayload{Version: 1, MoneyCents: &cents}, InputHash: strings.Repeat("a", 64), CorrelationID: "test-only"}
}

func TestLeagueEventAppendReadAndTriggers(t *testing.T) {
	t.Parallel()
	s, _, p := newStore(t)
	e, err := s.AppendEvent(t.Context(), eventFixture("first"), 0)
	if err != nil || e.Revision != 1 || !e.RecordedAt.Equal(t0()) {
		t.Fatalf("append %+v %v", e, err)
	}
	correction := eventFixture("second")
	correction.Supersedes = e.ID
	if _, err := s.AppendEvent(t.Context(), correction, 0); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("conflict: %v", err)
	}
	if _, err := s.AppendEvent(t.Context(), correction, 1); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{`UPDATE league_event SET actor='wrong'`, `DELETE FROM league_event`, `INSERT INTO league_event SELECT 'skip',league,stream,4,effective_at,recorded_at,actor,scope,kind,payload,input_hash,correlation_id,NULL FROM league_event WHERE id='first'`} {
		if _, err := p.Write().ExecContext(t.Context(), query); err == nil {
			t.Fatalf("forbidden statement passed: %s", query)
		}
	}
	rows, err := s.Events(t.Context(), "legacy", "contracts", 0)
	if err != nil || len(rows) != 2 || rows[1].Supersedes != "first" || *rows[0].Payload.MoneyCents != 123456789 {
		t.Fatalf("read %+v %v", rows, err)
	}
	rows, err = s.Events(t.Context(), "legacy", "contracts", 1)
	if err != nil || len(rows) != 1 || rows[0].Revision != 2 {
		t.Fatalf("cursor %+v %v", rows, err)
	}
	other := eventFixture("other")
	other.Stream = "rosters"
	if _, err := s.AppendEvent(t.Context(), other, 0); err != nil {
		t.Fatal(err)
	}
	cross := eventFixture("cross")
	cross.Supersedes = "other"
	if _, err := s.AppendEvent(t.Context(), cross, 2); err == nil {
		t.Fatal("cross-stream supersedes accepted")
	}
}

func TestLeagueEventConcurrentExpectedRevision(t *testing.T) {
	t.Parallel()
	s, _, p := newStore(t)
	second := New(p, s.Registry())
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i, store := range []*Store{s, second} {
		wg.Go(func() {
			_, err := store.AppendEvent(t.Context(), eventFixture([]string{"one", "two"}[i]), 0)
			results <- err
		})
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, ErrRevisionConflict):
			conflict++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success %d conflicts %d", success, conflict)
	}
}

func TestAuthorityDefaultsToMFLUntilCutover(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	for _, at := range []time.Time{t0(), t0().AddDate(100, 0, 0)} {
		authority, err := s.Authority(t.Context(), "legacy", "contracts", at)
		if err != nil || authority != "MFL" {
			t.Fatalf("default %s %v", authority, err)
		}
	}
	if err := s.SetAuthorityPolicy(t.Context(), AuthorityPolicy{League: "legacy", Domain: "contracts"}); err != nil {
		t.Fatal(err)
	}
	cutoff := t0().Add(time.Hour)
	if err := s.SetAuthorityPolicy(t.Context(), AuthorityPolicy{League: "legacy", Domain: "contracts", CutoverAt: &cutoff}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		at   time.Time
		want string
	}{{t0(), "MFL"}, {cutoff, "local"}, {cutoff.Add(time.Hour), "local"}} {
		got, err := s.Authority(t.Context(), "legacy", "contracts", tc.at)
		if err != nil || got != tc.want {
			t.Fatalf("%s: %s %v", tc.at, got, err)
		}
	}
}

func TestLeagueEventRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)
	for _, mutate := range []func(*LeagueEvent){func(e *LeagueEvent) { e.ID = "" }, func(e *LeagueEvent) { e.InputHash = "bad" }, func(e *LeagueEvent) { e.Payload.Version = 0 }, func(e *LeagueEvent) { e.EffectiveAt = time.Time{} }} {
		e := eventFixture("bad")
		mutate(&e)
		if _, err := s.AppendEvent(t.Context(), e, 0); err == nil {
			t.Fatal("invalid event accepted")
		}
	}
}

func TestLeagueEventRejectsReplacementAndUpsert(t *testing.T) {
	t.Parallel()
	for _, query := range []string{
		`INSERT OR REPLACE INTO league_event SELECT id,league,'other',1,effective_at,recorded_at,'wrong',scope,kind,payload,input_hash,correlation_id,NULL FROM league_event WHERE id='first'`,
		`INSERT INTO league_event SELECT id,league,'other',1,effective_at,recorded_at,'wrong',scope,kind,payload,input_hash,correlation_id,NULL FROM league_event WHERE id='first' ON CONFLICT(id) DO UPDATE SET actor=excluded.actor`,
		`INSERT OR REPLACE INTO league_event SELECT 'replacement',league,stream,2,effective_at,recorded_at,'wrong',scope,kind,payload,input_hash,correlation_id,NULL FROM league_event WHERE id='request'`,
	} {
		t.Run(query, func(t *testing.T) {
			s, _, p := newStore(t)
			if _, err := s.AppendEvent(t.Context(), eventFixture("first"), 0); err != nil {
				t.Fatal(err)
			}
			request := eventFixture("request")
			request.Stream = "commissioner"
			if _, err := s.AppendEvent(t.Context(), request, 0); err != nil {
				t.Fatal(err)
			}
			if _, err := p.Write().ExecContext(t.Context(), query); err == nil {
				t.Fatal("ledger replacement accepted")
			}
			var count int
			if err := p.Read().QueryRowContext(t.Context(), `SELECT COUNT(*) FROM league_event WHERE actor='commissioner' AND id IN ('first','request')`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 2 {
				t.Fatalf("unchanged events = %d, want 2", count)
			}
		})
	}
}
