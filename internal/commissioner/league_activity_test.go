package commissioner

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLeagueActivityAllowListOmitsPrivateAndUnknownEvents(t *testing.T) {
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	record := func(id, command string, args RawArgs, visibility, reason string, offset time.Duration) Record {
		return Record{Command: RawCommand{League: "native.activity", Command: command, Args: args,
			Visibility: visibility, Reason: reason}, Receipt: Receipt{Entry: id, RecordedAt: at.Add(offset)}}
	}
	records := []Record{
		record("notice", NoticePublish, RawArgs{Note: &RawNote{Text: "Public league notice"}}, "league", "secret commissioner reason", 0),
		record("note", NotePost, RawArgs{Note: &RawNote{Text: "secret league note"}}, "league", "", time.Second),
		record("bid", BidPlace, RawArgs{Bid: &RawBidArgs{Player: "private.player", Franchise: "franchise.01", Salary: 900000000, Years: 2}}, "private", "secret bid reason", 2*time.Second),
		record("offer", TradeOffer, RawArgs{TradeOutcome: &TradeState{ID: "offer", Status: "offered", Words: "secret pending offer", Parties: []string{"franchise.01", "franchise.02"}}}, "league", "secret pending reason", 3*time.Second),
		record("trade.done", TradeAccept, RawArgs{TradeOutcome: &TradeState{ID: "offer", Status: "approved", Words: "secret trade details", Parties: []string{"franchise.01", "franchise.02"}, Votes: []TradeVote{{Reason: "secret reviewer reason"}}}}, "league", "secret commissioner reason", 4*time.Second),
		record("drop", RosterCut, RawArgs{Waiver: &RawWaiverArgs{Franchise: "franchise.01"}, WaiverOutcome: &WaiverOutcome{Player: &NativePlayer{ID: "secret.drop.player", Name: "Secret Player Name"}}}, "league", "secret drop reason", 5*time.Second),
		record("claim", WaiverProcess, RawArgs{WaiverOutcome: &WaiverOutcome{Winner: "franchise.02", Claims: []WaiverClaimResult{{ID: "secret.claim.id"}}}}, "league", "secret claim reason", 6*time.Second),
		record("add", BidsProcess, RawArgs{BidOutcome: &BidOutcome{Winner: "franchise.03", WinningBid: "secret.winning.bid", Player: "secret.bid.player", Results: []BidResult{{ID: "secret.bid.id", Refusal: "secret bid refusal"}}}}, "league", "secret bid decision reason", 7*time.Second),
		record("future", "future.sensitive.event", RawArgs{Note: &RawNote{Text: "secret unknown event"}}, "league", "secret unknown reason", 8*time.Second),
	}

	reading, err := LeagueActivityReading(RawLeagueActivityScope{League: "native.activity"}, records)
	if err != nil {
		t.Fatal(err)
	}
	if reading.League != "native.activity" || reading.Revision == 0 || len(reading.Events) != 5 {
		t.Fatalf("activity = %+v", reading)
	}
	wantKinds := []string{ActivityRoster, ActivityRoster, ActivityRoster, ActivityTrade, ActivityNotice}
	for i, want := range wantKinds {
		if reading.Events[i].Kind != want {
			t.Errorf("event %d kind=%q, want %q", i, reading.Events[i].Kind, want)
		}
	}
	body, err := json.Marshal(reading)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secret league note", "private.player", "secret pending offer", "secret commissioner reason",
		"secret reviewer reason", "secret drop reason", "Secret Player Name", "secret.claim.id", "secret.winning.bid",
		"secret.bid.player", "secret.bid.id", "secret bid refusal", "secret unknown event", "secret unknown reason"} {
		if strings.Contains(string(body), secret) {
			t.Errorf("private content %q leaked in %s", secret, body)
		}
	}
	if reading.Events[0].Summary != "Player added to a roster" || reading.Events[1].Summary != "Waiver claim awarded" ||
		reading.Events[2].Summary != "Player released from a roster" || reading.Events[3].Summary != "Trade completed" ||
		reading.Events[4].Summary != "Public league notice" {
		t.Fatalf("unexpected summaries: %+v", reading.Events)
	}
	publicRecords := []Record{records[0], records[3], records[4], records[5], records[6], records[7]}
	publicOnly, err := LeagueActivityReading(RawLeagueActivityScope{League: "native.activity"}, publicRecords)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reading, publicOnly) {
		t.Fatalf("private and unknown records affected the public reading: mixed=%+v public=%+v", reading, publicOnly)
	}
}

func TestLeagueActivityFiltersAndRejectsUnknownKinds(t *testing.T) {
	at := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{Command: RawCommand{League: "native.activity", Command: NoticePublish, Visibility: "league", Args: RawArgs{Note: &RawNote{Text: "Notice"}}}, Receipt: Receipt{Entry: "notice", RecordedAt: at}},
		{Command: RawCommand{League: "native.activity", Command: RosterCut, Visibility: "league", Args: RawArgs{Waiver: &RawWaiverArgs{Franchise: "franchise.02"}, WaiverOutcome: &WaiverOutcome{Player: &NativePlayer{ID: "2345"}}}}, Receipt: Receipt{Entry: "cut.2", RecordedAt: at.Add(2 * time.Minute)}},
		{Command: RawCommand{League: "native.activity", Command: RosterCut, Visibility: "league", Args: RawArgs{Waiver: &RawWaiverArgs{Franchise: "franchise.01"}, WaiverOutcome: &WaiverOutcome{Player: &NativePlayer{ID: "1234"}}}}, Receipt: Receipt{Entry: "cut.1", RecordedAt: at.Add(time.Minute)}},
	}
	all, err := LeagueActivityReading(RawLeagueActivityScope{League: "native.activity"}, records)
	if err != nil || len(all.Events) != 3 || all.Events[0].ID != "cut.2" || all.Events[1].ID != "cut.1" {
		t.Fatalf("events not ordered by recorded time: %+v, %v", all, err)
	}
	before := at.Add(3 * time.Minute).Format(time.RFC3339Nano)
	reading, err := LeagueActivityReading(RawLeagueActivityScope{League: "native.activity", Filter: &RawLeagueActivityFilter{
		Kind: ActivityRoster, Franchise: "franchise.01", Before: before, Limit: 1,
	}}, records)
	if err != nil {
		t.Fatal(err)
	}
	if len(reading.Events) != 1 || reading.Events[0].ID != "cut.1" {
		t.Fatalf("filtered events = %+v", reading.Events)
	}
	for _, filter := range []RawLeagueActivityFilter{{Kind: "bids"}, {Limit: 101}, {Limit: -1}, {Before: "tomorrow"}} {
		if err := filter.Validate(); err == nil {
			t.Errorf("invalid filter accepted: %+v", filter)
		}
	}
	if err := (RawLeagueActivityScope{League: "native.activity", Filter: &RawLeagueActivityFilter{Kind: "new.kind"}}).Validate(); err == nil {
		t.Fatal("unknown event kind accepted")
	}
}
