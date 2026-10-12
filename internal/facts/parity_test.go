package facts

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type rawGoldenHeader struct {
	Table   string   `json:"table"`
	Kind    string   `json:"kind"`
	Columns []string `json:"columns"`
}
type rawFixtureInput struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
	Tokens []int  `json:"tokens"`
}
type rawInputFixture struct {
	Dictionary []string          `json:"dictionary"`
	Inputs     []rawFixtureInput `json:"inputs"`
}

func TestReducedReferenceParity(t *testing.T) {
	inputs := fixtureInputs(t)
	runParity(t, inputs, "../lab/testdata/facts/facts.jsonl.gz", true)
}

func TestFullReferenceParity(t *testing.T) {
	if os.Getenv("TWR_ARCHIVE_PARITY") != "1" {
		t.Skip("full archive parity is opt-in")
	}
	oracle := os.Getenv("TWR_FACTS_ORACLE")
	root := os.Getenv("TWR_FACTS_ARCHIVE")
	if oracle == "" || root == "" {
		t.Fatal("set TWR_FACTS_ORACLE and TWR_FACTS_ARCHIVE")
	}
	inputs := fixtureInputs(t)
	for i := range inputs {
		body, err := readWithin(root, inputs[i].File)
		if err != nil {
			t.Fatal(err)
		}
		inputs[i].Body = body
	}
	runParity(t, inputs, oracle, false)
}

func fixtureInputs(t *testing.T) []ResolvedInput {
	t.Helper()
	body := gunzipFile(t, "../lab/testdata/facts/inputs.json.gz")
	var fixture rawInputFixture
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatal(err)
	}
	var inputs []ResolvedInput
	for _, raw := range fixture.Inputs {
		var body []byte
		for _, token := range raw.Tokens {
			if token < 0 || token >= len(fixture.Dictionary) {
				t.Fatal("invalid fixture token")
			}
			body = append(body, fixture.Dictionary[token]...)
		}
		input := ResolvedInput{File: raw.File, SHA256: raw.SHA256, Bytes: raw.Bytes, Body: body}
		if err := input.Validate(); err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, input)
	}
	return inputs
}

func gunzipFile(t *testing.T, file string) []byte {
	t.Helper()
	body, err := readWithin(filepath.Dir(file), filepath.Base(file))
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := gz.Close(); err != nil {
			t.Error(err)
		}
	}()
	var result bytes.Buffer
	if _, err := result.ReadFrom(gz); err != nil {
		t.Fatal(err)
	}
	return result.Bytes()
}

