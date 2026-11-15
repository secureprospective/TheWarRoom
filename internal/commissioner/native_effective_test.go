package commissioner

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestNativeEffectiveCutBeforeAward(t *testing.T) {
	b := rolloverLedger(t)
	player := b.records[1].Command.Args.Native.Players[2]
	bid := b.add(RawCommand{League: rolloverLeague, Command: BidPlace,
		Args:      RawArgs{Bid: &RawBidArgs{Player: player.ID, Franchise: "franchise.01", Salary: 400, Years: 2}, BidOutcome: &BidOutcome{}},
		Effective: "now"}, seasonMoment(6))
	b.add(RawCommand{League: rolloverLeague, Command: BidsProcess,
		Args:      RawArgs{Bid: &RawBidArgs{Player: player.ID}, BidOutcome: &BidOutcome{Winner: "franchise.01", WinningBid: bid, Season: 2026}},
		Effective: "now"}, seasonMoment(9))
	b.add(RawCommand{League: rolloverLeague, Command: RosterCut,
		Args:      RawArgs{Waiver: &RawWaiverArgs{Player: player.ID, Franchise: player.Franchise}, WaiverOutcome: &WaiverOutcome{Player: &player, Season: 2026}},
		Effective: seasonMoment(5).Format(time.RFC3339Nano)}, seasonMoment(12))
	assertEffectiveOwners(t, b, player.ID, []rosterInstant{
		{"before cut", seasonMoment(5).Add(-time.Nanosecond), "franchise.02"},
		{"cut inclusive", seasonMoment(5), ""},
		{"before award", seasonMoment(9).Add(-time.Nanosecond), ""},
		{"award inclusive", seasonMoment(9), "franchise.01"},
		{"after receipt", seasonMoment(13), "franchise.01"},
	})
}

func TestNativeEffectiveTradeBetweenWeeks(t *testing.T) {
	b := rolloverLedger(t)
	offer := TradeState{Status: "offered", Assets: []TradeAsset{{Player: "0003", From: "franchise.02", To: "franchise.01"}}}
	id := b.add(RawCommand{League: rolloverLeague, Command: TradeOffer,
		Args: RawArgs{TradeOutcome: &offer}, Effective: "now"}, seasonMoment(4))
	offer.ID = id
	approved := offer.Clone()
	approved.Status, approved.Previous = "approved", TradeDigest(offer)
	b.add(RawCommand{League: rolloverLeague, Command: TradeReview,
		Args: RawArgs{TradeOutcome: &approved}, Effective: seasonMoment(7).Format(time.RFC3339Nano)}, seasonMoment(12))
	assertEffectiveOwners(t, b, "0003", []rosterInstant{
		{"earlier week", seasonMoment(4), "franchise.02"},
		{"before trade", seasonMoment(7).Add(-time.Nanosecond), "franchise.02"},
		{"trade inclusive", seasonMoment(7), "franchise.01"},
		{"later week", seasonMoment(11), "franchise.01"},
		{"after receipt", seasonMoment(13), "franchise.01"},
	})
}

type rosterInstant struct {
	name  string
	at    time.Time
	owner string
}

func assertEffectiveOwners(t *testing.T, b *ledgerBuilder, player string, cases []rosterInstant) {
	t.Helper()
	b.records = WithReplayCache(b.records)
	before, err := json.Marshal(b.records)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, read := range []func(string, []Record, time.Time) (NativeAssets, error){
				NativeAssetsAsOf,
				func(league string, records []Record, at time.Time) (NativeAssets, error) {
					return NativeAssetsReading(league, records, at)
				},
			} {
				assets, err := read(rolloverLeague, b.records, tc.at)
				if err != nil {
					t.Fatal(err)
				}
				owner := ""
				for _, p := range assets.Snapshot.Players {
					if p.ID == player {
						owner = p.Franchise
					}
				}
				if owner != tc.owner || assets.Revision != int64(len(b.records)) {
					t.Fatalf("owner=%q revision=%d, want %q revision=%d", owner, assets.Revision, tc.owner, len(b.records))
				}
			}
		})
	}
	after, err := json.Marshal(b.records)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("projection mutated durable records")
	}
}

func TestNativeEffectiveRevisionTie(t *testing.T) {
	b := rolloverLedger(t)
	player := b.records[1].Command.Args.Native.Players[2]
	b.add(RawCommand{League: rolloverLeague, Command: RosterReserve,
		Args:      RawArgs{Waiver: &RawWaiverArgs{Player: player.ID, Franchise: player.Franchise, Bucket: "ir"}, WaiverOutcome: &WaiverOutcome{Player: &player}},
		Effective: seasonMoment(5).Format(time.RFC3339Nano)}, seasonMoment(12))
	injured := player
	injured.Status = domain.RosterIR
	b.add(RawCommand{League: rolloverLeague, Command: RosterReserve,
		Args:      RawArgs{Waiver: &RawWaiverArgs{Player: player.ID, Franchise: player.Franchise, Bucket: "taxi"}, WaiverOutcome: &WaiverOutcome{Player: &injured}},
		Effective: seasonMoment(5).Format(time.RFC3339Nano)}, seasonMoment(13))
	for _, tc := range []struct {
		at     time.Time
		status domain.RosterStatus
	}{
		{seasonMoment(5).Add(-time.Nanosecond), domain.RosterActive},
		{seasonMoment(5), domain.RosterTaxi},
	} {
		assets, err := NativeAssetsAsOf(rolloverLeague, b.records, tc.at)
		if err != nil {
			t.Fatal(err)
		}
		i := assets.Snapshot.PlayerOf(player.Franchise, player.ID)
		if i < 0 || assets.Snapshot.Players[i].Status != tc.status {
			t.Fatal("equal-effective changes lost revision order")
		}
	}
}

func TestNativeEffectiveCorrectionAndCompensation(t *testing.T) {
	b := rolloverLedger(t)
	entry := b.add(RawCommand{League: rolloverLeague, Command: CorrectRoster,
		Args:      RawArgs{Correction: &RawCorrection{Player: "0003", Franchise: "franchise.02", Action: "remove"}},
		Effective: seasonMoment(5).Format(time.RFC3339Nano), Reason: "Repair week membership"}, seasonMoment(12))
	b.revert(entry, seasonMoment(13))
	assertEffectiveOwners(t, b, "0003", []rosterInstant{
		{"before correction", seasonMoment(4), "franchise.02"},
		{"correction learned later", seasonMoment(5), ""},
		{"compensation inclusive", seasonMoment(13), "franchise.02"},
	})
	assets, err := NativeAssetsAsOf(rolloverLeague, b.records, seasonMoment(6))
	if err != nil || len(assets.Corrections) != 1 || !assets.Corrections[0].When.Equal(seasonMoment(12)) {
		t.Fatalf("backdated correction must retain its receipt: %+v %v", assets.Corrections, err)
	}
}
