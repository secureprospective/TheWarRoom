package accounts

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestPasswordHashAndVerification(t *testing.T) {
	const password = "a correct horse battery staple"
	first, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || strings.Contains(first, password) {
		t.Fatal("password salt was reused or plaintext stored")
	}
	salt, hash, err := DecodePasswordHash(first)
	if err != nil {
		t.Fatal(err)
	}
	if len(salt) != 16 || len(hash) != 32 {
		t.Fatal("wrong salt or hash length")
	}
	expected := argon2.IDKey([]byte(password), salt, 3, 65536, 1, 32)
	if base64.RawStdEncoding.EncodeToString(expected) != base64.RawStdEncoding.EncodeToString(hash) {
		t.Fatal("hash does not match ruled Argon2id parameters")
	}
	for _, candidate := range []string{password, "wrong password", strings.ToUpper(password)} {
		valid, err := VerifyPassword(first, candidate)
		if err != nil || valid != (candidate == password) {
			t.Fatalf("verification: valid=%v error=%v", valid, err)
		}
	}
}

func TestMalformedHashesFailClosed(t *testing.T) {
	salt := base64.RawStdEncoding.EncodeToString(make([]byte, 16))
	hash := base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	valid := hashPrefix + salt + "$" + hash
	cases := []string{
		"", "plaintext", valid + "x", strings.Repeat("x", 10000),
		strings.Replace(valid, "argon2id", "argon2i", 1),
		strings.Replace(valid, "v=19", "v=20", 1),
		strings.Replace(valid, "m=65536", "m=99999", 1),
		strings.Replace(valid, "t=3", "t=0", 1),
		strings.Replace(valid, "p=1", "p=0", 1),
		hashPrefix + salt + hash + "x",
		hashPrefix + salt + "$" + strings.Repeat("!", 43),
		hashPrefix + salt + "$" + hash[:42] + "B",
		hashPrefix + salt + "$$" + hash[:42],
		hashPrefix + salt[:21] + "$" + hash + "x",
	}
	for i, encoded := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			if ok, err := VerifyPassword(encoded, "secret"); ok || err == nil {
				t.Fatal("malformed hash accepted")
			}
		})
	}
}

func FuzzDecodePasswordHash(f *testing.F) {
	f.Add("")
	f.Add(hashPrefix + strings.Repeat("A", 66))
	f.Add(hashPrefix + strings.Repeat("A", 22) + "$" + strings.Repeat("A", 43))
	f.Fuzz(func(t *testing.T, encoded string) {
		salt, hash, err := DecodePasswordHash(encoded)
		if err == nil && (len(salt) != 16 || len(hash) != 32) {
			t.Fatal("invalid hash decoded")
		}
	})
}

func TestRawCredentialsValidationAndRedaction(t *testing.T) {
	for _, raw := range []RawCredentials{
		{Username: "ab", Password: "secret"}, {Username: " owner", Password: "secret"},
		{Username: "ownér", Password: "secret"}, {Username: "owner", Password: ""},
		{Username: "Kowner", Password: "secret"}, {Username: strings.Repeat("a", 65), Password: "secret"},
		{Username: "owner", Password: strings.Repeat("x", 1025)}, {Username: "owner", Password: "bad\x00secret"},
		{Username: "owner", Password: "\xff"},
	} {
		if err := raw.Validate(); err == nil {
			t.Fatal("invalid credentials accepted")
		}
	}
	raw := RawCredentials{Username: "Owner.One", Password: "sentinel-password-123"}
	if err := raw.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"%v", "%+v", "%#v"} {
		if strings.Contains(fmt.Sprintf(format, raw), raw.Password) {
			t.Fatal("password exposed by formatting")
		}
	}
}
