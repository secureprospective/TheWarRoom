// Package madden is the Layer 1 fetcher for the MaddenFilm scouting signal — EA's
// Madden player ratings, which the rubrics use as a cross-position "regulator" of the
// subjective film grades (every blend table seats Madden at a small weight) and, for
// K, as the MAJORITY Layer-4 signal (DECISION-011). It reads the EA ratings-api and
// returns RAW, gsis-keyed records; it does NOT score and does NOT compose the per-
// position MaddenFilm value (the engine's job — Approach A, position-blind Profile).
//
// LOWER-DURABILITY FALLBACK, and STALE. Verified live 2026-06-21: the current-season
// slug (m26) 500s, and the prior slug (m25) returns an empty set — the only populated
// slug is m24 (the 2023-season ratings, two seasons stale). So MaddenFilm is a fallback
// behind FTN-charting (the primary, current veteran-film signal), and its rubric weight
// is a CALIBRATION concern carrying a staleness discount — do not treat it as a fresh
// signal. When EA unblocks a current slug, only RatingsURL changes.
//
// THE KEYING IS BRITTLE AND INJECTED. Madden records carry no gsis/espn id — only a
// name, team, position, and birthdate (and an EA-internal asset id with no public
// crosswalk). The bridge to a TheWarRoom player is therefore a name+birthdate match
// (birthdate disambiguates same-name players), which resolves ~77% of m24 live. To keep
// that brittleness OUT of this fetcher, the resolver is INJECTED (a GSISResolver func):
// this package passes EA's raw name+birthdate to it and trusts the result, so the match
// logic and its format-normalization live in the caller, are swappable, and are unit-
// testable here with a fake. A player who does not resolve is skipped — a clean miss,
// never a guessed attribution.
//
// ZERO-LEAK (hard constraint): Madden ratings are athletic/skill grades (speed, agility,
// awareness, throw accuracy, coverage, …) — no fantasy points, projected volume, or MFL
// scoring. RawMaddenRating holds only the integer rating attributes and identity; a leak
// is unrepresentable.
//
// EA is not CFBD and not MFL — no auth, no host discovery — so it uses neither cfbd.go
// nor mfl.Client. It is the FIRST EA caller, so its capped HTTP GET stays local here;
// extract a shared ingestion/ea.go only at a second EA caller (Codex M17).
package madden

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// RatingsURL is the only populated EA Madden slug (m24; m25 is empty, m26 500s — see
// the package doc). Fetch takes the base URL as an argument so tests point it at a
// fixture server and a future unblocked slug is a one-line change.
const RatingsURL = "https://ratings-api.ea.com/v2/entities/m24-ratings"

// pageLimit is the EA page size. The endpoint caps a page near 1000 records (1992 total
// live), so Fetch pages by offset until it has them all.
const pageLimit = 1000

// maxPageBytes bounds one EA page read (live ~3 MiB) so a runaway source cannot exhaust
// memory at the boundary (Codex M15); over the cap fails loud rather than truncating.
const maxPageBytes = 16 << 20

// ratingSuffix marks the integer attribute fields among EA's ~141 per-record columns;
// the matching "_diff" / identity columns are ignored. overallAttr is pulled out as a
// named field; the rest land in Attributes under their suffix-stripped names.
const (
	ratingSuffix = "_rating"
	overallAttr  = "overall"
)

// GSISResolver maps a Madden player's raw name + birthdate to an nflverse gsis id
// (ok=false on a miss). The caller supplies the match + normalization (e.g. from
// db_playerids name/birthdate columns); injecting it keeps the brittle matching out of
// this fetcher and lets tests pass a fake.
type GSISResolver func(fullName, birthdate string) (gsis string, ok bool)

// errEmpty guards a fetch that resolved zero gsis-keyed records — the populated slug
// carries ~1992 players and the resolver matches most, so an empty result means a
// truncated download, a dead slug, or a broken resolver, surfaced loudly.
var errEmpty = errors.New("madden: resolved zero gsis-keyed records")

// page is the EA ratings-api response envelope. docs are decoded as raw messages so
// each record can be read twice — once for its typed identity, once for its dynamic
// "_rating" attribute set — without an interface{} sink.
type page struct {
	Count int               `json:"count"`
	Docs  []json.RawMessage `json:"docs"`
}

// identity is the typed slice of a Madden record this fetcher names explicitly. The
// remaining "_rating" attributes are read dynamically into RawMaddenRating.Attributes.
type identity struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Team      string `json:"team"`
	Position  string `json:"position"`
	Birthdate string `json:"plyrBirthdate"`
}

// RawMaddenRating is one player's raw Madden ratings, keyed by the gsis the resolver
// maps the player to. Overall is the headline rating; Attributes holds every other
// integer "_rating" sub-attribute under its suffix-stripped name (e.g. "speed",
// "manCoverage") for the engine to compose per position. No field references fantasy
// points or MFL scoring (zero-leak). FullName/Birthdate are carried for traceability.
type RawMaddenRating struct {
	GSISID     string
	FullName   string
	Team       string
	Position   string
	Birthdate  string
	Overall    int
	Attributes map[string]int
}

