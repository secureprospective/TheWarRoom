package catalog

import (
	"encoding/json"
	"reflect"
)

func Bind0[R any](call func() (R, error)) Binding {
	return binding([]Field{}, func(_ []json.RawMessage) (json.RawMessage, error) {
		return result(call())
	})
}
func Bind0Error(call func() error) Binding {
	return binding([]Field{}, func(_ []json.RawMessage) (json.RawMessage, error) {
		return result(json.RawMessage("null"), call())
	})
}
func Bind0Value[R any](call func() R) Binding {
	return binding([]Field{}, func(_ []json.RawMessage) (json.RawMessage, error) {
		return result(call(), nil)
	})
}
func Bind1[A, R any](call func(A) (R, error), name0 string) Binding {
	return binding([]Field{{Name: name0, Schema: shape(reflect.TypeFor[A]())}}, func(args []json.RawMessage) (json.RawMessage, error) {
		var arg0 A
		if err := decodeJSON(args[0], &arg0); err != nil {
			return nil, err
		}
		return result(call(arg0))
	})
}
func Bind1Error[A any](call func(A) error, name0 string) Binding {
	return binding([]Field{{Name: name0, Schema: shape(reflect.TypeFor[A]())}}, func(args []json.RawMessage) (json.RawMessage, error) {
		var arg0 A
		if err := decodeJSON(args[0], &arg0); err != nil {
			return nil, err
		}
		return result(json.RawMessage("null"), call(arg0))
	})
}
func Bind1Value[A, R any](call func(A) R, name0 string) Binding {
	return binding([]Field{{Name: name0, Schema: shape(reflect.TypeFor[A]())}}, func(args []json.RawMessage) (json.RawMessage, error) {
		var arg0 A
		if err := decodeJSON(args[0], &arg0); err != nil {
			return nil, err
		}
		return result(call(arg0), nil)
	})
}
func Bind2[A, B, R any](call func(A, B) (R, error), name0 string, name1 string) Binding {
	return binding([]Field{{Name: name0, Schema: shape(reflect.TypeFor[A]())}, {Name: name1, Schema: shape(reflect.TypeFor[B]())}}, func(args []json.RawMessage) (json.RawMessage, error) {
		var arg0 A
		if err := decodeJSON(args[0], &arg0); err != nil {
			return nil, err
		}
		var arg1 B
		if err := decodeJSON(args[1], &arg1); err != nil {
			return nil, err
		}
		return result(call(arg0, arg1))
	})
}
func Bind2Error[A, B any](call func(A, B) error, name0 string, name1 string) Binding {
	return binding([]Field{{Name: name0, Schema: shape(reflect.TypeFor[A]())}, {Name: name1, Schema: shape(reflect.TypeFor[B]())}}, func(args []json.RawMessage) (json.RawMessage, error) {
		var arg0 A
		if err := decodeJSON(args[0], &arg0); err != nil {
			return nil, err
		}
		var arg1 B
		if err := decodeJSON(args[1], &arg1); err != nil {
			return nil, err
		}
		return result(json.RawMessage("null"), call(arg0, arg1))
	})
}
func Bind2Value[A, B, R any](call func(A, B) R, name0 string, name1 string) Binding {
	return binding([]Field{{Name: name0, Schema: shape(reflect.TypeFor[A]())}, {Name: name1, Schema: shape(reflect.TypeFor[B]())}}, func(args []json.RawMessage) (json.RawMessage, error) {
		var arg0 A
		if err := decodeJSON(args[0], &arg0); err != nil {
			return nil, err
		}
		var arg1 B
		if err := decodeJSON(args[1], &arg1); err != nil {
			return nil, err
		}
		return result(call(arg0, arg1), nil)
	})
}
func Bind3[A, B, C, R any](call func(A, B, C) (R, error), name0 string, name1 string, name2 string) Binding {
	return binding([]Field{{Name: name0, Schema: shape(reflect.TypeFor[A]())}, {Name: name1, Schema: shape(reflect.TypeFor[B]())}, {Name: name2, Schema: shape(reflect.TypeFor[C]())}}, func(args []json.RawMessage) (json.RawMessage, error) {
		var arg0 A
		if err := decodeJSON(args[0], &arg0); err != nil {
			return nil, err
		}
		var arg1 B
		if err := decodeJSON(args[1], &arg1); err != nil {
			return nil, err
		}
		var arg2 C
		if err := decodeJSON(args[2], &arg2); err != nil {
			return nil, err
		}
		return result(call(arg0, arg1, arg2))
	})
}
func Bind3Value[A, B, C, R any](call func(A, B, C) R, name0 string, name1 string, name2 string) Binding {
	return binding([]Field{{Name: name0, Schema: shape(reflect.TypeFor[A]())}, {Name: name1, Schema: shape(reflect.TypeFor[B]())}, {Name: name2, Schema: shape(reflect.TypeFor[C]())}}, func(args []json.RawMessage) (json.RawMessage, error) {
		var arg0 A
		if err := decodeJSON(args[0], &arg0); err != nil {
			return nil, err
		}
		var arg1 B
		if err := decodeJSON(args[1], &arg1); err != nil {
			return nil, err
		}
		var arg2 C
		if err := decodeJSON(args[2], &arg2); err != nil {
			return nil, err
		}
		return result(call(arg0, arg1, arg2), nil)
	})
}
func Bind4[A, B, C, D, R any](call func(A, B, C, D) (R, error), name0 string, name1 string, name2 string, name3 string) Binding {
	return binding([]Field{{Name: name0, Schema: shape(reflect.TypeFor[A]())}, {Name: name1, Schema: shape(reflect.TypeFor[B]())}, {Name: name2, Schema: shape(reflect.TypeFor[C]())}, {Name: name3, Schema: shape(reflect.TypeFor[D]())}}, func(args []json.RawMessage) (json.RawMessage, error) {
		var arg0 A
		if err := decodeJSON(args[0], &arg0); err != nil {
			return nil, err
		}
		var arg1 B
		if err := decodeJSON(args[1], &arg1); err != nil {
			return nil, err
		}
		var arg2 C
		if err := decodeJSON(args[2], &arg2); err != nil {
			return nil, err
		}
		var arg3 D
		if err := decodeJSON(args[3], &arg3); err != nil {
			return nil, err
		}
		return result(call(arg0, arg1, arg2, arg3))
	})
}
