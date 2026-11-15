package source

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/parquet-go/parquet-go"
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

// ReadDepthHistory validates every dated row without discarding earlier snapshots.
func ReadDepthHistory(body []byte) ([]RawDepth, error) {
	var rows []RawDepth
	if err := ScanDepthHistory(body, func(row RawDepth) { rows = append(rows, row) }); err != nil {
		return nil, err
	}
	return rows, nil
}

// ScanDepthHistory validates all dated rows in bounded batches. Consumers retain only the
// snapshots they need; the parquet reader must never allocate a season-sized row buffer.
func ScanDepthHistory(body []byte, accept func(RawDepth)) error {
	file, err := parquet.OpenFile(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("open depth parquet: %w", err)
	}
	reader := parquet.NewGenericReader[RawDepth](file)
	defer func() { _ = reader.Close() }()
	batch := make([]RawDepth, 256)
	for {
		n, err := reader.Read(batch)
		for _, row := range batch[:n] {
			if err := row.Validate(); err != nil {
				return err
			}
			accept(row)
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read depth parquet: %w", err)
		}
	}
}

// DepthHistoryURL identifies the archived season membership feed.
func DepthHistoryURL(season int) string {
	return fmt.Sprintf("https://github.com/nflverse/nflverse-data/releases/download/depth_charts/depth_charts_%d.parquet", season)
}
