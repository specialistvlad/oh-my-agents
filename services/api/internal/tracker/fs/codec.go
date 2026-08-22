package fs

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// encode renders an entity indented, because a store kept in files is meant
// to be read and edited by a person.
func encode(entity any) ([]byte, error) {
	body, err := json.MarshalIndent(entity, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("tracker/fs: encode: %w", err)
	}
	return append(body, '\n'), nil
}

// readAll loads every .json file in a directory, in name order so that a
// reload is deterministic. A missing directory holds nothing.
func readAll[T any](dir string) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tracker/fs: read %q: %w", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	out := make([]T, 0, len(names))
	for _, name := range names {
		path := filepath.Join(dir, name)
		body, err := os.ReadFile(path) //nolint:gosec // path is built from a fixed root
		if err != nil {
			return nil, fmt.Errorf("tracker/fs: read %q: %w", path, err)
		}
		var entity T
		if err := json.Unmarshal(body, &entity); err != nil {
			return nil, fmt.Errorf("tracker/fs: %q is not readable: %w", path, err)
		}
		out = append(out, entity)
	}
	return out, nil
}

// readList loads a single file holding a JSON array.
func readList[T any](path string) ([]T, error) {
	body, err := os.ReadFile(path) //nolint:gosec // path is built from a fixed root
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tracker/fs: read %q: %w", path, err)
	}
	var out []T
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("tracker/fs: %q is not readable: %w", path, err)
	}
	return out, nil
}

// readLines loads a file of one JSON object per line.
//
// A truncated final line — a crash mid-append — is refused rather than
// skipped. Silently dropping it would turn a damaged feed into a shorter one
// that looks fine, and the sequence numbers would then lie.
func readLines[T any](path string) ([]T, error) {
	file, err := os.Open(path) //nolint:gosec // path is built from a fixed root
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("tracker/fs: read %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	var out []T
	lines := bufio.NewScanner(file)
	lines.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for n := 1; lines.Scan(); n++ {
		line := strings.TrimSpace(lines.Text())
		if line == "" {
			continue
		}
		var entity T
		if err := json.Unmarshal([]byte(line), &entity); err != nil {
			return nil, fmt.Errorf("tracker/fs: %q line %d is not readable: %w", path, n, err)
		}
		out = append(out, entity)
	}
	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("tracker/fs: read %q: %w", path, err)
	}
	return out, nil
}

// writeAtomic writes through a temporary file in the same directory, so a
// reader sees the previous contents or the new ones and never a partial file.
func writeAtomic(path string, body []byte, encodeErr error) error {
	if encodeErr != nil {
		return encodeErr
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("tracker/fs: create temp for %q: %w", path, err)
	}
	name := tmp.Name()
	fail := func(cause error) error {
		_ = tmp.Close()
		_ = os.Remove(name)
		return cause
	}
	if _, err := tmp.Write(body); err != nil {
		return fail(fmt.Errorf("tracker/fs: write %q: %w", path, err))
	}
	if err := tmp.Sync(); err != nil {
		return fail(fmt.Errorf("tracker/fs: sync %q: %w", path, err))
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("tracker/fs: close %q: %w", path, err)
	}
	if err := os.Chmod(name, 0o644); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("tracker/fs: chmod %q: %w", path, err)
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("tracker/fs: replace %q: %w", path, err)
	}
	return nil
}
