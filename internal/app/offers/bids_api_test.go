package offers

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
)

func TestBidPreviewShowsGoFigures(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	p := a.TargetBidPreview(rawBid("5", 3))
	if p.Problem != "" || p.Points != "7.000" || p.Salary.Cents != 500_000_000 ||
		p.Player != "Smith, John (WR, DAL)" || p.Snapped != "" {
		t.Fatalf("preview = %+v", p)
	}
	if want := "Player: Smith, John (WR, DAL)\nContract: 3 years at $5,000,000 a year, flat\n" +
		"Cap hit: $5,000,000 a year\nPoints: 7.000"; p.Post != want {
		t.Errorf("post text:\n%s", p.Post)
	}
	snapped := a.TargetBidPreview(rawBid("5.004", 1))
	if snapped.Salary.Cents != 500_000_000 || !strings.Contains(snapped.Snapped, "Entered $5,004,000") ||
		snapped.Points != "5.000" {
		t.Errorf("snapped preview = %+v", snapped)
	}
	long := a.TargetBidPreview(rawBid("5", 5))
	if long.Problem != "" || long.Points != "Not scored" {
		t.Errorf("5-year preview = %+v", long)
	}
	for _, bad := range []RawBidInput{rawBid("", 1), rawBid("abc", 1), {FranchiseID: "1", Player: "13593", Salary: "5", Years: 1},
		{FranchiseID: "0001", Player: "99999", Salary: "5", Years: 1},
		{FranchiseID: "0001", Player: "13593", Salary: "5", Years: 1, Beat: "6.5551"},
		{FranchiseID: "0001", Player: "13593", Salary: "5", Years: 1, Closes: "tomorrow"}} {
		if got := a.TargetBidPreview(bad); got.Problem == "" {
			t.Errorf("bad input %+v gave %+v", bad, got)
		}
	}
}

func TestBidReviseAndWithdraw(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	plan, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	revised, err := a.TargetBidRevise(id, rawBid("6", 2))
	if err != nil || revised.Receipt.State != envelope.Ready || revised.Points != "7.200" {
		t.Fatalf("revised = %+v, %v", revised.Receipt.State, err)
	}
	var states []string
	for _, entry := range revised.Receipt.Audit {
		states = append(states, string(entry.To))
	}
	if strings.Join(states, ",") != "ready,stale,draft,ready" {
		t.Errorf("revision route = %v", states)
	}
	if _, err := a.TargetBidRevise(id, RawBidInput{FranchiseID: "0001", Player: "13594", Salary: "6", Years: 2}); err == nil {
		t.Error("a plan changed its player")
	}
	fixed, err := a.TargetBidRevise(id, rawBid("6", 4))
	if err != nil || fixed.Receipt.State != envelope.Ready || fixed.Facts.Years != 4 {
		t.Fatalf("fixed = %v, %v", fixed.Receipt.State, err)
	}
	gone, err := a.TargetBidWithdraw(id)
	if err != nil || gone.Receipt.State != envelope.Withdrawn || gone.State != "Withdrawn" ||
		gone.Status != "Removed by you before it was handed off." || len(gone.Actions) != 0 {
		t.Fatalf("withdrawn = %+v, %v", gone, err)
	}
	if _, err := a.TargetBidRevise(id, rawBid("6", 4)); err == nil {
		t.Error("a withdrawn plan was revised")
	}
}

func TestBidFactsFreezeOnceHandedOff(t *testing.T) {
	t.Parallel()
	a := bidTestApp(t)
	plan, err := a.TargetBidDraft(rawBid("5", 3))
	if err != nil {
		t.Fatal(err)
	}
	id := plan.Receipt.CorrelationID
	handed, err := a.TargetBidHandOff(id)
	if err != nil || handed.Receipt.State != envelope.HandedOff {
		t.Fatalf("hand-off = %v, %v", handed.Receipt.State, err)
	}
	if !strings.Contains(handed.Status, "nothing was posted for you") {
		t.Errorf("handed off: %q", handed.Status)
	}
	for _, try := range []func() error{
		func() error { _, err := a.TargetBidRevise(id, rawBid("9", 3)); return err },
		func() error { _, err := a.TargetBidHandOff(id); return err },
	} {
		if err := try(); err == nil {
			t.Error("a handed-off plan was changed")
		}
	}
	posted, err := a.TargetBidPosted(id)
	if err != nil || posted.Receipt.State != envelope.BidPending || !strings.Contains(posted.Status, "owner reports") {
		t.Fatalf("posted = %v, %v", posted.Receipt.State, err)
	}
	select {
	case <-a.core.MovesWake:
	default:
		t.Error("the watcher was not woken")
	}
}
