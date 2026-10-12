// Package manifest pins exactly which archived bodies a compile reads. Each entry names one
// archived MFL response by its body's sha256, so a compile resolves the very bytes it was pinned
// to, never the newest copy of a URL. A manifest is identified by the sha256 of its entries.
package manifest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguearchive"
)

// Entry is one pinned input. FetchID locates the archive row holding it in this database and is
// not part of the manifest's identity; it and CapturedAt are empty when the archive holds no
// body with that sha256.
type Entry struct {
	File       string `json:"file"`
	Export     string `json:"export"`
	League     string `json:"league"`
	Season     int    `json:"season"`
	Week       int    `json:"week"`
	Scope      string `json:"scope"`
	CapturedAt string `json:"capturedAt"`
	SHA256     string `json:"sha256"`
	MFLError   string `json:"mflError,omitempty"`
	FetchID    int64  `json:"fetchId"`
}

// Held reports whether the archive held the pinned body when the manifest was built.
func (e Entry) Held() bool { return e.FetchID != 0 }

// ScopeCurated marks an entry that is a human-curated input, not MFL data: its body ships with the
// app and resolves through Bodies like any other, by its sha256.
const ScopeCurated = "curated"

// Curated reports whether the entry is a curated input rather than an archived MFL response.
func (e Entry) Curated() bool { return e.Scope == ScopeCurated }

// WithCurated adds curated inputs to the manifest, each pinned by its own sha256, and re-derives
// the manifest's identity, so a changed curated file is a changed manifest.
func (m Manifest) WithCurated(files ...Pin) (Manifest, error) {
	out := Manifest{Name: m.Name, Entries: slices.Clone(m.Entries)}
	for _, f := range files {
		if decoded, err := hex.DecodeString(f.SHA256); err != nil || len(decoded) != sha256.Size || f.File == "" {
			return Manifest{}, fmt.Errorf("manifest %s: invalid curated input %q", m.Name, f.File)
		}
		out.Entries = append(out.Entries, Entry{File: f.File, Export: ScopeCurated, Scope: ScopeCurated, SHA256: f.SHA256})
	}
	slices.SortFunc(out.Entries, func(a, b Entry) int { return strings.Compare(a.File, b.File) })
	var err error
	if out.SHA256, err = identity(out.Entries); err != nil {
		return Manifest{}, fmt.Errorf("manifest %s: %w", m.Name, err)
	}
	return out, nil
}

// Manifest is a named, ordered set of pinned inputs.
type Manifest struct {
	Name    string
	Entries []Entry
	SHA256  string
}

// Pin is one input a compile must read: a file of the archive and the bytes it held.
type Pin struct {
	File   string
	SHA256 string
}

// Fetch is one archived response an entry may be pinned to.
type Fetch struct {
	ID        int64
	URL       string
	SHA256    string
	Error     string
	FetchedAt time.Time
}

// Build pins each input to the archived fetch of the same request with the same bytes, captured
// no later than cutoff. A pin no fetch matches stays in the manifest, unheld. rootSeason is the
// season of files kept at the archive root.
func Build(name string, pins []Pin, fetches []Fetch, rootSeason int, cutoff time.Time) (Manifest, error) {
	byBody := map[[2]string][]Fetch{}
	leagues := map[int]string{}
	for _, f := range fetches {
		e, league, err := leaguearchive.ParseURL(f.URL)
		if err != nil {
			return Manifest{}, fmt.Errorf("manifest %s: %w", name, err)
		}
		if league != "" {
			if previous := leagues[e.Season]; previous != "" && previous != league {
				return Manifest{}, fmt.Errorf("manifest %s: season %d has conflicting league ids %s and %s", name, e.Season, previous, league)
			}
			leagues[e.Season] = league
		}
		if !f.FetchedAt.After(cutoff) {
			k := [2]string{e.Key(), f.SHA256}
			byBody[k] = append(byBody[k], f)
		}
	}
	m := Manifest{Name: name, Entries: make([]Entry, 0, len(pins))}
	for _, p := range pins {
		e, err := leaguearchive.ParsePath(p.File, rootSeason)
		if err != nil {
			return Manifest{}, fmt.Errorf("manifest %s: %w", name, err)
		}
		entry := Entry{File: p.File, Export: e.Type, Season: e.Season, Week: e.Week(),
			Scope: string(leaguearchive.ScopeOf(e.Type)), SHA256: p.SHA256}
		if entry.Scope != string(leaguearchive.ScopeSite) {
			entry.League = leagues[e.Season]
		}
		if held := byBody[[2]string{e.Key(), p.SHA256}]; len(held) > 0 {
			first := slices.MinFunc(held, func(a, b Fetch) int { return a.FetchedAt.Compare(b.FetchedAt) })
			entry.FetchID, entry.MFLError = first.ID, first.Error
			entry.CapturedAt = first.FetchedAt.UTC().Format(time.RFC3339)
		}
		m.Entries = append(m.Entries, entry)
	}
	slices.SortFunc(m.Entries, func(a, b Entry) int { return strings.Compare(a.File, b.File) })
	var err error
	if m.SHA256, err = identity(m.Entries); err != nil {
		return Manifest{}, fmt.Errorf("manifest %s: %w", name, err)
	}
	return m, nil
}

