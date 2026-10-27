package comms

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

func TestMemberBoardRefusalAndOrderedReads(t *testing.T) {
	var boards [2]report.HeldFeed[[]ingestion.RawLeaguePost]
	for i := range boards {
		boards[i].Value = []ingestion.RawLeaguePost{{Title: "private"}}
		boards[i].FetchedAt = time.Now()
	}
	RefuseBoards(&boards, errors.New("locked"))
	if len(boards[0].Value) != 1 || boards[0].RefreshError != "locked" {
		t.Fatal(boards)
	}
	RefuseBoards(&boards, ErrNeedsKey)
	if boards[0].Value != nil || !boards[0].FetchedAt.IsZero() || boards[1].RefreshError != ErrNeedsKey.Error() {
		t.Fatal(boards)
	}
	if err := MemberKey(t.Context(), nil); err == nil || err.Error() != "MFL keyring is locked or unavailable" {
		t.Fatal(err)
	}
	var order []string
	RefreshBoards(t.Context(), func(context.Context) error { order = append(order, "key"); return nil }, func(error) { t.Fatal("refused") }, func() func(context.Context, time.Duration) error {
		order = append(order, "select pause")
		return func(context.Context, time.Duration) error { order = append(order, "pause"); return nil }
	}, func(_ context.Context, _ int, export string) { order = append(order, export) })
	if !reflect.DeepEqual(order, []string{"key", "select pause", "pause", "messageBoard", "select pause", "pause", "polls"}) {
		t.Fatal(order)
	}
	refused := false
	RefreshBoards(t.Context(), func(context.Context) error { return ErrNeedsKey }, func(err error) { refused = errors.Is(err, ErrNeedsKey) }, func() func(context.Context, time.Duration) error { t.Fatal("pause after refusal"); return nil }, func(context.Context, int, string) { t.Fatal("read after refusal") })
	if !refused {
		t.Fatal("missing refusal publication")
	}
}

func TestCommsRefreshFreshnessAndStartupAdmission(t *testing.T) {
	fresh := report.Feed{FetchedAt: time.Now()}
	for _, force := range []bool{false, true} {
		var order []string
		err := RefreshReading(force, func() error { order = append(order, "ready"); return nil }, func() (report.Feed, [2]report.Feed) {
			order = append(order, "capture")
			return fresh, [2]report.Feed{fresh, fresh}
		}, func() context.Context { order = append(order, "context"); return t.Context() }, func(run func()) { order = append(order, "lock"); run(); order = append(order, "unlock") }, func(context.Context) { order = append(order, "transactions") }, func(context.Context) { order = append(order, "boards") })
		want := []string{"ready", "capture", "context", "lock", "unlock"}
		if force {
			want = []string{"ready", "capture", "context", "lock", "transactions", "boards", "unlock"}
		}
		if err != nil || !reflect.DeepEqual(order, want) {
			t.Fatal(order, err)
		}
	}
	if err := RefreshReading(true, func() error { return errors.New("failed") }, func() (report.Feed, [2]report.Feed) { t.Fatal("early capture"); return fresh, [2]report.Feed{} }, nil, nil, nil, nil); err == nil || err.Error() != "comms refresh: failed" {
		t.Fatal(err)
	}
}
