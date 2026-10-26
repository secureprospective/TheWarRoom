package catalog

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Schema describes JSON shape, not business validity. Validator names the existing Raw check;
// optional means a field may be omitted, not that its zero value is legal for every command.
type Schema struct {
	Type      string
	Enum      []string
	Format    string
	Validator string
	Nullable  bool
	Fields    []Field
	Items     *Schema
}

type Field struct {
	Name     string
	Optional bool
	Schema   Schema
}

// Shape derives JSON field names and scalar types from the Raw type, avoiding a second field list.
// Unsupported types remain unset and cannot pass publication. RawMessage is checked by its owning
// Raw validator; it is not permission for a dispatcher to accept arbitrary values.
func Shape(t reflect.Type, validator string) Schema {
	schema := shape(t)
	schema.Validator = validator
	return schema
}

func shape(t reflect.Type) Schema {
	if t == nil {
		return Schema{Type: Unset}
	}
	if t.Kind() == reflect.Pointer {
		schema := shape(t.Elem())
		schema.Nullable = true
		return schema
	}
	switch t {
	case reflect.TypeFor[time.Time]():
		return Schema{Type: "string", Format: "date-time"}
	case reflect.TypeFor[json.RawMessage]():
		return Schema{Type: "json"}
	}
	switch t.Kind() {
	case reflect.String:
		return Schema{Type: "string"}
	case reflect.Bool:
		return Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return Schema{Type: "number"}
	case reflect.Slice, reflect.Array:
		item := shape(t.Elem())
		return Schema{Type: "array", Items: &item}
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return Schema{Type: Unset}
		}
		item := shape(t.Elem())
		return Schema{Type: "map", Items: &item}
	case reflect.Struct:
		return structShape(t)
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Pointer, reflect.UnsafePointer:
		return Schema{Type: Unset}
	}
	return Schema{Type: Unset}
}

func structShape(t reflect.Type) Schema {
	schema := Schema{Type: "object", Fields: []Field{}}
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() && (!field.Anonymous || field.Type.Kind() != reflect.Struct) {
			continue
		}
		tag := strings.Split(field.Tag.Get("json"), ",")
		if tag[0] == "-" {
			continue
		}
		if field.Anonymous && tag[0] == "" {
			schema.Fields = append(schema.Fields, shape(field.Type).Fields...)
			continue
		}
		name := tag[0]
		if name == "" {
			name = field.Name
		}
		optional := false
		for _, option := range tag[1:] {
			optional = optional || option == "omitempty" || option == "omitzero"
		}
		schema.Fields = append(schema.Fields, Field{Name: name, Optional: optional, Schema: shape(field.Type)})
	}
	return schema
}

func (s Schema) Validate() error {
	if strings.TrimSpace(s.Validator) == "" || s.Validator == Unset {
		return fmt.Errorf("missing Raw validator")
	}
	return s.validateShape()
}

func (s Schema) validateShape() error {
	switch s.Type {
	case "object":
		seen := map[string]bool{}
		for _, field := range s.Fields {
			if field.Name == "" || seen[field.Name] {
				return fmt.Errorf("missing or duplicate argument name")
			}
			seen[field.Name] = true
			if err := field.Schema.validateShape(); err != nil {
				return fmt.Errorf("field %s: %w", field.Name, err)
			}
		}
	case "array", "map":
		if s.Items == nil {
			return fmt.Errorf("missing item schema")
		}
		return s.Items.validateShape()
	case "string", "integer", "number", "boolean", "json":
	default:
		return fmt.Errorf("unsupported argument type %q", s.Type)
	}
	return nil
}
