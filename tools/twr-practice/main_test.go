package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

type practiceClient struct {
	address string
	http    *http.Client
	bearer  string
	proof   string
}

func (c *practiceClient) post(ctx context.Context, path string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.address+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	if c.proof != "" {
		req.Header.Set("X-TWR-Recent-Auth", c.proof)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func TestPracticeUsage(t *testing.T) {
	err := run(t.Context(), nil, &bytes.Buffer{})
	if err == nil || err.Error() != "usage: twr-practice <up [-bin directory]|code|down> <throwaway-directory>" {
		t.Fatalf("unexpected usage: %v", err)
	}
}

func sandboxHTTPSLifecycle(t *testing.T, bin string) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	if bin == "" {
		// Build from the repository root just as the documented invocation does.
		t.Chdir("../..")
	}
	dir := filepath.Join(t.TempDir(), "sandbox")
	var output bytes.Buffer
	args := []string{"up", dir}
	launcher := "go run ./tools/twr-practice"
	if bin != "" {
		args = []string{"up", "-bin", bin, dir}
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		launcher = quote(executable)
	}
	if err := run(ctx, args, &output); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := os.Stat(dir); err == nil {
			if err := down(context.Background(), dir); err != nil {
				t.Error(err)
			}
		}
	})
	if bin != "" {
		assertCopiedBinaries(t, dir, bin)
	}
	state, err := readState(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{state.Address, state.Password, state.Seed, launcher + " code " + quote(dir), launcher + " down " + quote(dir)} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("missing launch instruction %q", want)
		}
	}
	pem, err := os.ReadFile(filepath.Join(dir, "server.pem"))
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pem) {
		t.Fatal("invalid certificate")
	}
	client := &practiceClient{
		address: state.Address,
		http:    &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}},
	}
	defer client.http.CloseIdleConnections()
	loginBody, err := client.post(ctx, serverhttp.LoginPath, serverhttp.RawLogin{
		Protocol: 1, Username: "commissioner", Password: state.Password, Device: "practice test",
	})
	if err != nil {
		t.Fatalf("HTTPS login: %v", err)
	}
	var login struct {
		Bearer string `json:"bearer"`
	}
	if err := json.Unmarshal(loginBody, &login); err != nil || login.Bearer == "" {
		t.Fatalf("HTTPS login reply: %s %v", loginBody, err)
	}
	client.bearer = login.Bearer
	code, err := hostedfixture.Code(state.Seed, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	authBody, err := client.post(ctx, serverhttp.TOTPStepUpPath, serverhttp.RawAuthenticator{
		Protocol: 1, Username: "commissioner", Password: state.Password, Code: code,
	})
	if err != nil {
		t.Fatalf("HTTPS authenticator: %v", err)
	}
	var auth serverhttp.AuthenticatorResponse
	if err := json.Unmarshal(authBody, &auth); err != nil || auth.RecentAuth == "" {
		t.Fatalf("HTTPS authenticator reply: %s %v", authBody, err)
	}
	client.proof = auth.RecentAuth
	enter := func() (string, error) {
		body, err := client.post(ctx, serverhttp.PeoplePath, serverhttp.RawNativeRequest{Protocol: 1, League: "native.hosted"})
		if err != nil {
			return "", err
		}
		var view struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &view); err != nil {
			return "", err
		}
		return view.Name, nil
	}
	if name, err := enter(); err != nil || name != "Practice league" {
		t.Fatalf("enter hosted league: %q %v", name, err)
	}
	if err := run(ctx, []string{"code", dir}, &output); err != nil {
		t.Fatal(err)
	}
	if err := run(ctx, args, &output); err == nil {
		t.Fatal("accepted nonempty directory")
	}
	if _, err := enter(); err != nil {
		t.Fatalf("refusal damaged running sandbox: %v", err)
	}
	if err := run(ctx, []string{"down", dir}, &output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("directory retained: %v", err)
	}
	if _, err := enter(); err == nil {
		t.Fatal("server still serving after down")
	}
}

