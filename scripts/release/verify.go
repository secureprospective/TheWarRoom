// Command verify checks a published release using the desktop's TUF verifier.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/secureprospective/TheWarRoom/internal/releasetrust"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	rootPath := flags.String("root", "", "independently authenticated public root file")
	configPath := flags.String("config", "", "public client configuration file")
	cache := flags.String("cache", "", "persistent private update cache folder")
	target := flags.String("target", "", "release file to verify and download")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("read verification options: %w", err)
	}
	if flags.NArg() != 0 || *rootPath == "" || *configPath == "" || *cache == "" || *target == "" {
		return fmt.Errorf("provide -root, -config, -cache and -target")
	}
	root, err := os.ReadFile(*rootPath)
	if err != nil {
		return fmt.Errorf("read public trust root: %w", err)
	}
	data, err := os.ReadFile(*configPath)
	if err != nil {
		return fmt.Errorf("read public configuration: %w", err)
	}
	config, err := releasetrust.DecodeConfig(data)
	if err != nil {
		return fmt.Errorf("verify published release: %w", err)
	}
	client, err := releasetrust.New(releasetrust.Bootstrap{Config: config, Root: root}, *cache)
	if err != nil {
		return fmt.Errorf("verify published release: %w", err)
	}
	path, err := client.Download(*target)
	if err != nil {
		return fmt.Errorf("verify published release: %w", err)
	}
	if _, err := fmt.Fprintf(output, "Verified release file: %s\nNothing was installed or launched.\n", path); err != nil {
		return fmt.Errorf("report verified file: %w", err)
	}
	return nil
}
