package history

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

// OfficialSeason is one season of nflverse's official files, decompressed: weekly player statistics
// (stats_player_week) and play-by-play. At is when they were read; GSIS is the player directory's NFL id
// for each MFL id (Store.SourceIDs of the file holding the directory). Roster, nflverse's player list
// (players.csv), identifies a player the directory gives no NFL id; nil skips that.
type OfficialSeason struct {
	Weekly, Plays, Roster io.Reader
	At                    time.Time
	GSIS                  map[string]string
}

// OfficialPin is what PinOfficialStats did: the new evidence's digest (empty when nothing changed), the
// weeks it holds official statistics for, and each week it left out and why.
type OfficialPin struct {
	Digest  string
	Weeks   []int
	Skipped []string
}

// PinOfficialStats builds football evidence that is current's plus nflverse's official statistics for every
// finished regular-season week, imports its week parts and itself into s, and returns its digest. A week is
// finished when every game current's schedule lists for it has ended in the play-by-play and appears in
// the weekly statistics, and officialGrace has passed since its last kickoff. Each pin adds one statistic
// revision holding only the weeks that are new or changed (a stat correction) and keeps every earlier one,
// so a version the commissioner approved stays held. Players current names by MFL id alone take an NFL
// identity (officialIdentities). When there is no new week, correction or identity, nothing is written.
func (s *Store) PinOfficialStats(ctx context.Context, current *PinnedFootball, in OfficialSeason) (OfficialPin, error) {
	if current == nil || in.Weekly == nil || in.Plays == nil || in.At.IsZero() {
		return OfficialPin{}, fmt.Errorf("official statistics need pinned football, both season files and a read time")
	}
	next, change, out, err := current.raw.withOfficial(in)
	if err != nil || len(change.parts) == 0 && (change.linked == 0 || len(current.raw.Stats) == 0) {
		return out, err
	}
	for i, body := range change.bodies {
		if err := s.ImportFootballWeek(ctx, body, change.parts[i]); err != nil {
			return OfficialPin{}, err
		}
	}
	body, err := json.Marshal(next)
	if err != nil {
		return OfficialPin{}, fmt.Errorf("encode football evidence: %w", err)
	}
	out.Digest = footballDigest(body)
	if err := s.ImportFootball(ctx, body, out.Digest); err != nil {
		return OfficialPin{}, err
	}
	return out, nil
}

// officialChange is what new evidence adds: the week parts no revision holds yet, and how many players
// took an NFL identity.
type officialChange struct {
	parts  []string
	bodies [][]byte
	linked int
}

// withOfficial is r with official weeks: its players identified and, when a week part is new or changed,
// one new revision holding just those parts in front of r's.
func (r RawFootballEvidence) withOfficial(in OfficialSeason) (RawFootballEvidence, officialChange, OfficialPin, error) {
	var change officialChange
	weeks, err := splitOfficial(r.Season, in)
	if err != nil {
		return r, change, OfficialPin{}, err
	}
	roster, err := readRoster(in.Roster, r.Season)
	if err != nil {
		return r, change, OfficialPin{}, err
	}
	next := r
	next.Players, change.linked = officialIdentities(r.Players, in.GSIS, roster)
	parts, bodies, out := next.officialParts(weeks, in.At)
	change.parts, change.bodies = unheldParts(r.Stats, parts, bodies)
	at := in.At.UTC()
	if next.AsOf = at; at.Before(r.AsOf) {
		next.AsOf = r.AsOf
	}
	if !strings.Contains(next.Source, officialSource) {
		next.Source += "; " + officialSource
	}
	if len(change.parts) == 0 {
		return next, change, out, nil
	}
	if len(r.Stats) != 0 && !at.After(r.Stats[0].AsOf) || len(r.Stats) >= maxRevisions {
		return r, change, OfficialPin{}, fmt.Errorf("official statistics read at %s cannot follow the pin's %d revisions", at, len(r.Stats))
	}
	next.Stats = append([]RawFootballStats{{AsOf: at, Parts: change.parts}}, r.Stats...)
	return next, change, out, nil
}

