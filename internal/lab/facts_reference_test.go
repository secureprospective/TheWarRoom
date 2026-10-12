package lab

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type rawReferenceHeader struct {
	Table   string   `json:"table"`
	Kind    string   `json:"kind"`
	Columns []string `json:"columns"`
}

func TestFactsReferencePinsMatchSourceRows(t *testing.T) {
	checkFactsReference(t, "testdata/facts/facts.jsonl.gz", false)
}

func TestFullFactsReferencePinsMatchSourceRows(t *testing.T) {
	if os.Getenv("TWR_ARCHIVE_PARITY") != "1" {
		t.Skip("full facts oracle is opt-in")
	}
	path := os.Getenv("TWR_FACTS_ORACLE")
	if path == "" {
		t.Fatal("set TWR_FACTS_ORACLE")
	}
	checkFactsReference(t, path, true)
}

func checkFactsReference(t *testing.T, path string, full bool) {
	t.Helper()
	file, err := os.OpenInRoot(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		t.Fatal(err)
	}
	compressed, err := io.ReadAll(file)
	if closeErr := file.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	want := "5c2724cdb2ec0c4c19d413e8f5a399e826e031a5411d4c9d8e86b34ca28523e0"
	if full {
		want = "491d0e807ef040c9e6fa9fa0caecdede72a9a2289176d0a3bc114eea56bbf876"
	}
	sum := sha256.Sum256(compressed)
	if hex.EncodeToString(sum[:]) != want {
		t.Fatal("independent facts reference fingerprint changed")
	}
	if !full && len(compressed) > 1500000 {
		t.Fatal("reduced golden exceeds 1.5 MB")
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	}()
	counts, source := scanReference(t, reader)
	expected := map[string]int{"source_file": 315, "game": 1632, "lineup": 46609}
	if full {
		expected = map[string]int{"source_file": 1477, "game": 7616, "lineup": 307255}
	}
	if len(counts) != 39 {
		t.Fatalf("reference relation count changed: %d", len(counts))
	}
	for table, count := range expected {
		if counts[table] != count {
			t.Fatalf("%s rows=%d, want %d", table, counts[table], count)
		}
	}
	checkReferenceSources(t, source, full)
}

func selectedReferenceFile(file string, full bool) bool {
	return full || !strings.Contains(file, "/") || strings.HasPrefix(file, "2013/") || strings.HasPrefix(file, "2020/") || strings.HasPrefix(file, "2026/")
}

func checkReferenceSources(t *testing.T, source map[string]Input, full bool) {
	t.Helper()
	pins, err := Inputs()
	if err != nil {
		t.Fatal(err)
	}
	gaps, err := FrozenGaps()
	if err != nil {
		t.Fatal(err)
	}
	for _, gap := range gaps {
		if !selectedReferenceFile(gap.File, full) {
			continue
		}
		if _, exists := source[gap.File]; exists {
			t.Errorf("missing pin substituted: %s", gap.File)
		}
		source[gap.File] = Input{File: gap.File, Bytes: 175, SHA256: gap.SHA256}
	}
	selected := 0
	for _, pin := range pins {
		if !selectedReferenceFile(pin.File, full) {
			continue
		}
		selected++
		if source[pin.File] != pin {
			t.Errorf("reference source mismatch: %s", pin.File)
		}
	}
	if len(source) != selected {
		t.Fatalf("reference sources plus gaps=%d, pins=%d", len(source), selected)
	}
}

func scanReference(t *testing.T, reader *gzip.Reader) (map[string]int, map[string]Input) {
	t.Helper()
	counts, source := map[string]int{}, map[string]Input{}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var table string
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.HasPrefix(line, []byte("{")) {
			var header rawReferenceHeader
			if err := json.Unmarshal(line, &header); err != nil {
				t.Fatal(err)
			}
			if header.Table == "" || len(header.Columns) == 0 || (header.Kind != "table" && header.Kind != "view") {
				t.Fatalf("invalid reference header: %+v", header)
			}
			if _, exists := counts[header.Table]; exists {
				t.Fatalf("duplicate relation: %s", header.Table)
			}
			table = header.Table
			counts[table] = 0
			continue
		}
		if table == "" || !bytes.HasPrefix(line, []byte("[")) {
			t.Fatal("reference row without a relation")
		}
		counts[table]++
		if table == "source_file" {
			addReferenceSource(t, source, line)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return counts, source
}

func addReferenceSource(t *testing.T, source map[string]Input, line []byte) {
	t.Helper()
	var cells []json.RawMessage
	if err := json.Unmarshal(line, &cells); err != nil || len(cells) != 3 {
		t.Fatalf("invalid source_file row: %s, %v", line, err)
	}
	var pin Input
	for i, dest := range []*string{&pin.File, &pin.SHA256} {
		if err := json.Unmarshal(cells[i*2], dest); err != nil {
			t.Fatal(err)
		}
	}
	if err := json.Unmarshal(cells[1], &pin.Bytes); err != nil {
		t.Fatal(err)
	}
	if pin.File == "" || strings.Contains(pin.File, "..") || pin.Bytes < 0 || !shaPattern().MatchString(pin.SHA256) {
		t.Fatalf("invalid source pin: %+v", pin)
	}
	if _, exists := source[pin.File]; exists {
		t.Fatalf("duplicate source pin: %s", pin.File)
	}
	source[pin.File] = pin
}

func TestCuratedInputAndGapPolicyFingerprints(t *testing.T) {
	files := map[string]string{
		"testdata/note_readings.csv":  "b69a8d56549eb9680a803fc0b4f225bd3bbb2326cf8128a081294adef9517f7e",
		"testdata/missing-inputs.csv": "d7cc8069b97a0aad29769814c71f7096c1e78cba277f5fe067ad304c22f45085",
	}
	for file, want := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("%s: fingerprint = %s, want %s", file, got, want)
		}
	}
}
