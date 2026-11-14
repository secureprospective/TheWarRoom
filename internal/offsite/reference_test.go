package offsite

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"filippo.io/age"
)

// testManagers builds one manager per kind over the fake bucket; capture records the order of captures
// and calls after for each.
func testManagers(t *testing.T, r2 *R2, clock time.Time, after func(SetKind)) (map[SetKind]*Manager, *[]SetKind) {
	t.Helper()
	var mu sync.Mutex
	var order []SetKind
	out := map[SetKind]*Manager{}
	for _, kind := range setKinds() {
		capture := func(_ context.Context, stage string) (string, time.Time, error) {
			mu.Lock()
			order = append(order, kind)
			mu.Unlock()
			if after != nil {
				after(kind)
			}
			path := filepath.Join(stage, "set.age")
			return path, clock, os.WriteFile(path, []byte(kind.File), 0600)
		}
		m := NewKindManager(kind, r2, t.TempDir(), capture, func(context.Context) (int64, error) { return 1, nil })
		m.now = func() time.Time { return clock }
		out[kind] = m
	}
	return out, &order
}

// RunInOrder captures in the order given, every round: league, reference, sign-ins.
func TestRunInOrderCapturesInStatedOrder(t *testing.T) {
	r2, _ := testR2(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	managers, order := testManagers(t, r2, time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC), func(kind SetKind) {
		if kind == SignInSet {
			cancel() // the round finishes; RunInOrder returns at its wait
		}
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunInOrder(ctx, managers[LeagueSet], managers[ReferenceSet], managers[SignInSet])
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("RunInOrder did not return")
	}
	if len(*order) != 3 || (*order)[0] != LeagueSet || (*order)[1] != ReferenceSet || (*order)[2] != SignInSet {
		t.Fatalf("capture order %v", *order)
	}
}

// Reference uploads failing leave the league and sign-in kinds writable; only the reference kind
// reports itself behind. (No write guard reads the reference kind; see cmd/twr-server backup_hosted.go.)
func TestReferenceFailureLeavesOtherKindsWritable(t *testing.T) {
	r2, fake := testR2(t)
	fake.failPrefix = ReferenceSet.Prefix
	managers, _ := testManagers(t, r2, time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC), nil)
	for _, kind := range setKinds() {
		_ = managers[kind].Tick(t.Context())
	}
	if managers[LeagueSet].CheckWrite() != nil || managers[SignInSet].CheckWrite() != nil {
		t.Fatal("a failing reference upload paused league or sign-in writes")
	}
	if managers[ReferenceSet].CheckWrite() == nil || managers[ReferenceSet].Status().Failure == "" {
		t.Fatal("the failing reference kind does not report itself behind")
	}
}

// O4: a raw capture goes up once; a second Put of the same body sends nothing (the bucket refusing
// every upload does not matter), and the object decrypts to the body.
func TestRawCaptureUploadedOnce(t *testing.T) {
	r2, fake := testR2(t)
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	evidence := NewEvidence(r2, identity.Recipient().String())
	body := []byte("compressed raw capture")
	if err := evidence.Put(t.Context(), "abc", body); err != nil {
		t.Fatal(err)
	}
	fake.failPut = true
	if err := evidence.Put(t.Context(), "abc", body); err != nil {
		t.Fatalf("a known capture was uploaded again: %v", err)
	}
	object, err := r2.OpenEvidence(t.Context(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = object.Close() }()
	got, err := UnsealEvidence(object, identity)
	if err != nil || string(got) != string(body) {
		t.Fatalf("object %q: %v", got, err)
	}
	if len(fake.objects) != 1 {
		t.Fatalf("%d objects for one capture", len(fake.objects))
	}
}
