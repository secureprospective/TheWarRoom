package history

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type admissionFunc func(context.Context, commissioner.Access) error

func (f admissionFunc) Admit(ctx context.Context, access commissioner.Access) error {
	return f(ctx, access)
}

func admittedCommand(t *testing.T, command string, args commissioner.RawArgs, visibility string) commissioner.RawCommand {
	t.Helper()
	raw := commissioner.RawCommand{League: "native.test", Command: command, Args: args,
		Expected: 1, Request: "admission." + command, Reason: "Record the requested change", Effective: "now", Visibility: visibility}
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = commissioner.Decode(string(body))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func admissionCommands(t *testing.T) []commissioner.RawCommand {
	t.Helper()
	return []commissioner.RawCommand{
		reverseCommand("act.create", 1, "reverse"),
		admittedCommand(t, commissioner.TradeWithdraw, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{ID: "trade.other", Franchise: "franchise.01"}}, "league"),
		admittedCommand(t, commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: "window.other", Franchise: "franchise.01"}}, "private"),
		admittedCommand(t, commissioner.BidWithdraw, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Bid: "bid.other", Franchise: "franchise.01"}}, "private"),
	}
}

func assertAdmissionEvents(t *testing.T, s *Store, count int) {
	t.Helper()
	events, err := s.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil || len(events) != count {
		t.Fatalf("events=%d want=%d error=%v", len(events), count, err)
	}
}

// Holding the only writer connection proves the request reached transaction admission,
// rather than merely starting a goroutine that might not run until after revocation.
func queuedAdmissionWrite(ctx context.Context, t *testing.T, s *Store, raw commissioner.RawCommand) func() error {
	t.Helper()
	tx, err := s.pools.Write().BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	waits := s.pools.Write().Stats().WaitCount
	result := make(chan error, 1)
	go func() { _, err := s.ExecuteCommissioner(ctx, raw, "creator", nil); result <- err }()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for s.pools.Write().Stats().WaitCount == waits {
		select {
		case err := <-result:
			t.Fatalf("write did not queue: %v", err)
		case <-deadline.C:
			t.Fatal("write did not wait for the writer connection")
		case <-tick.C:
		}
	}
	return func() error {
		if err := tx.Rollback(); err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-result:
			return err
		case <-time.After(5 * time.Second):
			t.Fatal("queued write did not finish")
			return nil
		}
	}
}

func TestCredentialAdmissionQueuedFamiliesAndPreview(t *testing.T) {
	for _, change := range []string{"revoked", "role removed"} {
		t.Run(change, func(t *testing.T) {
			s, _, _ := newStore(t)
			if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
				t.Fatal(err)
			}
			for _, raw := range admissionCommands(t) {
				t.Run(raw.Command, func(t *testing.T) {
					var denied atomic.Bool
					var calls atomic.Int32
					gate := admissionFunc(func(_ context.Context, access commissioner.Access) error {
						calls.Add(1)
						if access.Raw.Command != raw.Command || access.Principal != "creator" || access.League != raw.League {
							return errors.New("wrong admission scope")
						}
						if denied.Load() {
							return errors.New(change)
						}
						return nil
					})
					ctx := commissioner.WithCredentialAdmission(t.Context(), gate)
					// Admission at the transport is not permission to append after waiting for the writer.
					if err := commissioner.AdmitCredential(ctx, commissioner.Access{Principal: "creator", League: raw.League, Raw: raw}); err != nil {
						t.Fatal(err)
					}
					resume := queuedAdmissionWrite(ctx, t, s, raw)
					denied.Store(true)
					if err := resume(); err == nil || !strings.Contains(err.Error(), change) {
						t.Fatalf("queued command: %v", err)
					}
					if calls.Load() != 2 {
						t.Fatalf("append did not recheck credential: %d", calls.Load())
					}
					preview, err := s.PreviewCommissioner(ctx, raw, "creator", nil, commissioner.PreviewCatalogue{})
					if err != nil || len(preview.Blockers) != 1 || !strings.Contains(strings.ToLower(preview.Blockers[0]), change) || len(preview.Fields) != 0 {
						t.Fatalf("preview disclosed change: %+v %v", preview, err)
					}
					assertAdmissionEvents(t, s, 1)
				})
			}
		})
	}
}

