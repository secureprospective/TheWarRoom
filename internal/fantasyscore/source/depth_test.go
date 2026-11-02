package source

import (
	"bytes"
	"testing"

	"github.com/parquet-go/parquet-go"
)

func TestDepthHistoryKeepsDatedMembershipAndRejectsInvalidRows(t *testing.T) {
	gsis := "00-0030001"
	rows := []RawDepth{
		{DT: "2026-10-07T12:00:00Z", Team: "NE", GSIS: &gsis, Group: "OFF", Position: "WR", Slot: 1, Rank: 1},
		{DT: "2026-10-14T12:00:00Z", Team: "NYJ", GSIS: &gsis, Group: "OFF", Position: "WR", Slot: 1, Rank: 1},
	}
	encode := func() []byte {
		t.Helper()
		var body bytes.Buffer
		if err := parquet.Write(&body, rows); err != nil {
			t.Fatal(err)
		}
		return body.Bytes()
	}
	got, err := ReadDepthHistory(encode())
	if err != nil || len(got) != 2 || got[0].Team != "NE" || got[1].Team != "NYJ" {
		t.Fatalf("dated membership lost: %+v %v", got, err)
	}
	rows[0].Rank = 0
	if got, err := ReadDepthHistory(encode()); err == nil || got != nil {
		t.Fatalf("invalid older row admitted: %+v %v", got, err)
	}
	if _, err := ReadDepthHistory([]byte("not parquet")); err == nil {
		t.Fatal("malformed parquet accepted")
	}
}
