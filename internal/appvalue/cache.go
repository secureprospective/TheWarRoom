package appvalue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/params"
)

// Cache retains one league's completed pass, shared by its franchise headers and inspectors.
// Database commits, changed settings/params and the next as-of observation invalidate it.
// Only the producer assembles evidence and runs the unchanged model for prepared screens.
type Cache struct {
	mu          sync.Mutex
	key         [sha256.Size]byte
	from, until time.Time
	reading     Reading
	ready       bool
	failure     error
	generation  uint64
}

func (c *Cache) get(key [sha256.Size]byte, at time.Time) (Reading, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ready || c.key != key || at.Before(c.from) || (!c.until.IsZero() && !at.Before(c.until)) {
		return Reading{}, false, nil
	}
	return cloneReading(c.reading), true, c.failure
}

func (src Source) cached(ctx context.Context, revision int64, set params.Set, franchise string, in Inputs) (Reading, error) {
	if err := ctx.Err(); err != nil {
		return Reading{}, fmt.Errorf("league values: %w", err)
	}
	key, err := cacheKey(revision, set, in)
	if err != nil {
		return Reading{}, err
	}
	if out, ok, failure := src.Cache.get(key, in.AsOf); ok && (failure == nil || src.PreparedOnly) {
		if failure != nil {
			return Reading{}, failure
		}
		return scoped(out, in, franchise), nil
	}
	if src.PreparedOnly {
		return preparing(in, franchise), nil
	}
	stable, err := src.stableKey(ctx, set, in)
	if err != nil {
		return Reading{}, err
	}
	if pass, ok := src.loadSaved(stable, in); ok {
		src.Cache.publish(key, pass.From, pass.Until, pass.Reading, nil)
		return scoped(pass.Reading, in, franchise), nil
	}
	until, err := src.NextFeatureAt(ctx, in.AsOf)
	if err != nil {
		return Reading{}, fmt.Errorf("league values expiry: %w", err)
	}
	out, valueErr := src.value(ctx, set, "", in)
	after, err := src.revision(ctx)
	if err != nil {
		return Reading{}, err
	}
	if after == revision {
		src.Cache.publish(key, in.AsOf, until, out, valueErr)
		if err := src.save(stable, in, in.AsOf, until, out, valueErr == nil); err != nil {
			return Reading{}, err
		}
	}
	if valueErr != nil {
		return Reading{}, valueErr
	}
	return scoped(out, in, franchise), nil
}

func (c *Cache) publish(key [sha256.Size]byte, from, until time.Time, out Reading, failure error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.key, c.from, c.until = key, from, until
	c.reading, c.ready, c.failure = cloneReading(out), true, failure
	c.generation++
}

func preparing(in Inputs, franchise string) Reading {
	return Reading{League: in.League, Season: in.Season, Franchise: franchise, Preparing: true,
		Players: []Player{}, Age: []Band{}, Notes: []string{},
		Best: Best{Franchise: franchise, Words: "Values are being prepared"}}
}

func cacheKey(revision any, set params.Set, in Inputs) ([sha256.Size]byte, error) {
	in.AsOf = time.Time{}
	body, err := json.Marshal(struct {
		Revision any
		Params   map[string]float64
		Inputs   Inputs
	}{revision, set.Values(), in})
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("league values cache key: %w", err)
	}
	return sha256.Sum256(body), nil
}

func cloneReading(in Reading) Reading {
	in.Players, in.Age, in.Notes = slices.Clone(in.Players), slices.Clone(in.Age), slices.Clone(in.Notes)
	return in
}

func scoped(out Reading, in Inputs, franchise string) Reading {
	all := out.Players
	if franchise != "" {
		out.Players = []Player{}
		for _, p := range all {
			if p.Franchise == franchise {
				out.Players = append(out.Players, p)
			}
		}
	}
	out.Franchise, out.Best, out.Age = franchise, best(all, in, franchise), bands(out.Players)
	return out
}

// Saved keeps completed passes across launches. A pass is reused only when its stable key
// (history evidence, params and league inputs) still matches and the as-of is inside its window.
type Saved interface {
	Load(scope Scope) ([]byte, error) // nil body, nil error when nothing is saved
	Save(scope Scope, body []byte) error
}

type savedPass struct {
	Key     string    `json:"key"`
	From    time.Time `json:"from"`
	Until   time.Time `json:"until"`
	Reading Reading   `json:"reading"`
}

// SavedFiles is Saved as one JSON file per league season in Dir.
type SavedFiles struct{ Dir string }

func (f SavedFiles) path(scope Scope) string {
	return filepath.Join(f.Dir, url.PathEscape(scope.League)+"-"+strconv.Itoa(scope.Season)+".json")
}

// Load reads a saved pass; a missing file is not an error.
func (f SavedFiles) Load(scope Scope) ([]byte, error) {
	body, err := os.ReadFile(f.path(scope))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("league values saved pass: %w", err)
	}
	return body, nil
}

// Save replaces the saved pass atomically.
func (f SavedFiles) Save(scope Scope, body []byte) error {
	if err := os.MkdirAll(f.Dir, 0o700); err != nil {
		return fmt.Errorf("league values saved pass: %w", err)
	}
	tmp, err := os.CreateTemp(f.Dir, ".pass-*")
	if err != nil {
		return fmt.Errorf("league values saved pass: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("league values saved pass: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("league values saved pass: %w", err)
	}
	if err := os.Rename(tmp.Name(), f.path(scope)); err != nil {
		return fmt.Errorf("league values saved pass: %w", err)
	}
	return nil
}

// stableKey is the relaunch-safe key; empty when saving is not wired.
func (src Source) stableKey(ctx context.Context, set params.Set, in Inputs) (string, error) {
	if src.Saved == nil || src.Evidence == nil {
		return "", nil
	}
	evidence, err := src.Evidence(ctx)
	if err != nil {
		return "", fmt.Errorf("league values evidence: %w", err)
	}
	key, err := cacheKey(evidence, set, in)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key[:]), nil
}

func (src Source) loadSaved(stable string, in Inputs) (savedPass, bool) {
	if stable == "" {
		return savedPass{}, false
	}
	body, err := src.Saved.Load(Scope{in.League, in.Season})
	if err != nil || body == nil {
		return savedPass{}, false
	}
	var pass savedPass
	if json.Unmarshal(body, &pass) != nil || pass.Key != stable || in.AsOf.Before(pass.From) ||
		(!pass.Until.IsZero() && !in.AsOf.Before(pass.Until)) {
		return savedPass{}, false
	}
	return pass, true
}

// save keeps only completed passes; a failed valuation is retried on the next launch.
func (src Source) save(stable string, in Inputs, from, until time.Time, out Reading, completed bool) error {
	if stable == "" || !completed {
		return nil
	}
	body, err := json.Marshal(savedPass{Key: stable, From: from, Until: until, Reading: out})
	if err != nil {
		return fmt.Errorf("league values saved pass: %w", err)
	}
	return src.Saved.Save(Scope{in.League, in.Season}, body) //nolint:wrapcheck // Saved names its failures.
}
