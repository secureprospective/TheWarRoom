package contracts

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/parquet-go/parquet-go"
)

// fileRow is a contract as the file holds it, with a column the reader does not decode.
type fileRow struct {
	Player     string   `parquet:"player"`
	GSIS       *string  `parquet:"gsis_id,optional"`
	YearSigned *int32   `parquet:"year_signed,optional"`
	Years      *int32   `parquet:"years,optional"`
	Value      *float64 `parquet:"value,optional"`
	Guaranteed *float64 `parquet:"guaranteed,optional"`
	CapPct     *float64 `parquet:"apy_cap_pct,optional"`
}

func ptr[T any](v T) *T { return &v }

func TestMapKeepsEachSeasonsLargerContract(t *testing.T) {
	var buf bytes.Buffer
	err := parquet.Write(&buf, []fileRow{
		{Player: "Burrow", GSIS: ptr("00-0036442"), YearSigned: ptr[int32](2020), Years: ptr[int32](4), Value: ptr(36.19),
			Guaranteed: ptr(36.19), CapPct: ptr(0.045)},
		{Player: "Burrow", GSIS: ptr("00-0036442"), YearSigned: ptr[int32](2023), Years: ptr[int32](5), Value: ptr(275.0),
			Guaranteed: ptr(146.51), CapPct: ptr(0.245)},
		{Player: "Burrow", GSIS: ptr("00-0036442"), YearSigned: ptr[int32](2023), Years: ptr[int32](1), Value: ptr(1.0)},
		{Player: "No id", YearSigned: ptr[int32](2023), Years: ptr[int32](1), Value: ptr(1.0)},
		{Player: "Unknown signing", GSIS: ptr("00-0036442"), YearSigned: ptr[int32](0), Years: ptr[int32](3), Value: ptr(9.0)},
		{Player: "Not in the directory", GSIS: ptr("00-0099999"), YearSigned: ptr[int32](2023), Years: ptr[int32](2)},
		{Player: "No cap share", GSIS: ptr("00-0011111"), YearSigned: ptr[int32](2024), Years: ptr[int32](2), Value: ptr(2.0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	keep := func(id string) bool { return id == "00-0036442" || id == "00-0011111" }
	b, err := Map(buf.Bytes(), keep)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range b.Facts {
		if f.IDType != IDType {
			t.Errorf("fact %+v carries id type %q", f, f.IDType)
		}
		got = append(got, f.ID+" "+strconv.Itoa(f.Season)+" "+f.Field+"="+f.Raw)
	}
	want := []string{
		"00-0011111 2024 contracts.value=2",
		"00-0011111 2024 contracts.years=2",
		"00-0036442 2020 contracts.apy_cap_pct=0.045",
		"00-0036442 2020 contracts.guaranteed=36.19",
		"00-0036442 2020 contracts.value=36.19",
		"00-0036442 2020 contracts.years=4",
		"00-0036442 2023 contracts.apy_cap_pct=0.245",
		"00-0036442 2023 contracts.guaranteed=146.51",
		"00-0036442 2023 contracts.value=275",
		"00-0036442 2023 contracts.years=5",
	}
	slices.Sort(got)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("facts:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if b.Source != Source || len(b.BodySHA256) != 64 {
		t.Errorf("source %q sha %q", b.Source, b.BodySHA256)
	}
}

func TestMapRejectsAFileThatIsNotParquet(t *testing.T) {
	if _, err := Map([]byte("player,gsis_id\n"), func(string) bool { return true }); err == nil {
		t.Fatal("a CSV body was read as contracts")
	}
}
