// Package leaguerules models commissioner-owned rules without I/O.
package leaguerules

import (
	"fmt"
	"reflect"
)

type State uint8

const (
	NeedsRuling State = iota
	IsKnown
)

// Decimal is exact ten-thousandths; Money is integer cents.
type Decimal int64
type Money int64
type Value[T any] struct {
	State    State
	Value    T
	Citation string
}

func Known[T any](v T) Value[T]             { return Value[T]{State: IsKnown, Value: v} }
func Needs[T any](citation string) Value[T] { return Value[T]{Citation: citation} }
func cited[T any](v T, citation string) Value[T] {
	return Value[T]{State: IsKnown, Value: v, Citation: citation}
}
func (v Value[T]) ruleSetting() {}
func (v Value[T]) Status() string {
	if v.State == NeedsRuling {
		return "Needs a ruling"
	}
	return "Set"
}

// Setting is sealed: changes carry an actual typed Value, never decoded arbitrary data.
type Setting interface{ ruleSetting() }
type FieldPath string

type FieldChange struct {
	Field  FieldPath
	Label  string
	Before Setting
	After  Setting
}

type ruleField interface {
	pathName() FieldPath
	labelText() string
	read(RuleSet) Setting
	apply(*RuleSet, Setting) error
	difference(RuleSet, RuleSet) (FieldChange, bool)
}
type field[T any] struct {
	path   FieldPath
	label  string
	access func(*RuleSet) *Value[T]
}

func (f field[T]) pathName() FieldPath    { return f.path }
func (f field[T]) labelText() string      { return f.label }
func (f field[T]) read(r RuleSet) Setting { return *f.access(&r) }
func (f field[T]) apply(r *RuleSet, s Setting) error {
	v, ok := s.(Value[T])
	if !ok {
		return fmt.Errorf("%s needs the correct kind of value", f.label)
	}
	if err := validateValue(reflect.ValueOf(v)); err != nil {
		return fmt.Errorf("%s: %w", f.label, err)
	}
	*f.access(r) = cloneValue(v)
	return nil
}
func (f field[T]) difference(a, b RuleSet) (FieldChange, bool) {
	before, after := *f.access(&a), *f.access(&b)
	if reflect.DeepEqual(before, after) {
		return FieldChange{}, false
	}
	return FieldChange{Field: f.path, Label: f.label, Before: cloneValue(before), After: cloneValue(after)}, true
}
func Diff(before, after RuleSet) []FieldChange {
	var result []FieldChange
	for _, f := range ruleFields() {
		if change, ok := f.difference(before, after); ok {
			result = append(result, change)
		}
	}
	return result
}
func cloneValue[T any](v T) T {
	return cloneReflect(reflect.ValueOf(v)).Interface().(T)
}
func cloneReflect(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Map:
		if v.IsNil() {
			return v
		}
		copyMap := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			copyMap.SetMapIndex(iter.Key(), cloneReflect(iter.Value()))
		}
		return copyMap
	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		result := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := range v.Len() {
			result.Index(i).Set(cloneReflect(v.Index(i)))
		}
		return result
	case reflect.Struct:
		result := reflect.New(v.Type()).Elem()
		result.Set(v)
		for i := range v.NumField() {
			if result.Field(i).CanSet() && v.Field(i).CanInterface() {
				result.Field(i).Set(cloneReflect(v.Field(i)))
			}
		}
		return result
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Pointer, reflect.String, reflect.UnsafePointer:
		return v
	}
	return v
}
