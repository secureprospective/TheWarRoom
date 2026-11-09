package composition

import (
	"strconv"
	"strings"
	"testing"
)

// R3 (10-30): every open bid counts against cap room until it is outbid or wins.
func TestNativeOpenBidsTogetherMustFitTheCap(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	committed := f.capOf(t, "franchise.01").CommittedCents
	if committed == nil {
		t.Fatal("the fixture's counted salary is not judged")
	}
	f.set(t, "cap.amount", strconv.FormatInt(int64(*committed)+8*million, 10), "now")
	f.place(t, "owner.one", "franchise.01", "5001", 5*million, 1)
	err := f.tryPlace(t, "owner.one", "franchise.01", "5002", 5*million, 1)
	if err == nil || !strings.Contains(err.Error(), "open bids would pass the salary cap together") {
		t.Fatalf("two open bids past the cap together were accepted: %v", err)
	}
	if err := f.tryPlace(t, "owner.one", "franchise.01", "5002", 3*million, 1); err != nil {
		t.Fatalf("open bids that fit the cap together were refused: %v", err)
	}
	if reading := f.bids(t, "owner.one", "franchise.01"); len(reading.Players) != 2 {
		t.Fatalf("want the two fitting bids recorded, got %d", len(reading.Players))
	}
}
