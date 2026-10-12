package facts

import (
	"cmp"
	"reflect"
	"sort"
)

func sortFacts(f *factRows) {
	rows := reflect.ValueOf(f).Elem()
	for i := 0; i < rows.NumField(); i++ {
		sortRelation(rows.Field(i))
	}
}

// sortRelation orders a slice of rows by every column in turn, nulls first, as SQLite orders them.
func sortRelation(relation reflect.Value) {
	sort.Sort(relationOrder{relation, reflect.Swapper(relation.Interface())})
}

type relationOrder struct {
	rows reflect.Value
	swap func(int, int)
}

func (r relationOrder) Len() int      { return r.rows.Len() }
func (r relationOrder) Swap(i, j int) { r.swap(i, j) }
func (r relationOrder) Less(i, j int) bool {
	a, b := r.rows.Index(i), r.rows.Index(j)
	for k := 0; k < a.NumField(); k++ {
		if order := compareCell(a.Field(k), b.Field(k)); order != 0 {
			return order < 0
		}
	}
	return false
}

func compareCell(a, b reflect.Value) int {
	if a.Kind() == reflect.Pointer {
		if a.IsNil() || b.IsNil() {
			if a.IsNil() && b.IsNil() {
				return 0
			}
			if a.IsNil() {
				return -1
			}
			return 1
		}
		a, b = a.Elem(), b.Elem()
	}
	switch {
	case a.Kind() == reflect.String:
		return cmp.Compare(a.String(), b.String())
	case a.Kind() == reflect.Int64:
		return cmp.Compare(a.Int(), b.Int())
	case a.Kind() == reflect.Float64:
		return cmp.Compare(a.Float(), b.Float())
	default:
		panic("unsupported facts column")
	}
}
func cmpInt(a, b int64) int { return cmp.Compare(a, b) }

func toColumns(rows factRows) Facts {
	var result Facts
	source := reflect.ValueOf(rows)
	target := reflect.ValueOf(&result).Elem()
	for i := 0; i < source.NumField(); i++ {
		setColumns(source.Field(i), target.Field(i))
	}
	return result
}

// setColumns copies a slice of rows into the matching struct of column slices.
func setColumns(relation, columns reflect.Value) {
	for col := 0; col < columns.NumField(); col++ {
		output := reflect.MakeSlice(columns.Field(col).Type(), relation.Len(), relation.Len())
		for row := 0; row < relation.Len(); row++ {
			output.Index(row).Set(relation.Index(row).Field(col))
		}
		columns.Field(col).Set(output)
	}
}
