// Package serverhttp exposes the loopback-only fixture HTTPS JSON protocol.
package serverhttp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

const FixturePath = "/v1/fixture/ledger"

// RawLedgerResponse is protocol v1's fixture-only reading, not a durable command version.
type RawLedgerResponse struct {
	Protocol int                 `json:"protocol"`
	Ledger   commissioner.Ledger `json:"ledger"`
}

func (r RawLedgerResponse) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported protocol major")
	}
	if r.Ledger.League != "native.https-fixture" || r.Ledger.Name == "" || r.Ledger.Revision < 1 || len(r.Ledger.Entries) == 0 {
		return fmt.Errorf("invalid fixture ledger")
	}
	for _, e := range r.Ledger.Entries {
		if e.ID == "" || e.Who == "" || e.What == "" || e.When.IsZero() || e.Undo != "" {
			return fmt.Errorf("invalid fixture ledger entry")
		}
	}
	return nil
}

// ListenLoopback rejects hostnames and wildcard/non-loopback IPs before opening a socket.
func ListenLoopback(ctx context.Context, address string) (net.Listener, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("parse listen address: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return nil, fmt.Errorf("fixture server requires a literal loopback IP")
	}
	var config net.ListenConfig
	listener, err := config.Listen(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen for fixture HTTPS: %w", err)
	}
	return listener, nil
}

// ServeFixture owns the listener and closes it on cancellation. It has no write routes.
func ServeFixture(ctx context.Context, listener net.Listener, certificate tls.Certificate) error {
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !address.IP.IsLoopback() {
		return fmt.Errorf("fixture listener is not loopback")
	}
	server := &http.Server{
		Handler: http.HandlerFunc(fixtureRead), ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second,
		MaxHeaderBytes: 16 << 10,
		TLSConfig:      &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}},
	}
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
			// There are no writes to drain; close active reads as well as the listener.
			if err := server.Close(); err != nil {
				log.Printf("close fixture server: %v", err)
			}
		case <-stopped:
		}
	}()
	err := server.ServeTLS(listener, "", "")
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("serve fixture HTTPS: %w", err)
}

// RawFixtureRequest contains only the route and shape of a read; it carries no authority.
type RawFixtureRequest struct {
	Method        string
	Path          string
	Query         string
	ContentLength int64
}

type requestRefusal struct {
	status  int
	message string
}

func (r requestRefusal) Error() string { return r.message }

func (r RawFixtureRequest) Validate() error {
	if r.Path != FixturePath {
		return requestRefusal{http.StatusUpgradeRequired, "unsupported protocol or route"}
	}
	if r.Method != http.MethodGet {
		return requestRefusal{http.StatusMethodNotAllowed, "read only"}
	}
	if r.Query != "" || r.ContentLength != 0 {
		return requestRefusal{http.StatusBadRequest, "fixture read takes no arguments"}
	}
	return nil
}

func fixtureRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.TLS == nil {
		http.Error(w, "HTTPS required", http.StatusBadRequest)
		return
	}
	raw := RawFixtureRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, ContentLength: r.ContentLength}
	if err := raw.Validate(); err != nil {
		var refusal requestRefusal
		if errors.As(err, &refusal) {
			if refusal.status == http.StatusMethodNotAllowed {
				w.Header().Set("Allow", "GET")
			}
			http.Error(w, refusal.message, refusal.status)
		} else {
			http.Error(w, "invalid request", http.StatusBadRequest)
		}
		return
	}
	ledger, err := leagueservice.FixtureLedger(r.Context())
	if err != nil {
		http.Error(w, "fixture read failed", http.StatusInternalServerError)
		return
	}
	body, err := json.Marshal(RawLedgerResponse{Protocol: 1, Ledger: ledger})
	if err != nil {
		http.Error(w, "fixture encoding failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		return
	}
}
