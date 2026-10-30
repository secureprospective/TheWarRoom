package serverhttp

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestBoundedServerSlowBodyAndHeaders(t *testing.T) {
	bodyResult := make(chan error, 1)
	handler := resourceLimits(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.Copy(io.Discard, r.Body)
		bodyResult <- err
		http.Error(w, "body refused", http.StatusBadRequest)
	}), netip.Addr{})
	server := httptest.NewUnstartedServer(handler)
	server.Config = boundedServer(handler, tls.Certificate{})
	if server.Config.ReadHeaderTimeout != 5*time.Second || server.Config.ReadTimeout != 10*time.Second || server.Config.WriteTimeout != 5*time.Second || server.Config.MaxHeaderBytes != 8<<10 {
		t.Fatal("server resource defaults changed")
	}
	// Exercise the same network deadlines without spending ten seconds per regression.
	server.Config.ReadHeaderTimeout = 50 * time.Millisecond
	server.Config.ReadTimeout = 100 * time.Millisecond
	server.Config.WriteTimeout = time.Second
	server.Config.TLSConfig.Certificates = nil
	server.TLS = server.Config.TLSConfig
	server.StartTLS()
	defer server.Close()
	clientTLS := server.Client().Transport.(*http.Transport).TLSClientConfig
	connection, err := tls.Dial("tcp", server.Listener.Addr().String(), clientTLS)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = connection.Close() }()
	if err := connection.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(connection, "POST /v1/native/execute HTTP/1.1\r\nHost: test\r\nContent-Length: 1000\r\n\r\nx"); err != nil {
		t.Fatal(err)
	}
	if err := <-bodyResult; err == nil {
		t.Fatal("slow body was accepted")
	}
	slowHeader, err := tls.Dial("tcp", server.Listener.Addr().String(), clientTLS)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = slowHeader.Close() }()
	if err := slowHeader.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(slowHeader, "POST / HTTP/1.1\r\nHost:"); err != nil {
		t.Fatal(err)
	}
	began := time.Now()
	_, err = io.ReadAll(slowHeader)
	if time.Since(began) >= 500*time.Millisecond {
		t.Fatalf("slow headers not closed promptly: %v", err)
	}
}

func TestHostedServerOversizeHeaders(t *testing.T) {
	server := httptest.NewUnstartedServer(resourceLimits(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }), netip.Addr{}))
	server.Config = boundedServer(server.Config.Handler, tls.Certificate{})
	server.Config.TLSConfig.Certificates = nil
	server.TLS = server.Config.TLSConfig
	server.StartTLS()
	defer server.Close()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Oversize", strings.Repeat("x", 32<<10))
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("oversize headers: %d", response.StatusCode)
	}
}

func TestBoundedServerHeaderCap(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	server := httptest.NewUnstartedServer(handler)
	server.Config = boundedServer(handler, tls.Certificate{})
	server.Config.TLSConfig.Certificates = nil
	server.TLS = server.Config.TLSConfig
	server.StartTLS()
	defer server.Close()
	// Rows run in order: the reused-connection rows follow an accepted request, where
	// net/http allows the most read-ahead. Neither path may admit headers over 16 KiB.
	for _, tc := range []struct {
		name  string
		size  int
		fresh bool
		want  int
	}{
		{"ordinary headers, new connection", 4 << 10, true, http.StatusNoContent},
		{"over 16 KiB, new connection", 17 << 10, true, http.StatusRequestHeaderFieldsTooLarge},
		{"ordinary headers, kept alive", 4 << 10, false, http.StatusNoContent},
		{"over 16 KiB, reused connection", 17 << 10, false, http.StatusRequestHeaderFieldsTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			request.Close = tc.fresh
			request.Header.Set("X-Pad", strings.Repeat("a", tc.size))
			response, err := server.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			_ = response.Body.Close()
			if response.StatusCode != tc.want {
				t.Fatalf("status %d, want %d", response.StatusCode, tc.want)
			}
		})
	}
}
