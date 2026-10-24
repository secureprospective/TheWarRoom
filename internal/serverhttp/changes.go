package serverhttp

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

const FixtureChangePath = "/v1/fixture/change"

// RawFixtureChange carries no principal or admission time. Durable command and protocol
// versions are independent; the fixture exposes only one completed mutation.
type RawFixtureChange struct {
	Protocol int                     `json:"protocol"`
	Command  commissioner.RawCommand `json:"command"`
}

func (r RawFixtureChange) Validate() error {
	if r.Protocol != 1 || r.Command.League != "native.https-fixture" || r.Command.Command != commissioner.NotePost {
		return fmt.Errorf("unsupported fixture change")
	}
	if err := r.Command.Validate(); err != nil {
		return fmt.Errorf("validate fixture change: %w", err)
	}
	return nil
}

type RawReceiptResponse struct {
	Protocol int                  `json:"protocol"`
	Receipt  commissioner.Receipt `json:"receipt"`
}

func (r RawReceiptResponse) Validate() error {
	if r.Protocol != 1 || r.Receipt.League != "native.https-fixture" || r.Receipt.Entry == "" || r.Receipt.Revision < 2 || r.Receipt.RecordedAt.IsZero() {
		return fmt.Errorf("invalid fixture receipt")
	}
	return nil
}

// ServeFixtureChanges is opt-in development authority, never a public or account route.
// The caller supplies a hosted service with fixture-only credential admission.
func ServeFixtureChanges(ctx context.Context, listener net.Listener, certificate tls.Certificate, service *leagueservice.Service) error {
	if service == nil {
		return fmt.Errorf("fixture service is required")
	}
	return serveFixture(ctx, listener, certificate, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixtureChange(w, r, service)
	}))
}

func fixtureChange(w http.ResponseWriter, r *http.Request, service *leagueservice.Service) {
	w.Header().Set("Cache-Control", "no-store")
	if r.TLS == nil || r.Header.Get("Origin") != "" || r.URL.RawQuery != "" {
		http.Error(w, "fixture request refused", http.StatusBadRequest)
		return
	}
	if r.URL.Path == FixturePath && r.Method == http.MethodGet && r.ContentLength == 0 {
		//nolint:contextcheck // The existing reading API owns its bounded host context; execute uses the request context.
		ledger, err := service.TargetCommissionerLedger("native.https-fixture")
		if err != nil {
			http.Error(w, "fixture read refused", http.StatusForbidden)
			return
		}
		writeFixtureJSON(w, RawLedgerResponse{Protocol: 1, Ledger: ledger})
		return
	}
	if r.URL.Path != FixtureChangePath || r.Method != http.MethodPost {
		http.Error(w, "unsupported protocol or route", http.StatusUpgradeRequired)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var raw RawFixtureChange
	if err := decoder.Decode(&raw); err != nil {
		http.Error(w, "invalid fixture change", http.StatusBadRequest)
		return
	}
	if err := raw.Validate(); err != nil {
		http.Error(w, "invalid fixture change", http.StatusBadRequest)
		return
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "invalid trailing data", http.StatusBadRequest)
		return
	}
	receipt, err := service.Execute(r.Context(), raw.Command)
	if err != nil {
		http.Error(w, "fixture change refused; reread before retrying", http.StatusConflict)
		return
	}
	writeFixtureJSON(w, RawReceiptResponse{Protocol: 1, Receipt: receipt})
}

// Both response shapes are closed protocol types, not arbitrary application objects.
type fixtureResponse interface {
	RawLedgerResponse | RawReceiptResponse
}

func writeFixtureJSON[T fixtureResponse](w http.ResponseWriter, response T) {
	body, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "fixture encoding failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		return
	}
}