func TestCredentialAdmissionRetryAndNilFailClosed(t *testing.T) {
	s, _, _ := newStore(t)
	raw := createCommand()
	var revoked atomic.Bool
	gate := admissionFunc(func(_ context.Context, access commissioner.Access) error {
		if revoked.Load() {
			return errors.New("session revoked")
		}
		if access.Mode != "execute" && access.Mode != "preview" {
			return errors.New("wrong mode")
		}
		return nil
	})
	ctx := commissioner.WithCredentialAdmission(t.Context(), gate)
	receipt, err := s.ExecuteCommissioner(ctx, raw, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := s.ExecuteCommissioner(ctx, raw, "creator", nil)
	if err != nil || receipt != retry {
		t.Fatalf("admitted retry: %+v %v", retry, err)
	}
	resume := queuedAdmissionWrite(ctx, t, s, raw)
	revoked.Store(true)
	if err := resume(); err == nil || !strings.Contains(err.Error(), "session revoked") {
		t.Fatalf("queued retry returned a receipt: %v", err)
	}
	for _, c := range []context.Context{ctx, commissioner.WithCredentialAdmission(t.Context(), nil)} {
		if _, err := s.ExecuteCommissioner(c, raw, "creator", nil); err == nil {
			t.Fatal("revoked/missing credential returned receipt")
		}
		p, err := s.PreviewCommissioner(c, raw, "creator", nil, commissioner.PreviewCatalogue{})
		if err != nil || len(p.Blockers) != 1 || len(p.Fields) != 0 {
			t.Fatalf("retry preview: %+v %v", p, err)
		}
	}
	assertAdmissionEvents(t, s, 1)
}

func TestCredentialAdmissionGetsAllReferencesAndTransactionRecords(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	for _, raw := range admissionCommands(t)[1:] {
		t.Run(raw.Command, func(t *testing.T) {
			gate := admissionFunc(func(_ context.Context, access commissioner.Access) error {
				if len(access.Records) != 1 || access.Records[0].Principal != "creator" {
					return errors.New("missing transaction records")
				}
				// A plausible selected franchise cannot disguise a reference to another franchise's object.
				switch {
				case access.Raw.Args.Trade != nil && access.Raw.Args.Trade.ID == "trade.other":
				case access.Raw.Args.Waiver != nil && access.Raw.Args.Waiver.Window == "window.other":
				case access.Raw.Args.Bid != nil && access.Raw.Args.Bid.Bid == "bid.other":
				default:
					return errors.New("missing object reference")
				}
				return errors.New("cross-franchise reference")
			})
			ctx := commissioner.WithCredentialAdmission(t.Context(), gate)
			if _, err := s.ExecuteCommissioner(ctx, raw, "creator", nil); err == nil || !strings.Contains(err.Error(), "cross-franchise reference") {
				t.Fatalf("reference admission: %v", err)
			}
			assertAdmissionEvents(t, s, 1)
		})
	}
}

func TestCredentialAdmissionCannotReplaceDomainAuthorization(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	ctx := commissioner.WithCredentialAdmission(t.Context(), admissionFunc(func(context.Context, commissioner.Access) error { return nil }))
	if _, err := s.ExecuteCommissioner(ctx, reverseCommand("act.create", 1, "reverse"), "outsider", nil); err == nil {
		t.Fatal("credential granted league authority")
	}
	assertAdmissionEvents(t, s, 1)
}

func TestCredentialAdmissionRechecksBeforeAppend(t *testing.T) {
	s, _, _ := newStore(t)
	calls := 0
	ctx := commissioner.WithCredentialAdmission(t.Context(), admissionFunc(func(context.Context, commissioner.Access) error {
		calls++
		if calls == 2 {
			return errors.New("revoked during planning")
		}
		return nil
	}))
	if _, err := s.ExecuteCommissioner(ctx, createCommand(), "creator", nil); err == nil || !strings.Contains(err.Error(), "revoked during planning") {
		t.Fatalf("final append admission: %v", err)
	}
	assertAdmissionEvents(t, s, 0)
}

func TestCredentialAdmissionDraftPickBeforeRetry(t *testing.T) {
	s, _, _ := newStore(t)
	if _, err := s.ExecuteCommissioner(t.Context(), createWith(commissioner.PresetLegacy, 0), "creator", nil); err != nil {
		t.Fatal(err)
	}
	runScoreCommand(t, s, ruleCommand("calendar.boundary", `"start_of_day"`, "now", "Open windows at the start of the shared date", 1), "creator")
	input := commissioner.DraftPickInput{League: "native.test", Season: 2026, Franchise: "franchise.01", Player: "1001", Request: "pick", Expected: 1}
	ctx := commissioner.WithCredentialAdmission(t.Context(), admissionFunc(func(_ context.Context, access commissioner.Access) error {
		if access.Operation != commissioner.DraftPick || access.DraftPick == nil || *access.DraftPick != input || len(access.Records) != 2 {
			return errors.New("missing draft scope")
		}
		return errors.New("session revoked")
	}))
	records, err := s.nativeRecords(t.Context(), input.League)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := commissioner.NativeActionGate(input.League, records, input.Franchise, "draft_pick", time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC), time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	permit, open := gate.Permit()
	if !open {
		t.Fatal(gate.Sentence)
	}
	if _, err := s.RecordDraftPick(ctx, permit, input, "creator", nil, time.UTC); err == nil || !strings.Contains(err.Error(), "session revoked") {
		t.Fatalf("draft admission: %v", err)
	}
	assertAdmissionEvents(t, s, 2)
}
