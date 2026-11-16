package serverhttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestWorkflowRefusalMapping(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		retry  string
	}{
		{fmt.Errorf("credential admission: %w", commissioner.Refused(context.DeadlineExceeded)), 503, "5"},
		{fmt.Errorf("cancelled: %w", context.Canceled), 503, "5"},
		{errors.New("rules refusal"), 409, ""},
	} {
		w := httptest.NewRecorder()
		workflowRefusal(w, tc.err)
		if w.Code != tc.status || w.Header().Get("Retry-After") != tc.retry {
			t.Fatalf("%v: status %d retry %q", tc.err, w.Code, w.Header().Get("Retry-After"))
		}
		if tc.status == 503 && strings.TrimSpace(w.Body.String()) != "The league server is busy. Try again in a few seconds." {
			t.Fatalf("busy message: %q", w.Body.String())
		}
	}
}

func TestWorkflowDeadlineReplyReachesTLSClient(t *testing.T) {
	handler := resourceLimits(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		workflowRefusal(w, fmt.Errorf("credential admission: %w", r.Context().Err()))
	}), netip.Addr{})
	server := httptest.NewUnstartedServer(handler)
	server.Config = boundedServer(handler, tls.Certificate{})
	server.Config.TLSConfig.Certificates = nil
	server.TLS = server.Config.TLSConfig
	server.StartTLS()
	defer server.Close()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+WorkflowPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("TLS response lost: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 503 || response.Header.Get("Retry-After") != "5" || strings.TrimSpace(string(body)) != "The league server is busy. Try again in a few seconds." {
		t.Fatalf("status=%d retry=%q body=%q", response.StatusCode, response.Header.Get("Retry-After"), body)
	}
}
