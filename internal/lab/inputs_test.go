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

func TestFrozenGapsAreExactNonDataPins(t *testing.T) {
	gaps, err := FrozenGaps()
	if err != nil || len(gaps) != 5 {
		t.Fatalf("gaps = %+v, %v", gaps, err)
	}
	if gaps[1].SHA256 != gaps[2].SHA256 || gaps[1].File != "2014/assets.json" || gaps[2].File != "2014/calendar.json" {
		t.Fatal("2014 assets and calendar must retain their shared frozen hash")
	}
	inputs, err := Inputs()
	if err != nil {
		t.Fatal(err)
	}
	bad := []byte("file,sha256,reason\n2013/assets.json," + gaps[0].SHA256 + ",reason\n2013/assets.json," + gaps[0].SHA256 + ",reason\n")
	if _, err := decodeGaps(bad, inputs); err == nil {
		t.Fatal("duplicate gap identity accepted")
	}
	if _, err := decodeGaps(missingInputsCSV, nil); err == nil {
		t.Fatal("gaps without matching frozen pins accepted")
	}
}
