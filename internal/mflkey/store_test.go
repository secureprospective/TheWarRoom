package mflkey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

func TestStore(t *testing.T) {
	keyring.MockInit()
	s := New("league", "2026")
	ctx := context.Background()
	if _, err := s.Get(ctx); !errors.Is(err, ErrNoKey) {
		t.Fatal(err)
	}
	if err := s.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	key := Key("test credential value")
	if err := s.Set(ctx, key); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx)
	if err != nil || got != key {
		t.Fatalf("get: %v %v", got, err)
	}
	if _, err := New("league", "2027").Get(ctx); !errors.Is(err, ErrNoKey) {
		t.Fatal(err)
	}
	if err := s.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	keyring.MockInitWithError(errors.New(string(key)))
	if _, err := s.Get(ctx); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	if err := s.Set(ctx, key); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	if err := s.Delete(ctx); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}

func TestTimeout(t *testing.T) {
	release := make(chan struct{})
	finished := make(chan struct{})
	s := New("league", "2026")
	s.get = func(string, string) (string, error) {
		<-release
		close(finished)
		return "late value", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := s.Get(ctx); !errors.Is(err, ErrTimeout) {
		t.Fatal(err)
	}
	close(release)
	<-finished
}

func TestRedaction(t *testing.T) {
	key := Key("private test value")
	for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%d", "%T"} {
		if strings.Contains(fmt.Sprintf(format, key), string(key)) {
			t.Fatal(format)
		}
	}
	body, err := json.Marshal(key)
	if err != nil || string(body) != `"[redacted]"` {
		t.Fatalf("marshal: %s %v", body, err)
	}
}

func TestMutationTimeouts(t *testing.T) {
	for _, operation := range []string{"set", "delete"} {
		t.Run(operation, func(t *testing.T) {
			release, finished := make(chan struct{}), make(chan struct{})
			block := func() error { <-release; close(finished); return nil }
			s := New("league", "2026")
			s.set = func(string, string, string) error { return block() }
			s.delete = func(string, string) error { return block() }
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()
			var err error
			if operation == "set" {
				err = s.Set(ctx, "candidate")
			} else {
				err = s.Delete(ctx)
			}
			if !errors.Is(err, ErrTimeout) {
				t.Fatal(err)
			}
			close(release)
			<-finished
		})
	}
}