// identity is the sha256 of the entries without their row locators.
func identity(entries []Entry) (string, error) {
	plain := slices.Clone(entries)
	for i := range plain {
		plain[i].FetchID = 0
	}
	b, err := json.Marshal(plain)
	if err != nil {
		return "", fmt.Errorf("encode entries: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Encode returns the entries as stored with a published manifest.
func (m Manifest) Encode() ([]byte, error) {
	b, err := json.Marshal(m.Entries)
	if err != nil {
		return nil, fmt.Errorf("manifest %s: encode: %w", m.Name, err)
	}
	return b, nil
}

// Decode reads a published manifest back and checks its sha256.
func Decode(name, sha string, entries []byte) (Manifest, error) {
	m := Manifest{Name: name, SHA256: sha}
	if err := json.Unmarshal(entries, &m.Entries); err != nil {
		return Manifest{}, fmt.Errorf("manifest %s: decode: %w", name, err)
	}
	got, err := identity(m.Entries)
	if err != nil {
		return Manifest{}, fmt.Errorf("manifest %s: %w", name, err)
	}
	if got != sha {
		return Manifest{}, fmt.Errorf("manifest %s: entries hash to %s, not %s", name, got, sha)
	}
	return m, nil
}

// Missing returns the archive entries the archive held no body for. Curated inputs are not
// archive entries.
func (m Manifest) Missing() []Entry {
	var out []Entry
	for _, e := range m.Entries {
		if !e.Held() && !e.Curated() {
			out = append(out, e)
		}
	}
	return out
}

// Bodies returns an archived body by its sha256.
type Bodies interface {
	Body(ctx context.Context, sha256 string) ([]byte, error)
}

// Resolved is one entry and its body. Body is nil for an entry whose body is an MFL error:
// it is pinned, but never read as data. Bytes is the pinned body's size either way.
type Resolved struct {
	Entry Entry
	Body  []byte
	Bytes int64
}

// Resolve reads every entry's pinned body. Every entry whose body is missing or does not match
// its sha256 is named in the error.
func (m Manifest) Resolve(ctx context.Context, bodies Bodies) ([]Resolved, error) {
	out, _, err := m.ResolveWithGaps(ctx, bodies, nil)
	return out, err
}

// Gap authorizes absence of one exact pin, never a replacement body.
type Gap struct {
	File   string
	SHA256 string
	Reason string
}

// ResolveWithGaps excludes only approved unheld pins. Held bytes still have to resolve
// and pass their hash, even if the same pin has an absence exception.
func (m Manifest) ResolveWithGaps(ctx context.Context, bodies Bodies, approved []Gap) ([]Resolved, []Gap, error) {
	allowed, err := gapIndex(approved)
	if err != nil {
		return nil, nil, err
	}
	out := make([]Resolved, 0, len(m.Entries))
	var gaps []Gap
	var errs []error
	for _, e := range m.Entries {
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("manifest %s: resolve: %w", m.Name, err)
		}
		if !e.Held() && !e.Curated() {
			if g, ok := allowed[[2]string{e.File, e.SHA256}]; ok {
				gaps = append(gaps, g)
				continue
			}
			errs = append(errs, fmt.Errorf("manifest %s: %s (sha256 %s): no pinned archive row", m.Name, e.File, e.SHA256))
			continue
		}
		body, err := bodies.Body(ctx, e.SHA256)
		if err == nil {
			sum := sha256.Sum256(body)
			if hex.EncodeToString(sum[:]) != e.SHA256 {
				err = errors.New("body does not match its sha256")
			}
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("manifest %s: %s (sha256 %s): %w", m.Name, e.File, e.SHA256, err))
			continue
		}
		size := int64(len(body))
		if e.MFLError != "" {
			body = nil
		}
		out = append(out, Resolved{Entry: e, Body: body, Bytes: size})
	}
	if len(errs) > 0 {
		return nil, gaps, errors.Join(errs...)
	}
	return out, gaps, nil
}

func gapIndex(approved []Gap) (map[[2]string]Gap, error) {
	out := make(map[[2]string]Gap, len(approved))
	for _, g := range approved {
		decoded, err := hex.DecodeString(g.SHA256)
		if g.File == "" || strings.TrimSpace(g.Reason) == "" || err != nil || len(decoded) != sha256.Size {
			return nil, fmt.Errorf("manifest: invalid gap for %q", g.File)
		}
		key := [2]string{g.File, g.SHA256}
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("manifest: duplicate gap for %q", g.File)
		}
		out[key] = g
	}
	return out, nil
}
