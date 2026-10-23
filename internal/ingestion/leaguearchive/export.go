// Package leaguearchive imports the league archive: every MFL export saved as a file, laid out
// as <season>/<export>[_<param>].json, <season>/weekly/<export>_W<nn>.json and
// <season>/messageBoardThread/<id>.json, plus allRules.json at the root. Each file is recorded
// in the fetch archive as the MFL request it answered, so the archive and the app's own fetches
// share one sha256-provenanced store.
package leaguearchive

import (
	"fmt"
	"maps"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/archive"
)

const (
	archiveLayoutMessage = "An archive file has an unexpected name or folder. Use the original archive names and keep weekly files in each season’s weekly folder."
)

// Scope says whose view of MFL a body is.
type Scope string

const (
	ScopeLeague Scope = "league" // the league's public exports
	ScopeOwner  Scope = "owner"  // exports MFL answers only for a logged-in owner
	ScopeSite   Scope = "site"   // MFL-wide or NFL exports, served from MFL's shared host
)

// ownerExports are the exports MFL refused in the archive without a login ("API requires logged
// in user"), and the message board's threads.
func ownerExports() []string {
	return []string{"assets", "calendar", "messageBoard", "messageBoardThread", "polls"}
}

// siteExports describe all of MFL or the NFL rather than one league, so they carry no league id.
func siteExports() []string {
	return []string{"injuries", "nflByeWeeks", "nflSchedule", "adp", "aav", "topAdds", "topDrops",
		"topStarters", "topOwns", "topTrades", "allRules"}
}

// ScopeOf returns the scope of an export.
func ScopeOf(export string) Scope {
	switch {
	case slices.Contains(siteExports(), export):
		return ScopeSite
	case slices.Contains(ownerExports(), export):
		return ScopeOwner
	}
	return ScopeLeague
}

// suffixParam is the query parameter a file name's suffix carries, by export. The suffix is the
// parameter's name and value run together (WYTD), except for a playoff bracket's bare id.
func suffixParam(export string) (key string, bare bool, ok bool) {
	switch export {
	case "players":
		return "DETAILS", false, true
	case "pool":
		return "POOLTYPE", false, true
	case "playoffBracket":
		return "BRACKET_ID", true, true
	case "playerScores", "weeklyResults", "rosters", "projectedScores", "injuries", "nflSchedule":
		return "W", false, true
	}
	return "", false, false
}

// fixedParams are the parameters an export was always requested with.
func fixedParams(export string) map[string]string {
	if export == "transactions" {
		return map[string]string{"TRANS_TYPE": "*"}
	}
	return nil
}

// Export is one MFL request: an export type for a season, with its extra parameters (never
// TYPE, L or JSON).
type Export struct {
	Season int
	Type   string
	Params map[string]string
}

// Week is the export's week, or 0 when it is not a weekly one.
func (e Export) Week() int {
	w, err := strconv.Atoi(e.Params["W"])
	if err != nil {
		return 0
	}
	return w
}

// Key identifies the request whatever host or league id it was sent with.
func (e Export) Key() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d/%s", e.Season, e.Type)
	for _, k := range slices.Sorted(maps.Keys(e.Params)) {
		fmt.Fprintf(&b, "&%s=%s", k, e.Params[k])
	}
	return b.String()
}

