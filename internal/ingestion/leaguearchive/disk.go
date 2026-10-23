package leaguearchive

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// DiskFS bounds one disk import, including repeated discovery reads. It is not safe for concurrent use.
// Static links are refused; concurrent path replacement requires server-mode safe opening.
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
		return nil, fmt.Errorf("archive: %s: invalid path", name)
	}
	parts := strings.Split(name, "/")
	if name != "." && len(parts) > 3 {
		return nil, fmt.Errorf("archive: %s: directory depth exceeded", name)
	}
	disk := g.root
	for i := -1; i < len(parts); i++ {
		if i >= 0 {
			disk = filepath.Join(disk, parts[i])
		}
		info, err := os.Lstat(disk)
		if err != nil {
			return nil, fmt.Errorf("archive: %s: %w", name, err)
		}
		if !info.IsDir() && (!info.Mode().IsRegular() || i < len(parts)-1) {
			return nil, fmt.Errorf("archive: %s: refusing non-regular path %s", name, disk)
		}
	}
	f, err := os.Open(disk)
	if err != nil {
		return nil, fmt.Errorf("archive: %s: %w", name, err)
	}
	info, err := f.Stat()
	if err != nil || (!info.IsDir() && !info.Mode().IsRegular()) {
		closeErr := f.Close()
		return nil, fmt.Errorf("archive: %s: refusing opened file (stat: %v, close: %v)", name, err, closeErr)
	}
	return f, nil
}

func (g *DiskFS) ReadFile(name string) ([]byte, error) {
	if g.files == 0 {
		return nil, fmt.Errorf("archive: %s: file count exceeded", name)
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
		return nil, fmt.Errorf("archive: %s: byte limit exceeded", name)
	}
	if readErr != nil {
		return nil, fmt.Errorf("archive: %s: %w", name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("archive: %s: %w", name, closeErr)
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
		return nil, fmt.Errorf("archive: %s: directory entry limit exceeded", name)
	}
	if readErr != nil && readErr != io.EOF {
		return nil, fmt.Errorf("archive: %s: %w", name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("archive: %s: %w", name, closeErr)
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}

// Check discovers leagues through the same bounded reads used by Import.
func (g *DiskFS) Check() error {
	_, err := readLeagues(g)
	return err
}
