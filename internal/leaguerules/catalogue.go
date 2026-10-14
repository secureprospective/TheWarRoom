package leaguerules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type ValueSchema struct {
	Kind    string        `json:"kind"`
	Minimum *int64        `json:"minimum,omitempty"`
	Maximum *int64        `json:"maximum,omitempty"`
	Choices []Choice      `json:"choices"`
	Fields  []SchemaField `json:"fields"`
	Key     *ValueSchema  `json:"key,omitempty"`
	Element *ValueSchema  `json:"element,omitempty"`
}

// Choice pairs a stored option with the words a GM reads.
type Choice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
type SchemaField struct {
	Name   string      `json:"name"`
	Label  string      `json:"label"`
	Schema ValueSchema `json:"schema"`
}
type CatalogueField struct {
	Path        FieldPath   `json:"path"`
	Label       string      `json:"label"`
	Section     string      `json:"section"`
	Schema      ValueSchema `json:"schema"`
	Competitive bool        `json:"competitive"`
}
type SettingReading struct {
	Words    string `json:"words"`
	State    string `json:"state"`
	Citation string `json:"citation"`
}

func Catalogue() []CatalogueField {
	result := make([]CatalogueField, 0, len(ruleFields()))
	for _, f := range ruleFields() {
		section, _, _ := strings.Cut(string(f.pathName()), ".")
		value := reflect.ValueOf(f.read(RuleSet{})).FieldByName("Value")
		schema := schemaFor(value.Type())
		fieldBounds(f.pathName(), &schema)
		// House rules are reference text; every other registered field changes competition.
		result = append(result, CatalogueField{f.pathName(), f.labelText(), sectionLabels()[section], schema,
			section != "house"})
	}
	return result
}

func sectionLabels() map[string]string {
	return map[string]string{"league": "League shape", "roster": "Roster", "lineup": "Lineup",
		"scoring": "Scoring", "standings": "Standings", "cap": "Salary cap", "contracts": "Contracts",
		"ir": "IR", "taxi": "Taxi squad", "waivers": "Waivers", "bids": "Free-agent bids", "trades": "Trades",
		"schedule": "Schedule", "playoffs": "Playoffs", "house": "House rules", "calendar": "Calendar",
		"draft": "Draft"}
}

// percentFields hold a share of one, which a GM reads as a percentage.
func percentFields() map[FieldPath]bool {
	return map[FieldPath]bool{"cap.irShare": true, "cap.taxiShare": true, "contracts.deadCap": true,
		"contracts.restructuredDeadCap": true}
}

func fieldBounds(path FieldPath, schema *ValueSchema) {
	switch path {
	case "trades.dotSeats":
		low, high := int64(4), int64(10)
		schema.Minimum, schema.Maximum = &low, &high
	case "cap.irShare", "cap.taxiShare":
		low, high := int64(0), int64(10000)
		schema.Minimum, schema.Maximum = &low, &high
	case "cap.quantum":
		low := int64(1)
		schema.Minimum = &low
	}
}

func schemaFor(t reflect.Type) ValueSchema {
	if isValueType(t) {
		return schemaFor(t.Field(1).Type)
	}
	s := ValueSchema{Kind: t.Kind().String(), Choices: []Choice{}, Fields: []SchemaField{}}
	switch t.Kind() {
	case reflect.Struct:
		if t == reflect.TypeFor[time.Time]() {
			s.Kind = "date"
			return s
		}
		s.Kind = "record"
		for i := range t.NumField() {
			f := t.Field(i)
			s.Fields = append(s.Fields, SchemaField{f.Name, fieldWords(f.Name), schemaFor(f.Type)})
		}
	case reflect.Map, reflect.Slice:
		s.Kind = "list"
		if t.Kind() == reflect.Map {
			s.Kind = "map"
		}
		child := schemaFor(t.Elem())
		s.Element = &child
		if t.Kind() == reflect.Map {
			key := schemaFor(t.Key())
			s.Key = &key
		}
	case reflect.Int, reflect.Int64:
		s.Kind = "count"
		low := int64(0)
		s.Minimum = &low
		switch t {
		case reflect.TypeFor[Money]():
			s.Kind = "cents"
		case reflect.TypeFor[Decimal]():
			s.Kind = "ten_thousandths"
			s.Minimum = nil
		case reflect.TypeFor[time.Duration]():
			s.Kind = "nanoseconds"
		}
	case reflect.String:
		s.Choices = catalogueChoices(t.Name())
	case reflect.Bool:
		s.Kind = "boolean"
	case reflect.Invalid, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Pointer, reflect.UnsafePointer:
	}
	return s
}