// ParsePath reads the request a file stands for from its path under the archive root.
// rootSeason is the season of files kept at the root.
func ParsePath(rel string, rootSeason int) (Export, error) {
	parts := strings.Split(rel, "/")
	name, ok := strings.CutSuffix(parts[len(parts)-1], ".json")
	if !ok || name == "" {
		return Export{}, archive.WithMessage(fmt.Errorf("leaguearchive: %s is not a .json export", rel), archiveLayoutMessage)
	}
	if len(parts) == 1 {
		return Export{Season: rootSeason, Type: name, Params: map[string]string{}}, nil
	}
	season, err := strconv.Atoi(parts[0])
	if err != nil || season < 1900 || len(parts) > 3 {
		return Export{}, archive.WithMessage(fmt.Errorf("leaguearchive: %s is not under a season folder", rel), archiveLayoutMessage)
	}
	if len(parts) == 3 && parts[1] == "messageBoardThread" {
		if _, err := strconv.Atoi(name); err != nil {
			return Export{}, archive.WithMessage(fmt.Errorf("leaguearchive: %s is not a thread id", rel), archiveLayoutMessage)
		}
		return Export{Season: season, Type: "messageBoardThread", Params: map[string]string{"THREAD": name}}, nil
	}
	e, err := parseName(season, name)
	if err != nil {
		return Export{}, archive.WithMessage(fmt.Errorf("leaguearchive: %s: %w", rel, err), archiveLayoutMessage)
	}
	weekly := len(parts) == 3 && parts[1] == "weekly"
	if len(parts) == 3 && !weekly {
		return Export{}, archive.WithMessage(fmt.Errorf("leaguearchive: %s is in an unknown folder", rel), archiveLayoutMessage)
	}
	if weekly != (e.Week() > 0) {
		return Export{}, archive.WithMessage(fmt.Errorf("leaguearchive: %s: a weekly file must name its week, and only a weekly file may", rel), archiveLayoutMessage)
	}
	return e, nil
}

func parseName(season int, name string) (Export, error) {
	typ, suffix, hasSuffix := strings.Cut(name, "_")
	e := Export{Season: season, Type: typ, Params: map[string]string{}}
	maps.Copy(e.Params, fixedParams(typ))
	if !hasSuffix {
		return e, nil
	}
	key, bare, ok := suffixParam(typ)
	if !ok {
		return Export{}, archive.WithMessage(fmt.Errorf("export %s takes no parameter", typ), archiveLayoutMessage)
	}
	value := suffix
	if !bare {
		if value, ok = strings.CutPrefix(suffix, key); !ok || value == "" {
			return Export{}, archive.WithMessage(fmt.Errorf("suffix %q does not carry %s", suffix, key), archiveLayoutMessage)
		}
	}
	if n, err := strconv.Atoi(value); err == nil {
		value = strconv.Itoa(n)
	}
	e.Params[key] = value
	return e, nil
}

// League is one season's league on MFL: its id and the host it was served from.
type League struct {
	ID   string
	Host string
}

// URL is the MFL request e stands for, sent to lg's host, or to MFL's shared host for a site
// export.
func (e Export) URL(lg League) string {
	q := url.Values{"TYPE": {e.Type}, "JSON": {"1"}}
	host := lg.Host
	if ScopeOf(e.Type) == ScopeSite {
		host = "api." + parentDomain(lg.Host)
	} else {
		q.Set("L", lg.ID)
	}
	for k, v := range e.Params {
		q.Set(k, v)
	}
	u := url.URL{Scheme: "https", Host: host, Path: path.Join("/", strconv.Itoa(e.Season), "export"), RawQuery: q.Encode()}
	return u.String()
}

func parentDomain(host string) string {
	if _, rest, ok := strings.Cut(host, "."); ok && strings.Contains(rest, ".") {
		return rest
	}
	return host
}

// ParseURL reads an archived MFL request back into the export it asked for and the league id
// it named (empty for a site export).
func ParseURL(raw string) (Export, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Export{}, "", archive.WithMessage(fmt.Errorf("leaguearchive: url %q: %w", raw, err), "A saved MFL address is not valid. Read the data again to replace it.")
	}
	seasonText, ok := strings.CutSuffix(strings.TrimPrefix(u.Path, "/"), "/export")
	season, err := strconv.Atoi(seasonText)
	q := u.Query()
	if !ok || err != nil || q.Get("TYPE") == "" {
		return Export{}, "", archive.WithMessage(fmt.Errorf("leaguearchive: %q is not an MFL export request", raw), "A saved MFL address is not valid. Read the data again to replace it.")
	}
	e := Export{Season: season, Type: q.Get("TYPE"), Params: map[string]string{}}
	for k := range q {
		if k != "TYPE" && k != "L" && k != "JSON" {
			e.Params[k] = q.Get(k)
		}
	}
	return e, q.Get("L"), nil
}
