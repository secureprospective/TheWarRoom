package researchfeeds

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

type RawDepth = source.RawDepth

func newestDepth(rows []RawDepth) ([]RawDepth, error) {
	var newest time.Time
	for _, r := range rows {
		if err := r.Validate(); err != nil {
			return nil, fmt.Errorf("depth row: %w", err)
		}
		at, err := time.Parse(time.RFC3339, r.DT)
		if err != nil {
			return nil, fmt.Errorf("depth date: %w", err)
		}
		if at.After(newest) {
			newest = at
		}
	}
	out := []RawDepth{}
	for _, r := range rows {
		at, err := time.Parse(time.RFC3339, r.DT)
		if err != nil {
			return nil, fmt.Errorf("depth date: %w", err)
		}
		if at.Equal(newest) {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Team != b.Team {
			return a.Team < b.Team
		}
		if a.GroupID != b.GroupID {
			return a.GroupID < b.GroupID
		}
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		return a.Rank < b.Rank
	})
	return out, nil
}
func ParseDepth(body []byte) ([]RawDepth, error) {
	var rows []RawDepth
	var newest time.Time
	err := source.ScanDepthHistory(body, func(row RawDepth) {
		at, _ := time.Parse(time.RFC3339, row.DT) // The scanner validated the timestamp.
		if at.After(newest) {
			newest, rows = at, rows[:0]
		}
		if at.Equal(newest) {
			rows = append(rows, row)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("depth charts: %w", err)
	}
	return newestDepth(rows)
}
func ParseDepthCSV(body []byte) ([]RawDepth, error) {
	reader := csv.NewReader(strings.NewReader(string(body)))
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("depth header: %w", err)
	}
	indexes := map[string]int{}
	for i, h := range header {
		indexes[h] = i
	}
	for _, h := range []string{"dt", "team", "player_name", "espn_id", "gsis_id", "pos_grp_id", "pos_grp", "pos_abb", "pos_slot", "pos_rank"} {
		if _, ok := indexes[h]; !ok {
			return nil, fmt.Errorf("depth: missing %s", h)
		}
	}
	var rows []RawDepth
	for {
		cells, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("depth row: %w", err)
		}
		get := func(key string) string { return cells[indexes[key]] }
		group, ge := strconv.ParseInt(get("pos_grp_id"), 10, 32)
		slot, se := strconv.ParseInt(get("pos_slot"), 10, 32)
		rank, re := strconv.ParseInt(get("pos_rank"), 10, 32)
		if ge != nil || se != nil || re != nil {
			return nil, fmt.Errorf("depth: invalid numeric formation or rank")
		}
		espn, gsis := get("espn_id"), get("gsis_id")
		rows = append(rows, RawDepth{DT: get("dt"), Team: get("team"), Name: get("player_name"), ESPN: &espn, GSIS: &gsis, GroupID: int32(group), Group: get("pos_grp"), Position: get("pos_abb"), Slot: int32(slot), Rank: int32(rank)})
	}
	return newestDepth(rows)
}
func DepthURL(season int) string {
	return source.DepthHistoryURL(season)
}
func FetchDepth(ctx context.Context, client *http.Client, season int) ([]RawDepth, error) {
	body, err := ingestion.Get(ctx, client, DepthURL(season), 32<<20)
	if err != nil {
		return nil, fmt.Errorf("depth charts: %w", err)
	}
	return ParseDepth(body)
}
