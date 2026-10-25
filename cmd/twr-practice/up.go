package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func up(ctx context.Context, dir, bin string, out io.Writer) (result error) {
	if err := claimDirectory(dir); err != nil {
		return err
	}
	var child *exec.Cmd
	defer func() {
		if result == nil {
			return
		}
		if child != nil && child.Process != nil {
			_ = child.Process.Kill()
			_ = child.Wait()
		}
		if err := os.RemoveAll(dir); err != nil { //nolint:gosec // Cleanup is registered only after successfully claiming an empty disposable directory.
			result = fmt.Errorf("%w; cleanup: %v", result, err)
		}
	}()
	if err := prepareFiles(ctx, dir, bin); err != nil {
		return err
	}
	password, seed, err := provision(ctx, dir)
	if err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(dir, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600) //nolint:gosec // Exclusive file inside the claimed empty directory.
	if err != nil {
		return fmt.Errorf("create server log: %w", err)
	}
	defer func() { _ = log.Close() }()
	child, err = practiceServer(dir)
	if err != nil {
		return err
	}
	child.Stdout, child.Stderr = log, log
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := child.Start(); err != nil {
		return fmt.Errorf("start hosted server: %w", err)
	}
	state := rawState{PID: child.Process.Pid, Password: password, Seed: seed, Directory: dir}
	if err := saveState(dir, state); err != nil {
		return err
	}
	state.Address, err = awaitAddress(ctx, dir)
	if err != nil {
		return err
	}
	if err := saveState(dir, state); err != nil {
		return err
	}
	if result = report(out, dir, state, bin); result == nil {
		go func() { _ = child.Wait() }()
	}
	return result
}

func practiceServer(dir string) (*exec.Cmd, error) {
	digest, err := os.ReadFile(filepath.Join(dir, "football.digest")) //nolint:gosec // Fixed file inside the claimed private sandbox.
	if err != nil {
		return nil, fmt.Errorf("read practice football digest: %w", err)
	}
	child := exec.Command(filepath.Join(dir, "twr-server"), "hosted", "-football", string(digest), "-db", filepath.Join(dir, "server.db"), "-listen", "127.0.0.1:0", "-tls-cert", filepath.Join(dir, "server.pem"), "-tls-key", filepath.Join(dir, "server.key"), "-totp-key-file", filepath.Join(dir, "authenticator.key"), "-preview-registry", filepath.Join(dir, "registry.csv")) //nolint:noctx,gosec // Fixed executable in the claimed sandbox, no shell; down owns the detached lifetime.
	return child, nil
}

func claimDirectory(dir string) error {
	if err := os.Mkdir(dir, 0700); err != nil { //nolint:gosec // Explicit canonicalized sandbox path; real profile locations are rejected.
		if !os.IsExist(err) {
			return fmt.Errorf("create sandbox: %w", err)
		}
		entries, readErr := os.ReadDir(dir)
		if readErr != nil || len(entries) != 0 {
			return fmt.Errorf("refusing existing non-empty sandbox directory")
		}
		if err := os.Chmod(dir, 0700); err != nil { //nolint:gosec // Directories require owner traversal, not file-mode 0600.
			return fmt.Errorf("protect sandbox: %w", err)
		}
	}
	return nil
}

func prepareFiles(ctx context.Context, dir, bin string) error {
	for _, name := range []string{"home", "config", "data", "cache", "runtime"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0700); err != nil { //nolint:gosec // Fixed subdirectory names inside the claimed sandbox.
			return fmt.Errorf("create isolated profile: %w", err)
		}
	}
	registryPath := "docs/ui/endpoint-registry.csv"
	if bin != "" {
		registryPath = filepath.Join(bin, "registry.csv")
	}
	registry, err := os.ReadFile(registryPath)
	if err != nil {
		return fmt.Errorf("read preview registry %s: %w", registryPath, err)
	}
	if err := writePrivate(filepath.Join(dir, "registry.csv"), registry); err != nil {
		return err
	}
	commands := [][]string{
		{"openssl", "req", "-x509", "-newkey", "ec", "-pkeyopt", "ec_paramgen_curve:P-256", "-nodes", "-keyout", filepath.Join(dir, "server.key"), "-out", filepath.Join(dir, "server.pem"), "-days", "1", "-subj", "/CN=Disposable loopback", "-addext", "subjectAltName=IP:127.0.0.1"},
		{"go", "build", "-tags", "practice", "-o", filepath.Join(dir, "twr-server"), "./cmd/twr-server"},
		{"go", "build", "-tags", "webkit2_41", "-o", filepath.Join(dir, "thewarroom"), "."},
	}
	if bin != "" {
		if err := copyBinaries(dir, bin); err != nil {
			return err
		}
		commands = commands[:1]
	}
	for _, args := range commands {
		if output, err := exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput(); err != nil { //nolint:gosec // Fixed build/TLS commands with argument arrays, never a shell.
			return fmt.Errorf("prepare %s: %w: %s", args[0], err, output)
		}
	}
	return nil
}

func awaitAddress(ctx context.Context, dir string) (string, error) {
	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		data, err := os.ReadFile(filepath.Join(dir, "server.log")) //nolint:gosec // Fixed startup log inside the claimed sandbox.
		if err != nil {
			return "", fmt.Errorf("read startup log: %w", err)
		}
		address, err := practiceOrigin(string(data))
		if err != nil {
			return "", err
		}
		if address != "" {
			return address, nil
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("wait for hosted server: %w", ctx.Err())
		case <-timer.C:
			return "", fmt.Errorf("hosted server did not start; log: %s", data)
		case <-ticker.C:
		}
	}
}

func copyBinaries(dir, bin string) error {
	for _, name := range []string{"twr-server", "thewarroom"} {
		source := filepath.Join(bin, name)
		file, err := os.Open(source)
		if err != nil {
			return fmt.Errorf("open prebuilt %s (provide an executable file): %w", source, err)
		}
		err = copyExecutable(dir, name, file)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return fmt.Errorf("close prebuilt %s: %w", source, closeErr)
		}
	}
	return nil
}

func copyExecutable(dir, name string, source *os.File) error {
	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("check prebuilt %s: %w", source.Name(), err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("prebuilt %s must be a regular executable file; check its permissions", source.Name())
	}
	destination, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700) //nolint:gosec // Exclusive executable in the claimed empty sandbox.
	if err != nil {
		return fmt.Errorf("create sandbox executable: %w", err)
	}
	_, copyErr := io.Copy(destination, source)
	closeErr := destination.Close()
	if copyErr != nil {
		return fmt.Errorf("copy prebuilt %s: %w", source.Name(), copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close sandbox executable: %w", closeErr)
	}
	return nil
}

func practiceOrigin(output string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "https://") {
			continue
		}
		parsed, err := url.Parse(line)
		if err != nil || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" {
			return "", fmt.Errorf("server reported invalid loopback origin")
		}
		return line, nil
	}
	return "", nil
}
