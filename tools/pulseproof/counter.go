// Package pulseproof contains development-only hosted transport instrumentation.
package pulseproof

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

// Counts reports completed HTTP responses, not attempted or failed network calls.
type Counts struct{ Calls, Unchanged int }

// Counter wraps a fixture client's transport. It never retains credentials or bodies.
type Counter struct {
	Transport http.RoundTripper
	mu        sync.Mutex
	counts    map[string]Counts
}

// Snapshot returns a copy suitable for comparing scenario steps.
func (c *Counter) Snapshot() map[string]Counts {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]Counts, len(c.counts))
	for k, v := range c.counts {
		out[k] = v
	}
	return out
}

// RoundTrip counts workflow responses; unchanged is observed as the client consumes the response.
func (c *Counter) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := c.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	if req.URL.Path != "/v1/native/workflow" {
		return transport.RoundTrip(req) //nolint:wrapcheck // Preserve transport errors unchanged.
	}
	command := "<unknown>"
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err == nil {
			var envelope struct {
				Workflow struct {
					Command string `json:"command"`
				} `json:"workflow"`
			}
			if json.NewDecoder(io.LimitReader(body, 1<<20)).Decode(&envelope) == nil && envelope.Workflow.Command != "" {
				command = envelope.Workflow.Command
			}
			_ = body.Close()
		}
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return resp, err //nolint:wrapcheck // Preserve transport errors unchanged.
	}
	c.mu.Lock()
	if c.counts == nil {
		c.counts = make(map[string]Counts)
	}
	count := c.counts[command]
	count.Calls++
	c.counts[command] = count
	c.mu.Unlock()
	resp.Body = &observedBody{ReadCloser: resp.Body, counter: c, command: command}
	return resp, nil
}

type observedBody struct {
	io.ReadCloser
	counter            *Counter
	command            string
	data               bytes.Buffer
	overflow, finished bool
}

func (b *observedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if !b.overflow {
		if b.data.Len()+n > 1<<20 {
			b.overflow = true
			b.data.Reset()
		} else {
			_, _ = b.data.Write(p[:n])
		}
	}
	if err == io.EOF && !b.finished {
		b.finished = true
		var result struct {
			Unchanged bool `json:"unchanged"`
		}
		if !b.overflow && json.Unmarshal(b.data.Bytes(), &result) == nil && result.Unchanged {
			b.counter.mu.Lock()
			count := b.counter.counts[b.command]
			count.Unchanged++
			b.counter.counts[b.command] = count
			b.counter.mu.Unlock()
		}
		b.data.Reset()
	}
	return n, err //nolint:wrapcheck // Preserve reader errors, including EOF.
}