// Fetch pages the EA ratings slug at baseURL, resolves each player to a gsis via
// resolve, and returns the gsis-keyed raw ratings. A player who does not resolve is
// skipped (clean miss); two players resolving to one gsis drop that gsis (ambiguous).
// A malformed rating value fails loud.
func Fetch(ctx context.Context, client *http.Client, baseURL string, resolve GSISResolver) (map[string]RawMaddenRating, error) {
	docs, err := fetchAll(ctx, client, baseURL)
	if err != nil {
		return nil, err
	}

	out := map[string]RawMaddenRating{}
	poisoned := map[string]bool{}
	for _, raw := range docs {
		rec, err := parseDoc(raw)
		if err != nil {
			return nil, err
		}
		gsis, ok := resolve(rec.FullName, rec.Birthdate)
		if !ok || poisoned[gsis] {
			continue
		}
		if _, dup := out[gsis]; dup {
			delete(out, gsis)
			poisoned[gsis] = true
			continue
		}
		rec.GSISID = gsis
		out[gsis] = rec
	}

	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}

// fetchAll pages the slug by offset until every record (per the response count) is
// collected, guarding against a non-advancing page so a misbehaving endpoint cannot
// loop forever.
func fetchAll(ctx context.Context, client *http.Client, baseURL string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	for offset := 0; ; {
		p, err := fetchPage(ctx, client, baseURL, offset)
		if err != nil {
			return nil, err
		}
		if len(p.Docs) == 0 {
			break
		}
		all = append(all, p.Docs...)
		offset += len(p.Docs)
		if offset >= p.Count {
			break
		}
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("madden: slug returned zero records (dead or empty slug)")
	}
	return all, nil
}

// fetchPage GETs one page (limit+offset), status-checked and byte-capped, and decodes
// the envelope. EA needs no auth header.
func fetchPage(ctx context.Context, client *http.Client, baseURL string, offset int) (page, error) {
	url := baseURL + "?limit=" + strconv.Itoa(pageLimit) + "&offset=" + strconv.Itoa(offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return page{}, fmt.Errorf("madden: build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return page{}, fmt.Errorf("madden: fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return page{}, fmt.Errorf("madden: %s unexpected status %d", url, resp.StatusCode)
	}

	lr := &io.LimitedReader{R: resp.Body, N: maxPageBytes + 1}
	body, err := io.ReadAll(lr)
	if err != nil {
		return page{}, fmt.Errorf("madden: read %s: %w", url, err)
	}
	if lr.N == 0 {
		return page{}, fmt.Errorf("madden: page %s exceeds %d-byte cap", url, maxPageBytes)
	}

	var p page
	if err := json.Unmarshal(body, &p); err != nil {
		return page{}, fmt.Errorf("madden: decode page: %w", err)
	}
	return p, nil
}

// parseDoc reads one record's typed identity and its dynamic "_rating" attribute set
// into a RawMaddenRating (GSISID left for the caller to fill on resolve). A present-but-
// unparseable rating fails loud (corruption, not zero).
func parseDoc(raw json.RawMessage) (RawMaddenRating, error) {
	var id identity
	if err := json.Unmarshal(raw, &id); err != nil {
		return RawMaddenRating{}, fmt.Errorf("madden: decode identity: %w", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return RawMaddenRating{}, fmt.Errorf("madden: decode fields: %w", err)
	}

	rec := RawMaddenRating{
		FullName:   strings.TrimSpace(id.FirstName + " " + id.LastName),
		Team:       id.Team,
		Position:   id.Position,
		Birthdate:  id.Birthdate,
		Attributes: map[string]int{},
	}
	if err := fillRatings(&rec, fields); err != nil {
		return RawMaddenRating{}, err
	}
	return rec, nil
}

// fillRatings pulls every integer "_rating" field into rec: "overall" becomes Overall,
// the rest go into Attributes under their suffix-stripped names.
func fillRatings(rec *RawMaddenRating, fields map[string]json.RawMessage) error {
	for key, val := range fields {
		name, isRating := strings.CutSuffix(key, ratingSuffix)
		if !isRating {
			continue
		}
		// EA encodes the numeric (athletic) ratings as JSON numbers and the descriptive
		// ones as JSON strings (e.g. runningStyle_rating = "Default Stride Awkward").
		// Skip the string-valued ratings — they are not engine attributes — while a
		// non-integer NUMBER (real corruption) still fails loud below.
		if len(val) > 0 && val[0] == '"' {
			continue
		}
		// A JSON null rating is neither a string nor a number, so it fell through
		// this guard into json.Unmarshal(null, &n), which succeeds with n=0 and no
		// error (Go spec) — a silent zero indistinguishable from a real 0 rating.
		// Madden is K's 0.60-majority Layer-4 signal (DECISION-011), so an unnoticed
		// null would corrupt a whole position (GLM whole-repo audit 2026-06-28,
		// Pass C MAJOR-1, MED confidence pending a live-response null check).
		if string(val) == "null" {
			continue
		}
		var n int
		if err := json.Unmarshal(val, &n); err != nil {
			return fmt.Errorf("madden: rating %q value %s: %w", key, val, err)
		}
		if name == overallAttr {
			rec.Overall = n
			continue
		}
		rec.Attributes[name] = n
	}
	return nil
}