func runParity(t *testing.T, inputs []ResolvedInput, oracle string, reduced bool) {
	t.Helper()
	curated, err := os.ReadFile("../lab/testdata/note_readings.csv")
	if err != nil {
		t.Fatal(err)
	}
	notes, err := ParseCurated(curated)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	facts, err := Compile(t.Context(), inputs, notes)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("compile wall=%s inputs=%d", time.Since(start), len(inputs))
	selectedTxns := selectedTransactions(t, inputs)
	file, err := os.OpenInRoot(filepath.Dir(oracle), filepath.Base(oracle))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	}()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 65536), 1024*1024)
	headers := map[string]rawGoldenHeader{}
	rows := map[string][][]json.RawMessage{}
	var table string
	for scanner.Scan() {
		line := scanner.Bytes()
		if line[0] == '{' {
			var h rawGoldenHeader
			if err := json.Unmarshal(line, &h); err != nil {
				t.Fatal(err)
			}
			headers[h.Table] = h
			table = h.Table
		} else {
			if !supportedRelation(table) {
				continue
			}
			var cells []json.RawMessage
			if err := json.Unmarshal(line, &cells); err != nil {
				t.Fatal(err)
			}
			rows[table] = append(rows[table], cells)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	compareRelations(t, reflect.ValueOf(facts), headers, rows, reduced, selectedTxns)
	start = time.Now()
	views := facts.Views()
	t.Logf("views wall=%s", time.Since(start))
	compareRelations(t, reflect.ValueOf(views), headers, rows, reduced, selectedTxns)
	if !reduced {
		compareChecks(t, facts, filepath.Join(filepath.Dir(oracle), "checks.json"))
	}
}

// compareChecks runs the nine checks on the full archive: every difference is pinned, and each
// check's counts equal the reference's checks.py result.
func compareChecks(t *testing.T, facts Facts, reference string) {
	t.Helper()
	results, err := facts.Check(pinnedExceptions(t))
	if err != nil {
		t.Fatal(err)
	}
	body, err := readWithin(filepath.Dir(reference), filepath.Base(reference))
	if err != nil {
		t.Fatal(err)
	}
	var want []struct {
		Name                                     string
		Passed, Checked, Agreed, Explained, Open int
	}
	if err := json.Unmarshal(body, &want); err != nil {
		t.Fatal(err)
	}
	byName := map[string]CheckResult{}
	for _, r := range results {
		byName[r.Name] = r
	}
	for _, w := range want {
		r := byName[w.Name]
		got := [4]int{r.Checked, r.Agreed, r.Explained, r.Review}
		if got != [4]int{w.Checked, w.Agreed, w.Explained, w.Open} || !r.Passed {
			t.Errorf("%s: %v, reference %+v", w.Name, got, w)
		}
		t.Logf("check %q checked=%d agreed=%d explained=%d review=%d", r.Name, r.Checked, r.Agreed, r.Explained, r.Review)
	}
}

func compareRelations(t *testing.T, relations reflect.Value, headers map[string]rawGoldenHeader, rows map[string][][]json.RawMessage, reduced bool, txns map[int64]bool) {
	t.Helper()
	for i := 0; i < relations.NumField(); i++ {
		relation := relations.Field(i)
		name := relations.Type().Field(i).Name
		table := tableName(name, headers)
		header, exists := headers[table]
		if !exists || relation.NumField() != len(header.Columns) {
			t.Fatalf("missing or changed schema %s", name)
		}
		for col, column := range header.Columns {
			if relation.Type().Field(col).Tag.Get("json") != column {
				t.Fatalf("%s column %d schema changed", table, col)
			}
		}
		actual := relationRows(t, relation, table, reduced, txns)
		expected := rows[table]
		differences := compareRows(t, table, actual, expected)
		t.Logf("%s rows=%d differences=%d", table, len(expected), differences)
	}
}

func tableName(name string, headers map[string]rawGoldenHeader) string {
	for table := range headers {
		if strings.EqualFold(strings.ReplaceAll(table, "_", ""), name) {
			return table
		}
	}
	return ""
}

func relationRows(t *testing.T, relation reflect.Value, table string, reduced bool, txns map[int64]bool) [][]json.RawMessage {
	t.Helper()
	var rows [][]json.RawMessage
	for row := 0; row < relation.Field(0).Len(); row++ {
		if reduced && !selectedRow(relation, row, table, txns) {
			continue
		}
		cells := make([]json.RawMessage, relation.NumField())
		for col := range cells {
			body, err := json.Marshal(relation.Field(col).Index(row).Interface())
			if err != nil {
				t.Fatal(err)
			}
			cells[col] = body
		}
		rows = append(rows, cells)
	}
	return rows
}

func selectedRow(relation reflect.Value, row int, table string, txns map[int64]bool) bool {
	if table == "source_file" || table == "load_note" {
		field := relation.FieldByName("Src").Index(row)
		if field.Kind() == reflect.Pointer {
			field = field.Elem()
		}
		src := field.String()
		return !strings.Contains(src, "/") || strings.HasPrefix(src, "2013/") || strings.HasPrefix(src, "2020/") || strings.HasPrefix(src, "2026/")
	}
	for _, name := range []string{"Year", "SnapshotYear"} {
		if field := relation.FieldByName(name); field.IsValid() {
			return selectedYear(field.Index(row).Int())
		}
	}
	if field := relation.FieldByName("TxnID"); field.IsValid() {
		return txns[field.Index(row).Int()]
	}
	return true
}
func selectedYear(year int64) bool { return year == 2013 || year == 2020 || year == 2026 }

func selectedTransactions(t *testing.T, inputs []ResolvedInput) map[int64]bool {
	t.Helper()
	c, err := newCompiler(t.Context(), inputs)
	if err != nil {
		t.Fatal(err)
	}
	selected := map[int64]bool{}
	var tid int64
	for _, year := range c.years {
		d, err := c.read(seasonFile(year, "transactions"))
		if err != nil {
			t.Fatal(err)
		}
		for range d.get("transactions", "transaction").rows() {
			tid++
			selected[tid] = selectedYear(year)
		}
	}
	return selected
}

func compareRows(t *testing.T, table string, got, want [][]json.RawMessage) int {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s rows %d != %d", table, len(got), len(want))
		return 1
	}
	differences := 0
	for row := range want {
		if len(got[row]) != len(want[row]) {
			t.Fatalf("%s column count changed", table)
		}
		for col := range want[row] {
			if !equalCell(got[row][col], want[row][col], false) {
				differences++
				if differences <= 5 {
					t.Errorf("%s row %d col %d: %s != %s", table, row, col, got[row][col], want[row][col])
				}
			}
		}
	}
	return differences
}

