package history

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestReverseReceiveFailurePreservesBothFences(t *testing.T) {
	source, sp := openMoveTest(t, filepath.Join(t.TempDir(), "source.db"))
	target, _ := openMoveTest(t, filepath.Join(t.TempDir(), "target.db"))
	if _, err := source.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
		t.Fatal(err)
	}
	original, _, anchor, err := source.CaptureTransfer(t.Context(), "native.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := target.StageTransfer(t.Context(), ImportReceipt{League: "native.test", SourceEpoch: "epoch", Content: "content"}, []byte("document"), original, nil); err != nil {
		t.Fatal(err)
	}
	peer, err := target.TransferIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	f, err := source.SealTransfer(t.Context(), TransferFence{League: "native.test", Epoch: "epoch", Content: "content", Peer: peer}, original, anchor)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.DecideTransfer(t.Context(), f, true, func() error { return target.DecideDestination(t.Context(), f, true, original, nil) }); err != nil {
		t.Fatal(err)
	}
	raw := commissioner.RawCommand{League: "native.test", Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: "cap.amount", Value: json.RawMessage(`100000000000`)}, Expected: 1, Request: "server.change", Reason: "Change on server", Effective: "now", Visibility: "league"}
	if _, err := target.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatal(err)
	}
	events, _, anchor, err := target.CaptureTransfer(t.Context(), "native.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	peer, err = source.TransferIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := target.SealTransfer(t.Context(), TransferFence{League: "native.test", Epoch: f.Move, Content: "reverse", Peer: peer}, events, anchor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sp.Write().ExecContext(t.Context(), `CREATE TRIGGER fail_reverse BEFORE INSERT ON league_event WHEN NEW.revision>1 BEGIN SELECT RAISE(ABORT,'injected reverse failure'); END`); err != nil {
		t.Fatal(err)
	}
	decide := func() error {
		return target.DecideTransfer(t.Context(), reverse, true, func() error { return source.DecideDestination(t.Context(), reverse, true, events, nil) })
	}
	if err := decide(); err == nil {
		t.Fatal("injected reverse failure ignored")
	}
	for _, store := range []*Store{source, target} {
		status, err := store.TransferStatus(t.Context(), "native.test")
		if err != nil || status.Writable {
			t.Fatal("failure lifted fence", status, err)
		}
	}
	held, _, _, err := source.CaptureTransfer(t.Context(), "native.test", nil)
	if err != nil || !reflect.DeepEqual(held, original) {
		t.Fatal("partial reverse history", err)
	}
	if _, err := sp.Write().ExecContext(t.Context(), `DROP TRIGGER fail_reverse`); err != nil {
		t.Fatal(err)
	}
	if err := decide(); err != nil {
		t.Fatal("reverse retry", err)
	}
	held, _, _, err = source.CaptureTransfer(t.Context(), "native.test", nil)
	if err != nil || !reflect.DeepEqual(held, events) {
		t.Fatal("reverse retry lost server history", err)
	}
}