// unheldParts keeps the parts (and their bodies) no revision lists yet.
func unheldParts(revisions []RawFootballStats, parts []string, bodies [][]byte) ([]string, [][]byte) {
	held := map[string]bool{}
	for _, revision := range revisions {
		for _, part := range revision.Parts {
			held[part] = true
		}
	}
	var keep []string
	var keepBodies [][]byte
	for i, part := range parts {
		if !held[part] {
			keep, keepBodies = append(keep, part), append(keepBodies, bodies[i])
		}
	}
	return keep, keepBodies
}

const (
	officialSource = "official nflverse statistics"
	// officialGrace is how long after a week's last kickoff its statistics may be taken as official.
	officialGrace = 6 * time.Hour
	maxRevisions  = 100 // scoringFiles' limit
)

// officialParts makes one week part per finished week and checks each the way a pin load reads it.
func (r RawFootballEvidence) officialParts(weeks map[int]*officialWeek, at time.Time) ([]string, [][]byte, OfficialPin) {
	var out OfficialPin
	var parts []string
	var bodies [][]byte
	for _, week := range r.Weeks {
		held := weeks[week.Number]
		if held == nil {
			continue
		}
		if why := held.unfinished(week, at); why != "" {
			out.Skipped = append(out.Skipped, fmt.Sprintf("week %d: %s", week.Number, why))
			continue
		}
		body, digest, err := r.officialPart(week.Number, held)
		if err != nil {
			out.Skipped = append(out.Skipped, fmt.Sprintf("week %d: %v", week.Number, err))
			continue
		}
		parts, bodies = append(parts, digest), append(bodies, body)
		out.Weeks = append(out.Weeks, week.Number)
	}
	return parts, bodies, out
}

func (r RawFootballEvidence) officialPart(number int, held *officialWeek) ([]byte, string, error) {
	held.weeklyCSV.Flush()
	held.playsCSV.Flush()
	body, err := json.Marshal(RawFootballWeek{Version: 1, Season: r.Season, Week: number,
		Weekly: held.weekly.String(), Plays: held.plays.String()})
	if err != nil {
		return nil, "", fmt.Errorf("encode the week: %w", err)
	}
	digest := footballDigest(body)
	if _, err := decodeFootballWeek(body); err != nil {
		return nil, "", err
	}
	trial := func(string) ([]byte, error) { return body, nil }
	if _, err := r.scoringFile(RawFootballStats{AsOf: r.AsOf, Parts: []string{digest}}, trial); err != nil {
		return nil, "", err
	}
	return body, digest, nil
}

// officialIdentities gives every player held by MFL id alone an NFL identity, and counts them: the one the
// player directory links him to, or, when it links none (or only an old pre-GSIS id), the one player of
// nflverse's list with his name and position group. An identity two players would take, or one another
// player already holds, is given to none.
func officialIdentities(held map[string]RawFootballPlayer, links map[string]string, roster nflRoster) (map[string]RawFootballPlayer, int) {
	found := map[string]string{}
	claims := map[string]int{}
	for id, player := range held {
		if !strings.HasPrefix(player.Identity, MFLIdentityPrefix) {
			claims[player.Identity] += 2
			continue
		}
		gsis := links[id]
		if !source.IsGSIS(gsis) {
			gsis = roster.match(player.Facts.Name, player.Facts.Position)
		}
		if gsis != "" {
			found[id] = gsis
			claims[gsis]++
		}
	}
	out := maps.Clone(held)
	linked := 0
	for id, player := range held {
		gsis := found[id]
		if gsis != "" && claims[gsis] == 1 {
			player.Identity = gsis
			out[id] = player
			linked++
		}
	}
	return out, linked
}

// officialWeek is one regular-season week of the season files: its rows re-written as their own CSVs, and
// the games the play-by-play holds, has seen end, and the weekly statistics report.
type officialWeek struct {
	weekly, plays             bytes.Buffer
	weeklyCSV, playsCSV       *csv.Writer
	games, ended, reported    map[string]bool
	weeklyHeader, playsHeader bool
}