func catalogueChoices(kind string) []Choice {
	candidates := []string{"head_to_head", "draft", "all", "keepers", "zero_top_starter", "zero_slot", "none",
		"hard", "soft", "IR", "PUP", "NFI", "Out", "suspended", "calendar_years", "career_games", "cut",
		"reverse_standings", "rolling", "blind", "open", "dot", "commissioner", "start_of_day", "end_of_day",
		"slow", "live", "skip", "autopick", "standings", "linear", "snake", "win_percentage", "head_to_head",
		"division_percentage", "conference_percentage", "all_play_percentage", "points", "power_rank",
		"QB", "RB", "WR", "TE", "PK", "DT", "DE", "LB", "CB", "S", "offense", "defense", "total"}
	choices := []Choice{}
	if validateMenu(kind, "not_an_option") == nil {
		return choices
	}
	for _, option := range candidates {
		if validateMenu(kind, option) == nil {
			choices = append(choices, Choice{option, choiceWords(option)})
		}
	}
	return choices
}

func DecodeSetting(path FieldPath, body json.RawMessage) (Setting, error) {
	for _, f := range ruleFields() {
		if f.pathName() != path {
			continue
		}
		v := reflect.New(reflect.TypeOf(f.read(RuleSet{}))).Elem()
		if err := decodeNatural(body, v); err != nil {
			return nil, fmt.Errorf("%s: %w", f.labelText(), err)
		}
		setting, ok := v.Interface().(Setting)
		if !ok {
			return nil, fmt.Errorf("choose the correct kind of rule value")
		}
		var r RuleSet
		if err := f.apply(&r, setting); err != nil {
			return nil, err
		}
		if err := validateRules(r); err != nil {
			return nil, err
		}
		return setting, nil
	}
	return nil, fmt.Errorf("choose a recognised league rule")
}

func isValueType(t reflect.Type) bool {
	return t.Kind() == reflect.Struct && strings.HasPrefix(t.Name(), "Value[")
}

func decodeNatural(body json.RawMessage, target reflect.Value) error {
	if len(body) == 0 || bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
		return fmt.Errorf("choose a value")
	}
	if isValueType(target.Type()) {
		target.FieldByName("State").SetUint(uint64(IsKnown))
		return decodeNatural(body, target.FieldByName("Value"))
	}
	if target.Type() == reflect.TypeFor[time.Time]() {
		return decodeTyped(body, target)
	}
	switch target.Kind() {
	case reflect.Struct:
		return decodeRecord(body, target)
	case reflect.Slice:
		return decodeList(body, target)
	case reflect.Map:
		return decodeMap(body, target)
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Pointer, reflect.String,
		reflect.UnsafePointer:
		return decodeTyped(body, target)
	}
	return fmt.Errorf("choose the correct kind of value")
}

func decodeTyped(body json.RawMessage, target reflect.Value) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target.Addr().Interface()); err != nil {
		return fmt.Errorf("choose %s", kindWords(target.Type()))
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("send one rule value")
	}
	return nil
}

func kindWords(t reflect.Type) string {
	schema := schemaFor(t)
	switch schema.Kind {
	case "count":
		return "a whole number"
	case "cents":
		return "an amount in cents"
	case "ten_thousandths":
		return "a number in ten-thousandths"
	case "nanoseconds":
		return "a length of time"
	case "boolean":
		return "on or off"
	case "date":
		return "a date and time"
	case "string":
		if len(schema.Choices) == 0 {
			return "text"
		}
		return "one of the listed options"
	}
	return "the correct kind of value"
}

func decodeRecord(body json.RawMessage, target reflect.Value) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("choose named rule values")
	}
	for key, value := range raw {
		found := false
		for i := range target.NumField() {
			if !strings.EqualFold(key, target.Type().Field(i).Name) {
				continue
			}
			found = true
			if err := decodeNatural(value, target.Field(i)); err != nil {
				return err
			}
			break
		}
		if !found {
			return fmt.Errorf("choose recognised rule details")
		}
	}
	return nil
}

func decodeList(body json.RawMessage, target reflect.Value) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("choose a list of values")
	}
	target.Set(reflect.MakeSlice(target.Type(), len(raw), len(raw)))
	for i, value := range raw {
		if err := decodeNatural(value, target.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func decodeMap(body json.RawMessage, target reflect.Value) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("choose named values")
	}
	target.Set(reflect.MakeMapWithSize(target.Type(), len(raw)))
	for key, value := range raw {
		k := reflect.New(target.Type().Key()).Elem()
		if err := decodeKey(key, k); err != nil {
			return err
		}
		v := reflect.New(target.Type().Elem()).Elem()
		if err := decodeNatural(value, v); err != nil {
			return err
		}
		target.SetMapIndex(k, v)
	}
	return nil
}

