package leaguerules

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// Current returns every catalogue field's in-force value as the plain JSON DecodeSetting accepts, in
// catalogue order. A field that Needs a ruling is null.
func Current(r RuleSet) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(ruleFields()))
	for _, f := range ruleFields() {
		body := json.RawMessage("null")
		if natural, ok := naturalValue(reflect.ValueOf(f.read(r))); ok {
			if encoded, err := json.Marshal(natural); err == nil {
				body = encoded
			}
		}
		out = append(out, body)
	}
	return out
}

// IsPercent says a field holds a share of one, which a GM reads and types as a percentage.
func IsPercent(path FieldPath) bool { return percentFields()[path] }

// naturalValue mirrors decodeNatural; ok is false for an unset value or an empty date.
func naturalValue(v reflect.Value) (any, bool) {
	if isValueType(v.Type()) {
		if v.FieldByName("State").Uint() != uint64(IsKnown) {
			return nil, false
		}
		return naturalValue(v.FieldByName("Value"))
	}
	if v.Type() == reflect.TypeFor[time.Time]() {
		at, ok := v.Interface().(time.Time)
		if !ok || at.IsZero() {
			return nil, false
		}
		return at.UTC().Format(time.RFC3339Nano), true
	}
	kind := v.Kind()
	if kind == reflect.Struct {
		return naturalRecord(v), true
	}
	if kind == reflect.Slice {
		items := make([]any, 0, v.Len())
		for i := range v.Len() {
			item, _ := naturalValue(v.Index(i))
			items = append(items, item)
		}
		return items, true
	}
	if kind == reflect.Map {
		entries := make(map[string]any, v.Len())
		for iter := v.MapRange(); iter.Next(); {
			entry, _ := naturalValue(iter.Value())
			entries[fmt.Sprint(iter.Key().Interface())] = entry
		}
		return entries, true
	}
	return naturalScalar(v)
}

func naturalRecord(v reflect.Value) map[string]any {
	fields := make(map[string]any, v.NumField())
	for i := range v.NumField() {
		if field, ok := naturalValue(v.Field(i)); ok {
			fields[v.Type().Field(i).Name] = field
		}
	}
	return fields
}

func naturalScalar(v reflect.Value) (any, bool) {
	if v.Kind() == reflect.Bool {
		return v.Bool(), true
	}
	if v.Kind() == reflect.String {
		return v.String(), true
	}
	if v.Kind() == reflect.Int || v.Kind() == reflect.Int64 {
		return v.Int(), true
	}
	return nil, false
}
