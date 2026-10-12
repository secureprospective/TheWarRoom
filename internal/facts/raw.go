package facts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

type RawValue struct {
	Object map[string]RawValue
	List   []RawValue
	Scalar *string
	JSON   []byte
}

func decodeRaw(body []byte) (RawValue, error) {
	var compact bytes.Buffer
	if err := json.Compact(&compact, body); err != nil {
		return RawValue{}, fmt.Errorf("archive JSON: %w", err)
	}
	return decodeValue(compact.Bytes())
}

func decodeValue(body []byte) (RawValue, error) {
	r := RawValue{JSON: body}
	switch body[0] {
	case '{':
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(body, &fields); err != nil {
			return r, fmt.Errorf("object: %w", err)
		}
		r.Object = make(map[string]RawValue, len(fields))
		for key, value := range fields {
			child, err := decodeValue(value)
			if err != nil {
				return r, err
			}
			r.Object[key] = child
		}
	case '[':
		var fields []json.RawMessage
		if err := json.Unmarshal(body, &fields); err != nil {
			return r, fmt.Errorf("list: %w", err)
		}
		for _, value := range fields {
			child, err := decodeValue(value)
			if err != nil {
				return r, err
			}
			r.List = append(r.List, child)
		}
	case '"':
		var value string
		if err := json.Unmarshal(body, &value); err != nil {
			return r, fmt.Errorf("string: %w", err)
		}
		r.Scalar = &value
	case 'n':
	default:
		value := string(body)
		r.Scalar = &value
	}
	return r, nil
}

func (r RawValue) Validate() error {
	if r.Object == nil {
		return fmt.Errorf("archive root must be an object")
	}
	return nil
}

func (r RawValue) get(keys ...string) RawValue {
	for _, key := range keys {
		r = r.Object[key]
	}
	return r
}

func (r RawValue) rows() []RawValue {
	if r.Object != nil {
		return []RawValue{r}
	}
	return r.List
}

func (r RawValue) str() *string { return r.Scalar }
func (r RawValue) text() *string {
	if r.Object != nil {
		return r.get("$t").str()
	}
	return nonempty(r.str())
}
func nonempty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}
func value(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func ptr[T string | int64 | float64](v T) *T { return &v }

func (r RawValue) num() *float64 {
	s := strings.NewReplacer("$", "", ",", "").Replace(value(r.str()))
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &n
}
func (r RawValue) whole() *int64 {
	if n := r.num(); n != nil {
		return ptr(int64(*n))
	}
	return nil
}

func (r RawValue) dump() string {
	if len(r.JSON) == 0 {
		return "null"
	}
	var out strings.Builder
	for i := 0; i < len(r.JSON); i++ {
		c := r.JSON[i]
		if c == '"' {
			end := quotedEnd(r.JSON, i)
			var s string
			// The decoder already validated this token.
			if err := json.Unmarshal([]byte(r.JSON[i:end]), &s); err != nil {
				panic(err)
			}
			out.WriteString(asciiString(s))
			i = end - 1
		} else {
			out.WriteByte(c)
			if c == ',' || c == ':' {
				out.WriteByte(' ')
			}
		}
	}
	return out.String()
}

func quotedEnd(s []byte, start int) int {
	for i := start + 1; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '"' {
			return i + 1
		}
	}
	return len(s)
}

func asciiString(s string) string {
	var out strings.Builder
	out.WriteByte('"')
	for _, c := range s {
		switch {
		case c == '"' || c == '\\':
			out.WriteByte('\\')
			out.WriteRune(c)
		case c == '\b':
			out.WriteString(`\b`)
		case c == '\f':
			out.WriteString(`\f`)
		case c == '\n':
			out.WriteString(`\n`)
		case c == '\r':
			out.WriteString(`\r`)
		case c == '\t':
			out.WriteString(`\t`)
		case c < 32 || c >= 127:
			for _, code := range utf16.Encode([]rune{c}) {
				hex := strconv.FormatUint(uint64(code), 16)
				out.WriteString(`\u` + strings.Repeat("0", 4-len(hex)) + hex)
			}
		default:
			out.WriteRune(c)
		}
	}
	out.WriteByte('"')
	return out.String()
}
