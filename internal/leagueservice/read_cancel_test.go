package leagueservice

import (
	"context"
	"errors"
	"testing"
)

func TestReadCancelsWithdrawRunningAndEarlyRequests(t *testing.T) {
	var r readCancels
	ctx, release := r.watch(context.Background(), "room-1")
	r.withdraw("room-1")
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("running read not cancelled: %v", ctx.Err())
	}
	release()

	r.withdraw("room-2") // arrives before its read starts
	early, release2 := r.watch(context.Background(), "room-2")
	defer release2()
	if early.Err() == nil {
		t.Fatal("a read withdrawn before it started must start cancelled")
	}

	other, release3 := r.watch(context.Background(), "room-3")
	defer release3()
	unnamed, release4 := r.watch(context.Background(), "")
	defer release4()
	r.withdraw("room-1") // finished: no effect on anyone else
	if other.Err() != nil || unnamed.Err() != nil {
		t.Fatal("withdrawing one request cancelled another")
	}
}

func TestCancelReadRefusesMalformedRequest(t *testing.T) {
	a := New(nil)
	for _, bad := range []string{"room 1", "room/1", string(make([]byte, 65))} {
		if err := a.TargetCancelRead(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	if err := a.TargetCancelRead("room-1"); err != nil {
		t.Fatal(err)
	}
}
