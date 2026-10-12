package lab

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestInputsAreTheFrozenDataSourceFiles(t *testing.T) {
	in, err := Inputs()
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != 1482 || in[0].File != "2013/aav.json" || in[0].Bytes != 98 {
		t.Fatalf("inputs = %d, first %+v; want 1482 starting 2013/aav.json", len(in), in[0])
	}
	if _, err := decodeInputs(append([]byte(nil), inputsCSV[:len(inputsCSV)-1]...), InputsSHA); err == nil {
		t.Error("a changed inputs list decoded under the frozen fingerprint")
	}
	bad := "file,bytes,sha256\n2013/x.json,1,abc\n"
	if _, err := decodeInputs([]byte(bad), sha(bad)); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("err = %v, want the bad line named", err)
	}
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
