// Package schooltier is the Layer 1 fetcher for the SchoolTier scouting signal — the
// college-competition tier (Power Four / Group of Five / FCS / non-FCS) a school
// belongs to, feeding the Breakout component. It reads the CollegeFootballData (CFBD)
// /teams endpoint and returns a school-name -> tier map.
//
// IT EMITS A FINAL VALUE, NOT RAW (deliberate deviation from its sibling fetchers).
// Every other ingestion fetcher emits raw and lets the engine normalize, because the
// scouting Profile is position-blind and those signals need POSITION-SPECIFIC
// normalization the fetcher structurally cannot do. SchoolTier is different: the tier
// is POSITION-INDEPENDENT (a school is the same tier for a QB or a CB), so there is no
// position-specific step reserved for the engine. The conference->tier classification
// is deterministic vocabulary mapping the source map explicitly assigns to this
// fetcher ("static logic ours"). Hence it imports internal/scouting for the enum and
// emits scouting.SchoolTier directly rather than duplicating the four enum constants.
// (depguard permits this: scouting is a leaf schema package, not an upward layer.)
//
// IT KEYS BY SCHOOL, NOT BY PLAYER. This sidesteps the rookie player-id keying gap
// that complicates CollegeProductionShare: the assembly layer attaches a tier to a
// player via that player's college (school name), so no gsis/MFL crosswalk is needed.
//
// CFBD IS AN AUTHED JSON API, NOT A STATIC CSV — it does NOT use extcsv.go. It needs a
// bearer token and (from CT105) HTTP/1.1 (see NewClient). The minimal CFBD client here
// is intentionally LOCAL to this package for now; when CollegeProductionShare (the 2nd
// CFBD fetcher) lands, extract the shared auth/JSON/HTTP-1.1 client to a shared
// ingestion/cfbd.go then, not before (Codex M17 — extract on the second instance).
//
// ZERO-LEAK (hard constraint): a competition tier carries no fantasy points, projected
// volume, or MFL scoring — a leak is structurally unrepresentable here.
package schooltier

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// TeamsURL is the CFBD teams endpoint. A data-source address, not a secret; Fetch
// takes the base URL as an argument so tests can point at a fixture server.
const TeamsURL = "https://api.collegefootballdata.com/teams"

// pac12PowerThroughYear is the last season the Pac-12 was a genuine power conference.
// From 2024 the conference is a two-team remnant (Oregon St / Washington St) after
// realignment, so a Pac-12 school is Power Four through this year and Group of Five
// after (Christopher's call, 2026-06-21 — CFBD reports the conference as-of the queried
// year, so this rule is applied per-year).
const pac12PowerThroughYear = 2023

// notreDame is the one FBS Independent treated as Power Four; all other independents
// (UConn, UMass, …) are Group of Five (Christopher's call, 2026-06-21).
const notreDame = "Notre Dame"

// errEmpty guards a fetch that classified zero schools — a glitch (wrong URL, an auth
// failure that still returned 200-with-empty, or a CFBD schema change) surfaced loudly
// rather than returned as "no schools".
var errEmpty = errors.New("schooltier: CFBD returned zero classifiable schools")

// team is the slice of a CFBD /teams record this fetcher needs. CFBD returns many more
// fields (logos, location, colors); decoding is LENIENT (no DisallowUnknownFields) per
// the external-3rd-party-boundary policy — only these three are bound.
type team struct {
	School         string `json:"school"`
	Conference     string `json:"conference"`
	Classification string `json:"classification"`
}

// NewClient returns an *http.Client for the CFBD API with HTTP/2 DISABLED.
// api.collegefootballdata.com returns an HTTP/2 PROTOCOL_ERROR from CT105 (verified
// live 2026-06-21: curl --http1.1 succeeds where default h2 fails); pinning the
// transport's TLSNextProto to a non-nil empty map forces HTTP/1.1.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
		},
	}
}

// Fetch retrieves the CFBD teams for a season and returns a school-name -> SchoolTier
// map. client, baseURL, apiKey, and year are injected (no package globals) so the
// fetcher is unit-testable against a fixture server and source-move safe. A school
// whose classification is unrecognized is skipped (lenient external boundary); a non-
// 200, a decode failure, or a zero-school result fails loud.
func Fetch(ctx context.Context, client *http.Client, baseURL, apiKey string, year int) (map[string]scouting.SchoolTier, error) {
	teams, err := getTeams(ctx, client, baseURL, apiKey, year)
	if err != nil {
		return nil, err
	}

	out := make(map[string]scouting.SchoolTier, len(teams))
	for _, t := range teams {
		if tier := classify(year, t.School, t.Conference, t.Classification); tier != scouting.SchoolUnset {
			out[t.School] = tier
		}
	}
	if len(out) == 0 {
		return nil, errEmpty
	}
	return out, nil
}

// getTeams performs the authed GET and decodes the team slice (lenient).
func getTeams(ctx context.Context, client *http.Client, baseURL, apiKey string, year int) ([]team, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?year="+strconv.Itoa(year), nil)
	if err != nil {
		return nil, fmt.Errorf("schooltier: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schooltier: fetch teams: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("schooltier: CFBD /teams unexpected status %d", resp.StatusCode)
	}

	var teams []team
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		return nil, fmt.Errorf("schooltier: decode teams: %w", err)
	}
	return teams, nil
}

// classify maps a CFBD team to its competition tier. An unrecognized classification
// returns SchoolUnset (the caller skips it).
func classify(year int, school, conference, classification string) scouting.SchoolTier {
	switch classification {
	case "fbs":
		return classifyFBS(year, school, conference)
	case "fcs":
		return scouting.SchoolFCS
	case "ii", "iii":
		return scouting.SchoolNonFCS
	default:
		return scouting.SchoolUnset
	}
}

// classifyFBS resolves the within-FBS tier, applying the Pac-12 year rule and the
// Notre Dame independent special-case.
func classifyFBS(year int, school, conference string) scouting.SchoolTier {
	switch conference {
	case "SEC", "Big Ten", "Big 12", "ACC":
		return scouting.SchoolPowerFour
	case "Pac-12":
		if year <= pac12PowerThroughYear {
			return scouting.SchoolPowerFour
		}
		return scouting.SchoolGroupOfFive
	case "FBS Independents":
		if school == notreDame {
			return scouting.SchoolPowerFour
		}
		return scouting.SchoolGroupOfFive
	default:
		return scouting.SchoolGroupOfFive
	}
}
