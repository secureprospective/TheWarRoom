package leaguearchive

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/secureprospective/TheWarRoom/internal/archive"
)

const (
	readArchiveMessage  = "The archive could not be read. Check that the folder and its files are available, then try again."
	archiveLinkMessage  = "The archive contains a link or something other than a normal file or folder. Choose a folder containing the actual archive files, not links."
	archiveLimitMessage = "The archive is too large or has too many files or nested folders to import safely. Choose a smaller archive with season folders directly inside it."
)

// DiskFS bounds one disk import, including repeated discovery reads. It is not safe for concurrent use.
// Opens stay inside a pinned directory handle, even if paths are replaced during import.
type DiskFS struct {
	root                 string
	fileLimit, remaining int64
	files, entries       int
}

func NewDiskFS(root string) *DiskFS {
	return &DiskFS{root: root, fileLimit: 256 << 20, remaining: 512 << 20, files: 10000, entries: 20000}
}

func (g *DiskFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: invalid path", name), "An archive file is outside the allowed folder layout. Choose an archive with season folders directly inside it.")
	}
	parts := strings.Split(name, "/")
	if name != "." && len(parts) > 3 {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: directory depth exceeded", name), archiveLimitMessage)
	}
	rootInfo, fileInfo, err := g.pathInfo(name, parts)
	if err != nil {
		return nil, err
	}
	return g.openChecked(name, rootInfo, fileInfo)
}

func (g *DiskFS) pathInfo(name string, parts []string) (fs.FileInfo, fs.FileInfo, error) {
	disk := g.root
	var rootInfo, fileInfo fs.FileInfo
	for i := -1; i < len(parts); i++ {
		if i >= 0 {
			disk = filepath.Join(disk, parts[i])
		}
		info, err := os.Lstat(disk)
		if err != nil {
			return nil, nil, archive.WithMessage(fmt.Errorf("archive: %s: %w", name, err), readArchiveMessage)
		}
		if i == -1 {
			rootInfo = info
		}
		fileInfo = info
		if !info.IsDir() && (!info.Mode().IsRegular() || i < len(parts)-1) {
			return nil, nil, archive.WithMessage(fmt.Errorf("archive: %s: refusing non-regular path %s", name, disk), archiveLinkMessage)
		}
	}
	return rootInfo, fileInfo, nil
}

func (g *DiskFS) openChecked(name string, rootInfo, fileInfo fs.FileInfo) (fs.File, error) {
	root, err := os.OpenRoot(g.root)
	if err != nil {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: %w", name, err), readArchiveMessage)
	}
	defer func() { _ = root.Close() }()
	pinned, err := root.Stat(".")
	if err != nil || !os.SameFile(rootInfo, pinned) {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: root changed during open", name), archiveLinkMessage)
	}
	// Nonblocking prevents a swapped FIFO from stalling before its handle is checked.
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: %w", name, err), readArchiveMessage)
	}
	info, err := f.Stat()
	if err != nil || !os.SameFile(fileInfo, info) || (!info.IsDir() && !info.Mode().IsRegular()) {
		closeErr := f.Close()
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: refusing opened file (stat: %v, close: %v)", name, err, closeErr), archiveLinkMessage)
	}
	return f, nil
}

func (g *DiskFS) ReadFile(name string) ([]byte, error) {
	if g.files == 0 {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: file count exceeded", name), archiveLimitMessage)
	}
	f, err := g.Open(name)
	if err != nil {
		return nil, err
	}
	g.files--
	limit := min(g.fileLimit, g.remaining)
	body, readErr := io.ReadAll(io.LimitReader(f, limit+1))
	closeErr := f.Close()
	g.remaining -= int64(len(body))
	if int64(len(body)) > limit {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: byte limit exceeded", name), archiveLimitMessage)
	}
	if readErr != nil {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: %w", name, readErr), readArchiveMessage)
	}
	if closeErr != nil {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: %w", name, closeErr), readArchiveMessage)
	}
	return body, nil
}

func (g *DiskFS) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := g.Open(name)
	if err != nil {
		return nil, err
	}
	entries, readErr := f.(fs.ReadDirFile).ReadDir(g.entries + 1)
	closeErr := f.Close()
	g.entries -= len(entries)
	if g.entries < 0 {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: directory entry limit exceeded", name), archiveLimitMessage)
	}
	if readErr != nil && readErr != io.EOF {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: %w", name, readErr), readArchiveMessage)
	}
	if closeErr != nil {
		return nil, archive.WithMessage(fmt.Errorf("archive: %s: %w", name, closeErr), readArchiveMessage)
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}

// Check discovers leagues through the same bounded reads used by Import.
func (g *DiskFS) Check() error {
	_, err := readLeagues(g)
	return err
}