func (w *officialWeek) unfinished(week leagueweek.Week, at time.Time) string {
	scheduled, last := len(week.Games), time.Time{}
	for _, game := range week.Games {
		if game.Kickoff.After(last) {
			last = game.Kickoff
		}
	}
	switch {
	case at.Before(last.Add(officialGrace)):
		return "its last game kicked off less than " + officialGrace.String() + " ago"
	case scheduled == 0 || len(w.games) < scheduled:
		return fmt.Sprintf("%d of %d games are in the play-by-play", len(w.games), scheduled)
	case len(w.ended) < len(w.games):
		return fmt.Sprintf("%d of %d games have ended", len(w.ended), len(w.games))
	case !w.weeklyHeader:
		return "the weekly statistics do not report it yet"
	}
	for game := range w.games {
		if !w.reported[game] {
			return "the weekly statistics do not report game " + game + " yet"
		}
	}
	return ""
}

// splitOfficial reads both season files once, keeping only the season's regular-season rows and the
// play-by-play columns scoring reads.
func splitOfficial(season int, in OfficialSeason) (map[int]*officialWeek, error) {
	weeks := map[int]*officialWeek{}
	week := func(n int) *officialWeek {
		if weeks[n] == nil {
			w := &officialWeek{games: map[string]bool{}, ended: map[string]bool{}, reported: map[string]bool{}}
			w.weeklyCSV, w.playsCSV = csv.NewWriter(&w.weekly), csv.NewWriter(&w.plays)
			weeks[n] = w
		}
		return weeks[n]
	}
	err := eachRegular(in.Weekly, season, nil, func(n int, header, row []string, at map[string]int) error {
		w := week(n)
		if !w.weeklyHeader {
			w.weeklyHeader = true
			if err := w.weeklyCSV.Write(header); err != nil {
				return err //nolint:wrapcheck // wrapped once by eachRegular's caller
			}
		}
		w.reported[row[at["game_id"]]] = true
		return w.weeklyCSV.Write(row)
	})
	if err != nil {
		return nil, fmt.Errorf("read official weekly statistics: %w", err)
	}
	err = eachRegular(in.Plays, season, source.PlayColumn, func(n int, header, row []string, at map[string]int) error {
		w := week(n)
		if !w.playsHeader {
			w.playsHeader = true
			if err := w.playsCSV.Write(header); err != nil {
				return err //nolint:wrapcheck // wrapped once below
			}
		}
		game := row[at["game_id"]]
		w.games[game] = true
		if row[at["desc"]] == "END GAME" {
			w.ended[game] = true
		}
		return w.playsCSV.Write(row)
	})
	if err != nil {
		return nil, fmt.Errorf("read official play-by-play: %w", err)
	}
	return weeks, nil
}

// eachRegular calls row for every regular-season row of season in a CSV, with only the columns keep
// allows (all when nil). header and at describe the kept columns.
func eachRegular(input io.Reader, season int, keep func(string) bool,
	row func(week int, header, cells []string, at map[string]int) error) error {
	reader := csv.NewReader(input)
	full, err := reader.Read()
	if err != nil {
		return fmt.Errorf("header: %w", err)
	}
	kept, header, at, err := keptColumns(full, keep)
	if err != nil {
		return err
	}
	wanted := strconv.Itoa(season)
	for {
		cells, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("row: %w", err)
		}
		out := make([]string, len(kept))
		for i, col := range kept {
			out[i] = cells[col]
		}
		if out[at["season"]] != wanted || out[at["season_type"]] != "REG" {
			continue
		}
		n, err := strconv.Atoi(out[at["week"]])
		if err != nil || n < 1 || n > 22 {
			return fmt.Errorf("a row names week %q", out[at["week"]])
		}
		if err := row(n, header, out, at); err != nil {
			return err
		}
	}
}

// keptColumns is the indexes and names of the columns keep allows (all when nil), and each kept name's
// place; the columns every row is sorted by must be among them.
func keptColumns(full []string, keep func(string) bool) ([]int, []string, map[string]int, error) {
	var kept []int
	var header []string
	at := map[string]int{}
	for i, name := range full {
		if keep == nil || keep(name) {
			at[name] = len(header)
			kept, header = append(kept, i), append(header, name)
		}
	}
	for _, name := range []string{"season", "week", "season_type", "game_id"} {
		if _, ok := at[name]; !ok {
			return nil, nil, nil, fmt.Errorf("the file has no %s column", name)
		}
	}
	return kept, header, at, nil
}
