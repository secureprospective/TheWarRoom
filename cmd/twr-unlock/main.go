// twr-unlock binds a LUKS key to this host's TPM, using the Clevis wire format.
package main

import (
	"fmt"
	"io"
	"os"

	clevis "github.com/anatol/clevis.go"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		// Library errors can include input; never print key or binding contents.
		fmt.Fprintln(os.Stderr, "Disk key unavailable. Use the sealed backup passphrase from the Proxmox console.")
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output io.Writer) error {
	if len(args) != 1 || (args[0] != "seal" && args[0] != "unseal") {
		return fmt.Errorf("usage: twr-unlock seal|unseal (standard input/output only)")
	}
	data, err := io.ReadAll(io.LimitReader(input, 65537))
	if err != nil {
		return fmt.Errorf("read disk key: %w", err)
	}
	defer clear(data)
	if len(data) == 0 || len(data) > 65536 {
		return fmt.Errorf("invalid disk key input length")
	}
	var result []byte
	if args[0] == "seal" {
		if len(data) != 32 {
			return fmt.Errorf("disk key must contain 32 bytes")
		}
		result, err = clevis.Encrypt(data, "tpm2", `{"pcr_bank":"sha256","pcr_ids":"7"}`)
	} else {
		result, err = clevis.Decrypt(data)
	}
	if err != nil {
		return fmt.Errorf("bind or recover disk key: %w", err)
	}
	defer clear(result)
	if args[0] == "unseal" && len(result) != 32 {
		return fmt.Errorf("recovered disk key must contain 32 bytes")
	}
	if _, err := output.Write(result); err != nil {
		return fmt.Errorf("write disk key: %w", err)
	}
	return nil
}
