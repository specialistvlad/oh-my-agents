// Package fs stores a tracker on the filesystem.
//
// It supplies persistence to [store.Store] rather than reimplementing the
// tracker's rules: the resolution gate, the cycle check and the version
// comparison live in one place and are shared, which is the difference
// between a second store and a second copy of the logic (ADR-0001).
//
// The layout is meant to be read by a person, because that is most of the
// point of keeping it in files at all:
//
//	<dir>/schema/<type-id>.json
//	<dir>/items/<item-id>.json
//	<dir>/comments/<comment-id>.json
//	<dir>/links.json
//	<dir>/events.jsonl
package fs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker/store"
)

// Directory names, one per kind of thing.
const (
	schemaDir   = "schema"
	itemsDir    = "items"
	commentsDir = "comments"
	linksFile   = "links.json"
	eventsFile  = "events.jsonl"
)

// Files is a [store.Persistence] over a directory.
type Files struct{ dir string }

// New returns a tracker stored under dir, holding whatever is already there.
func New(ctx context.Context, dir string, d store.Deps) (*store.Store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("tracker/fs: resolve %q: %w", dir, err)
	}
	d.Persistence = &Files{dir: abs}
	return store.New(ctx, d)
}

// Dir reports where this tracker is stored.
func (f *Files) Dir() string { return f.dir }

// Load implements [store.Persistence]. A directory that does not exist yet
// holds nothing, which is not an error: a project has a tracker from the
// moment it is asked for one, not from the moment someone writes to it.
func (f *Files) Load(ctx context.Context) (store.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return store.Snapshot{}, err
	}
	var held store.Snapshot
	types, err := readAll[tracker.ItemType](filepath.Join(f.dir, schemaDir))
	if err != nil {
		return store.Snapshot{}, err
	}
	held.Schema = tracker.Schema{Types: types}
	if held.Items, err = readAll[tracker.Item](filepath.Join(f.dir, itemsDir)); err != nil {
		return store.Snapshot{}, err
	}
	if held.Comments, err = readAll[tracker.Comment](filepath.Join(f.dir, commentsDir)); err != nil {
		return store.Snapshot{}, err
	}
	if held.Links, err = readList[tracker.Link](filepath.Join(f.dir, linksFile)); err != nil {
		return store.Snapshot{}, err
	}
	if held.Events, err = readLines[tracker.Event](filepath.Join(f.dir, eventsFile)); err != nil {
		return store.Snapshot{}, err
	}
	return held, nil
}

// SaveType implements [store.Persistence].
func (f *Files) SaveType(ctx context.Context, t tracker.ItemType) error {
	return f.write(ctx, schemaDir, string(t.ID), t)
}

// DeleteType implements [store.Persistence].
func (f *Files) DeleteType(ctx context.Context, id tracker.TypeID) error {
	return f.remove(ctx, schemaDir, string(id))
}

// SaveItem implements [store.Persistence].
func (f *Files) SaveItem(ctx context.Context, item tracker.Item) error {
	return f.write(ctx, itemsDir, string(item.ID), item)
}

// DeleteItem implements [store.Persistence].
func (f *Files) DeleteItem(ctx context.Context, id tracker.ItemID) error {
	return f.remove(ctx, itemsDir, string(id))
}

// SaveComment implements [store.Persistence].
func (f *Files) SaveComment(ctx context.Context, c tracker.Comment) error {
	return f.write(ctx, commentsDir, string(c.ID), c)
}

// DeleteComment implements [store.Persistence].
func (f *Files) DeleteComment(ctx context.Context, id tracker.CommentID) error {
	return f.remove(ctx, commentsDir, string(id))
}

// SaveLinks implements [store.Persistence], writing the whole set.
func (f *Files) SaveLinks(ctx context.Context, links []tracker.Link) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	body, err := encode(links)
	return writeAtomic(filepath.Join(f.dir, linksFile), body, err)
}

// AppendEvent implements [store.Persistence].
//
// One JSON object per line, appended. The feed only ever grows, so rewriting
// it whole on every event would make each write cost the history.
func (f *Files) AppendEvent(ctx context.Context, e tracker.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("tracker/fs: encode event %d: %w", e.Seq, err)
	}
	if err := os.MkdirAll(f.dir, 0o755); err != nil {
		return fmt.Errorf("tracker/fs: create %q: %w", f.dir, err)
	}
	path := filepath.Join(f.dir, eventsFile)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // path is built from a fixed root
	if err != nil {
		return fmt.Errorf("tracker/fs: open %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.Write(append(body, '\n')); err != nil {
		return fmt.Errorf("tracker/fs: append event %d: %w", e.Seq, err)
	}
	return file.Sync()
}

func (f *Files) write(ctx context.Context, dir, name string, entity any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if name == "" {
		return errors.New("tracker/fs: refusing to write a file with no name")
	}
	full := filepath.Join(f.dir, dir)
	if err := os.MkdirAll(full, 0o755); err != nil {
		return fmt.Errorf("tracker/fs: create %q: %w", full, err)
	}
	body, err := encode(entity)
	return writeAtomic(filepath.Join(full, name+".json"), body, err)
}

func (f *Files) remove(ctx context.Context, dir, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := os.Remove(filepath.Join(f.dir, dir, name+".json"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("tracker/fs: remove %s/%s: %w", dir, name, err)
	}
	return nil
}
