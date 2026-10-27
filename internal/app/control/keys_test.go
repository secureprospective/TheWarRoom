package control

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
)

func TestKeyWorkflowAdmissionAndVerificationBeforeStorage(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	store := mflkey.NewMemory("fixture", "2026")
	calls := []string{}
	verifyErr := error(nil)
	service := KeyWorkflow{Ready: func() error { return nil }, Lock: func() { calls = append(calls, "lock") }, Unlock: func() { calls = append(calls, "unlock") },
		Context: func() context.Context { return ctx }, Store: func() CredentialStore { return store },
		Status: func(state, detail string) MFLKeyStatus { return MFLKeyStatus{State: state, Detail: detail} },
		Verify: func(_ context.Context, key mflkey.Key) (string, error) {
			if key.Reveal() != strings.Repeat("x", 16) {
				t.Fatal("not trimmed")
			}
			calls = append(calls, "verify")
			return "verified", verifyErr
		},
		Lookup: func(ctx context.Context) {
			if _, err := store.Get(ctx); err != nil {
				t.Fatal("lookup before storage")
			}
			calls = append(calls, "lookup")
		},
		Verified: func(stamp string) {
			if stamp != at.Format(time.RFC3339) {
				t.Fatal(stamp)
			}
			calls = append(calls, "publish")
		},
		Clear: func() { calls = append(calls, "clear") }, Refresh: func() { calls = append(calls, "refresh") }, Now: func() time.Time { return at },
	}
	status, err := service.Set("short")
	if err != nil || status.State != "rejected" || !reflect.DeepEqual(calls, []string{"lock", "unlock"}) {
		t.Fatal(status, err, calls)
	}
	for _, failure := range []error{pendingtrades.ErrRejected, context.DeadlineExceeded} {
		calls = nil
		verifyErr = failure
		status, err = service.Set(" " + strings.Repeat("x", 16) + " ")
		want := "rejected"
		if errors.Is(failure, context.DeadlineExceeded) {
			want = "unreachable"
		}
		if err != nil || status.State != want || !reflect.DeepEqual(calls, []string{"lock", "verify", "unlock"}) {
			t.Fatal(status, err, calls)
		}
		if _, err := store.Get(ctx); !errors.Is(err, mflkey.ErrNoKey) {
			t.Fatal("rejected credential stored", err)
		}
	}
	calls = nil
	verifyErr = nil
	status, err = service.Set(strings.Repeat("x", 16))
	if err != nil || status.State != "connected" || !reflect.DeepEqual(calls, []string{"lock", "verify", "lookup", "publish", "refresh", "unlock"}) {
		t.Fatal(status, err, calls)
	}
	calls = nil
	if status, err = service.Delete(); err != nil || status.State != "absent" || !reflect.DeepEqual(calls, []string{"lock", "clear", "refresh", "unlock"}) {
		t.Fatal(status, err, calls)
	}
	if _, err := store.Get(ctx); !errors.Is(err, mflkey.ErrNoKey) {
		t.Fatal("deleted credential retained", err)
	}
	service.Ready = func() error { return context.Canceled }
	service.Store = func() CredentialStore { t.Fatal("captured before readiness"); return nil }
	if _, err := service.Read(); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := service.Set("short"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := service.Delete(); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestResolveFranchiseFailsClosed(t *testing.T) {
	ctx := context.Background()
	store := mflkey.NewMemory("fixture", "2026")
	lookup := func(context.Context) (string, error) { t.Fatal("looked up without key"); return "", nil }
	if got := ResolveFranchise(ctx, store, lookup); got != "" {
		t.Fatal(got)
	}
	if err := store.Set(ctx, "fixture"); err != nil {
		t.Fatal(err)
	}
	if got := ResolveFranchise(ctx, store, func(context.Context) (string, error) { return "0001", context.Canceled }); got != "" {
		t.Fatal(got)
	}
	if got := ResolveFranchise(ctx, store, func(context.Context) (string, error) { return "0001", nil }); got != "0001" {
		t.Fatal(got)
	}
}
