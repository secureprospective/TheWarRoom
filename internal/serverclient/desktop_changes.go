package serverclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

const DesktopChangesPath = "/desktop/changes"

type DesktopChanges struct {
	AuthGeneration string                    `json:"authGeneration"`
	Seq            uint64                    `json:"seq"`
	Topics         []serverhttp.TopicVersion `json:"topics"`
	State          string                    `json:"state"`
	ServerEpoch    string                    `json:"serverEpoch,omitempty"`
	League         string                    `json:"league,omitempty"`
}

type RawChanges struct {
	After *uint64 `json:"after"`
}

func (r RawChanges) Validate() error {
	if r.After == nil {
		return fmt.Errorf("after sequence required")
	}
	return nil
}

// A bounded hint history bridges slow polls. Overflow returns an empty-topic resync hint.
func (d *Desktop) changeLocked(state string, message RawInvalidation) {
	if d.changed == nil {
		d.changed = make(chan struct{})
	}
	d.changes.AuthGeneration = strconv.FormatUint(d.generation, 10)
	d.changes.Seq++
	d.changes.State = state
	d.changes.ServerEpoch, d.changes.League = message.ServerEpoch, message.League
	d.changes.Topics = append([]serverhttp.TopicVersion{}, message.Topics...)
	d.changeHistory = append(d.changeHistory, d.changes)
	if len(d.changeHistory) > 64 {
		d.changeHistory = d.changeHistory[1:]
	}
	close(d.changed)
	d.changed = make(chan struct{})
}

func (d *Desktop) stopWatchLocked() {
	if d.watchCancel != nil {
		d.watchCancel()
		d.watchCancel = nil
	}
	d.changeLocked("signed-out", RawInvalidation{})
}

func (d *Desktop) startWatchLocked() {
	if d.bearer == "" {
		return
	}
	d.stopWatchLocked()
	ctx, cancel := context.WithCancel(context.Background())
	d.watchCancel = cancel
	generation := d.generation
	connection := d.captureInvalidations()
	connection.state = func(state string) {
		d.mu.Lock()
		defer d.mu.Unlock()
		if d.generation != generation {
			return
		}
		if state == "signed-out" || state == "refused" {
			d.generation++
			if err := d.clearCredential(); err != nil {
				state = "refused"
			}
		}
		d.changeLocked(state, RawInvalidation{})
	}
	d.changeLocked("reconnecting", RawInvalidation{})
	go func() {
		err := connection.run(ctx, func(_ context.Context, message RawInvalidation) error {
			d.mu.Lock()
			defer d.mu.Unlock()
			if d.generation != generation {
				return fmt.Errorf("superseded invalidation session")
			}
			d.changeLocked("connected", message)
			return nil
		})
		if err != nil && ctx.Err() == nil {
			d.mu.Lock()
			if d.generation == generation {
				d.changeLocked("refused", RawInvalidation{})
			}
			d.mu.Unlock()
		}
	}()
}

func (d *Desktop) changesAfter(ctx context.Context, after uint64, wait time.Duration) DesktopChanges {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	for {
		d.mu.Lock()
		if d.changed == nil {
			d.changed = make(chan struct{})
		}
		result := d.changes
		result.AuthGeneration = strconv.FormatUint(d.generation, 10)
		result.Topics = append([]serverhttp.TopicVersion{}, result.Topics...)
		if result.State == "" {
			result.State = "signed-out"
		}
		if result.Seq > after {
			result.Topics = d.topicsAfterLocked(after)
		}
		news := d.changed
		d.mu.Unlock()
		if result.Seq > after {
			return result
		}
		select {
		case <-ctx.Done():
			result.Topics = []serverhttp.TopicVersion{}
			return result
		case <-timer.C:
			result.Topics = []serverhttp.TopicVersion{}
			return result
		case <-news:
		}
	}
}

func (d *Desktop) changesHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "use a JSON change request", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var raw RawChanges
	if err := decoder.Decode(&raw); err != nil || raw.Validate() != nil {
		http.Error(w, "invalid change request", http.StatusBadRequest)
		return
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		http.Error(w, "one change request required", http.StatusBadRequest)
		return
	}
	result := d.changesAfter(r.Context(), *raw.After, 25*time.Second)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (d *Desktop) topicsAfterLocked(after uint64) []serverhttp.TopicVersion {
	topics := []serverhttp.TopicVersion{}
	if len(d.changeHistory) == 0 || d.changeHistory[0].Seq > after+1 {
		return topics
	}
	for _, change := range d.changeHistory {
		if change.Seq <= after {
			continue
		}
		// Empty topics means everything in the authorized scope changed.
		if len(change.Topics) == 0 {
			return []serverhttp.TopicVersion{}
		}
		for _, topic := range change.Topics {
			index := -1
			for i, old := range topics {
				if old.Kind == topic.Kind && old.Franchise == topic.Franchise && old.Season == topic.Season && old.Week == topic.Week && old.Run == topic.Run {
					index = i
					break
				}
			}
			if index < 0 {
				topics = append(topics, topic)
			} else {
				topics[index] = topic
			}
		}
	}
	return topics
}
