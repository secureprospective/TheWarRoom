package serverclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

type RawInvalidation struct {
	Protocol int `json:"protocol"`
}

func (r RawInvalidation) Validate() error {
	if r.Protocol != 1 {
		return fmt.Errorf("unsupported invalidation protocol")
	}
	return nil
}

// WatchInvalidations runs in desktop Go, never the webview. Reread must replace the
// cached view with an authorized server read, not replay an event or execute a write.
// Every connection starts with a reread, including after loss/overflow. No cursor is kept.
func (d *Desktop) WatchInvalidations(ctx context.Context, reread func(context.Context) error) error {
	if reread == nil {
		return fmt.Errorf("invalidation reread required")
	}
	for {
		retry, err := d.watchInvalidations(ctx, reread)
		if !retry {
			return err
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("invalidation watch stopped: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

func (d *Desktop) watchInvalidations(ctx context.Context, reread func(context.Context) error) (bool, error) {
	d.mu.Lock()
	d.restore()
	bearer, proof := d.bearer, d.proof
	client := *d.client.http
	address, league := d.client.base, d.league
	d.mu.Unlock()
	if bearer == "" {
		return false, fmt.Errorf("sign in before watching native changes")
	}
	client.Timeout = 0 // The request context owns the stream lifetime; normal calls stay bounded.
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address+serverhttp.InvalidationPath, nil)
	if err != nil {
		return false, fmt.Errorf("create invalidation request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+bearer)
	request.Header.Set("X-TWR-Recent-Auth", proof)
	request.Header.Set("X-TWR-League", league)
	response, err := client.Do(request)
	if err != nil {
		return true, fmt.Errorf("connect invalidations: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return response.StatusCode >= 500 || response.StatusCode == 429, fmt.Errorf("invalidation access refused (HTTP %d)", response.StatusCode)
	}
	if response.Header.Get("Content-Type") != "text/event-stream" {
		return false, fmt.Errorf("invalid stream response")
	}
	return consumeInvalidations(ctx, response, reread)
}

func consumeInvalidations(ctx context.Context, response *http.Response, reread func(context.Context) error) (bool, error) {
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 1024), 4096)
	event, data := "", ""
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if event != "" || data != "" {
				if err := validateInvalidation(event, data); err != nil {
					return false, err
				}
				if err := reread(ctx); err != nil {
					return false, fmt.Errorf("reread native view: %w", err)
				}
			}
			event, data = "", ""
		case line == "event: invalidate" && event == "":
			event = "invalidate"
		case len(line) > 6 && line[:6] == "data: " && data == "":
			data = line[6:]
		case len(line) > 0 && line[0] == ':':
		default:
			return false, fmt.Errorf("unsupported stream field")
		}
	}
	if err := scanner.Err(); err != nil {
		return true, fmt.Errorf("read invalidations: %w", err)
	}
	return true, fmt.Errorf("invalidation connection closed")
}

func validateInvalidation(event, data string) error {
	var raw RawInvalidation
	if event != "invalidate" || json.Unmarshal([]byte(data), &raw) != nil || raw.Validate() != nil {
		return fmt.Errorf("invalid invalidation event")
	}
	return nil
}
