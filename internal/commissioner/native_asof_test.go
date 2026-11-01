package commissioner

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNativeAssetsAsOfReceiptHistory(t *testing.T) {
	b := rolloverLedger(t)
	remove := RawCommand{League: rolloverLeague, Command: CorrectRoster,
		Args:    RawArgs{Correction: &RawCorrection{Player: "0003", Franchise: "franchise.02", Action: "remove"}},
		Request: "remove", Reason: "Repair roster membership", Effective: "now", Visibility: "league"}
	if err := remove.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := CheckCorrectionAppend(remove, b.records, seasonMoment(5)); err != nil {
		t.Fatal(err)
	}
	entry := b.add(remove, seasonMoment(5))
	b.revert(entry, seasonMoment(7))
	b.records = WithReplayCache(b.records)
	original, err := json.Marshal(b.records)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		at      time.Time
		present bool
	}{
		{"observation before seed receipt", seasonMoment(1), true},
		{"before release", seasonMoment(5).Add(-time.Nanosecond), true},
		{"release instant inclusive", seasonMoment(5), false},
		{"before reversal", seasonMoment(7).Add(-time.Nanosecond), false},
		{"reversal instant inclusive", seasonMoment(7), true},
		{"after ledger", seasonMoment(8), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assets, readErr := NativeAssetsAsOf(rolloverLeague, b.records, tc.at)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if got := assets.Snapshot.PlayerOf("franchise.02", "0003") >= 0; got != tc.present {
				t.Fatalf("player present=%t, want %t", got, tc.present)
			}
			if assets.Revision != int64(len(b.records)) {
				t.Fatal("historical read lost the source ledger revision")
			}
		})
	}
	current, err := NativeAssetsReading(rolloverLeague, b.records)
	if err != nil {
		t.Fatal(err)
	}
	asOf, err := NativeAssetsAsOf(rolloverLeague, b.records, seasonMoment(8))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(current, asOf) {
		t.Fatal("current roster differs from complete receipt-history projection")
	}
	after, err := json.Marshal(b.records)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) {
		t.Fatal("historical projection changed the ledger")
	}
}

func TestNativeAssetsAsOfRejectsUnknownHistory(t *testing.T) {
	b := rolloverLedger(t)
	for _, tc := range []struct {
		name string
		at   time.Time
		want string
	}{
		{"zero instant", time.Time{}, "choose the roster observation time"},
		{"before observation", seasonMoment(1).Add(-time.Nanosecond), "before the starting observation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assets, err := NativeAssetsAsOf(rolloverLeague, b.records, tc.at)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
			if assets.Snapshot != nil {
				t.Fatal("unknown history returned a guessed roster")
			}
		})
	}
	if _, err := NativeAssetsAsOf(rolloverLeague, nil, seasonMoment(1)); err == nil {
		t.Fatal("missing ledger returned success")
	}
}
