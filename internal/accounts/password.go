// Package accounts owns account credentials, independently of league grants.
package accounts

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const hashPrefix = "$argon2id$v=19$m=65536,t=3,p=1$"

func validatePassword(password string) error {
	if len(password) == 0 || len(password) > 1024 || !utf8.ValidString(password) || strings.ContainsRune(password, 0) {
		return fmt.Errorf("password must be nonempty UTF-8 text of at most 1024 bytes")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, 32)
	return hashPrefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash), nil
}

// DecodePasswordHash accepts only the fixed resource budget this server writes.
func DecodePasswordHash(encoded string) (salt, hash []byte, err error) {
	if len(encoded) != len(hashPrefix)+22+1+43 || !strings.HasPrefix(encoded, hashPrefix) {
		return nil, nil, fmt.Errorf("invalid stored password hash")
	}
	parts := strings.Split(strings.TrimPrefix(encoded, hashPrefix), "$")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid stored password hash")
	}
	salt, saltErr := base64.RawStdEncoding.Strict().DecodeString(parts[0])
	hash, hashErr := base64.RawStdEncoding.Strict().DecodeString(parts[1])
	if saltErr != nil || hashErr != nil || len(salt) != 16 || len(hash) != 32 {
		return nil, nil, fmt.Errorf("invalid stored password hash")
	}
	return salt, hash, nil
}

func VerifyPassword(encoded, password string) (bool, error) {
	salt, expected, err := DecodePasswordHash(encoded)
	if err != nil {
		return false, err
	}
	if err := validatePassword(password); err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
