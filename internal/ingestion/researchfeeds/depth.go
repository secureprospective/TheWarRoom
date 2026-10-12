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

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/contracts"
)

type RawDepth struct {
	DT       string  `parquet:"dt"`
	Team     string  `parquet:"team"`
	Name     string  `parquet:"player_name"`
	ESPN     *string `parquet:"espn_id,optional"`
	GSIS     *string `parquet:"gsis_id,optional"`
	GroupID  int32   `parquet:"pos_grp_id"`
	Group    string  `parquet:"pos_grp"`
	Position string  `parquet:"pos_abb"`
	Slot     int32   `parquet:"pos_slot"`
	Rank     int32   `parquet:"pos_rank"`
}

func (r RawDepth) Validate() error {
	if _, err := time.Parse(time.RFC3339, r.DT); err != nil {
		return fmt.Errorf("depth snapshot: %w", err)
	}
	if r.Team == "" || r.Group == "" || r.Position == "" || r.Slot < 1 || r.Rank < 1 {
		return fmt.Errorf("depth chart: missing identity, formation or rank")
	}
	if r.Name == "" && (r.GSIS == nil || *r.GSIS == "") && (r.ESPN == nil || *r.ESPN == "") {
		return fmt.Errorf("depth chart: no player identity")
	}
	return nil
}
func newestDepth(rows []RawDepth) ([]RawDepth, error) {
	var newest time.Time
	for _, r := range rows {
		if err := r.Validate(); err != nil {
			return nil, err
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
	rows, err := contracts.ReadRows[RawDepth](body)
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
	return fmt.Sprintf("https://github.com/nflverse/nflverse-data/releases/download/depth_charts/depth_charts_%d.parquet", season)
}
func FetchDepth(ctx context.Context, client *http.Client, season int) ([]RawDepth, error) {
	body, err := ingestion.Get(ctx, client, DepthURL(season), 32<<20)
	if err != nil {
		return nil, fmt.Errorf("depth charts: %w", err)
	}
	return ParseDepth(body)
}