func equalCell(a, b json.RawMessage, derived bool) bool {
	if bytes.Equal(a, b) {
		return true
	}
	if bytes.Equal(a, []byte("null")) || bytes.Equal(b, []byte("null")) {
		return false
	}
	if len(a) == 0 || len(b) == 0 || a[0] == '"' || b[0] == '"' || bytes.Equal(a, []byte("null")) || bytes.Equal(b, []byte("null")) {
		var sa, sb string
		if json.Unmarshal(a, &sa) != nil || json.Unmarshal(b, &sb) != nil {
			return false
		}
		return sa == sb
	}
	if !derived {
		ra, okA := new(big.Rat).SetString(string(a))
		rb, okB := new(big.Rat).SetString(string(b))
		return okA && okB && ra.Cmp(rb) == 0
	}
	var fa, fb float64
	if json.Unmarshal(a, &fa) != nil || json.Unmarshal(b, &fb) != nil {
		return false
	}
	return math.Abs(fa-fb) <= 1e-9+1e-9*math.Max(math.Abs(fa), math.Abs(fb))
}

func TestComparisonRules(t *testing.T) {
	cases := []struct {
		a, b          string
		derived, want bool
	}{
		{"1.00", "1", false, true}, {"1.01", "1", false, false}, {`"0151"`, `"151"`, false, false},
		{"null", "0", false, false}, {"null", `""`, false, false}, {"100", "100.00000001", true, true},
		{"1", "1.00001", true, false}, {`"a"`, `"\u0061"`, false, true},
	}
	for _, c := range cases {
		t.Run(fmt.Sprint(c), func(t *testing.T) {
			if equalCell([]byte(c.a), []byte(c.b), c.derived) != c.want {
				t.Fatal("comparison rule")
			}
		})
	}
}

func readWithin(root, file string) ([]byte, error) {
	f, err := os.OpenInRoot(root, file)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(f)
	closeErr := f.Close()
	if readErr != nil {
		return nil, readErr
	}
	return body, closeErr
}

func supportedRelation(table string) bool {
	for _, typ := range []reflect.Type{reflect.TypeFor[Facts](), reflect.TypeFor[Views]()} {
		for i := 0; i < typ.NumField(); i++ {
			if strings.EqualFold(strings.ReplaceAll(table, "_", ""), typ.Field(i).Name) {
				return true
			}
		}
	}
	return false
}

// TestWriteFullCheckItems writes every difference the checks find on the full archive, for the
// exceptions file's provenance (TWR_FACTS_CHECK_ITEMS=<out file>, with the full-parity variables).
func TestWriteFullCheckItems(t *testing.T) {
	out := os.Getenv("TWR_FACTS_CHECK_ITEMS")
	if out == "" {
		t.Skip("opt-in")
	}
	inputs := fixtureInputs(t)
	for i := range inputs {
		body, err := readWithin(os.Getenv("TWR_FACTS_ARCHIVE"), inputs[i].File)
		if err != nil {
			t.Fatal(err)
		}
		inputs[i].Body = body
	}
	curated, err := os.ReadFile("../lab/testdata/note_readings.csv")
	if err != nil {
		t.Fatal(err)
	}
	notes, err := ParseCurated(curated)
	if err != nil {
		t.Fatal(err)
	}
	facts, err := Compile(t.Context(), inputs, notes)
	if err != nil {
		t.Fatal(err)
	}
	results, _ := facts.Check(Exceptions{})
	body, err := json.MarshalIndent(results, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(filepath.Dir(out))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := root.WriteFile(filepath.Base(out), body, 0o600); err != nil {
		t.Fatal(err)
	}
}
