// Package mflkey keeps MFL credentials in the OS keyring, never on disk.
package mflkey

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/zalando/go-keyring"
)

// Key redacts itself at every formatting boundary. Reveal is for storage and transport only.
type Key string

func (Key) String() string               { return "[redacted]" }
func (Key) GoString() string             { return "[redacted]" }
func (Key) MarshalJSON() ([]byte, error) { return []byte(`"[redacted]"`), nil }
func (Key) Format(s fmt.State, _ rune)   { _, _ = io.WriteString(s, "[redacted]") }
func (k Key) Reveal() string             { return string(k) }

// Sentinel errors deliberately omit provider errors, which may contain credentials.
var (
	ErrNoKey       = errors.New("MFL key not found")
	ErrUnavailable = errors.New("OS keyring unavailable or unlock dismissed")
	ErrTimeout     = errors.New("keyring not responding")
)

// Store identifies a league-season credential.
type Store struct {
	account string
	get     func(string, string) (string, error)
	set     func(string, string, string) error
	delete  func(string, string) error
}

func New(league, season string) *Store {
	return &Store{
		account: "mfl-apikey:" + league + ":" + season,
		get:     keyring.Get,
		set:     keyring.Set,
		delete:  keyring.Delete,
	}
}

type result struct {
	key Key
	err error
}

// go-keyring cannot be cancelled: a timed-out operation can finish later (including a write).
// A buffered result channel lets that goroutine exit without retaining a waiting caller.
func bounded(ctx context.Context, call func() result) (Key, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return "", ErrTimeout
	}
	done := make(chan result, 1)
	go func() { done <- call() }()
	select {
	case <-ctx.Done():
		return "", ErrTimeout
	case r := <-done:
		if errors.Is(r.err, keyring.ErrNotFound) {
			return "", ErrNoKey
		}
		if r.err != nil {
			return "", ErrUnavailable
		}
		return r.key, nil
	}
}

func (s *Store) Get(ctx context.Context) (Key, error) {
	return bounded(ctx, func() result {
		value, err := s.get("TheWarRoom", s.account)
		return result{key: Key(value), err: err}
	})
}

func (s *Store) Set(ctx context.Context, key Key) error {
	_, err := bounded(ctx, func() result {
		return result{err: s.set("TheWarRoom", s.account, key.Reveal())}
	})
	return err
}

func (s *Store) Delete(ctx context.Context) error {
	_, err := bounded(ctx, func() result { return result{err: s.delete("TheWarRoom", s.account)} })
	if errors.Is(err, ErrNoKey) {
		return nil
	}
	return err
}
