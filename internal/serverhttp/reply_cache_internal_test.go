package serverhttp

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func testScope(c *ReplyCache, head int64, epoch int, principal string) replyScope {
	return replyScope{cache: c, league: "native.l", head: head, epoch: epoch, key: replyKey{principal: principal, reading: "native.assets.read"}}
}

func TestReplyCacheKeepsOnlySuccessPerHeadEpochAndViewer(t *testing.T) {
	c := NewReplyCache(1 << 20)
	var runs atomic.Int32
	read := func(body string) func() ([]byte, error) {
		return func() ([]byte, error) { runs.Add(1); return []byte(body), nil }
	}
	failing := errors.New("refused")
	if _, err := testScope(c, 1, 0, "a").reply(func() ([]byte, error) { runs.Add(1); return nil, failing }); !errors.Is(err, failing) {
		t.Fatalf("a failed read must return its error, got %v", err)
	}
	for range 3 {
		if got, _ := testScope(c, 1, 0, "a").reply(read("A1")); string(got) != "A1" {
			t.Fatalf("got %q", got)
		}
	}
	if runs.Load() != 2 {
		t.Fatalf("a failure is never kept and a success is: %d runs, want 2", runs.Load())
	}
	if got, _ := testScope(c, 1, 0, "b").reply(read("B1")); string(got) != "B1" {
		t.Fatalf("viewer b got %q: one viewer's reply reached another", got)
	}
	if got, _ := testScope(c, 1, 1, "a").reply(read("A-epoch")); string(got) != "A-epoch" {
		t.Fatalf("a new epoch must read afresh, got %q", got)
	}
	if got, _ := testScope(c, 2, 1, "a").reply(read("A2")); string(got) != "A2" {
		t.Fatalf("a new head must read afresh, got %q", got)
	}
	if got, _ := testScope(c, 1, 0, "a").reply(read("A-old")); string(got) != "A-old" {
		t.Fatalf("an older head reads alone, got %q", got)
	}
	if got, _ := testScope(c, 2, 1, "a").reply(read("never")); string(got) != "A2" {
		t.Fatalf("an older head must not displace the current one, got %q", got)
	}
	if c.size != 2 {
		t.Fatalf("size %d, want the one kept reply", c.size)
	}
}

func TestReplyCacheLimit(t *testing.T) {
	c := NewReplyCache(10)
	read := func(body string) func() ([]byte, error) { return func() ([]byte, error) { return []byte(body), nil } }
	_, _ = testScope(c, 1, 0, "a").reply(read("123456"))
	_, _ = testScope(c, 1, 0, "b").reply(read("123456"))
	if c.size != 6 || len(c.leagues["native.l"].replies) != 1 {
		t.Fatalf("past the limit the others are dropped: size %d, %d replies", c.size, len(c.leagues["native.l"].replies))
	}
	if got, _ := testScope(c, 1, 0, "c").reply(read("12345678901")); string(got) != "12345678901" {
		t.Fatal("an oversized reply is still answered")
	}
	if c.size != 6 {
		t.Fatalf("an oversized reply is not kept: size %d", c.size)
	}
}

// TestReplyCacheConcurrentReaders runs under -race: concurrent viewers across moving heads each get their own reply.
func TestReplyCacheConcurrentReaders(t *testing.T) {
	c := NewReplyCache(1 << 10)
	var wg sync.WaitGroup
	for g := range 16 {
		wg.Go(func() {
			for i := range 200 {
				head := int64(i / 20)
				principal := string(rune('a' + g%4))
				want := principal + string(rune('0'+head))
				got, err := testScope(c, head, 0, principal).reply(func() ([]byte, error) { return []byte(want), nil })
				if err != nil || string(got) != want {
					t.Errorf("got %q want %q (%v)", got, want, err)
					return
				}
			}
		})
	}
	wg.Wait()
}