func decodeKey(key string, target reflect.Value) error {
	switch target.Kind() {
	case reflect.String:
		target.SetString(key)
		return validateValue(target)
	case reflect.Int:
		n, err := strconv.Atoi(key)
		if err != nil || strconv.Itoa(n) != key {
			return fmt.Errorf("choose whole numbers for this rule's entries")
		}
		target.SetInt(int64(n))
		return nil
	case reflect.Invalid, reflect.Bool, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32,
		reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Struct,
		reflect.UnsafePointer:
	}
	return fmt.Errorf("choose recognised entries for this rule")
}

// EncodeStored keeps a full typed value, including its state and citation, for durable replay.
func EncodeStored(s Setting) (json.RawMessage, error) {
	body, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("record rule value: %w", err)
	}
	return body, nil
}

// DecodeStored reads a value written by EncodeStored and validates it for its field.
func DecodeStored(path FieldPath, body json.RawMessage) (Setting, error) {
	for _, f := range ruleFields() {
		if f.pathName() != path {
			continue
		}
		v := reflect.New(reflect.TypeOf(f.read(RuleSet{})))
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(v.Interface()); err != nil {
			return nil, fmt.Errorf("%s: read recorded value: %w", f.labelText(), err)
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			return nil, fmt.Errorf("%s: send one recorded value", f.labelText())
		}
		setting, ok := v.Elem().Interface().(Setting)
		if !ok {
			return nil, fmt.Errorf("choose the correct kind of rule value")
		}
		var r RuleSet
		if err := f.apply(&r, setting); err != nil {
			return nil, err
		}
		return setting, nil
	}
	return nil, fmt.Errorf("choose a recognised league rule")
}

func ReadSettings(r RuleSet) []SettingReading {
	out := make([]SettingReading, 0, len(ruleFields()))
	for _, f := range ruleFields() {
		out = append(out, DescribeField(f.pathName(), f.read(r)))
	}
	return out
}

// DescribeField is DescribeSetting with the field's own unit, such as a percentage.
func DescribeField(path FieldPath, s Setting) SettingReading {
	reading := DescribeSetting(s)
	if v, ok := s.(Value[Decimal]); ok && v.State == IsKnown && percentFields()[path] {
		reading.Words = renderDecimal(int64(v.Value)*100) + "%"
	}
	return reading
}

func DescribeSetting(s Setting) SettingReading {
	v := reflect.ValueOf(s)
	if !v.IsValid() {
		return SettingReading{Words: "Needs a ruling", State: "Needs a ruling"}
	}
	state := "Set"
	if v.FieldByName("State").Uint() == uint64(NeedsRuling) {
		state = "Needs a ruling"
	}
	return SettingReading{Words: renderValue(v), State: state, Citation: v.FieldByName("Citation").String()}
}

func renderValue(v reflect.Value) string {
	if isValueType(v.Type()) {
		if v.FieldByName("State").Uint() == uint64(NeedsRuling) {
			return "Needs a ruling"
		}
		return renderValue(v.FieldByName("Value"))
	}
	if words, ok := renderSpecial(v); ok {
		return words
	}
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return "On"
		}
		return "Off"
	case reflect.String:
		if v.String() == "" {
			return "None"
		}
		if v.Type().Name() == "string" {
			return v.String()
		}
		return choiceWords(v.String())
	case reflect.Int, reflect.Int64:
		return renderNumber(v)
	case reflect.Struct:
		return renderRecord(v)
	case reflect.Slice:
		return renderList(v)
	case reflect.Map:
		return renderMap(v)
	case reflect.Invalid, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Pointer, reflect.UnsafePointer:
		return "Needs a ruling"
	}
	return "Needs a ruling"
}

func renderRecord(v reflect.Value) string {
	parts := []string{}
	for i := range v.NumField() {
		words := renderValue(v.Field(i))
		if v.Type().Field(i).Name == "ID" {
			words = plainName(words)
		}
		parts = append(parts, fieldWords(v.Type().Field(i).Name)+": "+words)
	}
	return strings.Join(parts, "; ")
}

func renderList(v reflect.Value) string {
	parts := []string{}
	for i := range v.Len() {
		parts = append(parts, renderValue(v.Index(i)))
	}
	if len(parts) == 0 {
		return "None"
	}
	return strings.Join(parts, "; ")
}

func renderMap(v reflect.Value) string {
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Kind() == reflect.Int {
			return keys[i].Int() < keys[j].Int()
		}
		return keys[i].String() < keys[j].String()
	})
	parts := []string{}
	for _, key := range keys {
		name := plainName(key.String())
		if key.Kind() == reflect.Int {
			name = strconv.FormatInt(key.Int(), 10)
		}
		parts = append(parts, name+": "+renderValue(v.MapIndex(key)))
	}
	if len(parts) == 0 {
		return "None"
	}
	return strings.Join(parts, "; ")
}

