package serverclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

type RawInvalidation struct {
	Protocol    int                       `json:"protocol"`
	ServerEpoch string                    `json:"serverEpoch"`
	League      string                    `json:"league"`
	Topics      []serverhttp.TopicVersion `json:"topics"`
}

func (r RawInvalidation) Validate() error {
	if r.Protocol == 1 {
		return nil
	}
	if r.Protocol != 2 || r.ServerEpoch == "" || !strings.HasPrefix(r.League, "native.") || len(r.Topics) > 64 {
		return fmt.Errorf("unsupported invalidation protocol")
	}
	for _, topic := range r.Topics {
		if err := validateChangeTopic(topic); err != nil {
			return err
		}
	}

	return nil
}

func validateChangeTopic(topic serverhttp.TopicVersion) error {
	if _, err := strconv.ParseUint(topic.Version, 10, 64); err != nil {
		return fmt.Errorf("invalid topic version: %w", err)
	}
	switch topic.Kind {
	case "league":
	case "team":
		if topic.Franchise == "" {
			return fmt.Errorf("missing franchise")
		}
	case "scores":
		if topic.Season < 1 || topic.Week < 1 {
			return fmt.Errorf("missing score scope")
		}
	case "values":
		if topic.Season < 1 || topic.Run == "" {
			return fmt.Errorf("missing value scope")
		}
	case "clock":
		if topic.NextBoundary != "" {
			if _, err := time.Parse(time.RFC3339, topic.NextBoundary); err != nil {
				return fmt.Errorf("invalid boundary: %w", err)
			}
		}
	default:
		return fmt.Errorf("invalid topic kind")
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
	d.mu.Lock()
	d.restore()
	connection := d.captureInvalidations()
	d.mu.Unlock()
	return connection.run(ctx, func(ctx context.Context, _ RawInvalidation) error { return reread(ctx) })
}

func (c invalidationConnection) run(ctx context.Context, receive func(context.Context, RawInvalidation) error) error {
	var backoff reconnectBackoff
	for {
		connected := time.Now()
		retry, err := c.watch(ctx, receive)
		if !retry {
			return err
		}
		timer := time.NewTimer(backoff.next(time.Since(connected)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("invalidation watch stopped: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

// Reconnects back off exponentially from 1 s to 30 s with full jitter, so devices dropped by one outage
// do not return in step; a stream that stayed up for streamHealthyAfter starts again from 1 s.
const (
	reconnectFirst     = time.Second
	reconnectCap       = 30 * time.Second
	streamHealthyAfter = time.Minute
)

type reconnectBackoff struct{ attempt int }

// next is the wait before the next connect, given how long the last stream stayed up.
func (b *reconnectBackoff) next(up time.Duration) time.Duration {
	if up >= streamHealthyAfter {
		b.attempt = 0
	}
	ceiling := reconnectCap
	if b.attempt < 5 { // 1 s << 5 = 32 s already passes the cap
		ceiling = min(reconnectFirst<<b.attempt, reconnectCap)
	}
	b.attempt++
	return rand.N(ceiling) + 1 //nolint:gosec // jitter, not a secret
}

type invalidationConnection struct {
	desktopSnapshot
	state func(string)
}

// captureInvalidations is called under the desktop lock; reconnect never recaptures credentials.
func (d *Desktop) captureInvalidations() invalidationConnection {
	return invalidationConnection{desktopSnapshot: d.snapshotLocked()}
}

func (c invalidationConnection) watch(ctx context.Context, receive func(context.Context, RawInvalidation) error) (bool, error) {
	bearer, proof, client, address, league := c.bearer, c.proof, *c.client, c.address, c.league
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
	request.Header.Set("X-TWR-Invalidation", "2")
	response, err := client.Do(request)
	if update, ok := updateRequired(err); ok {
		return false, update
	}
	if err != nil {
		return true, fmt.Errorf("connect invalidations: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if c.state != nil {
			switch response.StatusCode {
			case 401:
				c.state("signed-out")
			case 403:
				c.state("refused")
			}
		}
		return response.StatusCode >= 500 || response.StatusCode == 429, fmt.Errorf("invalidation access refused (HTTP %d)", response.StatusCode)
	}
	if response.Header.Get("Content-Type") != "text/event-stream" {
		return false, fmt.Errorf("invalid stream response")
	}
	if c.state != nil {
		c.state("connected")
	}
	defer func() {
		if c.state != nil && ctx.Err() == nil {
			c.state("reconnecting")
		}
	}()
	return consumeTopicInvalidations(ctx, response, receive)
}

func consumeInvalidations(ctx context.Context, response *http.Response, reread func(context.Context) error) (bool, error) {
	return consumeTopicInvalidations(ctx, response, func(ctx context.Context, _ RawInvalidation) error { return reread(ctx) })
}

func consumeTopicInvalidations(ctx context.Context, response *http.Response, receive func(context.Context, RawInvalidation) error) (bool, error) {
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 1024), 4096)
	event, data := "", ""
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if event != "" || data != "" {
				if err := receiveInvalidation(ctx, event, data, receive); err != nil {
					return false, err
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

func receiveInvalidation(ctx context.Context, event, data string, receive func(context.Context, RawInvalidation) error) error {
	if err := validateInvalidation(event, data); err != nil {
		return err
	}
	var raw RawInvalidation
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return fmt.Errorf("decode invalidation: %w", err)
	}
	if err := receive(ctx, raw); err != nil {
		return fmt.Errorf("receive invalidation: %w", err)
	}
	return nil
}
