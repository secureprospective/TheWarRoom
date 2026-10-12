package labcompile

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/facts"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

type rawInput struct {
	File   string
	SHA256 string
	Bytes  int64
	Tokens []int
}
type rawFixture struct {
	Dictionary []string
	Inputs     []rawInput
}

func (r rawFixture) Validate() error {
	for _, i := range r.Inputs {
		for _, token := range i.Tokens {
			if token < 0 || token >= len(r.Dictionary) {
				return fmt.Errorf("invalid fixture token %d", token)
			}
		}
	}
	return nil
}
func readGzip(t *testing.T, path string) []byte {
	t.Helper()
	body, err := readWithin(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		t.Fatal(err)
	}
	r, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return result
}
func referenceFacts(t *testing.T, full bool) facts.Facts {
	t.Helper()
	var fixture rawFixture
	if err := json.Unmarshal(readGzip(t, "../lab/testdata/facts/inputs.json.gz"), &fixture); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Validate(); err != nil {
		t.Fatal(err)
	}
	inputs := make([]facts.ResolvedInput, 0, len(fixture.Inputs))
	for _, r := range fixture.Inputs {
		var body []byte
		for _, token := range r.Tokens {
			body = append(body, fixture.Dictionary[token]...)
		}
		if full {
			var err error
			body, err = readWithin(os.Getenv("TWR_FACTS_ARCHIVE"), r.File)
			if err != nil {
				t.Fatal(err)
			}
		}
		input := facts.ResolvedInput{File: r.File, SHA256: r.SHA256, Bytes: r.Bytes, Body: body}
		if err := input.Validate(); err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, input)
	}
	body, err := os.ReadFile("../lab/testdata/note_readings.csv")
	if err != nil {
		t.Fatal(err)
	}
	notes, err := facts.ParseCurated(body)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	f, err := facts.Compile(t.Context(), inputs, notes)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("facts compile %s", time.Since(start))
	return f
}
func frozenEvidence(t *testing.T) (Evidence, map[string]json.RawMessage) {
	t.Helper()
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(readGzip(t, "../lab/testdata/lab.json.gz"), &doc); err != nil {
		t.Fatal(err)
	}
	var times struct {
		Built, ArchiveThrough time.Time
		Source                struct{ Pages PageExam }
	}
	body, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &times); err != nil {
		t.Fatal(err)
	}
	return Evidence{times.Built, times.ArchiveThrough, times.Source.Pages}, doc
}
func TestFullDocumentParity(t *testing.T) {
	if os.Getenv("TWR_ARCHIVE_PARITY") != "1" {
		t.Skip("full archive parity is opt-in")
	}
	if os.Getenv("TWR_FACTS_ARCHIVE") == "" || os.Getenv("TWR_LAB_ANALYSIS") == "" {
		t.Fatal("set TWR_FACTS_ARCHIVE and TWR_LAB_ANALYSIS")
	}
	f := referenceFacts(t, true)
	in, want := frozenEvidence(t)
	var raw RawAnalysis
	if err := json.Unmarshal(readGzip(t, os.Getenv("TWR_LAB_ANALYSIS")), &raw); err != nil {
		t.Fatal(err)
	}
	if err := raw.Validate(); err != nil {
		t.Fatal(err)
	}
	a := Analysis(raw)
	before, stopRSS, err := sampleRSS()
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	body, err := compile(t.Context(), f, in, func(_ context.Context, _ *Lab, _ []Trade, _ []Pick, _, _ map[teamYear]int64) (Analysis, error) {
		return a, nil
	})
	rss := stopRSS()
	if rss.Err != nil {
		t.Fatal(rss.Err)
	}
	t.Logf("document RSS baseline=%d KiB peak=%d KiB rise=%d KiB (1ms samples, analysis decode excluded)", before, rss.Peak, rss.Peak-before)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("document compile %s", time.Since(start))
	status, err := os.ReadFile("/proc/self/status")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(status), "\n") {
		if strings.HasPrefix(line, "VmHWM:") {
			t.Log(line)
		}
	}
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	decoded, err := lab.DecodeGzip(compressed.Bytes(), hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Lab decode trades=%d legs=%d draft=%d", len(decoded.Trades), len(decoded.Legs), len(decoded.Draft))
	var got map[string]json.RawMessage
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range sortedRawKeys(want) {
		count, differences := compareJSON(t, got[key], want[key], key)
		t.Logf("P4a %s values=%d differences=%d", key, count, differences)
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			t.Errorf("unexpected key %s", key)
		}
	}
}
func sortedRawKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sortStrings(keys)
	return keys
}
func compareJSON(t *testing.T, a, b json.RawMessage, path string) (int, int) {
	t.Helper()
	a, b = bytes.TrimSpace(a), bytes.TrimSpace(b)
	if len(a) == 0 || len(b) == 0 {
		t.Errorf("%s missing value", path)
		return 0, 1
	}
	if a[0] == '{' && b[0] == '{' {
		return compareObjects(t, a, b, path)
	}
	if a[0] == '[' && b[0] == '[' {
		return compareArrays(t, a, b, path)
	}
	if !sameJSONScalar(a, b, path) {
		t.Errorf("%s got=%s want=%s", path, a, b)
		return 1, 1
	}
	return 1, 0
}
func compareObjects(t *testing.T, a, b json.RawMessage, path string) (int, int) {
	t.Helper()
	var am, bm map[string]json.RawMessage
	if err := json.Unmarshal(a, &am); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &bm); err != nil {
		t.Fatal(err)
	}
	n, d := 0, 0
	for _, key := range sortedRawKeys(bm) {
		nn, dd := compareJSON(t, am[key], bm[key], path+"."+key)
		n += nn
		d += dd
	}
	for key := range am {
		if _, ok := bm[key]; !ok {
			t.Errorf("%s unexpected key %s", path, key)
			d++
		}
	}
	return n, d
}
func compareArrays(t *testing.T, a, b json.RawMessage, path string) (int, int) {
	t.Helper()
	var aa, bb []json.RawMessage
	if err := json.Unmarshal(a, &aa); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &bb); err != nil {
		t.Fatal(err)
	}
	n, d := 0, 0
	if len(aa) != len(bb) {
		t.Errorf("%s length got=%d want=%d", path, len(aa), len(bb))
		d++
	}
	for i := 0; i < min(len(aa), len(bb)); i++ {
		nn, dd := compareJSON(t, aa[i], bb[i], fmt.Sprintf("%s[%d]", path, i))
		n += nn
		d += dd
	}
	return n, d
}
func sameJSONScalar(a, b []byte, path string) bool {
	if bytes.Equal(a, b) {
		return true
	}
	if a[0] == '"' && b[0] == '"' {
		var x, y string
		if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
			return false
		}
		return x == y
	}
	if a[0] == 'n' || b[0] == 'n' || a[0] == 't' || b[0] == 't' || a[0] == 'f' || b[0] == 'f' {
		return false
	}
	if (!bytes.ContainsAny(a, ".eE") && !bytes.ContainsAny(b, ".eE")) || strings.Contains(path, ".salary[") || strings.HasSuffix(path, ".cap") {
		ar, aok := new(big.Rat).SetString(string(a))
		br, bok := new(big.Rat).SetString(string(b))
		return aok && bok && ar.Cmp(br) == 0
	}
	var x, y float64
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return math.Abs(x-y) <= max(1e-9, 1e-9*max(math.Abs(x), math.Abs(y)))
}

func readWithin(root, file string) ([]byte, error) {
	f, err := os.OpenInRoot(root, file)
	if err != nil {
		return nil, fmt.Errorf("open input: %w", err)
	}
	body, readErr := io.ReadAll(f)
	closeErr := f.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read input: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close input: %w", closeErr)
	}
	return body, nil
}
