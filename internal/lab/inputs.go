package lab

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"regexp"
	"slices"
	"strconv"
)

// InputsSHA fingerprints the list of archive files the frozen Lab data was compiled from.
const InputsSHA = "e1604241e10f6bc78a1aa212a5ae4a42cb4e46f43bf0b6ce386237232d4080aa"

//go:embed testdata/inputs.csv
var inputsCSV []byte

// Input is one archive file the frozen Lab data read, by its path under the archive root.
type Input struct {
	File   string
	Bytes  int64
	SHA256 string
}

func shaPattern() *regexp.Regexp { return regexp.MustCompile(`^[0-9a-f]{64}$`) }

// Inputs returns the frozen Lab data's input files, checked against their fingerprint.
func Inputs() ([]Input, error) { return decodeInputs(inputsCSV, InputsSHA) }

func decodeInputs(b []byte, wantSHA string) ([]Input, error) {
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != wantSHA {
		return nil, fmt.Errorf("lab inputs fingerprint mismatch")
	}
	rows, err := csv.NewReader(bytes.NewReader(b)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read lab inputs: %w", err)
	}
	if len(rows) == 0 || !slices.Equal(rows[0], []string{"file", "bytes", "sha256"}) {
		return nil, fmt.Errorf("lab inputs: header is not file,bytes,sha256")
	}
	sha := shaPattern()
	out := make([]Input, 0, len(rows)-1)
	for i, r := range rows[1:] {
		n, err := strconv.ParseInt(r[1], 10, 64)
		if r[0] == "" || err != nil || n < 0 || !sha.MatchString(r[2]) {
			return nil, fmt.Errorf("lab inputs line %d: %q is not a file, size and sha256", i+2, r)
		}
		out = append(out, Input{File: r[0], Bytes: n, SHA256: r[2]})
	}
	return out, nil
}
