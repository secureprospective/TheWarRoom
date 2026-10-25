package leaguetransfer

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

const maxExportBytes = 64 << 20
const envelope = "TWR-LEAGUE-XCHACHA20POLY1305-v1\n"

// Encrypt authenticates the whole bounded document before any plaintext is released to a reader.
// A random 192-bit nonce allows independent exports with the same offline key.
func Encrypt(ctx context.Context, output io.Writer, raw RawExport, key [32]byte) error {
	if err := raw.Validate(ctx); err != nil {
		return err
	}
	if key == [32]byte{} {
		return fmt.Errorf("choose a random, nonempty league export key")
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("encode league export: %w", err)
	}
	if len(body) > maxExportBytes {
		return fmt.Errorf("the league export exceeds the 64 MiB safety limit")
	}
	cipher, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return fmt.Errorf("prepare league encryption: %w", err)
	}
	nonce := make([]byte, cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("create league encryption nonce: %w", err)
	}
	data := append([]byte(envelope), nonce...)
	data = cipher.Seal(data, nonce, body, []byte(envelope))
	if n, err := output.Write(data); err != nil {
		return fmt.Errorf("save encrypted league: %w", err)
	} else if n != len(data) {
		return fmt.Errorf("save encrypted league: %w", io.ErrShortWrite)
	}
	return nil
}

// Decrypt checks authentication before JSON decoding, then checks every referenced body and replay.
func Decrypt(ctx context.Context, input io.Reader, key [32]byte) (RawExport, error) {
	var raw RawExport
	data, err := io.ReadAll(io.LimitReader(input, maxExportBytes+1024))
	if err != nil {
		return raw, fmt.Errorf("read encrypted league: %w", err)
	}
	if len(data) > maxExportBytes+len(envelope)+chacha20poly1305.NonceSizeX+chacha20poly1305.Overhead || len(data) < len(envelope)+chacha20poly1305.NonceSizeX+chacha20poly1305.Overhead || string(data[:len(envelope)]) != envelope {
		return raw, fmt.Errorf("this is not a supported encrypted league export")
	}
	cipher, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return raw, fmt.Errorf("prepare league decryption: %w", err)
	}
	nonce := data[len(envelope) : len(envelope)+cipher.NonceSize()]
	body, err := cipher.Open(nil, nonce, data[len(envelope)+cipher.NonceSize():], []byte(envelope))
	if err != nil {
		return raw, fmt.Errorf("the export key is wrong or this file has changed; ask for the original file and key")
	}
	return decode(ctx, body)
}
