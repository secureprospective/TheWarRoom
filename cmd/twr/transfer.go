package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguetransfer"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/spf13/cobra"
)

type exportOptions struct {
	path, league, epoch, football, keyFile, output string
	additional                                     []string
}

func (e env) transferCommand() *cobra.Command {
	command := &cobra.Command{Use: "league-export", Short: "Save or check an encrypted copy for moving a league; does not move authority"}
	var options exportOptions
	save := &cobra.Command{Use: "save", Short: "Save an encrypted league from this computer (keep the source generation UUID)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return e.saveExport(cmd, options) }}
	save.Flags().StringVar(&options.path, "db", "", "existing league database on this computer")
	save.Flags().StringVar(&options.league, "league", "", "league ID (native.…)")
	save.Flags().StringVar(&options.epoch, "source-generation", "", "source authority generation UUID; keep it for this move")
	save.Flags().StringVar(&options.football, "football", "", "selected self-contained football evidence digest")
	save.Flags().StringSliceVar(&options.additional, "past-football", nil, "pinned evidence digests for previous seasons (comma separated)")
	save.Flags().StringVar(&options.keyFile, "key-file", "", "private file containing a 64-character random hexadecimal key")
	save.Flags().StringVar(&options.output, "output", "", "new encrypted export filename")
	var checkFile, checkKey string
	check := &cobra.Command{Use: "check", Short: "Check an encrypted league without importing or activating it", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return e.checkExport(cmd, checkFile, checkKey) }}
	check.Flags().StringVar(&checkFile, "file", "", "encrypted league filename")
	check.Flags().StringVar(&checkKey, "key-file", "", "private export key file")
	command.AddCommand(save, check)
	return command
}

func (e env) saveExport(cmd *cobra.Command, options exportOptions) error {
	key, err := readExportKey(options.keyFile)
	if err != nil {
		return err
	}
	info, err := os.Lstat(options.path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("choose an existing league database; links and missing files cannot be exported")
	}
	pools, err := db.Open(cmd.Context(), options.path, history.SchemaVersion())
	if err != nil {
		return fmt.Errorf("open the league to export: %w", err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		return fmt.Errorf("load league measures: %w", err)
	}
	// Like matched-set backups, hold the single writer while reading linked identities and history.
	writer, err := pools.Write().Conn(cmd.Context())
	if err != nil {
		return fmt.Errorf("pause changes for export: %w", err)
	}
	defer func() { _ = writer.Close() }()
	store := history.New(pools, registry)
	linked, err := accountstore.New(pools).Aliases(cmd.Context())
	if err != nil {
		return fmt.Errorf("read the league's linked identities: %w", err)
	}
	aliases := make([]governance.VerifiedAlias, 0, len(linked))
	for _, alias := range linked {
		aliases = append(aliases, governance.VerifiedAlias{League: options.league, Alias: alias.Alias, Account: alias.Account})
	}
	raw, err := leaguetransfer.Capture(cmd.Context(), store, options.league, options.epoch, options.football, time.Now().UTC(), aliases, options.additional...)
	if err != nil {
		return fmt.Errorf("prepare league export: %w", err)
	}
	if err := writeExport(cmd, options.output, raw, key); err != nil {
		return err
	}
	_, err = fmt.Fprintln(e.out, "Encrypted league saved. The source is still writable; this file does not activate another server.")
	if err != nil {
		return fmt.Errorf("report league export: %w", err)
	}
	return nil
}

func writeExport(cmd *cobra.Command, path string, raw leaguetransfer.RawExport, key [32]byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("choose a new export filename: %w", err)
	}
	complete := false
	defer func() {
		_ = file.Close()
		if !complete {
			_ = os.Remove(path)
		}
	}()
	if err := leaguetransfer.Encrypt(cmd.Context(), file, raw, key); err != nil {
		return fmt.Errorf("encrypt league export: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("save league export to disk: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close league export: %w", err)
	}
	complete = true
	return nil
}

func (e env) checkExport(cmd *cobra.Command, path, keyPath string) error {
	key, err := readExportKey(keyPath)
	if err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open encrypted league: %w", err)
	}
	defer func() { _ = file.Close() }()
	raw, err := leaguetransfer.Decrypt(cmd.Context(), file, key)
	if err != nil {
		return fmt.Errorf("check encrypted league: %w", err)
	}
	_, err = fmt.Fprintf(e.out, "League %s matches its saved roster, cap, lineups, bids, results and standings. Nothing was imported or activated.\n", raw.League)
	if err != nil {
		return fmt.Errorf("report export check: %w", err)
	}
	return nil
}

func readExportKey(path string) ([32]byte, error) {
	var key [32]byte
	file, err := os.Open(path)
	if err != nil {
		return key, fmt.Errorf("open the private export key file: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return key, fmt.Errorf("check the private export key file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return key, fmt.Errorf("keep the export key in a private file (permissions 0600)")
	}
	data, err := io.ReadAll(io.LimitReader(file, 67))
	if err != nil {
		return key, fmt.Errorf("read the export key: %w", err)
	}
	decoded, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(decoded) != 32 {
		return key, fmt.Errorf("the export key must contain exactly 64 hexadecimal characters")
	}
	copy(key[:], decoded)
	if key == [32]byte{} {
		return key, fmt.Errorf("use a randomly generated export key, not an empty key")
	}
	return key, nil
}
