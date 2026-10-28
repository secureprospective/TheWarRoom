package composition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
)

// TestReportWireTwinsMatch proves wireBoard and wireChoices lose nothing: a board with every
// field set, and one left empty, encode the same before and after conversion. A field added to
// one type set and not the other, a renamed JSON tag, or a field the converter skips fails here.
func TestReportWireTwinsMatch(t *testing.T) {
	t.Parallel()
	var full report.Board
	n := 0
	fillWire(reflect.ValueOf(&full).Elem(), &n)
	choices := []report.Choice{{Value: "0001", Label: "One"}}
	for _, c := range []struct {
		name       string
		built, got any
	}{
		{"full board", full, wireBoard(full)},
		{"empty board", report.Board{}, wireBoard(report.Board{})},
		{"empty report", report.Board{Reports: []report.Report{{}}}, wireBoard(report.Board{Reports: []report.Report{{}}})},
		{"choices", choices, wireChoices(choices)},
		{"nil choices", []report.Choice(nil), wireChoices(nil)},
	} {
		want, err := json.Marshal(c.built)
		if err != nil {
			t.Fatal(err)
		}
		got, err := json.Marshal(c.got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("%s: wire twin encodes differently\n built %s\n wire  %s", c.name, want, got)
		}
	}
}

// fillWire sets every field reachable from v to a distinct non-zero value.
func fillWire(v reflect.Value, n *int) {
	*n++
	kind := v.Kind()
	if kind == reflect.Struct {
		for i := range v.NumField() {
			fillWire(v.Field(i), n)
		}
		return
	}
	if kind == reflect.Pointer {
		v.Set(reflect.New(v.Type().Elem()))
		fillWire(v.Elem(), n)
		return
	}
	if kind == reflect.Slice {
		v.Set(reflect.MakeSlice(v.Type(), 2, 2))
		for i := range v.Len() {
			fillWire(v.Index(i), n)
		}
		return
	}
	scalars := map[reflect.Kind]func(){
		reflect.String:  func() { v.SetString(fmt.Sprint("text ", *n)) },
		reflect.Float64: func() { v.SetFloat(float64(*n) + 0.25) },
		reflect.Int:     func() { v.SetInt(int64(*n)) },
		reflect.Int64:   func() { v.SetInt(int64(*n)) },
		reflect.Bool:    func() { v.SetBool(true) },
	}
	set, ok := scalars[kind]
	if !ok {
		panic("fillWire: add a case for " + kind.String())
	}
	set()
}
