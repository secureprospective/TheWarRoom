package serverstatus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"time"

	"golang.org/x/sys/unix"
)

const (
	lowDisk     = 1 << 30
	certWarning = 30 * 24 * time.Hour
	waitAgain   = "Wait five minutes and look again: the server restarts itself. If it is still here, reboot the container " +
		"(in Proxmox select it, then Reboot), wait five minutes and look again. If it is still here, call Christopher."
)

// Console is the console command's view: everything checkable from inside the CT, even when
// the league server is down. It contacts only the private socket and two loopback
// addresses on this container; Validate refuses anything else, so the dashboard can
// never become an outbound notifier.
type Console struct {
	Socket string
	// Origin is the league server's loopback TLS address, trusted through OriginCA as localhost.
	Origin   string
	OriginCA *x509.CertPool
	// Tunnel is cloudflared's loopback metrics address, which answers /ready.
	Tunnel string
	// State is the folder holding the league; backups stage there too. Space reads it (FreeSpace).
	State string
	Space func(path string) (free, total uint64, err error)
	Clock func() error
	Dial  func(ctx context.Context, network, address string) (net.Conn, error)
	Now   func() time.Time
}

// Validate refuses any network destination outside this container.
func (c Console) Validate() error {
	for _, address := range []string{c.Origin, c.Tunnel} {
		parsed, err := netip.ParseAddrPort(address)
		if err != nil || !parsed.Addr().IsLoopback() {
			return fmt.Errorf("status checks only reach loopback IP:port addresses on this container, not %q", address)
		}
	}
	if c.Socket == "" || c.OriginCA == nil || c.State == "" || c.Space == nil || c.Clock == nil || c.Dial == nil || c.Now == nil {
		return fmt.Errorf("status checks need the socket, origin certificate, state folder, free-space reader, clock and dialer")
	}
	return nil
}

// Checks reads the dashboard in the order a person fixes things.
func (c Console) Checks(ctx context.Context) ([]Check, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	checks := []Check{c.origin(ctx), c.tunnel(ctx)}
	server, err := c.server(ctx)
	if err != nil {
		checks = append(checks, Check{Name: "Backups and history", State: Problem,
			Detail: "Could not be read, because the league server's private status did not answer (" + err.Error() + ").", Action: waitAgain})
	}
	var notes []Check
	for _, check := range server {
		if check.State == Note {
			notes = append(notes, check)
		} else {
			checks = append(checks, check)
		}
	}
	checks = append(checks, c.clock(), c.disk())
	return append(checks, notes...), nil
}

func (c Console) client(transport *http.Transport) *http.Client {
	transport.Proxy = nil // never route a status check through an environment proxy
	transport.DialContext = c.Dial
	return &http.Client{Transport: transport, Timeout: 90 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func (c Console) get(ctx context.Context, client *http.Client, url string, header http.Header) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build status request: %w", err)
	}
	for name, values := range header {
		request.Header[name] = values
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("status request: %w", err)
	}
	return response, nil
}

// origin repeats the installer's readiness probe: TLS verified for the loopback IP, and a real
// application reply (404 for an unknown route, past the Tunnel peer check).
func (c Console) origin(ctx context.Context) Check {
	check := Check{Name: "League server", State: Problem, Action: waitAgain}
	client := c.client(&http.Transport{TLSClientConfig: &tls.Config{RootCAs: c.OriginCA, MinVersion: tls.VersionTLS12}})
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	response, err := c.get(ctx, client, "https://"+c.Origin+"/not-a-route", http.Header{"Cf-Connecting-Ip": {"127.0.0.1"}})
	if err != nil {
		check.Detail = "Not answering on this machine, so nobody can use the league (" + err.Error() + ")."
		return check
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode != http.StatusNotFound {
		check.Detail = fmt.Sprintf("Answering, but not normally (HTTP %d), so owners may not get in.", response.StatusCode)
		return check
	}
	expires := response.TLS.PeerCertificates[0].NotAfter
	if expires.Sub(c.Now()) < certWarning {
		check.Detail = "Answering, but its private certificate for the Tunnel expires " + expires.UTC().Format(time.DateOnly) + "."
		check.Action = "Call Christopher before that date."
		return check
	}
	return Check{Name: "League server", State: OK, Detail: "Running and answering securely on this machine."}
}

func (c Console) tunnel(ctx context.Context) Check {
	check := Check{Name: "Tunnel", State: Problem, Detail: "Not connected to Cloudflare, so owners cannot reach the league from the internet.",
		Action: "Check that the house internet works. If it does and this stays for 30 minutes, reboot the container " +
			"(in Proxmox select it, then Reboot), wait five minutes and look again. If it is still here, call Christopher."}
	client := c.client(&http.Transport{})
	defer client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	response, err := c.get(ctx, client, "http://"+c.Tunnel+"/ready", nil)
	if err != nil {
		check.Detail += " The Tunnel program is not answering (" + err.Error() + ")."
		return check
	}
	defer func() { _ = response.Body.Close() }()
	var ready struct {
		Connections int `json:"readyConnections"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&ready) != nil || ready.Connections < 1 {
		return check
	}
	return Check{Name: "Tunnel", State: OK, Detail: fmt.Sprintf("Connected to Cloudflare (%d connections), so owners can reach the league.", ready.Connections)}
}

func (c Console) server(ctx context.Context) ([]Check, error) {
	transport := &http.Transport{}
	client := c.client(transport)
	defer client.CloseIdleConnections()
	transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return c.Dial(ctx, "unix", c.Socket)
	}
	response, err := c.get(ctx, client, "http://status"+StatusPath, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	var report Report
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&report) != nil || report.Version != reportVersion {
		return nil, fmt.Errorf("unreadable reply (HTTP %d)", response.StatusCode)
	}
	return report.Checks, nil
}

func (c Console) clock() Check {
	if err := c.Clock(); err != nil {
		return Check{Name: "Clock", State: Problem,
			Detail: "Not synchronised (" + err.Error() + "), so bid, waiver and other deadlines are on hold. Nothing is lost while they wait.",
			Action: "Call Christopher: the clock comes from the Proxmox machine, not this container."}
	}
	return Check{Name: "Clock", State: OK, Detail: "Synchronised, so deadlines run on time."}
}

func (c Console) disk() Check {
	free, total, err := c.Space(c.State)
	switch {
	case err != nil:
		return Check{Name: "Disk", State: Problem, Detail: "Free space could not be read (" + err.Error() + ").", Action: "Call Christopher."}
	case free < lowDisk:
		return Check{Name: "Disk", State: Problem, Detail: fmt.Sprintf("Only %s free of %s. Backups need room to work.", size(free), size(total)),
			Action: "Call Christopher this week; nothing needs to be deleted by hand."}
	}
	return Check{Name: "Disk", State: OK, Detail: fmt.Sprintf("%s free of %s.", size(free), size(total))}
}

// FreeSpace reports bytes available to the unprivileged service user, and the filesystem
// size. The league server runs only on Linux; there is no Windows build of it.
func FreeSpace(path string) (free, total uint64, err error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0, fmt.Errorf("read free space: %w", err)
	}
	block := uint64(stat.Bsize) //nolint:gosec // Bsize is a positive block size from the kernel.
	return stat.Bavail * block, stat.Blocks * block, nil
}

func size(bytes uint64) string {
	return fmt.Sprintf("%.1f GB", float64(bytes)/1e9)
}
