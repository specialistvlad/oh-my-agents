package settings

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// DirName is the workspace folder's name wherever it appears.
const DirName = ".oma"

// DefaultDir is where the workspace lives when nothing overrides it: .oma in
// the user's home directory, so every project on the machine shares one
// workspace rather than scattering a folder per working directory.
//
// It is a function, not a constant, because the answer depends on the user
// the process runs as and cannot be baked in at compile time.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("settings: locate home directory: %w", err)
	}
	return filepath.Join(home, DirName), nil
}

// resolveDir turns a configured directory into an absolute path. An empty
// value means [DefaultDir]; a leading "~" is expanded here rather than left
// to a shell, because the value often arrives from a container's environment
// where nothing has expanded it.
func resolveDir(dir string) (string, error) {
	switch {
	case dir == "":
		var err error
		if dir, err = DefaultDir(); err != nil {
			return "", err
		}
	case dir == "~" || strings.HasPrefix(dir, "~/"):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("settings: expand %q: %w", dir, err)
		}
		dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("settings: resolve %q: %w", dir, err)
	}
	return abs, nil
}

// subdir keeps settings in their own corner of the root, leaving .oma free
// to hold other things later without a migration.
const subdir = "settings"

// ext is the file extension every document is stored under.
const ext = ".json"

// FS keeps each setting in its own file under a root directory.
//
// Nothing is created until the first write, so constructing an FS has no
// side effects and a process that only reads leaves no trace on disk.
//
// The mutex serializes this process against itself. Two processes sharing a
// root can still interleave writes — file locking is not implemented, and
// would be the thing to add before that stops being hypothetical.
type FS struct {
	root string
	mu   sync.RWMutex
}

// NewFS returns a store rooted at dir, which is resolved to an absolute path
// immediately: an empty value becomes [DefaultDir], a leading "~" is
// expanded, and a relative path is bound to the current working directory.
//
// A store that cannot work out where to write says so now rather than at the
// first write, which is the difference between a failed boot and a confusing
// runtime error.
func NewFS(dir string) (*FS, error) {
	root, err := resolveDir(dir)
	if err != nil {
		return nil, err
	}
	return &FS{root: root}, nil
}

// Dir reports the absolute root this store writes under, so a process can log
// what it resolved at boot.
func (s *FS) Dir() string { return s.root }

// Get implements [Reader].
func (s *FS) Get(ctx context.Context, key Key) (Document, error) {
	path, err := s.path(ctx, key)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, err := os.ReadFile(path) //nolint:gosec // path is built from a validated key under a fixed root
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %q", ErrNotFound, key)
	}
	if err != nil {
		return nil, fmt.Errorf("settings: read %q: %w", key, err)
	}
	return doc, nil
}

// Set implements [Writer]. The document is written to a temporary file and
// renamed into place, so a reader sees the old bytes or the new ones and
// never a half-written file.
func (s *FS) Set(ctx context.Context, key Key, doc Document) error {
	path, err := s.path(ctx, key)
	if err != nil {
		return err
	}
	if err := doc.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("settings: create %q: %w", dir, err)
	}
	return writeAtomic(path, doc)
}

// Delete implements [Writer].
func (s *FS) Delete(ctx context.Context, key Key) error {
	path, err := s.path(ctx, key)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	err = os.Remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %q", ErrNotFound, key)
	}
	if err != nil {
		return fmt.Errorf("settings: delete %q: %w", key, err)
	}
	return nil
}

// Keys implements [Lister]. A root that does not exist yet holds nothing,
// which is not an error.
func (s *FS) Keys(ctx context.Context) ([]Key, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	base := filepath.Join(s.root, subdir)
	keys, err := collectKeys(base)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("settings: list %q: %w", base, err)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys, nil
}

// path validates the key and maps it onto a file beneath the root.
func (s *FS) path(ctx context.Context, key Key) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := key.Validate(); err != nil {
		return "", err
	}
	rel := filepath.FromSlash(string(key)) + ext
	return filepath.Join(s.root, subdir, rel), nil
}

// collectKeys walks base and turns every document path back into its key.
func collectKeys(base string) ([]Key, error) {
	var keys []Key
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ext) {
			return nil
		}
		rel, err := filepath.Rel(base, strings.TrimSuffix(path, ext))
		if err != nil {
			return err
		}
		keys = append(keys, Key(filepath.ToSlash(rel)))
		return nil
	})
	return keys, err
}