func renderNumber(v reflect.Value) string {
	switch v.Type() {
	case reflect.TypeFor[Money]():
		return renderMoney(v.Int())
	case reflect.TypeFor[Decimal]():
		return renderDecimal(v.Int())
	case reflect.TypeFor[time.Duration]():
		return renderDuration(time.Duration(v.Int()))
	default:
		return strconv.FormatInt(v.Int(), 10)
	}
}

func groupThousands(n int64) string {
	digits := strconv.FormatInt(n, 10)
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return digits
}

// renderMoney shows exact cents; whole dollars drop the cents.
func renderMoney(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	if cents%100 == 0 {
		return sign + "$" + groupThousands(cents/100)
	}
	return fmt.Sprintf("%s$%s.%02d", sign, groupThousands(cents/100), cents%100)
}

// renderDecimal shows exact ten-thousandths without binary rounding.
func renderDecimal(n int64) string {
	sign := ""
	if n < 0 {
		sign, n = "-", -n
	}
	fraction := strings.TrimRight(fmt.Sprintf("%04d", n%10000), "0")
	if fraction == "" {
		return sign + strconv.FormatInt(n/10000, 10)
	}
	return sign + strconv.FormatInt(n/10000, 10) + "." + fraction
}

func renderDuration(d time.Duration) string {
	parts := []string{}
	for _, unit := range []struct {
		size time.Duration
		name string
	}{{24 * time.Hour, "day"}, {time.Hour, "hour"}, {time.Minute, "minute"}, {time.Second, "second"}} {
		if n := d / unit.size; n > 0 {
			name := unit.name
			if n != 1 {
				name += "s"
			}
			parts = append(parts, fmt.Sprintf("%d %s", n, name))
			d -= n * unit.size
		}
	}
	if len(parts) == 0 {
		return "No time"
	}
	return strings.Join(parts, " ")
}

// renderSpecial gives ranges and dates their natural wording; ok is false for other values.
func renderSpecial(v reflect.Value) (string, bool) {
	if !v.CanInterface() {
		return "", false
	}
	switch value := v.Interface().(type) {
	case time.Time:
		return value.Format("2 Jan 2006 15:04 MST"), true
	case MonthDay:
		if value.Month.State != IsKnown || value.Day.State != IsKnown {
			return "Needs a ruling", true
		}
		return time.Date(2000, time.Month(value.Month.Value), value.Day.Value, 0, 0, 0, 0, time.UTC).
			Format("2 January"), true
	case Bounds:
		return renderRange(renderValue(reflect.ValueOf(value.Min)), renderValue(reflect.ValueOf(value.Max)),
			value.Min.State == IsKnown, value.Max.State == IsKnown), true
	}
	return "", false
}

func renderRange(low, high string, hasLow, hasHigh bool) string {
	switch {
	case hasLow && hasHigh && low == high:
		return "Exactly " + low
	case hasLow && hasHigh:
		return low + " to " + high
	case hasLow:
		return "At least " + low
	case hasHigh:
		return "At most " + high
	}
	return "Needs a ruling"
}

func choiceWords(option string) string {
	if option == string(DOTReview) {
		return "DOT"
	}
	return plainName(option)
}

func fieldWords(name string) string {
	switch name {
	case "Min":
		return "Minimum"
	case "Max":
		return "Maximum"
	case "ID":
		return "Game"
	}
	return plainName(name)
}

// plainName turns an identifier into words, keeping acronyms such as QB or UFA intact.
func plainName(s string) string {
	runes := []rune(strings.ReplaceAll(s, "_", " "))
	var words []string
	start := 0
	for i := 1; i <= len(runes); i++ {
		boundary := i == len(runes) || runes[i] == ' '
		if !boundary && unicode.IsUpper(runes[i]) {
			boundary = unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) ||
				(i+1 < len(runes) && unicode.IsUpper(runes[i-1]) && unicode.IsLower(runes[i+1]))
		}
		if !boundary {
			continue
		}
		if word := strings.TrimSpace(string(runes[start:i])); word != "" {
			words = append(words, word)
		}
		start = i
	}
	for i, word := range words {
		if !isAcronym(word) {
			word = strings.ToLower(word)
		}
		if i == 0 {
			word = strings.ToUpper(word[:1]) + word[1:]
		}
		words[i] = word
	}
	return strings.Join(words, " ")
}

func isAcronym(word string) bool {
	return len(word) > 1 && strings.ToUpper(word) == word
}
