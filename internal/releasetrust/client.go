package releasetrust

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/theupdateframework/go-tuf/v2/metadata/config"
	"github.com/theupdateframework/go-tuf/v2/metadata/fetcher"
	"github.com/theupdateframework/go-tuf/v2/metadata/updater"
)

// Client owns a private persistent update cache, separate from native league data.
// Keep one Client per cache, under the desktop's single-instance lock. Never reset
// this cache on a verification error: its versions prevent rollback after restart.
type Client struct {
	mu        sync.Mutex
	bootstrap Bootstrap
	directory string
	fetcher   fetcher.Fetcher
}

// New uses system HTTPS trust, not a certificate leaf pin. Bootstrap must already
// have been authenticated by the install handoff. Cache contents are local trusted state.
func New(bootstrap Bootstrap, directory string) (*Client, error) {
	if err := bootstrap.Validate(); err != nil {
		return nil, err
	}
	if directory == "" {
		return nil, fmt.Errorf("choose a private update cache folder")
	}
	f := fetcher.NewDefaultFetcher()
	f.SetHTTPClient(&http.Client{Timeout: 30 * time.Second, CheckRedirect: publicRedirect})
	return &Client{bootstrap: Bootstrap{Config: bootstrap.Config, Root: append([]byte(nil), bootstrap.Root...)}, directory: directory, fetcher: f}, nil
}

func publicRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return fmt.Errorf("too many update redirects")
	}
	// Redirects include a filename, unlike bootstrap repository addresses.
	u := *req.URL
	u.Path, u.RawPath = "", ""
	raw := RawConfig{ServerURL: u.String(), MetadataURL: u.String(), TargetsURL: u.String()}
	return raw.Validate()
}

func (c *Client) updater() (*updater.Updater, error) {
	root, err := os.ReadFile(filepath.Join(c.directory, "metadata", "root.json"))
	if errors.Is(err, os.ErrNotExist) {
		// A missing root in an existing cache is damage, not permission to re-bootstrap.
		if _, statErr := os.Stat(filepath.Join(c.directory, "metadata")); !errors.Is(statErr, os.ErrNotExist) {
			return nil, fmt.Errorf("update trust cache is incomplete; restore it before updating")
		}
		root = c.bootstrap.Root
	} else if err != nil {
		return nil, fmt.Errorf("read saved update trust: %w", err)
	}
	cfg, err := config.New(c.bootstrap.Config.MetadataURL, root)
	if err != nil {
		return nil, fmt.Errorf("prepare update verification: %w", err)
	}
	cfg.LocalMetadataDir = filepath.Join(c.directory, "metadata")
	cfg.LocalTargetsDir = filepath.Join(c.directory, "targets")
	cfg.RemoteTargetsURL = c.bootstrap.Config.TargetsURL
	cfg.Fetcher = c.fetcher
	up, err := updater.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("open update verifier: %w", err)
	}
	return up, nil
}

// Download refreshes trust and returns a verified file in the update cache. It
// never installs, launches or applies configuration. A failed refresh returns no file.
func (c *Client) Download(name string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !safeTarget(name) {
		return "", fmt.Errorf("choose a release file name, not a path")
	}
	up, err := c.updater()
	if err != nil {
		return "", err
	}
	if err := up.Refresh(); err != nil {
		return "", fmt.Errorf("the update could not be verified; keep the installed version: %w", err)
	}
	info, err := up.GetTargetInfo(name)
	if err != nil {
		return "", fmt.Errorf("find signed release file: %w", err)
	}
	if info.Length < 1 || info.Length > 512*1024*1024 {
		return "", fmt.Errorf("release file size is outside the supported range")
	}
	path, _, err := up.DownloadTarget(info, "", "")
	if err != nil {
		return "", fmt.Errorf("the release file could not be verified: %w", err)
	}
	return path, nil
}

func safeTarget(name string) bool {
	if len(name) == 0 || len(name) > 160 || strings.Contains(name, "..") {
		return false
	}
	for _, c := range name {
		if !targetCharacter(c) {
			return false
		}
	}
	return name[0] != '.'
}

func targetCharacter(c rune) bool {
	return publicNameCharacter(c) || c >= 'A' && c <= 'Z' || c == '_' || c == '.'
}
