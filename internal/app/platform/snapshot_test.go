package platform

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func TestSnapshotCaptureOrderAndFailureRetention(t *testing.T) {
	failure := errors.New("build failed")
	for _, failed := range []bool{false, true} {
		calls := []string{}
		_, err := ReadSnapshot(context.Background(), func(context.Context) snapshot.Directory {
			calls = append(calls, "directory")
			return snapshot.Directory{}
		},
			func(context.Context) snapshot.Injuries { calls = append(calls, "injuries"); return snapshot.Injuries{} },
			func(snapshot.Directory, snapshot.Injuries) (snapshot.Snapshot, error) {
				calls = append(calls, "build")
				if failed {
					return snapshot.Snapshot{}, failure
				}
				return snapshot.Snapshot{}, nil
			},
			func(snapshot.Snapshot) { calls = append(calls, "publish") })
		want := []string{"directory", "injuries", "build", "publish"}
		if failed {
			want = want[:3]
		}
		if !reflect.DeepEqual(calls, want) || errors.Is(err, failure) != failed {
			t.Fatal(calls, err)
		}
	}
}

func TestSnapshotDirectoryProvenanceAndInjuryLaziness(t *testing.T) {
	at := time.Now().UTC().Truncate(time.Second)
	readAt := func() time.Time { return at }
	live := SnapshotDirectory([]normalize.Lookup{{}}, domain.Freshness{State: domain.FreshLive, FetchedAt: at.Add(time.Hour).Format(time.RFC3339)}, nil, readAt)
	if live.Provenance.Freshness.FetchedAt != at.Format(time.RFC3339) || live.Provenance.Source != "mfl-players" {
		t.Fatal(live)
	}
	archived := SnapshotDirectory([]normalize.Lookup{{}}, domain.Freshness{State: domain.FreshStale}, nil, func() time.Time { t.Fatal("sampled live directory for archive"); return at })
	if archived.Provenance.Source != "mfl-players-archive" {
		t.Fatal(archived)
	}
	failed := SnapshotDirectory(nil, domain.Freshness{}, errors.New("offline"), nil)
	if failed.Provenance.Freshness.State != domain.FreshFail || failed.Provenance.Freshness.Note != "offline" {
		t.Fatal(failed)
	}
	reads, refreshes := 0, 0
	for _, held := range []bool{true, false} {
		SnapshotInjuries(context.Background(), held, time.Minute, func(ctx context.Context) {
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("unbounded injury refresh")
			}
			refreshes++
		}, func() snapshot.Injuries { reads++; return snapshot.Injuries{} })
	}
	if reads != 2 || refreshes != 1 {
		t.Fatal(reads, refreshes)
	}
}

func TestRosterDraftRefusesBeforeLocksAndWrites(t *testing.T) {
	ownErr := errors.New("not yours")
	s := RosterDraft{Ready: func() error { return nil }, Own: func(string) error { return ownErr }, Snapshot: func() (snapshot.Snapshot, bool) {
		t.Fatal("captured unauthorized snapshot")
		return snapshot.Snapshot{}, false
	}}
	if _, err := s.Draft("roster.ir", "other", "0042"); !errors.Is(err, ownErr) {
		t.Fatal(err)
	}
	s.Own = func(string) error { return nil }
	if _, err := s.Draft("roster.ir", "mine", "bad"); err == nil {
		t.Fatal("invalid player accepted")
	}
	s.Snapshot = func() (snapshot.Snapshot, bool) { return snapshot.Snapshot{}, false }
	s.Lock = func() { t.Fatal("locked with no snapshot") }
	if _, err := s.Draft("roster.ir", "mine", "0042"); err == nil {
		t.Fatal("missing snapshot accepted")
	}
	receipts := []envelope.Receipt{{Spec: envelope.Spec{Intent: envelope.BidIntent}}, {Spec: envelope.Spec{Intent: "roster.ir"}}, {Spec: envelope.Spec{Intent: envelope.CutIntent}}, {Spec: envelope.Spec{Intent: "contract.extend"}}, {Spec: envelope.Spec{Intent: "mfl.trade.accept"}}}
	rail := RosterRail(receipts)
	if len(rail) != 2 || rail[0].Spec.Intent != "roster.ir" || rail[1].Spec.Intent != "mfl.trade.accept" {
		t.Fatal(rail)
	}
}
