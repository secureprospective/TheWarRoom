package accounts

import (
	"testing"
	"time"
)

func TestTOTPRFC6238SHA1Profile(t *testing.T) {
	// RFC 6238 appendix B's eight-digit values, reduced to the six-digit profile.
	for _, tt := range []struct {
		second int64
		code   string
	}{
		{59, "287082"}, {1111111109, "081804"}, {1111111111, "050471"},
		{1234567890, "005924"}, {2000000000, "279037"}, {20000000000, "353130"},
	} {
		if got := totpCode([]byte("12345678901234567890"), tt.second/30); got != tt.code {
			t.Fatalf("%d: %s", tt.second, got)
		}
	}
	seed := []byte("12345678901234567890")
	now := time.Unix(1234567890, 0)
	for delta := int64(-2); delta <= 2; delta++ {
		counter := now.Unix()/30 + delta
		got := matchingCounter(seed, totpCode(seed, counter), now)
		if (got == counter) != (delta >= -1 && delta <= 1) {
			t.Fatalf("window %d: %d", delta, got)
		}
	}
}

func TestRawTOTPValidation(t *testing.T) {
	for _, code := range []string{"", "12345", "1234567", " 12345", "１２３４５６", "12345a", "-12345"} {
		if err := (RawTOTP{Code: code}).Validate(); err == nil {
			t.Fatal("invalid code accepted")
		}
	}
	if err := (RawTOTP{Code: "000001"}).Validate(); err != nil {
		t.Fatal(err)
	}
}