func TestSandboxRefusesUnownedPaths(t *testing.T) {
	dir := t.TempDir()
	sentinel := filepath.Join(dir, "keep")
	if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(t.Context(), []string{"up", dir}, &bytes.Buffer{}); err == nil {
		t.Fatal("accepted occupied directory")
	}
	if err := down(t.Context(), dir); err == nil {
		t.Fatal("deleted unmarked directory")
	}
	if data, err := os.ReadFile(sentinel); err != nil || string(data) != "keep" {
		t.Fatal("damaged existing directory")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	if _, err := sandboxPath(link); err == nil {
		t.Fatal("accepted symlink")
	}
	if _, err := sandboxProcess(dir, os.Getpid()); err == nil {
		t.Fatal("accepted unrelated process")
	}
}

func TestAuthenticatorRFC6238(t *testing.T) {
	got, err := hostedfixture.Code("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Unix(59, 0))
	if err != nil || got != "287082" {
		t.Fatalf("RFC vector: %s %v", got, err)
	}
}

func TestSandboxRejectsRealProfiles(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(home, "league-archive"), filepath.Join(home, "league-archive", "sandbox"), filepath.Join(home, ".config", "TheWarRoom")} {
		if _, err := sandboxPath(path); err == nil {
			t.Fatalf("accepted real profile path %s", path)
		}
	}
}

func TestStartupFailureRemovesClaimedDirectory(t *testing.T) {
	t.Chdir(t.TempDir()) // No registry: preparation must fail before any process starts.
	dir := filepath.Join(t.TempDir(), "failed")
	if err := up(t.Context(), dir, "", &bytes.Buffer{}); err == nil {
		t.Fatal("unexpected startup success")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("failed startup left partial sandbox: %v", err)
	}
}

func TestPrebuiltSandboxHTTPSLifecycleWithoutGo(t *testing.T) {
	t.Chdir("../..")
	bin := t.TempDir()
	for _, args := range [][]string{
		{"build", "-tags", "practice", "-o", filepath.Join(bin, "twr-server"), "./cmd/twr-server"},
	} {
		if output, err := exec.CommandContext(t.Context(), "go", args...).CombinedOutput(); err != nil {
			t.Fatalf("build bundle: %v: %s", err, output)
		}
	}
	registry, err := os.ReadFile("docs/ui/endpoint-registry.csv")
	if err != nil {
		t.Fatal(err)
	}
	if err := writePrivate(filepath.Join(bin, "registry.csv"), registry); err != nil {
		t.Fatal(err)
	}
	openssl, err := exec.LookPath("openssl")
	if err != nil {
		t.Fatal(err)
	}
	tools := t.TempDir()
	if err := os.Symlink(openssl, filepath.Join(tools, "openssl")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tools)
	if _, err := exec.LookPath("go"); err == nil {
		t.Fatal("Go still available")
	}
	t.Chdir(t.TempDir()) // No checkout or relative registry on the test machine.
	sandboxHTTPSLifecycle(t, bin)
}

func TestPrebuiltSandboxRefusesInvalidBinaries(t *testing.T) {
	for _, name := range []string{"twr-server"} {
		for _, kind := range []string{"missing", "not executable", "directory"} {
			t.Run(name+"/"+kind, func(t *testing.T) {
				bin := t.TempDir()
				if err := writePrivate(filepath.Join(bin, "registry.csv"), []byte("unused")); err != nil {
					t.Fatal(err)
				}
				for _, executable := range []string{"twr-server"} {
					path := filepath.Join(bin, executable)
					if executable == name && kind == "missing" {
						continue
					}
					if executable == name && kind == "directory" {
						if err := os.Mkdir(path, 0700); err != nil {
							t.Fatal(err)
						}
						continue
					}
					mode := os.FileMode(0700)
					if executable == name && kind == "not executable" {
						mode = 0600
					}
					if err := os.WriteFile(path, []byte("preserve bundle"), mode); err != nil {
						t.Fatal(err)
					}
				}
				dir := filepath.Join(t.TempDir(), "sandbox")
				err := run(t.Context(), []string{"up", "-bin", bin, dir}, &bytes.Buffer{})
				if err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "executable") {
					t.Fatalf("expected plain executable refusal: %v", err)
				}
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Fatalf("partial sandbox retained: %v", err)
				}
				if data, err := os.ReadFile(filepath.Join(bin, "registry.csv")); err != nil || string(data) != "unused" {
					t.Fatalf("bundle damaged: %s %v", data, err)
				}
			})
		}
	}
}

func assertCopiedBinaries(t *testing.T, dir, bin string) {
	t.Helper()
	for _, name := range []string{"twr-server"} {
		source, err := os.ReadFile(filepath.Join(bin, name))
		if err != nil {
			t.Fatal(err)
		}
		copied, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(source, copied) {
			t.Fatalf("%s differs from bundle", name)
		}
		sourceInfo, err := os.Stat(filepath.Join(bin, name))
		if err != nil {
			t.Fatal(err)
		}
		copiedInfo, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if os.SameFile(sourceInfo, copiedInfo) || copiedInfo.Mode().Perm() != 0700 {
			t.Fatalf("%s is not an independent owner-only executable copy", name)
		}
	}
}
