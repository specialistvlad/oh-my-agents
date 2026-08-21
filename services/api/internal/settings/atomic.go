package settings

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeAtomic writes data to path so that a concurrent reader sees either the
// previous contents or the new ones, never a partial file.
//
// The temporary file is created in the destination directory rather than the
// system temp dir, because rename is only atomic within a filesystem and the
// two are not always the same one. It is removed on every failure path, so a
// crash mid-write leaves at worst one stray file and never a corrupt setting.
func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("settings: create temp for %q: %w", path, err)
	}
	name := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		closeAndRemove(tmp, name)
		return fmt.Errorf("settings: write %q: %w", path, err)
	}
	// Flush to disk before the rename: without it a crash can leave the
	// entry pointing at a file whose contents never landed.
	if err := tmp.Sync(); err != nil {
		closeAndRemove(tmp, name)
		return fmt.Errorf("settings: sync %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("settings: close %q: %w", path, err)
	}
	if err := os.Chmod(name, 0o644); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("settings: chmod %q: %w", path, err)
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("settings: replace %q: %w", path, err)
	}
	return nil
}

func closeAndRemove(f *os.File, name string) {
	_ = f.Close()
	_ = os.Remove(name)
}
