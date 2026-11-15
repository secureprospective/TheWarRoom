package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/feedwire"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// CheckAPIError rejects an error envelope before a feed is decoded.
func CheckAPIError(body []byte) error {
	if err := feedwire.CheckAPIError(body); err != nil {
		return fmt.Errorf("feed envelope: %w", err)
	}
	return nil
}

// MFLRefusal is archive.Transport's Judge for MFL: an MFL host's HTTP-200 error envelope is a refused fetch,
// named by MFL's own text; any other body or host is data. "No Valid Player IDs" is MFL's answer about the
// players asked for (it holds no profile for one of them), not a refusal of the source: the reader isolates them.
func MFLRefusal(rawURL string, body []byte) string {
	u, err := url.Parse(rawURL)
	if err != nil || !strings.HasSuffix(u.Hostname(), "myfantasyleague.com") {
		return ""
	}
	if err := CheckAPIError(body); err != nil && !UnknownPlayerIDs(err) {
		return err.Error()
	}
	return ""
}

// UnknownPlayerIDs reports MFL's "No Valid Player IDs!" answer to a request naming a player it holds no
// record for.
func UnknownPlayerIDs(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no valid player ids")
}

// MFLList accepts singleton and array feed rows.
type MFLList[T any] = feedwire.MFLList[T]

// LeagueExport fetches one league-scoped MFL export and returns its body. It fails on a transport
// error, a non-200 status, or MFL's HTTP-200 error envelope, so an outage never reads as "no
// data". The caller has already run DiscoverHost.
func LeagueExport(ctx context.Context, c *mfl.Client, export, year, leagueID string, extra map[string]string) ([]byte, error) {
	return leagueExport(ctx, c, mfl.Request{Type: export, Year: year}, leagueID, extra)
}

// KeyedLeagueExport fetches an export MFL serves only to a logged-in league member: the client adds the owner's
// key privately (mfl.Request.Keyed) and needs the league host already discovered.
func KeyedLeagueExport(ctx context.Context, c *mfl.Client, export, year, leagueID string, extra map[string]string) ([]byte, error) {
	return leagueExport(ctx, c, mfl.Request{Type: export, Year: year, Keyed: true}, leagueID, extra)
}

func leagueExport(ctx context.Context, c *mfl.Client, req mfl.Request, leagueID string, extra map[string]string) ([]byte, error) {
	export := req.Type
	req.Params = map[string]string{"L": leagueID}
	for k, v := range extra {
		req.Params[k] = v
	}
	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", export, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: unexpected status %d", export, resp.StatusCode)
	}
	if err := CheckAPIError(resp.Body); err != nil {
		return nil, fmt.Errorf("%s: %w", export, err)
	}
	return resp.Body, nil
}

// FetchLeagueExport discovers the league host, fetches export and decodes it into Env.
func FetchLeagueExport[Env any](ctx context.Context, c *mfl.Client, export, year, leagueID string, extra map[string]string) (Env, error) {
	var env Env
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return env, fmt.Errorf("discover host: %w", err)
	}
	body, err := LeagueExport(ctx, c, export, year, leagueID, extra)
	if err != nil {
		return env, err
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return env, fmt.Errorf("decode %s: %w", export, err)
	}
	return env, nil
}

func SameExport(raw, export string, season int) bool {
	return sameExport(raw, export, season, false)
}

// SameLeaguelessExport matches an archived copy of one of MFL's league-less exports, fetched
// without L (the request the client sends to the api host). A copy carrying the league's own
// id also matches: MFL serves the same NFL-wide feed to every league, and older archives may
// hold either form.
func SameLeaguelessExport(raw, export string, season int) bool {
	return sameExport(raw, export, season, true)
}

func sameExport(raw, export string, season int, leagueless bool) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	q := u.Query()
	if q.Get("TYPE") != export || u.Path != "/"+strconv.Itoa(season)+"/export" {
		return false
	}
	// playerScores can serve a different score season through the current league's host.
	if year := q.Get("YEAR"); export == "playerScores" && year != "" && year != strconv.Itoa(season) {
		return false
	}
	l := q.Get("L")
	return l == LeagueID || (leagueless && l == "")
}
